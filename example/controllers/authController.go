package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	goOauth2 "github.com/gobeam/golang-oauth"
	"github.com/gobeam/golang-oauth/example/core/models"
	"github.com/gobeam/golang-oauth/example/shared/passhash"
	"net/http"
)

type AuthController struct {
	store *goOauth2.Store
	Controller
}

func NewAuthController(store *goOauth2.Store) *AuthController {
	return &AuthController{store: store}
}

func (controller AuthController) Register(c *gin.Context) {
	user := models.User{}
	if err := c.ShouldBindBodyWith(&user, binding.JSON); err != nil {
		_ = c.AbortWithError(http.StatusUnprocessableEntity, err).SetType(gin.ErrorTypeBind)
		return
	}
	pwd, _ := passhash.HashPassword(user.Password)
	user.Password = string(pwd)
	user.Create()
	controller.SuccessResponse(c, map[string]interface{}{"email": user.Email})
}

func (controller AuthController) Client(c *gin.Context) {
	client, err := controller.store.CreateClient(1,"test app")
	if err != nil {
		controller.ErrorResponse(c, http.StatusUnauthorized, err.Error())
		return
	}
	controller.SuccessResponse(c, client)
}

func (controller AuthController) Token(c *gin.Context) {
	token, exists := c.Get("accessToken")
	if !exists {
		msg := "Invalid Credentials!"
		controller.ErrorResponse(c, http.StatusUnauthorized, msg)
		return
	}
	controller.SuccessResponse(c, token)
}

func (controller AuthController) Profile(c *gin.Context) {
	profile, exists := c.Get("user")
	if !exists {
		msg := "Invalid Token!"
		controller.ErrorResponse(c, http.StatusUnauthorized, msg)
		return
	}
	controller.SuccessResponse(c, profile)
}
