package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "paradigm-reboot-prober-go/docs"
)

// SetupRouter initializes the routes for the application
func SetupRouter() *gin.Engine {
	r := gin.Default()

	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "up",
		})
	})

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := r.Group("/api/v1")
	{
		users := v1.Group("/users")
		{
			users.GET("/profile", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "user profile"})
			})
		}
	}

	return r
}
