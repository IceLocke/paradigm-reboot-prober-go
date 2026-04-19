package main

import (
	"context"
	"log/slog"
	"net/http"
	"os/signal"
	"paradigm-reboot-prober-go/config"
	"paradigm-reboot-prober-go/internal/logging"
	"paradigm-reboot-prober-go/internal/metrics"
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
		slog.Error("JWT secret key must be changed from default value")
		panic("JWT secret key must be changed from default value")
	}

	// Initialize structured logging
	logCloser, err := logging.Setup(
		config.GlobalConfig.Logging.Output,
		config.GlobalConfig.Logging.File,
		config.GlobalConfig.Logging.Format,
	)
	if err != nil {
		panic(err)
	}
	defer func() { _ = logCloser.Close() }()

	// Initialize Database
	util.InitDB()

	r := router.SetupRouter(util.DB)

	srv := &http.Server{
		Addr:    config.GlobalConfig.Server.Port,
		Handler: r,
	}

	// Start API server in a goroutine
	go func() {
		slog.Info("server starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("failed to start server", "error", err)
			panic(err)
		}
	}()

	// Start dedicated metrics server on a separate port so Prometheus scraping
	// is not exposed on the public API port.
	var metricsSrv *http.Server
	if config.GlobalConfig.Metrics.Enabled {
		mux := http.NewServeMux()
		mux.Handle(config.GlobalConfig.Metrics.Path, metrics.Handler())
		metricsSrv = &http.Server{
			Addr:              config.GlobalConfig.Metrics.Addr,
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		}
		go func() {
			slog.Info("metrics server starting",
				"addr", metricsSrv.Addr,
				"path", config.GlobalConfig.Metrics.Path,
			)
			if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				slog.Error("failed to start metrics server", "error", err)
				panic(err)
			}
		}()
	}

	// Wait for interrupt signal
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	slog.Info("shutting down server...")

	// Give outstanding requests 10 seconds to complete
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced shutdown", "error", err)
		panic(err)
	}
	if metricsSrv != nil {
		if err := metricsSrv.Shutdown(shutdownCtx); err != nil {
			slog.Error("metrics server forced shutdown", "error", err)
		}
	}
	slog.Info("server exited")
}
