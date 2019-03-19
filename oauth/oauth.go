package oauth

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/json-iterator/go"
	"github.com/roshanr83/go-oauth2/util"
	"gopkg.in/gorp.v2"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"time"
)

const (
	PublicPem    = "public.pem"
	PrivatePem   = "private.pem"
	AccessTokenTable  = "oauth_access_tokens"
	RefreshTokenTable = "oauth_refresh_tokens"
	ClientTable       = "oauth_clients"
	BitSize       = 2048
)

// Default Model struct
type Model struct {
	ID        uuid.UUID `db:"id,primarykey"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// Oauth Access Token
type AccessTokens struct {
	Model
	AccessTokenPayload
	Name    string `db:"name"`
	Revoked bool   `db:"revoked"`
}

// Payload to encrypt of access token
type AccessTokenPayload struct {
	UserId    int64 `db:"user_id"`
	ClientId  int64 `db:"client_id"`
	ExpiredAt int64 `db:"expired_at"`
}

// payload to encrypt for refresh token
type RefreshTokenPayload struct {
	AccessTokenId uuid.UUID `db:"access_token_id"`
}

// Oauth Refresh Tokens
type RefreshTokens struct {
	Model
	RefreshTokenPayload
	Revoked bool `db:"revoked"`
}

//Oauth Clients
type Clients struct {
	Model
	UserId   int64  `db:"user_id"`
	Name     string `db:"name"`
	Secret   string `db:"secret"`
	Revoked  bool   `db:"revoked"`
	Redirect string `db:"redirect"`
}

// Store mysql token store
type Store struct {
	clientTable  string
	accessTable  string
	refreshTable string
	db           *gorp.DbMap
	stdout       io.Writer
	ticker       *time.Ticker
}

// NewStore create mysql store instance,
// config mysql configuration,
// tableName table name (default oauth2_token),
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
// db sql.DB
func NewStoreWithDB(db *sql.DB, gcInterval int) *Store {
	store := &Store{
		db:     &gorp.DbMap{Db: db, Dialect: gorp.MySQLDialect{Engine: "InnoDB", Encoding: "UTF8"}},
		accessTable: AccessTokenTable,
		clientTable: ClientTable,
		refreshTable: RefreshTokenTable,
		stdout: os.Stderr,
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

// NewConfig create mysql configuration instance
func NewConfig(dsn string) *Config {
	return &Config{
		DSN:          dsn,
		MaxLifetime:  time.Hour * 2,
		MaxOpenConns: 50,
		MaxIdleConns: 25,
	}
}

// Config mysql configuration
type Config struct {
	DSN          string
	MaxLifetime  time.Duration
	MaxOpenConns int
	MaxIdleConns int
}

// Token response after creating both access token and refresh token
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// NewDefaultStore create mysql store instance
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

func (s *Store) errorf(format string, args ...interface{}) {
	if s.stdout != nil {
		buf := fmt.Sprintf("[OAUTH2-MYSQL-ERROR]: "+format, args...)
		s.stdout.Write([]byte(buf))
	}
}

// create client
func (s *Store) CreateClient(userId int64) (Clients, error) {
	var client Clients
	if userId == 0 {
		return client, errors.New("user id cannot be empty")
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
		priv, pub := util.GenerateKeyPair(BitSize)
		util.SavePEMKey(PrivatePem, priv)
		util.SavePublicPEMKey(PublicPem, pub)
	}
	tokenResp := TokenResponse{}

	//check if valid client
	query := fmt.Sprintf("SELECT * FROM %s WHERE id=? AND secret=? LIMIT 1", s.clientTable)
	var client Clients
	dbErr := s.db.SelectOne(&client, query, info.GetClientID(), info.GetClientSecret())
	if dbErr != nil {
		if sql.ErrNoRows != nil {
			return tokenResp, errors.New("invalid client")
		}
		return tokenResp, dbErr
	}
	if client.ID == uuid.Nil {
		return tokenResp, errors.New("invalid client")
	}

	pubKeyFile, err := ioutil.ReadFile(PublicPem) // just pass the file name
	if err != nil {
		return tokenResp, err
	}
	pubkey := util.BytesToPublicKey(pubKeyFile)
	if err != nil {
		return tokenResp, err
	}

	accessTokenPayload := AccessTokenPayload{}
	accessId, err := uuid.NewRandom()
	refreshId, err := uuid.NewRandom()
	if err != nil {
		return tokenResp, err
	}
	i, err := strconv.ParseInt(info.GetUserID(), 10, 64)
	if err != nil {
		return tokenResp, err
	}
	cid, err := strconv.ParseInt(info.GetClientID(), 10, 64)
	if err != nil {
		return tokenResp, err
	}
	accessTokenPayload.UserId = i
	accessTokenPayload.ClientId = cid
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
	accessToken, err := util.EncryptWithPublicKey(accessByte.Bytes(), pubkey)
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

	refToken, err := util.EncryptWithPublicKey(refreshTokenByte.Bytes(), pubkey)
	tokenResp.RefreshToken = refToken
	if err != nil {
		return tokenResp, err
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

// GetByAccess use the access token for token information data
func (s *Store) GetByAccess(access string) (*AccessTokens, error) {
	accessToken, err := decryptAccessToken(access)
	if err != nil {
		return nil, err
	}
	currentTime := time.Now().Unix()
	if accessToken.ExpiredAt < currentTime {
		return nil, errors.New("access token has expired")
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
		return nil, errors.New("access token already revoked")
	}
	return &item, nil
}

// GetByRefresh use the refresh token for token information data
func (s *Store) GetByRefresh(refresh string) (*RefreshTokens, error) {
	accessToken, err := decryptRefreshToken(refresh)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT * FROM %s WHERE access_token_id=? LIMIT 1", s.refreshTable)
	var item RefreshTokens
	dbErr := s.db.SelectOne(&item, query, accessToken.AccessTokenId)
	if dbErr != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, dbErr
	}
	if item.Revoked == true {
		return nil, errors.New("refresh token already revoked")
	}

	updateQuery := fmt.Sprintf("UPDATE %s SET `revoke`=? WHERE access_token_id IN (?)", s.refreshTable)
	_, updateErr := s.db.Exec(updateQuery, 1, accessToken.AccessTokenId)
	if updateErr != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, updateErr
	}
	return &item, nil
}


// Clear all token related to user
func (s *Store) ClearByAccessToken(info TokenInfo) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE user_id=?", s.accessTable)
	_, err := s.db.Exec(query, info.GetUserID())
	if err != nil && err == sql.ErrNoRows {
		return nil
	}
	return err
}

// revoke from RefreshToken
func (s *Store) RevokeRefreshToken(accessTokenId string) error {
	query := fmt.Sprintf("UPDATE %s SET `revoke`=? WHERE accessTokenId IN (?)", s.refreshTable)
	_, err := s.db.Exec(query, 1, accessTokenId)
	if err != nil && err == sql.ErrNoRows {
		return nil
	}
	return err
}

// revoke from accessToken
func (s *Store) RevokeByAccessTokens(userId string) error {
	query := fmt.Sprintf("UPDATE %s SET `revoke`=? WHERE user_id IN (?)", s.accessTable)
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
	prikey := util.BytesToPrivateKey(privKey)
	if err != nil {
		return &tm, err
	}
	dec, err := util.DecryptWithPrivateKey(token, prikey)
	jsoniter.Unmarshal([]byte(dec), &tm)
	if tm.UserId == 0 {
		return &tm, errors.New("invalid access token")
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
	prikey := util.BytesToPrivateKey(privKey)
	if err != nil {
		return &tm, err
	}
	dec, err := util.DecryptWithPrivateKey(token, prikey)
	jsoniter.Unmarshal([]byte(dec), &tm)
	if tm.AccessTokenId == uuid.Nil {
		return &tm, errors.New("invalid refresh token")
	}
	return &tm, nil
}
