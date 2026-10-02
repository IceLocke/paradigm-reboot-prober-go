package main

// The analyze subcommand replays the full fitting population without writes,
// then displays score buckets, robust statistics and the final target estimate.

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"paradigm-reboot-prober-go/config"
	"paradigm-reboot-prober-go/internal/fitting"
	"paradigm-reboot-prober-go/internal/model"
	"paradigm-reboot-prober-go/internal/util"
)

func cmdAnalyze(args []string) {
	fs := flag.NewFlagSet("analyze", flag.ExitOnError)
	configPath := fs.String("config", "config/config.yaml", "Path to config file")
	chartID := fs.Int("chart", 0, "Chart ID to analyze (required)")
	_ = fs.Parse(args)
	if *chartID == 0 {
		fmt.Fprintln(os.Stderr, "error: -chart is required")
		os.Exit(2)
	}

	config.LoadConfig(*configPath)
	util.ConnectDB()

	ctx := context.Background()

	// 1. Load chart metadata.
	var chart model.Chart
	if err := util.DB.WithContext(ctx).
		Select("id, level, song_id, difficulty").
		First(&chart, *chartID).Error; err != nil {
		fmt.Fprintf(os.Stderr, "failed to load chart %d: %v\n", *chartID, err)
		os.Exit(1)
	}
	fmt.Printf("=== chart %d | level=%.1f | difficulty=%s ===\n\n", chart.ID, chart.Level, chart.Difficulty)

	base := configuredParams()
	runner := fitting.NewRunner(util.DB, base, fitting.RunnerConfig{})
	analysis, err := runner.AnalyzeChart(ctx, *chartID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load samples: %v\n", err)
		os.Exit(1)
	}
	samples := analysis.Samples
	fmt.Printf("total samples: %d; skill top-K: %d\n\n", len(samples), base.SkillTopK)
	analyzePrintBuckets(samples)
	fmt.Printf("\nPublished population: %d charts; official total: %.6f; fitting total: %.6f; offset: %+.6f\n",
		analysis.Balance.Charts, analysis.Balance.OfficialSum, analysis.Balance.FittingSum, analysis.Balance.Offset)
	result := analysis.Result
	fmt.Printf("Raw inversions: %d; N_eff: %.2f; median: %.4f; mean: %.4f; SD: %.4f; MAD: %.4f\n",
		result.SampleCount, result.EffectiveSampleSize, result.WeightedMedian, result.WeightedMean, result.StdDev, result.MAD)
	if result.FittingLevel == nil {
		fmt.Println("Fitting level: nil (insufficient samples or peer support)")
	} else {
		independent := fitting.ComputeFitting(chart.Level, samples, base)
		fmt.Printf("Independent estimate: %.6f; final fitting level: %.6f\n", *independent.FittingLevel, *result.FittingLevel)
	}
}

// Score buckets show raw inverse levels and the supported cohort corrections.
// AP is excluded from calibrated fitting and appears here for diagnostics.
func analyzePrintBuckets(samples []fitting.Sample) {
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

	fmt.Println("=== per-score-bucket breakdown (AP excluded in calibrated mode) ===")
	fmt.Printf("%-20s %-6s %-10s %-10s %-12s %-10s\n",
		"score bucket", "n", "avg_skill", "avg_infL", "reference_n", "avg_corrL")
	fmt.Println(strings.Repeat("-", 78))
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
		corrected := "n/a"
		if supported > 0 {
			corrected = fmt.Sprintf("%.3f", correctedSum/float64(supported))
		}
		fmt.Printf("%-20s %-6d %-10.2f %-10.3f %-12d %-10s\n",
			bucket.label, count, skillSum/float64(count), rawSum/float64(count), supported, corrected)
	}
}
