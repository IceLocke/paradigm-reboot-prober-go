package main

import (
	"context"
	"io"
	"log/slog"

	"paradigm-reboot-prober-go/config"
	"paradigm-reboot-prober-go/internal/logging"
)

// setupFittingLogging applies the shared logging config and reports the
// effective fitting settings after file and environment overrides.
func setupFittingLogging(command string) (context.Context, io.Closer) {
	cfg := config.GlobalConfig
	closer, err := logging.Setup(cfg.Logging.Output, cfg.Logging.File, cfg.Logging.Format)
	if err != nil {
		panic(err)
	}
	ctx := logging.AppendCtx(context.Background(),
		slog.String("component", "fitting"),
		slog.String("command", command),
	)
	fp := cfg.Fitting
	slog.InfoContext(ctx, "fitting configuration loaded",
		"enabled", fp.Enabled,
		"interval", config.FittingIntervalDuration.String(),
		"skill_top_k", fp.SkillTopK,
		"calibration_enabled", fp.CalibrationEnabled,
		"calibration_noise_penalty", fp.CalibrationNoisePenalty,
		"balance_total", fp.BalanceTotal,
		"min_samples", fp.MinSamples,
		"min_player_records", fp.MinPlayerRecords,
		"sample_halflife_days", fp.SampleHalflifeDays,
		"proximity_sigma", fp.ProximitySigma,
		"high_skill_sigma_ratio", fp.HighSkillSigmaRatio,
		"prior_strength", fp.PriorStrength,
		"deviation_penalty", fp.DeviationPenalty,
		"max_deviation", fp.MaxDeviation,
		"max_deviation_low", fp.MaxDeviationLow,
		"max_deviation_low_at", fp.MaxDeviationLowAt,
		"max_deviation_high_at", fp.MaxDeviationHighAt,
	)
	return ctx, closer
}
