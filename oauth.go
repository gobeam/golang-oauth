package go_oauth2

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/json-iterator/go"
	"gopkg.in/gorp.v2"
	"io/ioutil"
	"os"
	"time"
)

// NewStore create mysql store instance,
// config mysql configuration,
// GC time interval (in seconds, default 600)
func NewStore(config *Config, gcInterval int) *Store {
	db, err := sql.Open("mysql", config.DSN)
	if err != nil {
		panic(err)
	}

	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.MaxLifetime)
	return NewStoreWithDB(db, gcInterval)
}

// NewStoreWithDB create mysql store instance,
// db sql.DB,
// GC time interval (in seconds, default 600)
func NewStoreWithDB(db *sql.DB, gcInterval int) *Store {
	store := &Store{
		db:           &gorp.DbMap{Db: db, Dialect: gorp.MySQLDialect{Engine: "InnoDB", Encoding: "UTF8"}},
		accessTable:  AccessTokenTable,
		clientTable:  ClientTable,
		refreshTable: RefreshTokenTable,
		stdout:       os.Stderr,
	}

	store.db.AddTableWithName(AccessTokens{}, store.accessTable)
	store.db.AddTableWithName(Clients{}, store.clientTable)
	store.db.AddTableWithName(RefreshTokens{}, store.refreshTable)

	err := store.db.CreateTablesIfNotExists()
	if err != nil {
		panic(err)
	}
	store.db.CreateIndex()

	interval := 600
	if gcInterval > 0 {
		interval = gcInterval
	}
	store.ticker = time.NewTicker(time.Second * time.Duration(interval))
	go store.gc()
	return store
}

// NewConfig create mysql configuration instance,
// dsn mysql database credential
func NewConfig(dsn string) *Config {
	return &Config{
		DSN:          dsn,
		MaxLifetime:  time.Hour * 2,
		MaxOpenConns: 50,
		MaxIdleConns: 25,
	}
}

// NewDefaultStore create mysql store instance,
// config mysql configuration,
func NewDefaultStore(config *Config) *Store {
	return NewStore(config, 0)
}

// Close close the store
func (s *Store) Close() {
	s.ticker.Stop()
	s.db.Db.Close()
}

func (s *Store) gc() {
	for range s.ticker.C {
		s.clean()
	}
}

// Method to clean expired and revoked access token and refresh token during creation of mysql store instance
func (s *Store) clean() {
	now := time.Now().Unix()
	_, accessErr := s.db.Exec(fmt.Sprintf("DELETE FROM %s WHERE expired_at<=? OR (revoked='1')", s.accessTable), now)
	_, refreshErr := s.db.Exec(fmt.Sprintf("DELETE FROM %s WHERE revoked='1'", s.refreshTable), now)
	if accessErr != nil {
		s.errorf(accessErr.Error())
	}
	if refreshErr != nil {
		s.errorf(refreshErr.Error())
	}
}

// Log error
func (s *Store) errorf(format string, args ...interface{}) {
	if s.stdout != nil {
		buf := fmt.Sprintf("[OAUTH2-MYSQL-ERROR]: "+format, args...)
		s.stdout.Write([]byte(buf))
	}
}

// create client,
// userId user's id who created the client
func (s *Store) CreateClient(userId int64) (Clients, error) {
	var client Clients
	if userId == 0 {
		return client, errors.New(EmptyUserID)
	}
	client.ID = uuid.New()
	client.Secret = RandomKey(20)
	client.UserId = userId
	client.CreatedAt = time.Now()
	client.UpdatedAt = time.Now()
	err := s.db.Insert(&client)
	if err != nil {
		return client, err
	}
	return client, nil
}

