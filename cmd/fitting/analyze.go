package main

// The `analyze` subcommand is a READ-ONLY diagnostic tool.
//
// Usage:
//
//	go run ./cmd/fitting analyze -chart 51
//
// It connects to the same database as the `run` subcommand, loads all
// best_play_records for the given chart joined with the per-player skill
// snapshot, and prints:
//
//  1. A per-score-bucket breakdown of the sample set (count, avg skill,
//     average inferred level) so you can see WHERE the bias lives.
//  2. The output of fitting.ComputeFitting under several diagnostic Params
//     configurations (status quo vs candidate fixes), side-by-side.
//
// The subcommand writes nothing back to the database. It is safe to run
// against production. It is intentionally not driven by any scheduler — it
// exists to debug distribution problems uncovered during tuning. If the
// tool ever becomes obsolete, delete this file; none of its symbols are
// referenced from the `run` subcommand.

import (
	"context"
	"flag"
	"fmt"
	"math"
	"os"

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
	samples, err := runner.LoadChartSamples(ctx, *chartID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load samples: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("total samples: %d; skill top-K: %d\n\n", len(samples), base.SkillTopK)
	analyzePrintBuckets(chart.Level, samples, base)
	type cfg struct {
		name   string
		params fitting.Params
	}
	legacy := base
	legacy.CalibrationEnabled = false
	unscaled := base
	unscaled.CalibrationScale = 1
	configs := []cfg{
		{"configured", base},
		{"legacy (same top-K)", legacy},
		{"calibrated, gain=1", unscaled},
	}
	fmt.Println("\n=== ComputeFitting results ===")
	fmt.Println()
	fmt.Printf("%-32s %-8s %-8s %-8s %-8s %-8s %-8s\n",
		"config", "raw", "nEff", "wmed", "wmean", "sd", "fit")
	fmt.Println(analyzeRepeat("-", 84))
	for _, c := range configs {
		r := fitting.ComputeFitting(chart.Level, samples, c.params)
		fit := "nil"
		if r.FittingLevel != nil {
			fit = fmt.Sprintf("%.3f", *r.FittingLevel)
		}
		fmt.Printf("%-32s %-8d %-8.1f %-8.3f %-8.3f %-8.3f %-8s\n",
			c.name, r.SampleCount, r.EffectiveSampleSize,
			r.WeightedMedian, r.WeightedMean, r.StdDev, fit)
	}
}

// analyzePrintBuckets splits samples by score into canonical buckets and
// reports, per bucket: count, avg player skill, avg inferred level, and
// avg configured proximity weight. This is the single
// most useful view for seeing WHY fitting is being pulled away from the
// official level.
func analyzePrintBuckets(official float64, samples []fitting.Sample, params fitting.Params) {
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

	fmt.Printf("=== per-score-bucket breakdown (official level = %.1f) ===\n\n", official)
	fmt.Printf("%-20s %-6s %-10s %-10s %-12s\n",
		"score bucket", "n", "avg_skill", "avg_infL", "avg_prox")
	fmt.Println(analyzeRepeat("-", 62))

	for _, b := range buckets {
		var n int
		var sumSkill, sumInfL, sumProx float64
		for _, s := range samples {
			if s.Score < b.lo || s.Score > b.hi {
				continue
			}
			inferred, ok := fitting.InverseLevel(s.Score, s.PlayerSkill)
			if !ok {
				continue
			}
			diff := s.PlayerSkill - 10.0*official
			sigma := params.ProximitySigma
			if diff > 0 && params.HighSkillSigmaRatio > 0 {
				sigma *= params.HighSkillSigmaRatio
			}
			prox := math.Exp(-(diff * diff) / (2.0 * sigma * sigma))
			if math.Abs(diff) > 2.5*sigma {
				prox = 0
			}
			n++
			sumSkill += s.PlayerSkill
			sumInfL += inferred
			sumProx += prox
		}
		if n == 0 {
			continue
		}
		fmt.Printf("%-20s %-6d %-10.2f %-10.3f %-12.3f\n",
			b.label, n, sumSkill/float64(n), sumInfL/float64(n), sumProx/float64(n))
	}
}

func analyzeRepeat(s string, n int) string {
	out := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}
