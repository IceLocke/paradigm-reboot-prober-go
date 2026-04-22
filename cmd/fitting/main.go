// Package main is the entry point of the fitting-calculator microservice.
//
// The fitting calculator is intentionally a SEPARATE binary from cmd/server
// so that the probe service ("查分器") remains single-purpose. See AGENTS.md
// → "保持查分器本体的单纯性" for the design principle. This binary:
//
//   - Reads best_play_records and chart metadata directly from the shared
//     database using the shared config + model layers, but has NO
//     dependency on internal/service or internal/repository (no HTTP
//     handlers, no caches, no auth logic).
//   - Runs on a configurable ticker interval (config.fitting.interval,
//     typically hours) or once with the --once flag.
//   - Persists results into charts.fitting_level and a dedicated
//     chart_statistics table for offline analysis.
package main

import (
	"context"
	"flag"
	"log/slog"
	"os/signal"
	"syscall"
	"time"

	"paradigm-reboot-prober-go/config"
	"paradigm-reboot-prober-go/internal/fitting"
	"paradigm-reboot-prober-go/internal/logging"
	"paradigm-reboot-prober-go/internal/util"
)

func main() {
	configPath := flag.String("config", "config/config.yaml", "Path to config file")
	once := flag.Bool("once", false, "Run the calculator once and exit (ignores the ticker loop)")
	flag.Parse()

	// 1. Shared config (same file as cmd/server).
	config.LoadConfig(*configPath)

	// 2. Shared structured logging.
	logCloser, err := logging.Setup(
		config.GlobalConfig.Logging.Output,
		config.GlobalConfig.Logging.File,
		config.GlobalConfig.Logging.Format,
	)
	if err != nil {
		panic(err)
	}
	defer func() { _ = logCloser.Close() }()

	// Attach a stable component attribute so fitting logs are easy to filter.
	baseCtx := logging.AppendCtx(context.Background(),
		slog.String("component", "fitting"),
	)

	// 3. Master switch: when disabled we still initialize the DB so
	//    AutoMigrate keeps chart_statistics in sync with the schema, but we
	//    skip all actual work.
	if !config.GlobalConfig.Fitting.Enabled && !*once {
		slog.InfoContext(baseCtx, "fitting disabled by config.fitting.enabled; exiting")
		return
	}

	// 4. Open the shared DB (AutoMigrate applied, including chart_statistics).
	util.InitDB()

	// 5. Build the runner.
	fp := config.GlobalConfig.Fitting
	params := fitting.Params{
		MinEffectiveSamples: fp.MinSamples,
		ProximitySigma:      fp.ProximitySigma,
		VolumeFullAt:        fp.VolumeFullAt,
		PriorStrength:       fp.PriorStrength,
		MaxDeviation:        fp.MaxDeviation,
		MinScore:            fp.MinScore,
		TukeyK:              fp.TukeyK,
		MinPlayerRecords:    fp.MinPlayerRecords,
	}
	cfg := fitting.RunnerConfig{
		ChartBatchSize:  fp.ChartBatchSize,
		PlayerBatchSize: fp.PlayerBatchSize,
		BatchPause:      config.FittingBatchPauseDuration,
	}
	runner := fitting.NewRunner(util.DB, params, cfg)

	// 6. --once: run a single pass and exit with status 0 on success.
	if *once {
		runCtx, cancel := signal.NotifyContext(baseCtx, syscall.SIGINT, syscall.SIGTERM)
		defer cancel()
		report, err := runner.Run(runCtx)
		if err != nil {
			slog.ErrorContext(runCtx, "fitting run failed",
				"err", err,
				"duration_ms", report.Duration.Milliseconds(),
			)
			panic(err)
		}
		return
	}

	// 7. Continuous mode: run once immediately, then every Fitting.Interval.
	loopCtx, stop := signal.NotifyContext(baseCtx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	interval := config.FittingIntervalDuration
	slog.InfoContext(loopCtx, "fitting loop starting",
		"interval", interval.String(),
	)

	// Immediate first run so there is no long initial idle delay after boot.
	runOnce(loopCtx, runner)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-loopCtx.Done():
			slog.InfoContext(loopCtx, "fitting loop shutting down")
			return
		case <-ticker.C:
			runOnce(loopCtx, runner)
		}
	}
}

// runOnce invokes one fitting pass and logs any error, never panicking in the
// continuous loop — a single transient DB hiccup should not kill the
// microservice.
func runOnce(ctx context.Context, runner *fitting.Runner) {
	report, err := runner.Run(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "fitting run failed",
			"err", err,
			"duration_ms", report.Duration.Milliseconds(),
		)
		return
	}
}
