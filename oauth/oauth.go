package oauth

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/json-iterator/go"
	"github.com/roshanr83/go-oauth2/util"
	"gopkg.in/gorp.v2"
	"gopkg.in/oauth2.v3"
	"gopkg.in/oauth2.v3/models"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"time"
)

// Default Model struct
type Model struct {
	ID        uuid.UUID     `db:"id,primarykey"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// Oauth Access Token
type AccessTokens struct {
	Model
	UserId    int64  `db:"user_id"`
	ClientId  int64  `db:"client_id"`
	Name      string `db:"name"`
	Revoked   bool   `db:"revoked"`
	ExpiredAt int64  `db:"expired_at"`
}

// Oauth Refresh Tokens
type RefreshTokens struct {
	Model
	AccessTokenId uuid.UUID `db:"access_token_id"`
	Revoked       bool  `db:"revoked"`
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
		db:           &gorp.DbMap{Db: db, Dialect: gorp.MySQLDialect{Engine: "InnoDB", Encoding: "UTF8"}},
		accessTable:  "oauth_access_tokens",
		clientTable:  "oauth_clients",
		refreshTable: "oauth_refresh_tokens",
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

	pubKey, err := ioutil.ReadFile("public.pem") // just pass the file name
	if err != nil {
		fmt.Print(err)
	}
	pkey := util.BytesToPublicKey(pubKey)
	if err != nil {
		fmt.Print(err)
	}
	msg := []byte("hello man")
	data := util.EncryptWithPublicKey(msg, pkey)
	if err != nil {
		fmt.Print(err)
	}

	fmt.Println(fmt.Sprintf("%s", data))

	oauthAccess := &AccessTokens{}
	accessId, err := uuid.NewRandom()
	refreshId, err := uuid.NewRandom()
	if err != nil {
		s.errorf(err.Error())
	}
	oauthAccess.ID = accessId
	i, err := strconv.ParseInt(info.GetUserID(), 10, 64)
	if err != nil {

	}
	oauthAccess.UserId = i
	cid, err := strconv.ParseInt(info.GetClientID(), 10, 64)
	if err != nil {

	}
	oauthAccess.ClientId = cid
	oauthAccess.ExpiredAt = info.GetAccessCreateAt().Add(info.GetAccessExpiresIn()).Unix()
	oauthAccess.CreatedAt = time.Now()
	oauthAccess.UpdatedAt = time.Now()

	refreshToken := &RefreshTokens{}

	if refresh := info.GetRefresh(); refresh != "" {
		refreshToken.ID = refreshId
		refreshToken.AccessTokenId = accessId
		refreshToken.CreatedAt = time.Now()
		refreshToken.UpdatedAt = time.Now()
	}
	err2 := s.db.Insert(oauthAccess)
	fmt.Println(err2)

	err1 := s.db.Insert(refreshToken)
	fmt.Println(err1)
	return nil
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
