package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/gobeam/golang-oauth/example/core/models"
	"net/http"
	"strconv"
	"time"
)

type CategoryController struct {
	Controller
}

func (controller CategoryController) Index(c *gin.Context) {
	var categories models.Categories
	categories.Get()
	controller.SuccessResponse(c, categories)
}

func (controller CategoryController) View(c *gin.Context) {
	var category models.Category
	id := c.Param("id")
	todoId, _ := strconv.ParseInt(id, 10, 64)
	category.FindById(uint(todoId))
	if category.ID != 0 {
		controller.SuccessResponse(c, category)
	}
	controller.ErrorResponse(c, http.StatusNotFound, "not found")
}

func (controller CategoryController) Store(c *gin.Context) {
	var category models.Category
	if err := c.ShouldBindBodyWith(&category, binding.JSON); err != nil {
		_ = c.AbortWithError(http.StatusUnprocessableEntity, err).SetType(gin.ErrorTypeBind)
		return
	}
	category.CreatedAt = time.Now()
	category.Create()
	controller.SuccessResponse(c, category)
}

func (controller CategoryController) Update(c *gin.Context) {
	var category models.Category
	if err := c.ShouldBindBodyWith(&category, binding.JSON); err != nil {
		_ = c.AbortWithError(http.StatusUnprocessableEntity, err).SetType(gin.ErrorTypeBind)
		return
	}
	var orginalCategory models.Category
	id := c.Param("id")
	catId, _ := strconv.ParseInt(id, 10, 64)
	orginalCategory.FindById(uint(catId))
	if orginalCategory.ID != 0 {
		orginalCategory.Name = category.Name
		orginalCategory.Status = category.Status
		orginalCategory.Label = category.Label
		orginalCategory.UpdatedAt = time.Now()
		orginalCategory.Update()
	}
	controller.SuccessResponse(c, orginalCategory)
}

func (controller CategoryController) Destroy(c *gin.Context) {
	var category models.Category
	id := c.Param("id")
	todoId, _ := strconv.ParseInt(id, 10, 64)
	category.FindById(uint(todoId))
	if category.ID != 0 {
		category.Delete()
	}
	controller.Deleted(c)
}

func NewCategoryController() *CategoryController {
	return &CategoryController{}
}