// Create create and store the new token information
func (s *Store) Create(info TokenInfo) (TokenResponse, error) {

	var publicPemNotExist bool
	var privatePemNotExist bool
	// check if Public and Private key exists File is present
	if _, err := os.Stat(PublicPem); os.IsNotExist(err) {
		publicPemNotExist = true
	}
	if _, err := os.Stat(PrivatePem); os.IsNotExist(err) {
		privatePemNotExist = true
	}
	if publicPemNotExist || privatePemNotExist {
		priv, pub := GenerateKeyPair(BitSize)
		SavePEMKey(PrivatePem, priv)
		SavePublicPEMKey(PublicPem, pub)
	}
	tokenResp := TokenResponse{}
	if info.GetUserID() == 0 {
		return tokenResp, errors.New(EmptyUserID)
	}

	//check if valid client
	query := fmt.Sprintf("SELECT * FROM %s WHERE id=? AND secret=? LIMIT 1", s.clientTable)
	var client Clients
	dbErr := s.db.SelectOne(&client, query, info.GetClientID(), info.GetClientSecret())
	if dbErr != nil {
		if sql.ErrNoRows != nil {
			return tokenResp, errors.New(InvalidClient)
		}
		return tokenResp, dbErr
	}
	if client.ID == uuid.Nil {
		return tokenResp, errors.New(InvalidClient)
	}

	//create rsa pub
	pubKeyFile, err := ioutil.ReadFile(PublicPem)
	if err != nil {
		return tokenResp, err
	}
	pubkey := BytesToPublicKey(pubKeyFile)
	if err != nil {
		return tokenResp, err
	}

	accessTokenPayload := AccessTokenPayload{}
	accessId, err := uuid.NewRandom()
	refreshId, err := uuid.NewRandom()
	if err != nil {
		return tokenResp, err
	}
	accessTokenPayload.UserId = info.GetUserID()
	accessTokenPayload.ClientId = info.GetClientID()
	accessTokenPayload.ExpiredAt = info.GetAccessCreateAt().Add(info.GetAccessExpiresIn()).Unix()
	oauthAccess := &AccessTokens{
		Model{
			ID:        accessId,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		accessTokenPayload,
		"",
		false,
	}
	accessByte := new(bytes.Buffer)
	json.NewEncoder(accessByte).Encode(accessTokenPayload)
	accessToken, err := EncryptWithPublicKey(accessByte.Bytes(), pubkey)
	if err != nil {
		return tokenResp, err
	}
	tokenResp.AccessToken = accessToken

	// set refresh
	refreshTokenPayload := RefreshTokenPayload{}
	refreshTokenPayload.AccessTokenId = accessId
	refreshToken := &RefreshTokens{
		Model{
			ID:        refreshId,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		refreshTokenPayload,
		false,
	}

	refreshTokenByte := new(bytes.Buffer)
	json.NewEncoder(refreshTokenByte).Encode(refreshTokenPayload)

	refToken, err := EncryptWithPublicKey(refreshTokenByte.Bytes(), pubkey)
	tokenResp.RefreshToken = refToken
	if err != nil {
		return tokenResp, err
	}

	//revoke all old access tokens
	updateQuery := fmt.Sprintf("UPDATE %s SET `revoked`=? WHERE user_id = ?", s.accessTable)
	_, updateErr := s.db.Exec(updateQuery, 1, info.GetUserID())
	if updateErr != nil {
		if err == sql.ErrNoRows {
			return tokenResp, err
		}
		return tokenResp, updateErr
	}

	accessErr := s.db.Insert(oauthAccess)
	if accessErr != nil {
		return tokenResp, accessErr
	}

	refErr := s.db.Insert(refreshToken)
	if accessErr != nil {
		return tokenResp, refErr
	}
	return tokenResp, nil
}

// GetByAccess use the access token for token information data,
// access Access token string
func (s *Store) GetByAccess(access string) (*AccessTokens, error) {
	accessToken, err := decryptAccessToken(access)
	if err != nil {
		return nil, err
	}
	currentTime := time.Now().Unix()
	if accessToken.ExpiredAt < currentTime {
		return nil, errors.New(AccessTokenExpired)
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE user_id=? AND expired_at=? LIMIT 1", s.accessTable)
	var item AccessTokens
	dbErr := s.db.SelectOne(&item, query, accessToken.UserId, accessToken.ExpiredAt)
	if dbErr != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, dbErr
	}
	if item.Revoked == true {
		return nil, errors.New(AccessTokenRevoked)
	}
	return &item, nil
}

// GetByRefresh use the refresh token for token information data,
// refresh Refresh token string
func (s *Store) GetByRefresh(refresh string) (*RefreshTokens, error) {
	accessToken, err := decryptRefreshToken(refresh)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT * FROM %s WHERE access_token_id=? LIMIT 1", s.refreshTable)
	var refreshToken RefreshTokens
	dbErr := s.db.SelectOne(&refreshToken, query, accessToken.AccessTokenId)
	if dbErr != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, dbErr
	}
	if refreshToken.Revoked == true {
		return nil, errors.New(RefreshTokenRevoked)
	}

	//check if associated access token is revoked or not
	checkAccessTokenquery := fmt.Sprintf("SELECT * FROM %s WHERE id=? LIMIT 1", s.accessTable)
	var accessTokenData AccessTokens
	findErr := s.db.SelectOne(&accessTokenData, checkAccessTokenquery, accessToken.AccessTokenId)
	if findErr != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New(InvalidRefreshToken)
		}
		return nil, dbErr
	}
	if accessTokenData.Revoked == true {
		return nil, errors.New(InvalidRefreshToken)
	}

	// revoke refresh token after one time use
	updateQuery := fmt.Sprintf("UPDATE %s SET `revoked`=? WHERE access_token_id IN (?)", s.refreshTable)
	_, updateErr := s.db.Exec(updateQuery, 1, accessToken.AccessTokenId)
	if updateErr != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, updateErr
	}

	// revoke associated access token after use
	updateAccessTokenQuery := fmt.Sprintf("UPDATE %s SET `revoked`=? WHERE id=?", s.accessTable)
	_, updateAccessErr := s.db.Exec(updateAccessTokenQuery, 1, accessToken.AccessTokenId)
	if updateAccessErr != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, updateAccessErr
	}

	return &refreshToken, nil
}

