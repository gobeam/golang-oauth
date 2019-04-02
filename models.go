package goOauth2

import (
	"github.com/google/uuid"
	"gopkg.in/gorp.v2"
	"io"
	"time"
)

// Model is default model
type Model struct {
	ID        uuid.UUID `db:"id,primarykey"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// AccessTokens is model for Oauth Access Token
type AccessTokens struct {
	Model
	AccessTokenPayload
	Name    string `db:"name"`
	Revoked bool   `db:"revoked"`
}

// AccessTokenPayload is data that will be encrypted by RSA encryption
type AccessTokenPayload struct {
	UserId    int64     `db:"user_id"`
	ClientId  uuid.UUID `db:"client_id"`
	ExpiredAt int64     `db:"expired_at"`
}

// RefreshTokenPayload is model for oauth refresh token
type RefreshTokenPayload struct {
	AccessTokenId uuid.UUID `db:"access_token_id"`
}

// RefreshTokens is model for oauth refresh token
type RefreshTokens struct {
	Model
	RefreshTokenPayload
	Revoked bool `db:"revoked"`
}

// Clients is model for oauth clients
type Clients struct {
	Model
	UserId   int64  `db:"user_id"`
	Name     string `db:"name"`
	Secret   string `db:"secret"`
	Revoked  bool   `db:"revoked"`
	Redirect string `db:"redirect"`
}

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

// TokenResponse model after creating access token and refresh token
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiredAt int64 `json:"expired_at"`
}

// Token struct which hold token details
type Token struct {
	ClientID         uuid.UUID     `bson:"ClientID"`
	ClientSecret     string        `bson:"ClientSecret"`
	UserID           int64         `bson:"UserID"`
	RedirectURI      string        `bson:"RedirectURI"`
	Scope            string        `bson:"Scope"`
	AccessCreateAt   time.Time     `bson:"AccessCreateAt"`
	AccessExpiresIn  time.Duration `bson:"AccessExpiresIn"`
	RefreshCreateAt  time.Time     `bson:"RefreshCreateAt"`
	RefreshExpiresIn time.Duration `bson:"RefreshExpiresIn"`
}

// TokenInfo the token information model interface
type TokenInfo interface {
	New() TokenInfo

	GetClientID() uuid.UUID
	SetClientID(uuid.UUID)
	GetClientSecret() string
	SetClientSecret() string
	GetUserID() int64
	SetUserID(int64)
	GetRedirectURI() string
	SetRedirectURI(string)
	GetScope() string
	SetScope(string)

	GetAccessCreateAt() time.Time
	SetAccessCreateAt(time.Time)
	GetAccessExpiresIn() time.Duration
	SetAccessExpiresIn(time.Duration)

	GetRefreshCreateAt() time.Time
	SetRefreshCreateAt(time.Time)
	GetRefreshExpiresIn() time.Duration
	SetRefreshExpiresIn(time.Duration)
}

// NewToken create to token model instance
func NewToken() *Token {
	return &Token{}
}

// New create to token model instance
func (t *Token) New() TokenInfo {
	return NewToken()
}

// GetClientID the client id
func (t *Token) GetClientID() uuid.UUID {
	return t.ClientID
}

// GetClientSecret the client id
func (t *Token) GetClientSecret() string {
	return t.ClientSecret
}

// SetClientSecret the client id
func (t *Token) SetClientSecret() string {
	return t.ClientSecret
}

// SetClientID the client id
func (t *Token) SetClientID(clientID uuid.UUID) {
	t.ClientID = clientID
}

// GetUserID the user id
func (t *Token) GetUserID() int64 {
	return t.UserID
}

// SetUserID the user id
func (t *Token) SetUserID(userID int64) {
	t.UserID = userID
}

// GetRedirectURI redirect URI
func (t *Token) GetRedirectURI() string {
	return t.RedirectURI
}

// SetRedirectURI redirect URI
func (t *Token) SetRedirectURI(redirectURI string) {
	t.RedirectURI = redirectURI
}

// GetScope get scope of authorization
func (t *Token) GetScope() string {
	return t.Scope
}

// SetScope get scope of authorization
func (t *Token) SetScope(scope string) {
	t.Scope = scope
}

// GetAccessCreateAt create Time
func (t *Token) GetAccessCreateAt() time.Time {
	return t.AccessCreateAt
}

// SetAccessCreateAt create Time
func (t *Token) SetAccessCreateAt(createAt time.Time) {
	t.AccessCreateAt = createAt
}

// GetAccessExpiresIn the lifetime in seconds of the access token
func (t *Token) GetAccessExpiresIn() time.Duration {
	return t.AccessExpiresIn
}

// SetAccessExpiresIn the lifetime in seconds of the access token
func (t *Token) SetAccessExpiresIn(exp time.Duration) {
	t.AccessExpiresIn = exp
}

// GetRefreshCreateAt create Time
func (t *Token) GetRefreshCreateAt() time.Time {
	return t.RefreshCreateAt
}

// SetRefreshCreateAt create Time
func (t *Token) SetRefreshCreateAt(createAt time.Time) {
	t.RefreshCreateAt = createAt
}

// GetRefreshExpiresIn the lifetime in seconds of the refresh token
func (t *Token) GetRefreshExpiresIn() time.Duration {
	return t.RefreshExpiresIn
}

// SetRefreshExpiresIn the lifetime in seconds of the refresh token
func (t *Token) SetRefreshExpiresIn(exp time.Duration) {
	t.RefreshExpiresIn = exp
}
