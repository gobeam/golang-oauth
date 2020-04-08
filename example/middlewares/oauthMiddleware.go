package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	goOauth2 "github.com/gobeam/golang-oauth"
	"github.com/google/uuid"
	"github.com/gobeam/golang-oauth/example/core/models"
	"github.com/gobeam/golang-oauth/example/shared/passhash"
	"net/http"
	"strings"
	"time"
)

func oAuthAbort(c *gin.Context, msg string) {
	c.JSON(http.StatusUnauthorized, gin.H{
		"status":  "error",
		"error":   true,
		"message": msg,
	})
	c.Abort()
}

const (
	InvalidClient      = "Invalid client credential!"
	InvalidUser        = "Invalid resource owner credential!"
	InvalidGrantType   = "Invalid grant type!"
	InvalidAccessToken = "Invalid access token!"
	EmptyHeader        = "Authorization header is not included!"
	InvalidHeader      = "Authorization header is invalid!"
	RefreshToken       = "refresh_token"
	Password           = "password"
	Expiry             = 3600
)

type PasswordCredential struct {
	ClientID     string `json:"client_id" binding:"required"`
	ClientSecret string `json:"client_secret" binding:"required"`
	Scope        string `json:"scope" binding:"required"`
	Username     string `json:"username" binding:"required"`
	Password     string `json:"Password" binding:"required"`
}

type RefreshTokenCredential struct {
	ClientID     string `json:"client_id" binding:"required"`
	ClientSecret string `json:"client_secret" binding:"required"`
	RefreshToken string `json:"refresh_token" binding:"required"`
	Scope        string `json:"scope"`
}

type AccessTokenPayload struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiryTime   int64  `json:"expiry_time"`
}

type Profile struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	ID    uint   `json:"id"`
}

type GrantType struct {
	GrantType string `json:"grant_type" binding:"required"`
}

type Tokens interface {
	GetScope() string
}

func (rtc RefreshTokenCredential) GetScope() string {
	return rtc.Scope
}

func (pc PasswordCredential) GetScope() string {
	return pc.Scope
}

func OauthMiddleware(store *goOauth2.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.Request.Header.Get("Authorization")
		if authHeader == "" {
			oAuthAbort(c, EmptyHeader)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			oAuthAbort(c, InvalidHeader)
			return
		}

		tokenInfo, err := store.GetByAccess(parts[1])
		if err != nil || tokenInfo == nil {
			oAuthAbort(c, InvalidAccessToken)
			return
		}

		var user models.User
		user.ID = uint(tokenInfo.UserId)
		user.FindById()
		if user.ID < 1 {
			oAuthAbort(c, InvalidUser)
			return
		}
		c.Set("user", Profile{
			Email: user.Email,
			Name:  user.Name,
			ID:    user.ID,
		})
		c.Next()
	}
}

func AccessToken(store *goOauth2.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var grant GrantType
		if err := c.ShouldBindBodyWith(&grant, binding.JSON); err != nil {
			_ = c.AbortWithError(422, err).SetType(gin.ErrorTypeBind)
			return
		}
		if grant.GrantType == Password {
			var credential PasswordCredential
			if err := c.ShouldBindBodyWith(&credential, binding.JSON); err != nil {
				_ = c.AbortWithError(422, err).SetType(gin.ErrorTypeBind)
				return
			}

			user := models.User{Email: credential.Username}
			user.FindByEmail()

			if user.ID < 1 {
				oAuthAbort(c, InvalidUser)
				return
			}

			err := passhash.VerifyPassword(user.Password, credential.Password)
			if err != nil {
				oAuthAbort(c, InvalidUser)
				return
			}
			_, err = uuid.Parse(credential.ClientID)

			if err != nil {
				oAuthAbort(c, InvalidClient)
				return
			}

			accessToken := createToken(credential, user)

			token, err := store.Create(accessToken)
			if err != nil {
				oAuthAbort(c, err.Error())
				return
			}
			c.Set("accessToken", AccessTokenPayload{
				AccessToken:  token.AccessToken,
				RefreshToken: token.RefreshToken,
				ExpiryTime:   token.ExpiredAt,
			})
		} else if grant.GrantType == RefreshToken {
			var credential RefreshTokenCredential
			if err := c.ShouldBindBodyWith(&credential, binding.JSON); err != nil {
				_ = c.AbortWithError(422, err).SetType(gin.ErrorTypeBind)
				return
			}
			refreshToken, err := store.GetByRefresh(credential.RefreshToken)
			if err != nil {
				oAuthAbort(c, err.Error())
				return
			}
			accessToken := &goOauth2.Token{
				ClientID:        uuid.MustParse(credential.ClientID),
				ClientSecret:    credential.ClientSecret,
				UserID:          int64(refreshToken.UserId),
				Scope:           "*",
				AccessCreateAt:  time.Now(),
				AccessExpiresIn: time.Second * Expiry,
				RefreshCreateAt: time.Now(),
			}

			ctoken, err := store.Create(accessToken)
			if err != nil {
				oAuthAbort(c, err.Error())
				return
			}
			c.Set("accessToken", AccessTokenPayload{
				AccessToken:  ctoken.AccessToken,
				RefreshToken: ctoken.RefreshToken,
				ExpiryTime:   ctoken.ExpiredAt,
			})
		} else {
			oAuthAbort(c, InvalidGrantType)
			return
		}
		c.Next()
	}
}

func createToken(cred PasswordCredential, user models.User) (accessToken *goOauth2.Token) {
	accessToken = &goOauth2.Token{
		ClientID:        uuid.MustParse(cred.ClientID),
		ClientSecret:    cred.ClientSecret,
		UserID:          int64(user.ID),
		Scope:           "*",
		AccessCreateAt:  time.Now(),
		AccessExpiresIn: time.Second * Expiry,
		RefreshCreateAt: time.Now(),
	}
	return
}
