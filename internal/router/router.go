package router

import (
	"net/http"
	"paradigm-reboot-prober-go/internal/controller"
	"paradigm-reboot-prober-go/internal/middleware"
	"paradigm-reboot-prober-go/internal/repository"
	"paradigm-reboot-prober-go/internal/service"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"

	_ "paradigm-reboot-prober-go/docs"
)

// SetupRouter initializes the routes for the application
func SetupRouter(db *gorm.DB) *gin.Engine {
	r := gin.Default()

	// Initialize Repositories
	userRepo := repository.NewUserRepository(db)
	songRepo := repository.NewSongRepository(db)
	recordRepo := repository.NewRecordRepository(db)

	// Initialize Services
	userService := service.NewUserService(userRepo)
	songService := service.NewSongService(songRepo)
	recordService := service.NewRecordService(recordRepo, songRepo)

	// Initialize Controllers
	userCtrl := controller.NewUserController(userService)
	songCtrl := controller.NewSongController(songService)
	recordCtrl := controller.NewRecordController(recordService, userService)
	uploadCtrl := controller.NewUploadController(userService)

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
		// Public routes
		v1.POST("/user/register", userCtrl.Register)
		v1.POST("/user/login", userCtrl.Login)
		v1.GET("/songs", songCtrl.GetAllSongLevels)
		v1.GET("/songs/:song_id", songCtrl.GetSingleSongInfo)

		// Protected routes
		auth := v1.Group("")
		auth.Use(middleware.AuthMiddleware())
		{
			// User routes
			auth.GET("/user/me", userCtrl.GetMe)
			auth.PATCH("/user/me", userCtrl.UpdateMe)
			auth.POST("/user/me/upload-token", userCtrl.RefreshUploadToken)

			// Record routes
			auth.POST("/records", recordCtrl.UploadRecords)
			auth.GET("/records/b50", recordCtrl.GetB50)
			auth.GET("/records/best", recordCtrl.GetBestRecords)
			auth.GET("/records/all", recordCtrl.GetAllRecords)

			// Upload routes
			auth.POST("/upload/csv", uploadCtrl.UploadCSV)
			auth.POST("/upload/img", uploadCtrl.UploadImg)

			// Admin routes (Admin check should be inside controller or another middleware)
			auth.POST("/songs", songCtrl.CreateSong)
			auth.PATCH("/songs", songCtrl.UpdateSong)
		}
	}

	return r
}
