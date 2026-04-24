package fitting

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"paradigm-reboot-prober-go/internal/model"

	"gorm.io/gorm"
)

// RunnerConfig bundles the non-Params runtime knobs (things that change
// *how* the run iterates the DB rather than *what* fitting values come out).
type RunnerConfig struct {
	ChartBatchSize  int
	PlayerBatchSize int
	BatchPause      time.Duration
}

// Runner orchestrates a single offline fitting pass across the entire charts
// table. Exposed as a reusable type so that the cmd/fitting binary can use
// it from both a ticker loop and a one-shot execution (`--once`).
type Runner struct {
	db      *gorm.DB
	params  Params
	cfg     RunnerConfig
	nowFunc func() time.Time // injectable "now" for testing sample-age decay; defaults to time.Now
}

// NewRunner constructs a runner. `db` is expected to already have the shared
// schema (util.InitDB or equivalent AutoMigrate).
func NewRunner(db *gorm.DB, params Params, cfg RunnerConfig) *Runner {
	if cfg.ChartBatchSize <= 0 {
		cfg.ChartBatchSize = 200
	}
	if cfg.PlayerBatchSize <= 0 {
		cfg.PlayerBatchSize = 500
	}
	return &Runner{db: db, params: params, cfg: cfg, nowFunc: time.Now}
}

// now returns the runner's reference time. Uses nowFunc when set, otherwise
// falls back to time.Now so a zero-valued Runner still works (defensive).
func (r *Runner) now() time.Time {
	if r.nowFunc != nil {
		return r.nowFunc()
	}
	return time.Now()
}

// RunReport summarizes one execution, useful for logging and tests.
type RunReport struct {
	Started           time.Time
	Completed         time.Time
	Duration          time.Duration
	PlayersConsidered int
	ChartsTotal       int
	ChartsProcessed   int
	ChartsPublished   int // FittingLevel persisted to charts.fitting_level
	ChartsAbstained   int // insufficient samples → nil fitting
	ChartsEmpty       int // no samples at all
	ErrorsEncountered int
}

// Run executes one pass: build player-skill cache → iterate charts in
// batches → compute & persist fitting levels + statistics. Any context
// cancellation aborts promptly; partial progress stays persisted (updates
// are committed per-chart, not per-batch).
//
// Named returns so the deferred finalizer can inspect err and emit a
// per-outcome log line (errors → ERROR, otherwise INFO), plus unconditionally
// stamp report.Completed / report.Duration regardless of exit path.
func (r *Runner) Run(ctx context.Context) (report RunReport, err error) {
	report.Started = time.Now()
	slog.InfoContext(ctx, "fitting run starting",
		"chart_batch_size", r.cfg.ChartBatchSize,
		"player_batch_size", r.cfg.PlayerBatchSize,
		"batch_pause_ms", r.cfg.BatchPause.Milliseconds(),
	)
	defer func() {
		report.Completed = time.Now()
		report.Duration = report.Completed.Sub(report.Started)
		attrs := []any{
			"duration_ms", report.Duration.Milliseconds(),
			"players_considered", report.PlayersConsidered,
			"charts_total", report.ChartsTotal,
			"charts_processed", report.ChartsProcessed,
			"charts_published", report.ChartsPublished,
			"charts_abstained", report.ChartsAbstained,
			"charts_empty", report.ChartsEmpty,
			"errors", report.ErrorsEncountered,
		}
		if err != nil {
			slog.ErrorContext(ctx, "fitting run failed", append(attrs, "err", err)...)
		} else {
			slog.InfoContext(ctx, "fitting run completed", attrs...)
		}
	}()

	// 1. Player-skill snapshot (single pass over best_play_records).
	skills, err := r.collectPlayerSkills(ctx)
	if err != nil {
		return report, fmt.Errorf("collect player skills: %w", err)
	}
	report.PlayersConsidered = len(skills)
	slog.InfoContext(ctx, "player skills collected", "players", len(skills))

	// 2. Pull the full chart list up front (bounded — charts is a small table).
	charts, err := r.fetchChartsSorted(ctx)
	if err != nil {
		return report, fmt.Errorf("fetch charts: %w", err)
	}
	report.ChartsTotal = len(charts)

	// 3. Batch-process charts.
	for start := 0; start < len(charts); start += r.cfg.ChartBatchSize {
		if err := ctx.Err(); err != nil {
			return report, err
		}
		end := start + r.cfg.ChartBatchSize
		if end > len(charts) {
			end = len(charts)
		}
		batch := charts[start:end]

		chartIDs := make([]int, len(batch))
		levelByID := make(map[int]float64, len(batch))
		for i, c := range batch {
			chartIDs[i] = c.ID
			levelByID[c.ID] = c.Level
		}

		samplesByChart, err := r.fetchBestSamples(ctx, chartIDs, skills)
		if err != nil {
			slog.ErrorContext(ctx, "fetch best samples batch failed",
				"batch_start", start, "err", err)
			report.ErrorsEncountered++
			continue
		}

		for _, c := range batch {
			if err := ctx.Err(); err != nil {
				return report, err
			}
			samples := samplesByChart[c.ID]
			res := ComputeFitting(c.Level, samples, r.params)
			report.ChartsProcessed++
			if len(samples) == 0 {
				report.ChartsEmpty++
			} else if res.FittingLevel == nil {
				report.ChartsAbstained++
			} else {
				report.ChartsPublished++
			}

			if err := r.persist(ctx, c.ID, c.Level, res); err != nil {
				slog.ErrorContext(ctx, "persist fitting result failed",
					"chart_id", c.ID, "err", err)
				report.ErrorsEncountered++
			}
		}

		if r.cfg.BatchPause > 0 && end < len(charts) {
			select {
			case <-ctx.Done():
				return report, ctx.Err()
			case <-time.After(r.cfg.BatchPause):
			}
		}
	}

	return report, nil
}

