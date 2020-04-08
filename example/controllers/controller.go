package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type ResourceController interface {
	Index(c *gin.Context)
	View(c *gin.Context)
	Store(c *gin.Context)
	Update(c *gin.Context)
	Destroy(c *gin.Context)
}

type Controller struct{}

func (ctl Controller) success(data interface{}) map[string]interface{} {
	return gin.H{
		"status": "success",
		"error":  false,
		"data":   data,
	}
}

func (ctl Controller) error(message string) map[string]interface{} {
	return gin.H{
		"status":  "error",
		"error":   true,
		"message": message,
	}
}

func (ctl Controller) Deleted(c *gin.Context) {
	data := gin.H{
		"status":  "success",
		"error":   false,
		"message": "Successfully deleted!",
	}
	c.JSON(http.StatusAccepted, data)
}

func (ctl Controller) SuccessResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, ctl.success(data))
}

func (ctl Controller) ErrorResponse(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, ctl.error(message))
}
