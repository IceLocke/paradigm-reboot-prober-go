package main

// The `analyze` subcommand is a READ-ONLY diagnostic tool.
//
// Usage:
//
//	go run cmd/fitting/main.go analyze -chart 51
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
	"sort"

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
	util.InitDB()

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

	// 2. Load samples and skills (inline queries — we don't need paging for a single chart).
	samples := analyzeLoadSamples(ctx, *chartID)
	fmt.Printf("total raw samples: %d\n\n", len(samples))

	// 3. Bucket breakdown: who's playing, what they're scoring, what their skill is.
	analyzePrintBuckets(chart.Level, samples)

	// 4. Run ComputeFitting under several configs.
	type cfg struct {
		name   string
		params fitting.Params
		filter func(fitting.Sample) bool // optional pre-filter applied BEFORE ComputeFitting
	}
	// The "base" Params below are pulled from the loaded config (same values the
	// `run` subcommand uses in production), so the diagnostic reflects the
	// shipping algorithm. To probe knob changes, derive a Params from `base` and
	// override just the field(s) under investigation.
	fp := config.GlobalConfig.Fitting
	base := fitting.Params{
		MinEffectiveSamples: fp.MinSamples,
		ProximitySigma:      fp.ProximitySigma,
		HighSkillSigmaRatio: fp.HighSkillSigmaRatio,
		VolumeFullAt:        fp.VolumeFullAt,
		PriorStrength:       fp.PriorStrength,
		DeviationPenalty:    fp.DeviationPenalty,
		MaxDeviation:        fp.MaxDeviation,
		MaxDeviationLow:     fp.MaxDeviationLow,
		MaxDeviationLowAt:   fp.MaxDeviationLowAt,
		MaxDeviationHighAt:  fp.MaxDeviationHighAt,
		MinScore:            fp.MinScore,
		TukeyK:              fp.TukeyK,
		MinPlayerRecords:    fp.MinPlayerRecords,
	}
	withRatio := func(p fitting.Params, r float64) fitting.Params { p.HighSkillSigmaRatio = r; return p }
	flatCap := base // pre-ramp behaviour (flat ±MaxDeviation); useful for before/after comparison
	flatCap.MaxDeviationLow = 0

	configs := []cfg{
		{fmt.Sprintf("base (α=%.2f, ramp)", base.HighSkillSigmaRatio), base, nil},
		{fmt.Sprintf("base (α=%.2f, flat cap)", base.HighSkillSigmaRatio), flatCap, nil},
		{"α=0.3", withRatio(base, 0.3), nil},
		{"α=0.2", withRatio(base, 0.2), nil},
		{"α=0.15", withRatio(base, 0.15), nil},
		{"α=0.1", withRatio(base, 0.1), nil},
		{"score<1000000 only", base, func(s fitting.Sample) bool { return s.Score < 1000000 }},
		{"score<1005000 only", base, func(s fitting.Sample) bool { return s.Score < 1005000 }},
	}

	fmt.Println("\n=== ComputeFitting results ===")
	fmt.Println()
	fmt.Printf("%-32s %-8s %-8s %-8s %-8s %-8s %-8s\n",
		"config", "raw", "nEff", "wmed", "wmean", "sd", "fit")
	fmt.Println(analyzeRepeat("-", 84))
	for _, c := range configs {
		in := samples
		if c.filter != nil {
			filt := make([]fitting.Sample, 0, len(samples))
			for _, s := range samples {
				if c.filter(s) {
					filt = append(filt, s)
				}
			}
			in = filt
		}
		r := fitting.ComputeFitting(chart.Level, in, c.params)
		fit := "nil"
		if r.FittingLevel != nil {
			fit = fmt.Sprintf("%.3f", *r.FittingLevel)
		}
		fmt.Printf("%-32s %-8d %-8.1f %-8.3f %-8.3f %-8.3f %-8s\n",
			c.name, r.SampleCount, r.EffectiveSampleSize,
			r.WeightedMedian, r.WeightedMean, r.StdDev, fit)
	}
}

