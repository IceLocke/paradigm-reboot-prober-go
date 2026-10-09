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
	"paradigm-reboot-prober-go/internal/util"
)

// cmdRun executes the `run` subcommand: either a single fitting pass
// (--once) or the continuous mode driven by config.fitting.interval.
//
// This is also the default subcommand invoked when no subcommand keyword
// is present on the command line, so `./fitting`, `./fitting --once`, and
// `go run ./cmd/fitting --config foo.yaml` all route here unchanged from
// the pre-subcommand behaviour.
func cmdRun(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	configPath := fs.String("config", "config/config.yaml", "Path to config file")
	once := fs.Bool("once", false, "Run the calculator once and exit (ignores the ticker loop)")
	_ = fs.Parse(args)

	// 1. Shared config (same file as cmd/server).
	config.LoadConfig(*configPath)

	// 2. Shared structured logging.
	baseCtx, logCloser := setupFittingLogging("run")
	defer func() { _ = logCloser.Close() }()

	// 3. The master switch skips continuous mode; --once runs explicitly.
	if !config.GlobalConfig.Fitting.Enabled && !*once {
		slog.InfoContext(baseCtx, "fitting disabled by config.fitting.enabled; exiting")
		return
	}

	// 4. Open the shared DB (AutoMigrate applied, including chart_statistics).
	util.InitDB()

	// 5. Build the runner.
	params := configuredParams()
	fp := config.GlobalConfig.Fitting
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
	runTick(loopCtx, runner)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-loopCtx.Done():
			slog.InfoContext(loopCtx, "fitting loop shutting down")
			return
		case <-ticker.C:
			runTick(loopCtx, runner)
		}
	}
}

// runTick invokes one fitting pass and logs any error, never panicking in the
// continuous loop — a single transient DB hiccup should not kill the
// microservice.
func runTick(ctx context.Context, runner *fitting.Runner) {
	report, err := runner.Run(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "fitting run failed",
			"err", err,
			"duration_ms", report.Duration.Milliseconds(),
		)
		return
	}
}

// configuredParams is shared by run and analyze.
func configuredParams() fitting.Params {
	fp := config.GlobalConfig.Fitting
	return fitting.Params{
		MinEffectiveSamples:     fp.MinSamples,
		CalibrationEnabled:      fp.CalibrationEnabled,
		CalibrationNoisePenalty: fp.CalibrationNoisePenalty,
		BalanceTotal:            fp.BalanceTotal,
		SkillTopK:               fp.SkillTopK,
		SampleHalflifeDays:      fp.SampleHalflifeDays,
		ProximitySigma:          fp.ProximitySigma,
		HighSkillSigmaRatio:     fp.HighSkillSigmaRatio,
		VolumeFullAt:            fp.VolumeFullAt,
		PriorStrength:           fp.PriorStrength,
		DeviationPenalty:        fp.DeviationPenalty,
		MaxDeviation:            fp.MaxDeviation,
		MaxDeviationLow:         fp.MaxDeviationLow,
		MaxDeviationLowAt:       fp.MaxDeviationLowAt,
		MaxDeviationHighAt:      fp.MaxDeviationHighAt,
		MinScore:                fp.MinScore,
		ScoreFloorAt:            fp.ScoreFloorAt,
		ScoreGoodAt:             fp.ScoreGoodAt,
		ScoreFullAt:             fp.ScoreFullAt,
		ScoreGoodWeight:         fp.ScoreGoodWeight,
		TukeyK:                  fp.TukeyK,
		MinPlayerRecords:        fp.MinPlayerRecords,
	}
}
