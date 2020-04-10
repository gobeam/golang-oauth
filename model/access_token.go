package model

import "github.com/google/uuid"

// AccessTokens is model for Oauth Access Token
type AccessTokens struct {
	Model
	AccessTokenPayload
	Name    string `db:"name"`
	Scope   string `db:"scope"`
	Revoked bool   `db:"revoked"`
}

// AccessTokenPayload is data that will be encrypted by RSA encryption
type AccessTokenPayload struct {
	UserId    int64     `db:"user_id"`
	ClientId  uuid.UUID `db:"client_id"`
	ExpiredAt int64     `db:"expired_at"`
}
