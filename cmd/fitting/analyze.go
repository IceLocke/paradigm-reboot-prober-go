package main

// The analyze subcommand replays the full fitting population without writes,
// then displays score buckets, robust statistics and the final target estimate.

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"paradigm-reboot-prober-go/config"
	"paradigm-reboot-prober-go/internal/fitting"
	"paradigm-reboot-prober-go/internal/logging"
	"paradigm-reboot-prober-go/internal/model"
	"paradigm-reboot-prober-go/internal/util"
)

func cmdAnalyze(args []string) {
	fs := flag.NewFlagSet("analyze", flag.ExitOnError)
	configPath := fs.String("config", "config/config.yaml", "Path to config file")
	chartID := fs.Int("chart", 0, "Chart ID to analyze (required)")
	_ = fs.Parse(args)
	if *chartID == 0 {
		slog.Error("-chart is required")
		os.Exit(2)
	}

	config.LoadConfig(*configPath)
	ctx, logCloser := setupFittingLogging("analyze")
	defer func() { _ = logCloser.Close() }()
	ctx = logging.AppendCtx(ctx, slog.Int("chart_id", *chartID))
	util.ConnectDB()

	// 1. Load chart metadata.
	var chart model.Chart
	if err := util.DB.WithContext(ctx).
		Select("id, level, song_id, difficulty").
		First(&chart, *chartID).Error; err != nil {
		slog.ErrorContext(ctx, "failed to load chart", "err", err)
		os.Exit(1)
	}
	slog.InfoContext(ctx, "analyzing chart", "official_level", chart.Level, "difficulty", chart.Difficulty)

	base := configuredParams()
	runner := fitting.NewRunner(util.DB, base, fitting.RunnerConfig{})
	analysis, err := runner.AnalyzeChart(ctx, *chartID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to analyze chart", "err", err)
		os.Exit(1)
	}
	samples := analysis.Samples
	slog.InfoContext(ctx, "chart samples loaded", "samples", len(samples))
	analyzeLogBuckets(ctx, samples)
	slog.InfoContext(ctx, "published population",
		"charts", analysis.Balance.Charts,
		"official_sum", analysis.Balance.OfficialSum,
		"fitting_sum", analysis.Balance.FittingSum,
		"balance_offset", analysis.Balance.Offset,
	)
	result := analysis.Result
	slog.InfoContext(ctx, "chart statistics",
		"sample_count", result.SampleCount,
		"effective_sample_size", result.EffectiveSampleSize,
		"weighted_median", result.WeightedMedian,
		"weighted_mean", result.WeightedMean,
		"stddev", result.StdDev,
		"mad", result.MAD,
	)
	if result.FittingLevel == nil {
		slog.InfoContext(ctx, "chart fitting result",
			"fitting_level", nil,
			"reason", "insufficient samples or peer support",
		)
	} else {
		independent := fitting.ComputeFitting(chart.Level, samples, base)
		var independentLevel any
		if independent.FittingLevel != nil {
			independentLevel = *independent.FittingLevel
		}
		slog.InfoContext(ctx, "chart fitting result",
			"independent_fitting_level", independentLevel,
			"fitting_level", *result.FittingLevel,
		)
	}
}

// Score buckets show raw inverse levels and the supported cohort corrections.
// AP is excluded from calibrated fitting and appears here for diagnostics.
func analyzeLogBuckets(ctx context.Context, samples []fitting.Sample) {
	buckets := []struct {
		label string
		lo    int
		hi    int // inclusive
	}{
		{"< 900000", 0, 899999},
		{"900000-990000", 900000, 989999},
		{"990000-1000000", 990000, 999999},
		{"1000000-1005000", 1000000, 1004999},
		{"1005000-1008000", 1005000, 1007999},
		{"1008000-1009000", 1008000, 1008999},
		{"1009000-1009500", 1009000, 1009499},
		{"1009500-1009999", 1009500, 1009999},
		{"AP (=1010000)", 1010000, 1010000},
	}

	for _, bucket := range buckets {
		var count, supported int
		var skillSum, rawSum, correctedSum float64
		for _, sample := range samples {
			if sample.Score < bucket.lo || sample.Score > bucket.hi {
				continue
			}
			inferred, ok := fitting.InverseLevel(sample.Score, sample.PlayerSkill)
			if !ok {
				continue
			}
			count++
			skillSum += sample.PlayerSkill
			rawSum += inferred
			if sample.LevelCorrection != nil && sample.Score < 1010000 {
				supported++
				correctedSum += inferred - *sample.LevelCorrection
			}
		}
		if count == 0 {
			continue
		}
		var corrected any
		if supported > 0 {
			corrected = correctedSum / float64(supported)
		}
		slog.InfoContext(ctx, "score bucket",
			"score_bucket", bucket.label,
			"samples", count,
			"avg_skill", skillSum/float64(count),
			"avg_inferred_level", rawSum/float64(count),
			"reference_samples", supported,
			"avg_corrected_level", corrected,
		)
	}
}
