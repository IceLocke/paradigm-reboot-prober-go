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

	_ "paradigm-reboot-prober-go/docs" // Swagger docs
)

// SetupRouter initializes the routes for the application
func SetupRouter(db *gorm.DB) *gin.Engine {
	r := gin.Default() // gin.Default() already includes Logger and Recovery middleware

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
	recordCtrl := controller.NewRecordController(recordService, userService, songService)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "up",
		})
	})

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v2 := r.Group("/api/v2")
	{
		// Public routes
		v2.POST("/user/register", userCtrl.Register)
		v2.POST("/user/login", userCtrl.Login)
		v2.GET("/songs", songCtrl.GetAllCharts)
		v2.GET("/songs/:song_id", songCtrl.GetSingleSongInfo)

		// Routes with optional auth
		optionalAuth := v2.Group("")
		optionalAuth.Use(middleware.OptionalAuthMiddleware())
		{
			optionalAuth.GET("/records/:username", recordCtrl.GetPlayRecords)
			optionalAuth.GET("/records/:username/song/:song_addr", recordCtrl.GetSongRecords)
			optionalAuth.GET("/records/:username/chart/:chart_addr", recordCtrl.GetChartRecords)

			// Record upload: under optional auth so upload-token-based auth works
			// (handler performs its own authorization check)
			optionalAuth.POST("/records/:username", recordCtrl.UploadRecords)
		}

		// Protected routes
		auth := v2.Group("")
		auth.Use(middleware.AuthMiddleware())
		{
			// User routes
			auth.GET("/user/me", userCtrl.GetMe)
			auth.PUT("/user/me", userCtrl.UpdateMe)
			auth.POST("/user/me/upload-token", userCtrl.RefreshUploadToken)
			auth.PUT("/user/me/password", userCtrl.ChangePassword)

			// Admin routes (with admin middleware)
			admin := auth.Group("")
			admin.Use(middleware.AdminMiddleware(userService))
			{
				admin.POST("/songs", songCtrl.CreateSong)
				admin.PUT("/songs", songCtrl.UpdateSong)
				admin.POST("/user/reset-password", userCtrl.ResetPassword)
			}
		}
	}

	return r
}
