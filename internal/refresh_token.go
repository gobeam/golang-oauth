package internal

import "github.com/google/uuid"

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
