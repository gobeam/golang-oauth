package routers

import (
	"github.com/gin-gonic/gin"
	oauth2 "github.com/gobeam/golang-oauth"
	"github.com/gobeam/golang-oauth/example/controllers"
	"github.com/gobeam/golang-oauth/example/middlewares"
)

var Router *gin.Engine

func init() {
	Router = gin.Default()
}

func ResourceFulRouter(r gin.IRouter, controller controllers.ResourceController) {
	r.GET("", controller.Index)
	r.GET("/:id", controller.View)
	r.POST("", controller.Store)
	r.PUT("/:id", controller.Update)
	r.DELETE("/:id", controller.Destroy)
}

func SetupRouter(store *oauth2.Store) *gin.Engine {
	router := gin.Default()
	router.Use(middleware.CORS())
	authController := controllers.NewAuthController(store)

	pub := router.Group("/api/v1")
	pub.Use(middleware.Errors())
	{
		authorized := pub.Group("/auth")
		authorized.Use(middleware.AccessToken(store))
		{
			authorized.POST("/token", authController.Token)
		}

		pub.POST("/register", authController.Register)
		pub.POST("/client", authController.Client)

		priv := pub.Group("/")
		priv.Use(middleware.OauthMiddleware(store))
		{
			priv.GET("/profile", authController.Profile)

			postController := controllers.NewPostController()
			ResourceFulRouter(priv.Group("/post"), postController)
			categoryController := controllers.NewCategoryController()
			ResourceFulRouter(priv.Group("/category"), categoryController)
		}
	}
	return router
}
