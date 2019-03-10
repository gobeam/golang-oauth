package main

import (
	"database/sql"
	"fmt"
	"gopkg.in/gorp.v2"
	"io"
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
	AccessTokenId    int64  `db:"access_token_id"`
	Revoked   bool   `db:"revoked"`
	ExpiredAt int64  `db:"expired_at"`
}

//Oauth Clients
type OauthClients struct {
	Model
	UserId    int64  `db:"user_id"`
	Name      string `db:"name"`
	Secret  string  `db:"secret"`
	Revoked   bool   `db:"revoked"`
	Redirect   string   `db:"redirect"`
}


// Store mysql token store
type Store struct {
	tableName string
	db        *gorp.DbMap
	stdout    io.Writer
	ticker    *time.Ticker
}


// NewStore create mysql store instance,
// config mysql configuration,
// tableName table name (default oauth2_token),
// GC time interval (in seconds, default 600)
func NewStore(config *Config) {
	db, err := sql.Open("mysql", config.DSN)
	if err != nil {
		panic(err)
	}

	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.MaxLifetime)

	NewStoreWithDB(db)
}


// NewStoreWithDB create mysql store instance,
// db sql.DB
func NewStoreWithDB(db *sql.DB) {
	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.MySQLDialect{Encoding: "UTF8", Engine: "MyISAM"}}

	dbmap.AddTableWithName(OauthAccessTokens{}, "oauth_access_tokens")
	dbmap.AddTableWithName(OauthClients{}, "oauth_clients")
	 dbmap.AddTableWithName(OauthRefreshTokens{}, "oauth_refresh_tokens")
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
func NewDefaultStore(config *Config) {
	 NewStore(config)
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
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE expired_at<=? OR (code='' AND access='' AND refresh='')", s.tableName)
	n, err := s.db.SelectInt(query, now)
	if err != nil || n == 0 {
		if err != nil {
			s.errorf(err.Error())
		}
		return
	}

	_, err = s.db.Exec(fmt.Sprintf("DELETE FROM %s WHERE expired_at<=? OR (code='' AND access='' AND refresh='')", s.tableName), now)
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


