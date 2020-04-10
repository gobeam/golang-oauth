package util

// constants
const (
	PublicPem           = "public.pem"
	PrivatePem          = "private.pem"
	AccessTokenTable    = "oauth_access_tokens"
	RefreshTokenTable   = "oauth_refresh_tokens"
	ClientTable         = "oauth_clients"
	BitSize             = 2048
	RefreshTokenRevoked = "refresh token already been revoked"
	AccessTokenRevoked  = "access token has already been revoked"
	AccessTokenExpired  = "access token has already been expired"
	InvalidRefreshToken = "invalid refresh token"
	InvalidAccessToken  = "invalid access token"
	InvalidClient       = "invalid client"
	EmptyUserID         = "user id cannot be empty"
	Label               = "OAEP Encrypted"
	PublicKey           = "PUBLIC KEY"
	PrivateKey          = "PRIVATE KEY"
	DbConfig            = "root:@tcp(127.0.0.1:3306)/goauth?charset=utf8&parseTime=True&loc=Local"
)
