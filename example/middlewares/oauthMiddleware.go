package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	oauth2 "github.com/gobeam/golang-oauth"
	"github.com/gobeam/golang-oauth/example/core/models"
	"github.com/gobeam/golang-oauth/example/shared/passhash"
	"github.com/gobeam/golang-oauth/model"

	"github.com/google/uuid"
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
	InvalidScope       = "Invalid scope!"
	EmptyHeader        = "Authorization header is not included!"
	InvalidHeader      = "Authorization header is invalid!"
	RefreshToken       = "refresh_token"
	Password           = "password"
	Expiry             = 3600
)

type PasswordCredential struct {
	ClientID     string `json:"client_id" binding:"required"`
	ClientSecret string `json:"client_secret" binding:"required"`
	Scope        string `json:"scope,omitempty" binding:"required"`
	Username     string `json:"username" binding:"required"`
	Password     string `json:"Password" binding:"required"`
}

type RefreshTokenCredential struct {
	ClientID     string `json:"client_id" binding:"required"`
	ClientSecret string `json:"client_secret" binding:"required"`
	RefreshToken string `json:"refresh_token" binding:"required"`
	Scope        string `json:"scope,omitempty"`
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

func getRequestPath(c *gin.Context) []string {
	path := c.Request.URL.Path
	for _, param := range c.Params {
		path = strings.Replace(path, param.Value, ":"+param.Key, -1)
	}
	return strings.Split(path, "/")
}

func findInSlice(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}


// Check If access token is valid and have proper scope
func OauthMiddleware(store *oauth2.Store) gin.HandlerFunc {
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
		if err != nil {
			oAuthAbort(c, err.Error())
			return
		}
		if tokenInfo == nil {
			oAuthAbort(c, InvalidAccessToken)
			return
		}

		// checking scope
		if !strings.Contains(tokenInfo.Scope, "*") {
			scopeArr := strings.Fields(tokenInfo.Scope)
			requestUrlArr := getRequestPath(c)
			validScope := false
			for _, scope := range scopeArr {
				if findInSlice(requestUrlArr, scope) {
					validScope = true
				}
			}
			if !validScope {
				oAuthAbort(c, InvalidScope)
				return
			}
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

// Return Access Token for valid client and user credential
func AccessToken(store *oauth2.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var grant GrantType
		if err := c.ShouldBindBodyWith(&grant, binding.JSON); err != nil {
			_ = c.AbortWithError(422, err).SetType(gin.ErrorTypeBind)
			return
		}
		switch grant.GrantType {
		case Password:
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
		case RefreshToken:
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
			var scope string
			if credential.Scope != "" {
				scope = credential.Scope
			} else {
				scope = refreshToken.Scope
			}
			accessToken := &model.Token{
				ClientID:        uuid.MustParse(credential.ClientID),
				ClientSecret:    credential.ClientSecret,
				UserID:          int64(refreshToken.UserId),
				Scope:           scope,
				AccessCreateAt:  time.Now(),
				AccessExpiresIn: time.Second * Expiry,
				RefreshCreateAt: time.Now(),
			}

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
		default:
			oAuthAbort(c, InvalidGrantType)
			return
		}
		c.Next()
	}
}

func createToken(cred PasswordCredential, user models.User) (accessToken *model.Token) {
	accessToken = &model.Token{
		ClientID:        uuid.MustParse(cred.ClientID),
		ClientSecret:    cred.ClientSecret,
		UserID:          int64(user.ID),
		Scope:           cred.Scope,
		AccessCreateAt:  time.Now(),
		AccessExpiresIn: time.Second * Expiry,
		RefreshCreateAt: time.Now(),
	}
	return
}
