package golang_oauth

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql" //mysql driver for NewStore
	"github.com/gobeam/golang-oauth/internal"
	"github.com/gobeam/golang-oauth/util"
	"github.com/google/uuid"
	"github.com/json-iterator/go"
	"gopkg.in/gorp.v2"
	"io"
	"io/ioutil"
	"os"
	"time"
)

// Store mysql token store model
type Store struct {
	clientTable  string
	accessTable  string
	refreshTable string
	db           *gorp.DbMap
	stdout       io.Writer
	ticker       *time.Ticker
}

// Config mysql configuration
type Config struct {
	DSN          string
	MaxLifetime  time.Duration
	MaxOpenConns int
	MaxIdleConns int
}

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
		accessTable:  util.AccessTokenTable,
		clientTable:  util.ClientTable,
		refreshTable: util.RefreshTokenTable,
		stdout:       os.Stderr,
	}

	store.db.AddTableWithName(internal.AccessTokens{}, store.accessTable)
	store.db.AddTableWithName(internal.Clients{}, store.clientTable)
	store.db.AddTableWithName(internal.RefreshTokens{}, store.refreshTable)

	err := store.db.CreateTablesIfNotExists()
	if err != nil {
		panic(err)
	}
	_ = store.db.CreateIndex()

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
	_ = s.db.Db.Close()
}

func (s *Store) gc() {
	for range s.ticker.C {
		s.clean()
	}
}

// clean is method to clean expired and revoked access token and refresh token during creation of mysql store instance
func (s *Store) clean() {
	now := time.Now().Unix()
	_, accessErr := s.db.Exec(fmt.Sprintf("DELETE FROM %s WHERE (revoked='1')", s.accessTable))
	_, refreshErr := s.db.Exec(fmt.Sprintf("DELETE FROM %s WHERE revoked='1'", s.refreshTable), now)
	if accessErr != nil {
		s.errorf(accessErr.Error())
	}
	if refreshErr != nil {
		s.errorf(refreshErr.Error())
	}
}

// errorf logs error
func (s *Store) errorf(format string, args ...interface{}) {
	if s.stdout != nil {
		buf := fmt.Sprintf("[OAUTH2-MYSQL-ERROR]: "+format, args...)
		_, _ = s.stdout.Write([]byte(buf))
	}
}