// analyzeLoadSamples reads best_play_records + play_records for `chartID`
// and joins with each player's top-50 average rating (same definition the
// runner uses). Returns a ready-to-feed []fitting.Sample.
//
// Prefixed `analyze*` so the symbol cannot collide with anything in the
// sibling run.go file.
func analyzeLoadSamples(ctx context.Context, chartID int) []fitting.Sample {
	type row struct {
		Username string
		Score    int
	}
	var raw []row
	if err := util.DB.WithContext(ctx).
		Table("best_play_records").
		Select("best_play_records.username AS username, play_records.score AS score").
		Joins("JOIN play_records ON play_records.id = best_play_records.play_record_id").
		Where("best_play_records.chart_id = ?", chartID).
		Where("best_play_records.deleted_at IS NULL").
		Where("play_records.deleted_at IS NULL").
		Scan(&raw).Error; err != nil {
		fmt.Fprintf(os.Stderr, "fetch samples failed: %v\n", err)
		os.Exit(1)
	}

	if len(raw) == 0 {
		return nil
	}

	userSet := make(map[string]struct{}, len(raw))
	for _, r := range raw {
		userSet[r.Username] = struct{}{}
	}
	usernames := make([]string, 0, len(userSet))
	for u := range userSet {
		usernames = append(usernames, u)
	}
	sort.Strings(usernames)

	type ratingRow struct {
		Username string
		Rating   int
	}
	var ratings []ratingRow
	if err := util.DB.WithContext(ctx).
		Table("play_records").
		Select("play_records.username AS username, play_records.rating AS rating").
		Joins("JOIN best_play_records ON best_play_records.play_record_id = play_records.id").
		Where("play_records.username IN ?", usernames).
		Where("play_records.deleted_at IS NULL").
		Where("best_play_records.deleted_at IS NULL").
		Order("play_records.username ASC, play_records.rating DESC").
		Scan(&ratings).Error; err != nil {
		fmt.Fprintf(os.Stderr, "fetch ratings failed: %v\n", err)
		os.Exit(1)
	}

	type skill struct {
		avgRating float64
		n         int
	}
	skills := make(map[string]skill, len(usernames))
	curUser := ""
	buf := make([]int, 0, 64)
	total := 0
	flush := func() {
		if curUser == "" {
			return
		}
		k := buf
		if len(k) > 50 {
			k = k[:50]
		}
		sum := 0
		for _, v := range k {
			sum += v
		}
		avg := 0.0
		if len(k) > 0 {
			avg = float64(sum) / float64(len(k)) / 100.0
		}
		skills[curUser] = skill{avg, total}
	}
	for _, r := range ratings {
		if r.Username != curUser {
			flush()
			curUser = r.Username
			buf = buf[:0]
			total = 0
		}
		total++
		if len(buf) < 50 {
			buf = append(buf, r.Rating)
		}
	}
	flush()

	out := make([]fitting.Sample, 0, len(raw))
	for _, r := range raw {
		s, ok := skills[r.Username]
		if !ok {
			continue
		}
		out = append(out, fitting.Sample{
			Username:      r.Username,
			Score:         r.Score,
			PlayerSkill:   s.avgRating,
			PlayerRecords: s.n,
		})
	}
	return out
}

// analyzePrintBuckets splits samples by score into canonical buckets and
// reports, per bucket: count, avg player skill, avg inferred level, and
// avg (current default) proximity weight with α=0.5. This is the single
// most useful view for seeing WHY fitting is being pulled away from the
// official level.
func analyzePrintBuckets(official float64, samples []fitting.Sample) {
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
		"score bucket", "n", "avg_skill", "avg_infL", "avg_prox(α=0.5)")
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
			sigma := 20.0
			if diff > 0 {
				sigma = 10.0 // α=0.5
			}
			prox := math.Exp(-(diff * diff) / (2.0 * sigma * sigma))
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
