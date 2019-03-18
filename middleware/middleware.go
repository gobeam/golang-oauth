package middleware


import (
"fmt"
"github.com/ekbana/golang-oauth/core/models"
"github.com/ekbana/golang-oauth/libs/oauth"
"github.com/ekbana/golang-oauth/shared/passhash"
"github.com/gin-gonic/gin"
"github.com/gin-gonic/gin/binding"
models2 "gopkg.in/oauth2.v3/models"
"net/http"
"strconv"
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
	InvalidClient       = "Invalid client credential!"
	InvalidUser         = "Invalid resource owner credential!"
	InvalidGrantType    = "Invalid grant type!"
	InvalidRefreshToken = "Invalid refresh token!"
	InvalidAccessToken = "Invalid access token!"
	TokenExpired = "Token has expired!"
	EmptyHeader         = "Authorization header is not included!"
	InvalidHeader       = "Authorization header is invalid!"
	RefreshToken        = "refresh_token"
	ConversionError            = "Conversion error!"
	Password            = "password"
	Expiry = 300
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

func AuthenticationHandler(store *oauth.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.Request.Header.Get("Authorization")
		fmt.Println(authHeader)
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
		expiryTime := tokenInfo.GetRefreshCreateAt().Add(tokenInfo.GetAccessExpiresIn()).Unix()
		currentTime := time.Now().Unix()
		fmt.Println("subtract: ",expiryTime-currentTime)
		if expiryTime < currentTime {
			oAuthAbort(c, TokenExpired)
			return
		}
		userId , err := strconv.ParseUint(tokenInfo.GetUserID(), 10, 64)
		if err != nil {
			oAuthAbort(c, ConversionError)
			return
		}

		var user models.User
		user.ID = uint(userId)
		user.FindById()
		if user.ID < 1 {
			oAuthAbort(c, InvalidUser)
			return
		}
		c.Set("user", user)
		c.Next()

	}
}

func TokenHandler(store *oauth.Store) gin.HandlerFunc {
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

			client := models.Client{}
			client.FindByIdAndSecret(&models.Client{
				ClientID:     credential.ClientID,
				ClientSecret: credential.ClientSecret,
			})
			if client.ID < 1 {
				oAuthAbort(c, InvalidClient)
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
			userId := fmt.Sprint(user.ID)
			accessToken := createToken(client, credential, userId)
			store.RevokeAccessTokens(userId)
			store.Create(accessToken)
			c.Set("accessToken", AccessTokenPayload{
				AccessToken:  accessToken.Access,
				RefreshToken: accessToken.Refresh,
				ExpiryTime:   accessToken.GetAccessCreateAt().Add(accessToken.GetAccessExpiresIn()).Unix(),
			})
		} else if grant.GrantType == RefreshToken {
			var credential RefreshTokenCredential
			if err := c.ShouldBindBodyWith(&credential, binding.JSON); err != nil {
				_ = c.AbortWithError(422, err).SetType(gin.ErrorTypeBind)
				return
			}
			client := models.Client{}
			client.FindByIdAndSecret(&models.Client{
				ClientID:     credential.ClientID,
				ClientSecret: credential.ClientSecret,
			})
			if client.ID < 1 {
				oAuthAbort(c, InvalidClient)
				return
			}
			tokenInfo, err := store.GetByRefresh(credential.RefreshToken)
			if err != nil || tokenInfo == nil {
				oAuthAbort(c, InvalidRefreshToken)
				return
			}
			userId := tokenInfo.GetUserID()
			scope := tokenInfo.GetScope()
			credential.Scope = scope
			store.RemoveByRefresh(credential.RefreshToken)
			accessToken := createToken(client, credential, userId)
			store.Create(accessToken)
			c.Set("accessToken", AccessTokenPayload{
				AccessToken:  accessToken.Access,
				RefreshToken: accessToken.Refresh,
				ExpiryTime:   accessToken.GetAccessCreateAt().Add(accessToken.GetAccessExpiresIn()).Unix(),
			})
		} else {
			oAuthAbort(c, InvalidGrantType)
			return
		}
		c.Next()
	}
}

func createToken(client models.Client, credential Tokens, id string) (accessToken *models2.Token) {
	token := passhash.RandomKey(100)
	refreshToken := passhash.RandomKey(100)
	accessToken = &models2.Token{
		ClientID:        client.ClientID,
		UserID:          id,
		Scope:           credential.GetScope(),
		Access:          token,
		AccessCreateAt:  time.Now(),
		AccessExpiresIn: time.Second * Expiry,
		Refresh:         refreshToken,
		RefreshCreateAt: time.Now(),
	}
	return
}
