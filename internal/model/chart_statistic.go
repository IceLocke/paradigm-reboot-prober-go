package model

import "time"

// ChartStatistic stores per-chart aggregate statistics derived from best_play_records.
//
// It is produced by the fitting-calculator microservice (cmd/fitting) and is
// not consumed by the main probe service (cmd/server) — the table exists
// purely for offline analysis and observability. The score probe ("查分器")
// remains decoupled from fitting-level computation; see AGENTS.md
// (“保持查分器本体的单纯性”).
//
// One row per chart; ChartID is the primary key.
type ChartStatistic struct {
	BaseModel
	ChartID int `gorm:"primaryKey;column:chart_id" json:"chart_id"`

	// OfficialLevel is a snapshot of charts.level at the time of computation.
	OfficialLevel float64 `gorm:"not null;column:official_level" json:"official_level"`

	// FittingLevel mirrors the value persisted to charts.fitting_level. Nil
	// when the sample was insufficient to publish a level.
	FittingLevel *float64 `gorm:"column:fitting_level" json:"fitting_level"`

	// SampleCount is the raw number of best_play_records considered for this chart
	// (before robust trimming / weighting).
	SampleCount int `gorm:"not null;column:sample_count" json:"sample_count"`

	// EffectiveSampleSize (N_eff) is computed via Kish's formula:
	//   N_eff = (Σ w_i)² / Σ w_i²
	// where w_i is the final composite weight of each sample. It reflects how
	// many "ideal" samples the weighted aggregation is equivalent to.
	EffectiveSampleSize float64 `gorm:"not null;column:effective_sample_size" json:"effective_sample_size"`

	// WeightedMean is the weighted arithmetic mean of per-sample inferred
	// levels, before shrinkage.
	WeightedMean float64 `gorm:"not null;column:weighted_mean" json:"weighted_mean"`

	// WeightedMedian is the weighted median of per-sample inferred levels;
	// also serves as the initial anchor for robust (Tukey biweight) trimming.
	WeightedMedian float64 `gorm:"not null;column:weighted_median" json:"weighted_median"`

	// StdDev is the weighted standard deviation of inferred levels (post trim).
	StdDev float64 `gorm:"not null;column:std_dev" json:"std_dev"`

	// MAD is the median absolute deviation of inferred levels, used as the
	// robust dispersion estimator inside the Tukey biweight step.
	MAD float64 `gorm:"not null;column:mad" json:"mad"`

	// LastComputedAt is the wall-clock time of the most recent computation.
	LastComputedAt time.Time `gorm:"not null;column:last_computed_at" json:"last_computed_at"`
}

// TableName specifies the table name for GORM.
func (ChartStatistic) TableName() string { return "chart_statistics" }
