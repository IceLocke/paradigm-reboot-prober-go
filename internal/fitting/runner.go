package fitting

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"paradigm-reboot-prober-go/internal/model"

	"github.com/samber/lo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
// cancellation aborts promptly; partial progress stays persisted (each
// completed batch is committed before the run moves on).
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

		persistItems := make([]persistItem, 0, len(batch))
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

			persistItems = append(persistItems, persistItem{chartID: c.ID, officialLevel: c.Level, result: res})
		}

		if err := r.persistBatch(ctx, persistItems); err != nil {
			slog.ErrorContext(ctx, "persist fitting batch failed",
				"batch_start", start, "batch_size", len(persistItems), "err", err)
			report.ErrorsEncountered += len(persistItems)
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

type persistItem struct {
	chartID       int
	officialLevel float64
	result        Result
}

// persistBatch writes one computed chart batch in a short transaction. The
// VALUES update and conflict-aware statistics insert reduce a cold start from
// several statements per chart to two statements per batch, which matters
// much more than local query time when the analytical binary and PostgreSQL
// are on different hosts.
func (r *Runner) persistBatch(ctx context.Context, items []persistItem) error {
	if len(items) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Portable across PostgreSQL and SQLite: both name the columns of a
		// bare VALUES derived table column1/column2 (column-alias lists like
		// `AS v(id, fitting_level)` are PostgreSQL-only), and the CASTs give
		// PostgreSQL's parameter-type inference the column types the bare
		// VALUES list cannot (no-op under SQLite's dynamic typing).
		placeholders := make([]string, len(items))
		args := make([]any, 0, len(items)*2)
		for i, item := range items {
			placeholders[i] = "(CAST(? AS bigint), CAST(? AS double precision))"
			var level any
			if item.result.FittingLevel != nil {
				level = *item.result.FittingLevel
			}
			args = append(args, item.chartID, level)
		}
		updateSQL := fmt.Sprintf(
			"UPDATE charts SET fitting_level = v.column2 FROM (VALUES %s) AS v WHERE charts.id = v.column1 AND charts.deleted_at IS NULL",
			strings.Join(placeholders, ","))
		if err := tx.Exec(updateSQL, args...).Error; err != nil {
			return fmt.Errorf("update fitting-level batch: %w", err)
		}

		now := time.Now()
		stats := lo.Map(items, func(item persistItem, _ int) model.ChartStatistic {
			res := item.result
			return model.ChartStatistic{
				ChartID:             item.chartID,
				OfficialLevel:       item.officialLevel,
				FittingLevel:        res.FittingLevel,
				SampleCount:         res.SampleCount,
				EffectiveSampleSize: res.EffectiveSampleSize,
				WeightedMean:        res.WeightedMean,
				WeightedMedian:      res.WeightedMedian,
				StdDev:              res.StdDev,
				MAD:                 res.MAD,
				LastComputedAt:      now,
			}
		})
		updateColumns := []string{
			"official_level", "fitting_level", "sample_count",
			"effective_sample_size", "weighted_mean", "weighted_median",
			"std_dev", "mad", "last_computed_at", "updated_at",
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "chart_id"}},
			DoUpdates: clause.AssignmentColumns(updateColumns),
		}).Create(&stats).Error; err != nil {
			return fmt.Errorf("upsert chart-statistics batch: %w", err)
		}
		return nil
	})
}
