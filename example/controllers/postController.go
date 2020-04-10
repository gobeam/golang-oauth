package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gobeam/golang-oauth/example/core/models"
	"github.com/google/uuid"
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

type PostValidate struct {
	Title       string                `form:"title" binding:"required,max=100,min=2"`
	Image       *multipart.FileHeader `form:"image" binding:"required"`
	Description string                `form:"description" binding:"required,max=100,min=2"`
	Body        string                `form:"body" binding:"required,min=2"`
	Category    string                `form:"category" binding:"required"`
}

type PostValidateUpdate struct {
	Title       string `form:"title" binding:"required,max=100,min=2"`
	Description string `form:"description" binding:"required,max=100,min=2"`
	Body        string `form:"body" binding:"required,min=2"`
	Category    string `form:"category" binding:"required"`
}

// Index return all posts
func (controller PostController) Index(c *gin.Context) {
	posts := models.Posts{}
	posts.Get()
	controller.SuccessResponse(c, posts)
	return
}

// Store stores new post
func (controller PostController) Store(c *gin.Context) {
	var postRequest PostValidate
	if err := c.ShouldBind(&postRequest); err != nil {
		_ = c.AbortWithError(http.StatusUnprocessableEntity, err).SetType(gin.ErrorTypeBind)
		return
	}

	imagName, err := storeIfImagePresent(c, postRequest.Image)
	if err != nil {
		controller.ErrorResponse(c, http.StatusUnprocessableEntity, err.Error())
		return
	}

	var post models.Post
	post.Image = imagName
	post.Title = postRequest.Title
	post.Body = postRequest.Body
	post.Description = postRequest.Description
	post.CreatedAt = time.Now()
	post.Create()

	var ints []uint
	_ = json.Unmarshal([]byte(postRequest.Category), &ints)

	post.AssignCategory(ints, false)

	controller.SuccessResponse(c, post)
}

// View return post by id
func (controller PostController) View(c *gin.Context) {
	post := models.Post{}
	id := c.Param("id")
	postId, _ := strconv.ParseInt(id, 10, 64)
	post.FindById(postId)
	if post.ID != 0 {
		categories := models.PostCategories{}
		categories.GetByPostId(postId)
		if len(categories) > 0 {
			catIds := make([]uint, len(categories))
			for k, v := range categories {
				catIds[k] = v.CategoryId
			}
			//post.CategoryId = catIds
		}
		controller.SuccessResponse(c, post)
		return
	}
	controller.ErrorResponse(c, http.StatusNotFound, "not found")
}

// Update updates post by id
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
		controller.ErrorResponse(c, http.StatusNotFound, "post not found")
		return
	}
	file, _ := c.FormFile("image")
	if file != nil {
		// delete old image
		filePath := fmt.Sprintf("%s%s", "./public/uploads/", post.Image)
		err := os.Remove(filePath)
		if err != nil {
			controller.ErrorResponse(c, http.StatusInternalServerError, err.Error())
			return
		}

		// add new Image
		imageName, err := storeIfImagePresent(c, file)
		if err != nil {
			controller.ErrorResponse(c, http.StatusUnprocessableEntity, err.Error())
			return
		}
		post.Image = imageName
	}

	post.Title = postRequest.Title
	post.Body = postRequest.Body
	post.Description = postRequest.Description
	post.CreatedAt = time.Now()
	post.Update()

	var ints []uint
	_ = json.Unmarshal([]byte(postRequest.Category), &ints)

	post.AssignCategory(ints, true)

	controller.SuccessResponse(c, post)
}

//Destroy deletes post by id
func (controller PostController) Destroy(c *gin.Context) {
	post := models.Post{}
	id := c.Param("id")
	postId, _ := strconv.ParseInt(id, 10, 64)
	post.FindById(postId)
	if post.ID != 0 {
		dirname, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		err = os.Remove(path.Join(dirname, fmt.Sprintf("%s%s", "/public/uploads/", post.Image)))
		if err != nil {
			panic(err)
		}
		post.Delete()
		controller.Deleted(c)
		return
	}
	controller.ErrorResponse(c, http.StatusNotFound, "not found")
}

// storeIfImagePresent stores image if present
func storeIfImagePresent(c *gin.Context, file *multipart.FileHeader) (imgName string, err error) {
	fileType := file.Header.Get("Content-Type")
	if fileType == "" || !strings.Contains(fileType, "image") {
		msg := "invalid image file"
		return "", errors.New(msg)
	}

	filename := filepath.Base(file.Filename)
	ext := strings.Split(filename, ".")
	actualName := fmt.Sprintf("%s.%s", uuid.New(), ext[1])
	filename = fmt.Sprintf("%s%s", "./public/uploads/", actualName)

	if err := c.SaveUploadedFile(file, filename); err != nil {
		return "", errors.New(fmt.Sprintf("upload file err: %s", err.Error()))
	}
	return actualName, nil
}