// CreateClient creates new client,
// userId user's id who created the client
func (s *Store) CreateClient(userId int64) (internal.Clients, error) {
	var client internal.Clients
	if userId == 0 {
		return client, errors.New(util.EmptyUserID)
	}
	client.ID = uuid.New()
	client.Secret = util.RandomKey(20)
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
func (s *Store) Create(info internal.TokenInfo) (internal.TokenResponse, error) {
	_, publicPemNotExistserr := os.Stat(util.PublicPem)
	_, privatePemNotExistserr := os.Stat(util.PublicPem)

	// check if Public and Private key exists File is present
	if os.IsNotExist(publicPemNotExistserr) || os.IsNotExist(privatePemNotExistserr) {
		priv, pub := util.GenerateKeyPair(util.BitSize)
		util.SavePEMKey(util.PrivatePem, priv)
		util.SavePublicPEMKey(util.PublicPem, pub)
	}

	tokenResp := internal.TokenResponse{}
	if info.GetUserID() == 0 {
		return tokenResp, errors.New(util.EmptyUserID)
	}

	//check if valid client
	query := fmt.Sprintf("SELECT * FROM %s WHERE id=? AND secret=? LIMIT 1", s.clientTable)
	var client internal.Clients
	err := s.db.SelectOne(&client, query, info.GetClientID(), info.GetClientSecret())
	if err != nil {
		return tokenResp, errors.New(util.InvalidClient)
	}
	if client.ID == uuid.Nil {
		return tokenResp, errors.New(util.InvalidClient)
	}

	//create rsa pub
	pubKeyFile, err := ioutil.ReadFile(util.PublicPem)
	if err != nil {
		return tokenResp, err
	}
	pubkey := util.BytesToPublicKey(pubKeyFile)
	accessTokenPayload := internal.AccessTokenPayload{}
	accessId := uuid.New()
	accessTokenPayload.UserId = info.GetUserID()
	accessTokenPayload.ClientId = info.GetClientID()
	accessTokenPayload.ExpiredAt = info.GetAccessCreateAt().Add(info.GetAccessExpiresIn()).Unix()
	tokenResp.ExpiredAt = info.GetAccessCreateAt().Add(info.GetAccessExpiresIn()).Unix()
	oauthAccess := &internal.AccessTokens{
		internal.Model{
			ID:        accessId,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		accessTokenPayload,
		"",
		false,
	}
	accessByte := new(bytes.Buffer)
	_ = json.NewEncoder(accessByte).Encode(accessTokenPayload)
	accessToken, err := util.EncryptWithPublicKey(accessByte.Bytes(), pubkey)
	if err != nil {
		return tokenResp, err
	}
	tokenResp.AccessToken = accessToken
	tokenResp.ExpiredAt = accessTokenPayload.ExpiredAt

	// set refresh
	refreshTokenPayload := internal.RefreshTokenPayload{}
	refreshTokenPayload.AccessTokenId = accessId
	refreshToken := &internal.RefreshTokens{
		internal.Model{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		refreshTokenPayload,
		false,
	}

	refreshTokenByte := new(bytes.Buffer)
	_ = json.NewEncoder(refreshTokenByte).Encode(refreshTokenPayload)

	refToken, err := util.EncryptWithPublicKey(refreshTokenByte.Bytes(), pubkey)
	tokenResp.RefreshToken = refToken
	if err != nil {
		return tokenResp, err
	}

	//revoke all old access tokens
	//updateQuery := fmt.Sprintf("UPDATE %s SET `revoked`=? WHERE user_id = ?", s.accessTable)
	//_, updateErr := s.db.Exec(updateQuery, 1, info.GetUserID())
	//if updateErr != nil {
	//	return tokenResp, updateErr
	//}

	accessErr := s.db.Insert(oauthAccess)
	if accessErr != nil {
		return tokenResp, accessErr
	}

	refErr := s.db.Insert(refreshToken)
	if refErr != nil {
		return tokenResp, refErr
	}
	return tokenResp, nil
}

// GetByAccess use the access token for token information data,
// access Access token string
func (s *Store) GetByAccess(access string) (*internal.AccessTokens, error) {
	accessToken, err := decryptAccessToken(access)
	if err != nil {
		return nil, err
	}
	currentTime := time.Now().Unix()
	if accessToken.ExpiredAt < currentTime {
		return nil, errors.New(util.AccessTokenExpired)
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE user_id=? AND expired_at=? LIMIT 1", s.accessTable)
	var item internal.AccessTokens
	err = s.db.SelectOne(&item, query, accessToken.UserId, accessToken.ExpiredAt)
	if err != nil {
		return nil, errors.New(util.InvalidAccessToken)
	}
	if item.Revoked == true {
		return nil, errors.New(util.AccessTokenRevoked)
	}
	return &item, nil
}

// GetByRefresh use the refresh token for token information data,
// refresh Refresh token string
func (s *Store) GetByRefresh(refresh string) (*internal.AccessTokens, error) {
	accessToken, err := decryptRefreshToken(refresh)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT * FROM %s WHERE access_token_id=? LIMIT 1", s.refreshTable)
	var refreshToken internal.RefreshTokens
	err = s.db.SelectOne(&refreshToken, query, accessToken.AccessTokenId)
	if err != nil {
		return nil, errors.New(util.InvalidRefreshToken)
	}
	if refreshToken.Revoked == true {
		return nil, errors.New(util.RefreshTokenRevoked)
	}

	//check if associated access token is revoked or not
	checkAccessTokenquery := fmt.Sprintf("SELECT * FROM %s WHERE id=? LIMIT 1", s.accessTable)
	var accessTokenData internal.AccessTokens
	err = s.db.SelectOne(&accessTokenData, checkAccessTokenquery, accessToken.AccessTokenId)
	if err != nil {
		return nil, errors.New(util.InvalidRefreshToken)
	}
	if accessTokenData.Revoked == true {
		return nil, errors.New(util.InvalidRefreshToken)
	}

	// revoke refresh token after one time use
	updateQuery := fmt.Sprintf("UPDATE %s SET `revoked`=? WHERE access_token_id IN (?)", s.refreshTable)
	_, err = s.db.Exec(updateQuery, 1, accessToken.AccessTokenId)
	if err != nil {
		return nil, err
	}

	// revoke associated access token after use
	updateAccessTokenQuery := fmt.Sprintf("UPDATE %s SET `revoked`=? WHERE id=?", s.accessTable)
	_, err = s.db.Exec(updateAccessTokenQuery, 1, accessToken.AccessTokenId)
	if err != nil {
		return nil, err
	}

	return &accessTokenData, nil
}

// ClearByAccessToken clears all token related to user,
// userId id of user whose access token needs to be cleared
func (s *Store) ClearByAccessToken(userId int64) error {
	checkAccessTokenquery := fmt.Sprintf("SELECT * FROM %s WHERE user_id=? ", s.accessTable)
	var accessTokenData []internal.AccessTokens
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

// RevokeRefreshToken revokes token from RefreshToken,
func (s *Store) RevokeRefreshToken(accessTokenId string) error {
	query := fmt.Sprintf("UPDATE %s SET `revoked`=? WHERE access_token_id IN (?)", s.refreshTable)
	_, err := s.db.Exec(query, 1, accessTokenId)
	if err != nil && err == sql.ErrNoRows {
		return nil
	}
	return err
}

// RevokeByAccessTokens revokes token from accessToken
func (s *Store) RevokeByAccessTokens(userId int64) error {
	query := fmt.Sprintf("UPDATE %s SET `revoked`=? WHERE user_id IN (?)", s.accessTable)
	_, err := s.db.Exec(query, 1, userId)
	if err != nil && err == sql.ErrNoRows {
		return nil
	}
	return err
}

// decryptAccessToken decrypts given access token
func decryptAccessToken(token string) (*internal.AccessTokenPayload, error) {
	var tm internal.AccessTokenPayload
	privKey, err := ioutil.ReadFile(util.PrivatePem)
	if err != nil {
		return &tm, err
	}
	prikey := util.BytesToPrivateKey(privKey)
	dec, err := util.DecryptWithPrivateKey(token, prikey)
	if err != nil {
		return &tm, err
	}
	_ = jsoniter.Unmarshal([]byte(dec), &tm)
	if tm.UserId == 0 {
		return &tm, errors.New(util.InvalidAccessToken)
	}
	return &tm, nil
}

// decryptRefreshToken decrypts given refresh token
func decryptRefreshToken(token string) (*internal.RefreshTokenPayload, error) {
	var tm internal.RefreshTokenPayload
	privKey, err := ioutil.ReadFile(util.PrivatePem)
	if err != nil {
		return &tm, err
	}
	prikey := util.BytesToPrivateKey(privKey)
	decypher, err := util.DecryptWithPrivateKey(token, prikey)
	if err != nil {
		return &tm, err
	}
	_ = jsoniter.Unmarshal([]byte(decypher), &tm)
	if tm.AccessTokenId == uuid.Nil {
		return &tm, errors.New(util.InvalidRefreshToken)
	}
	return &tm, nil
}
