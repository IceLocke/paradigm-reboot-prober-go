package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"paradigm-reboot-prober-go/config"
	"paradigm-reboot-prober-go/internal/router"
	"paradigm-reboot-prober-go/internal/util"
	"syscall"
	"time"
)

// @title           Paradigm: Reboot Prober API
// @version         2
// @description     This is the API documentation for the Paradigm: Reboot Prober service.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      api.prp.icel.site
// @schemes   https
// @BasePath  /api/v2
func main() {
	// Load Configuration
	config.LoadConfig("config/config.yaml")

	// Validate JWT secret key
	if config.GlobalConfig.Auth.SecretKey == "your_secret_key_here" {
		log.Fatal("JWT secret key must be changed from default value")
	}

	// Initialize Database
	util.InitDB()

	r := router.SetupRouter(util.DB)

	srv := &http.Server{
		Addr:    config.GlobalConfig.Server.Port,
		Handler: r,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	log.Println("Shutting down server...")

	// Give outstanding requests 10 seconds to complete
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced shutdown: %v", err)
	}
	log.Println("Server exited")
}