// persist writes charts.fitting_level and upserts chart_statistics. It runs
// inside a short per-chart transaction so a long run does not hold large
// locks; the main probe server keeps serving live queries.
func (r *Runner) persist(ctx context.Context, chartID int, officialLevel float64, res Result) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Update charts.fitting_level. A nil FittingLevel persists NULL,
		//    explicitly signalling "abstained" to downstream consumers.
		if err := tx.Model(&model.Chart{}).
			Where("id = ?", chartID).
			Update("fitting_level", res.FittingLevel).Error; err != nil {
			return fmt.Errorf("update chart %d: %w", chartID, err)
		}

		// 2. Upsert chart_statistics. We use two-step read-modify-write so the
		//    logic is identical across SQLite and PostgreSQL (no Clauses/
		//    OnConflict syntax divergence). Contention is irrelevant here —
		//    only the fitting binary writes this table.
		now := time.Now()
		stat := model.ChartStatistic{
			ChartID:             chartID,
			OfficialLevel:       officialLevel,
			FittingLevel:        res.FittingLevel,
			SampleCount:         res.SampleCount,
			EffectiveSampleSize: res.EffectiveSampleSize,
			WeightedMean:        res.WeightedMean,
			WeightedMedian:      res.WeightedMedian,
			StdDev:              res.StdDev,
			MAD:                 res.MAD,
			LastComputedAt:      now,
		}
		var existing model.ChartStatistic
		err := tx.Where("chart_id = ?", chartID).First(&existing).Error
		switch {
		case err == nil:
			stat.CreatedAt = existing.CreatedAt // preserve initial observation time
			if err := tx.Model(&existing).Updates(map[string]interface{}{
				"official_level":        stat.OfficialLevel,
				"fitting_level":         stat.FittingLevel,
				"sample_count":          stat.SampleCount,
				"effective_sample_size": stat.EffectiveSampleSize,
				"weighted_mean":         stat.WeightedMean,
				"weighted_median":       stat.WeightedMedian,
				"std_dev":               stat.StdDev,
				"mad":                   stat.MAD,
				"last_computed_at":      stat.LastComputedAt,
			}).Error; err != nil {
				return fmt.Errorf("update chart_statistics %d: %w", chartID, err)
			}
		case errors.Is(err, gorm.ErrRecordNotFound):
			if err := tx.Create(&stat).Error; err != nil {
				return fmt.Errorf("insert chart_statistics %d: %w", chartID, err)
			}
		default:
			return fmt.Errorf("read chart_statistics %d: %w", chartID, err)
		}
		return nil
	})
}
