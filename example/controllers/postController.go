package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gobeam/golang-oauth/example/core/models"
	"github.com/gobeam/golang-oauth/example/messagingService"
	"github.com/gobeam/golang-oauth/example/middlewares"
	"github.com/gobeam/golang-oauth/example/common/configHelper"
	"github.com/gobeam/golang-oauth/example/postService/model"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type PostController struct {
	Controller
}

func NewPostController() *PostController {
	return &PostController{}
}

var (
	routingKey       = "app.service.post.evaluate"
	eventType        = "topic"
	createImageRoute = "app.service.image.create"
)

type PostValidate struct {
	Title       string                `form:"title" binding:"required,max=100,min=2"`
	Image       *multipart.FileHeader `form:"image" binding:"required"`
	Description string                `form:"description" binding:"required,max=100,min=2"`
	OgType      string                `form:"og_type" binding:"required,max=100,min=2"`
	OgUrl       string                `form:"og_url" binding:"required,max=100,min=2"`
	Body        string                `form:"body" binding:"required,min=2"`
	CategoryId  string                 `form:"category_id" binding:"required"`
}

type PostValidateUpdate struct {
	Title       string                `form:"title" binding:"required,max=100,min=2"`
	Description string                `form:"description" binding:"required,max=100,min=2"`
	OgType      string                `form:"og_type" binding:"required,max=100,min=2"`
	OgUrl       string                `form:"og_url" binding:"required,max=100,min=2"`
	Body        string                `form:"body" binding:"required,min=2"`
	CategoryId  string                 `form:"category_id" binding:"required"`
}

type BannerValidate struct {
	Title string                `form:"title" binding:"required"`
	Label string                `form:"label" binding:"required"`
	File  *multipart.FileHeader `form:"file" binding:"required"`
}


func (controller PostController) Index(c *gin.Context) {
	var posts models.Posts
	posts.Get()
	SuccessResponse(c, posts)
	return
}

func (controller PostController) Destroy(c *gin.Context) {
	var post models.Post
	id := c.Param("id")
	postId, _ := strconv.ParseInt(id, 10, 64)
	post.FindById(postId)
	if post.ID != 0 {
		dirname, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		err = os.Remove(path.Join(dirname, fmt.Sprintf("%s%s","/public/uploads/", post.Image)))
		if err != nil {
			panic(err)
		}
		post.Delete()
	}
	Deleted(c)
}


func (controller PostController) View(c *gin.Context) {
	var post models.Post
	id := c.Param("id")
	postId, _ := strconv.ParseInt(id, 10, 64)
	post.FindById(postId)
	if post.ID != 0 {
		var categorys models.PostCategorys
		categorys.GetByPostId(postId)
		if categorys != nil {
			catIds := make([]uint,len(categorys))
			for k,v := range categorys {
				catIds[k] = v.CategoryId
			}
			post.CategoryId = catIds
		}
		SuccessResponse(c, post)
		return
	}
	ErrorResponse(c, http.StatusNotFound, "not found")
}

func (controller PostController) Update(c *gin.Context) {
	var postRequest PostValidateUpdate
	if err := c.ShouldBind(&postRequest); err != nil {
		_ = c.AbortWithError(http.StatusUnprocessableEntity, err).SetType(gin.ErrorTypeBind)
		return
	}
	var post models.Post
	id := c.Param("id")
	postId, _ := strconv.ParseInt(id, 10, 64)
	post.FindById(postId)
	if post.ID < 1 {
		ErrorResponse(c, http.StatusNotFound, "post not found")
		return
	}
	file, _ := c.FormFile("image")
	if file != nil {
		fileType := file.Header.Get("Content-Type")
		if fileType == "" || !strings.Contains(fileType, "image") {
			msg := "Invalid Image File!"
			ErrorResponse(c, http.StatusUnprocessableEntity, msg)
			return
		}

		filename := filepath.Base(file.Filename)
		ext := strings.Split(filename,".")
		actualName := fmt.Sprintf("%s.%s", uuid.New(),ext[1])
		filename = fmt.Sprintf("%s%s", "./public/uploads/", actualName)

		if err := c.SaveUploadedFile(file, filename); err != nil {
			c.String(http.StatusBadRequest, fmt.Sprintf("upload file err: %s", err.Error()))
			return
		}
		post.Image = actualName
	}

	stringSlice := strings.Split(postRequest.CategoryId, ",")
	ary := make([]uint, len(stringSlice))
	for i := range ary {
		u64, _ := strconv.ParseUint(stringSlice[i], 10, 32)
		ary[i] = uint(u64)
	}
	post.Title = postRequest.Title
	post.OgType = postRequest.OgType
	post.OgUrl = postRequest.OgUrl
	post.Body = postRequest.Body
	post.Description = postRequest.Description
	post.CategoryId = ary
	post.CreatedAt = time.Now()
	post.Update()
	SuccessResponse(c, post)
}

func (controller PostController) Store(c *gin.Context) {
	profile, exists := c.Get("user")
	if !exists {
		msg := "Invalid Token!"
		ErrorResponse(c, http.StatusUnauthorized, msg)
		return
	}
	var postRequest PostValidate
	if err := c.ShouldBind(&postRequest); err != nil {
		_ = c.AbortWithError(http.StatusUnprocessableEntity, err).SetType(gin.ErrorTypeBind)
		return
	}

	fileType := postRequest.Image.Header.Get("Content-Type")
	if fileType == "" || !strings.Contains(fileType, "image") {
		msg := "Invalid Image File!"
		ErrorResponse(c, http.StatusUnprocessableEntity, msg)
		return
	}

	filename := filepath.Base(postRequest.Image.Filename)
	ext := strings.Split(filename,".")
	actualName := fmt.Sprintf("%s.%s", uuid.New(),ext[1])
	filename = fmt.Sprintf("%s%s", "./public/uploads/", actualName)

	if err := c.SaveUploadedFile(postRequest.Image, filename); err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("upload file err: %s", err.Error()))
		return
	}

	stringSlice := strings.Split(postRequest.CategoryId, ",")
	ary := make([]uint, len(stringSlice))
	for i := range ary {
		u64, _ := strconv.ParseUint(stringSlice[i], 10, 32)
		ary[i] = uint(u64)
	}

	var post model.Post
	post.Image = actualName
	post.Title = postRequest.Title
	post.OgType = postRequest.OgType
	post.OgUrl = postRequest.OgUrl
	post.Body = postRequest.Body
	post.Description = postRequest.Description
	post.CategoryId = ary
	post.CreatedAt = time.Now()

	user := profile.(middleware.Profile)
	b, _ := json.Marshal(post)
	err := messagingService.MessagingClient.Publish([]byte(b), configHelper.GetConfig("amqp", "exchange_name").String(), routingKey, eventType, strconv.FormatUint(uint64(user.ID), 10))
	if err != nil {
		ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	SuccessResponse(c, post)
}
