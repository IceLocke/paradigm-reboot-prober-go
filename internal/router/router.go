package router

import (
	"net/http"
	"paradigm-reboot-prober-go/config"
	"paradigm-reboot-prober-go/internal/controller"
	"paradigm-reboot-prober-go/internal/metrics"
	"paradigm-reboot-prober-go/internal/middleware"
	"paradigm-reboot-prober-go/internal/repository"
	"paradigm-reboot-prober-go/internal/service"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"

	_ "paradigm-reboot-prober-go/docs" // Swagger docs
)

const (
	MaxRawRequestBodySize = 10 << 20 // 10MB

	RegisterEndpointRequestPerMinute = 2
	LoginEndpointRequestPerMinute    = 10
)

// SetupRouter initializes the routes for the application
func SetupRouter(db *gorm.DB) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.SlogRequestMiddleware(config.GlobalConfig.Logging.ExcludePaths))

	// HTTP metrics middleware — kept as one of the outermost middlewares so the
	// recorded duration/response size reflect what the client actually saw,
	// including gzip, CORS and auth overhead. The /metrics endpoint itself is
	// served on a separate HTTP server (see cmd/server/main.go).
	if config.GlobalConfig.Metrics.Enabled {
		r.Use(metrics.Middleware(config.GlobalConfig.Metrics.ExcludePaths))
	}

	// CORS middleware
	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Content-Encoding", "If-None-Match"},
		ExposeHeaders:    []string{"Content-Length", "Content-Encoding", "ETag"},
		AllowCredentials: false,
	}))

	// Gzip middleware – decompress incoming gzip request bodies and
	// compress outgoing responses when the client supports it.
	r.Use(middleware.GzipRequestMiddleware())
	r.Use(middleware.MaxRequestBodyMiddleware(MaxRawRequestBodySize)) // 10 MB after decompression
	r.Use(middleware.GzipResponseMiddleware())

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

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "up",
		})
	})

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v2 := r.Group("/api/v2")
	{
		// Public routes
		v2.POST("/user/register", middleware.RateLimitMiddleware(RegisterEndpointRequestPerMinute, time.Minute), userCtrl.Register)
		v2.POST("/user/login", middleware.RateLimitMiddleware(LoginEndpointRequestPerMinute, time.Minute), userCtrl.Login)
		v2.POST("/user/refresh", userCtrl.RefreshToken)
		v2.GET("/songs", songCtrl.GetAllCharts)
		v2.GET("/songs/:song_id", songCtrl.GetSingleSongInfo)

		// Routes with optional auth
		optionalAuth := v2.Group("")
		optionalAuth.Use(middleware.OptionalAuthMiddleware(userService))
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
		auth.Use(middleware.AuthMiddleware(userService))
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
