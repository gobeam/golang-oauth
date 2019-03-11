package main

import (
	"database/sql"
	"fmt"
	"github.com/json-iterator/go"
	"gopkg.in/gorp.v2"
	"gopkg.in/oauth2.v3"
	"gopkg.in/oauth2.v3/models"
	"io"
	"os"
	"strconv"
	"time"
)

// Default Model struct
type Model struct {
	ID        int64     `db:"id,primarykey,autoincrement"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"DEFAULT:current_timestamp"`
}

// Oauth Access Token
type OauthAccessTokens struct {
	Model
	UserId    int64  `db:"user_id"`
	ClientId  int64  `db:"client_id"`
	Name      string `db:"name"`
	Revoked   bool   `db:"revoked"`
	ExpiredAt int64  `db:"expired_at"`
}

// Oauth Refresh Tokens
type OauthRefreshTokens struct {
	Model
	AccessTokenId int64 `db:"access_token_id"`
	Revoked       bool  `db:"revoked"`
	ExpiredAt     int64 `db:"expired_at"`
}

//Oauth Clients
type OauthClients struct {
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

// StoreItem data item
type StoreItem struct {
	ID        int64  `db:"id,primarykey,autoincrement"`
	ExpiredAt int64  `db:"expired_at"`
	UserId    string `db:"user_id"`
	Revoke    bool   `db:"revoke"`
	Code      string `db:"code,size:255"`
	Access    string `db:"access,size:255"`
	Refresh   string `db:"refresh,size:255"`
	Data      string `db:"data,size:2048"`
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
		db:           &gorp.DbMap{Db: db, Dialect: gorp.MySQLDialect{Encoding: "UTF8", Engine: "MyISAM"}},
		accessTable:  "oauth_access_tokens",
		clientTable:  "oauth_clients",
		refreshTable: "oauth_refresh_tokens",
		stdout:       os.Stderr,
	}

	store.db.AddTableWithName(OauthAccessTokens{}, store.accessTable)
	store.db.AddTableWithName(OauthClients{}, store.clientTable)
	store.db.AddTableWithName(OauthRefreshTokens{}, store.refreshTable)

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
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE expired_at<=? OR (code='' AND access='' AND refresh='')", s.accessTable)
	n, err := s.db.SelectInt(query, now)
	if err != nil || n == 0 {
		if err != nil {
			s.errorf(err.Error())
		}
		return
	}

	_, err = s.db.Exec(fmt.Sprintf("DELETE FROM %s WHERE expired_at<=? OR (code='' AND access='' AND refresh='')", s.accessTable), now)
	if err != nil {
		s.errorf(err.Error())
	}
}

func (s *Store) errorf(format string, args ...interface{}) {
	if s.stdout != nil {
		buf := fmt.Sprintf("[OAUTH2-MYSQL-ERROR]: "+format, args...)
		s.stdout.Write([]byte(buf))
	}
}

// Create create and store the new token information
func (s *Store) Create(info oauth2.TokenInfo) error {
	//buf, _ := jsoniter.Marshal(info)
	item := &OauthAccessTokens{}
	i, err := strconv.ParseInt(info.GetUserID(), 10, 64)
	if err != nil {

	}
	item.UserId = i
	cid, err := strconv.ParseInt(info.GetClientID(), 10, 64)
	if err != nil {

	}
	item.ClientId = cid
	item.ExpiredAt = info.GetAccessCreateAt().Add(info.GetAccessExpiresIn()).Unix()
	if refresh := info.GetRefresh(); refresh != "" {
		item.Refresh = info.GetRefresh()
		item.ExpiredAt = info.GetRefreshCreateAt().Add(info.GetRefreshExpiresIn()).Unix()
	}

	return s.db.Insert(item)
}

func (s *Store) ClearAccessToken(info oauth2.TokenInfo) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE user_id=?", s.accessTable)
	_, err := s.db.Exec(query, info.GetUserID())
	if err != nil && err == sql.ErrNoRows {
		return nil
	}
	return err
}

func (s *Store) RevokeAccessTokens(id string) error {
	query := fmt.Sprintf("UPDATE %s SET `revoke`=? WHERE user_id IN (?)", s.accessTable)
	_, err := s.db.Exec(query, 1, id)
	if err != nil && err == sql.ErrNoRows {
		return nil
	}
	return err
}

// RemoveByCode delete the authorization code
func (s *Store) RemoveByCode(id string) error {
	query := fmt.Sprintf("UPDATE %s SET code='' WHERE code=? LIMIT 1", s.accessTable)
	_, err := s.db.Exec(query, id)
	if err != nil && err == sql.ErrNoRows {
		return nil
	}
	return err
}

// RemoveByAccess use the access token to delete the token information
func (s *Store) RemoveByAccess(access string) error {
	query := fmt.Sprintf("UPDATE %s SET access='', refresh='' WHERE access=? LIMIT 1", s.accessTable)
	_, err := s.db.Exec(query, access)
	if err != nil && err == sql.ErrNoRows {
		return nil
	}
	return err
}

// RemoveByRefresh use the refresh token to delete the token information
func (s *Store) RemoveByRefresh(refresh string) error {
	query := fmt.Sprintf("UPDATE %s SET refresh='', access='' WHERE refresh=? LIMIT 1", s.accessTable)
	_, err := s.db.Exec(query, refresh)
	if err != nil && err == sql.ErrNoRows {
		return nil
	}
	return err
}

func (s *Store) toTokenInfo(data string) oauth2.TokenInfo {
	var tm models.Token
	jsoniter.Unmarshal([]byte(data), &tm)
	return &tm
}

// GetByCode use the authorization code for token information data
func (s *Store) GetByCode(code string) (oauth2.TokenInfo, error) {
	if code == "" {
		return nil, nil
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE code=? LIMIT 1", s.accessTable)
	var item StoreItem
	err := s.db.SelectOne(&item, query, code)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return s.toTokenInfo(item.Data), nil
}

// GetByAccess use the access token for token information data
func (s *Store) GetByAccess(access string) (oauth2.TokenInfo, error) {
	if access == "" {
		return nil, nil
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE access=? LIMIT 1", s.accessTable)
	var item StoreItem
	err := s.db.SelectOne(&item, query, access)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return s.toTokenInfo(item.Data), nil
}

// GetByRefresh use the refresh token for token information data
func (s *Store) GetByRefresh(refresh string) (oauth2.TokenInfo, error) {
	if refresh == "" {
		return nil, nil
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE refresh=? LIMIT 1", s.accessTable)
	var item StoreItem
	err := s.db.SelectOne(&item, query, refresh)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return s.toTokenInfo(item.Data), nil
}