// Clear all token related to user,
// userId id of user whose access token needs to be cleared
func (s *Store) ClearByAccessToken(userId int64) error {
	checkAccessTokenquery := fmt.Sprintf("SELECT * FROM %s WHERE user_id=? ", s.accessTable)
	var accessTokenData []AccessTokens
	_, err := s.db.Select(&accessTokenData, checkAccessTokenquery, userId)
	if err != nil {
		return err
	}

	//delete all related refreshtoken
	for _, value := range accessTokenData {
		query := fmt.Sprintf("DELETE FROM %s WHERE access_token_id=?", s.refreshTable)
		_, err := s.db.Exec(query, value.ID)
		if err != nil {
			return err
		}
	}

	//delete all access token related to user
	query := fmt.Sprintf("DELETE FROM %s WHERE user_id=?", s.accessTable)
	_, err = s.db.Exec(query, userId)
	if err != nil && err == sql.ErrNoRows {
		return nil
	}
	return err
}

// revoke from RefreshToken,
func (s *Store) RevokeRefreshToken(accessTokenId string) error {
	query := fmt.Sprintf("UPDATE %s SET `revoked`=? WHERE access_token_id IN (?)", s.refreshTable)
	_, err := s.db.Exec(query, 1, accessTokenId)
	if err != nil && err == sql.ErrNoRows {
		return nil
	}
	return err
}

// revoke from accessToken
func (s *Store) RevokeByAccessTokens(userId int64) error {
	query := fmt.Sprintf("UPDATE %s SET `revoked`=? WHERE user_id IN (?)", s.accessTable)
	_, err := s.db.Exec(query, 1, userId)
	if err != nil && err == sql.ErrNoRows {
		return nil
	}
	return err
}

//Decrypt Access Token
func decryptAccessToken(token string) (*AccessTokenPayload, error) {
	var tm AccessTokenPayload
	privKey, err := ioutil.ReadFile(PrivatePem)
	if err != nil {
		return &tm, err
	}
	prikey := BytesToPrivateKey(privKey)
	if err != nil {
		return &tm, err
	}
	dec, err := DecryptWithPrivateKey(token, prikey)
	jsoniter.Unmarshal([]byte(dec), &tm)
	if tm.UserId == 0 {
		return &tm, errors.New(InvalidAccessToken)
	}
	return &tm, nil
}

// Decrypt Refresh Token
func decryptRefreshToken(token string) (*RefreshTokenPayload, error) {
	var tm RefreshTokenPayload
	privKey, err := ioutil.ReadFile(PrivatePem)
	if err != nil {
		return &tm, err
	}
	prikey := BytesToPrivateKey(privKey)
	if err != nil {
		return &tm, err
	}
	dec, err := DecryptWithPrivateKey(token, prikey)
	jsoniter.Unmarshal([]byte(dec), &tm)
	if tm.AccessTokenId == uuid.Nil {
		return &tm, errors.New(InvalidRefreshToken)
	}
	return &tm, nil
}
