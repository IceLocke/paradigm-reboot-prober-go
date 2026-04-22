package fitting

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"paradigm-reboot-prober-go/config"
	"paradigm-reboot-prober-go/internal/model"
	"paradigm-reboot-prober-go/pkg/rating"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

var runnerTestDBCounter atomic.Int64

// setupTestDB mirrors the repository-layer pattern: fresh in-memory SQLite
// with every model auto-migrated, including the new ChartStatistic table.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	config.InitDefaults()
	dsn := fmt.Sprintf("file:memdb_fitting_%d?mode=memory&cache=shared", runnerTestDBCounter.Add(1))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Song{},
		&model.Chart{},
		&model.PlayRecord{},
		&model.BestPlayRecord{},
		&model.ChartStatistic{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// TestRunner_EndToEnd runs the full pipeline against an in-memory DB seeded
// with a single chart whose *true* level is 13.0 but officially 14.0, plus
// many simulated best_play_records produced via the real SingleRating formula.
// After Run() we expect charts.fitting_level to be populated and pulled
// toward 13.0 (bounded by the Bayesian prior).
func TestRunner_EndToEnd(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	// --- Seed a song + chart ---
	song := model.Song{
		SongBase: model.SongBase{
			WikiID:      "test_song",
			Title:       "Test Song",
			Artist:      "Tester",
			Genre:       "Test",
			Cover:       "cover.png",
			Illustrator: "Art",
			Version:     "1.0.0",
			B15:         false,
			Album:       "Album",
			BPM:         "120",
			Length:      "3:00",
		},
	}
	if err := db.Create(&song).Error; err != nil {
		t.Fatalf("create song: %v", err)
	}
	chart := model.Chart{
		SongID:     song.ID,
		Difficulty: model.DifficultyMassive,
		Level:      14.0, // official
		Notes:      1000,
	}
	if err := db.Create(&chart).Error; err != nil {
		t.Fatalf("create chart: %v", err)
	}
	// Seed a filler chart so player skill B50 is meaningful — we'll add many
	// records for it too.
	fillerChart := model.Chart{
		SongID:     song.ID,
		Difficulty: model.DifficultyInvaded,
		Level:      12.0,
		Notes:      800,
	}
	if err := db.Create(&fillerChart).Error; err != nil {
		t.Fatalf("create filler chart: %v", err)
	}

	// --- Seed 30 players, each with a best record on both charts. The test
	// chart is "truly" level 13.0 (1 level easier than official). ---
	const trueLevel = 13.0
	for i := 0; i < 30; i++ {
		username := fmt.Sprintf("player%02d", i)
		user := model.User{
			UserBase: model.UserBase{
				Username:    username,
				Email:       username + "@example.com",
				Nickname:    username,
				UploadToken: fmt.Sprintf("tok_%02d", i),
				IsActive:    true,
			},
			EncodedPassword: "x",
		}
		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("create user: %v", err)
		}
		skill := 125.0 + float64(i)*0.8 // rating range [125, 148.2]

		// Best record on the *test* chart. Score is synthesized to match
		// SingleRating(trueLevel, score) ≈ skill.
		score1 := simulateScore(trueLevel, skill)
		seedBestRecord(t, db, username, chart.ID, score1, chart.Level)

		// Best record on the filler chart (for player B50 averaging).
		score2 := simulateScore(fillerChart.Level, skill)
		seedBestRecord(t, db, username, fillerChart.ID, score2, fillerChart.Level)
	}

	// --- Configure fitting params + run ---
	cfg := config.GlobalConfig.Fitting
	params := Params{
		MinEffectiveSamples: 3.0,
		ProximitySigma:      cfg.ProximitySigma,
		VolumeFullAt:        5, // test data has few records per player
		PriorStrength:       1.0,
		MaxDeviation:        1.5,
		MinScore:            cfg.MinScore,
		TukeyK:              cfg.TukeyK,
	}
	config.GlobalConfig.Fitting.MinPlayerRecords = 1 // admit all players
	runner := NewRunner(db, params, RunnerConfig{
		ChartBatchSize:  10,
		PlayerBatchSize: 50,
	})
	report, err := runner.Run(ctx)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	assert.Greater(t, report.ChartsProcessed, 0)
	assert.Greater(t, report.ChartsPublished, 0)
	assert.Greater(t, report.PlayersConsidered, 0)

	// --- Inspect results ---
	var updated model.Chart
	if err := db.First(&updated, chart.ID).Error; err != nil {
		t.Fatalf("reload chart: %v", err)
	}
	if !assert.NotNil(t, updated.FittingLevel, "fitting_level should be populated") {
		return
	}
	// Expect fitting to sit between trueLevel (13.0) and officialLevel (14.0),
	// closer to true because we have many samples and small prior.
	assert.InDelta(t, trueLevel, *updated.FittingLevel, 0.6)
	assert.Less(t, *updated.FittingLevel, updated.Level)

	var stat model.ChartStatistic
	if err := db.Where("chart_id = ?", chart.ID).First(&stat).Error; err != nil {
		t.Fatalf("reload chart_statistics: %v", err)
	}
	assert.Equal(t, chart.ID, stat.ChartID)
	assert.Equal(t, chart.Level, stat.OfficialLevel)
	assert.NotNil(t, stat.FittingLevel)
	assert.Greater(t, stat.SampleCount, 0)
	assert.Greater(t, stat.EffectiveSampleSize, 0.0)
}

// TestRunner_InsufficientSamples ensures that a chart with no best records
// leaves its fitting_level untouched (NULL) and writes a stats row with
// zero-valued fields.
func TestRunner_InsufficientSamples(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	song := model.Song{SongBase: model.SongBase{
		WikiID: "lone_song", Title: "Lone", Artist: "A", Genre: "G", Cover: "c",
		Illustrator: "I", Version: "V", Album: "Al", BPM: "100", Length: "1:00",
	}}
	if err := db.Create(&song).Error; err != nil {
		t.Fatalf("create song: %v", err)
	}
	chart := model.Chart{
		SongID: song.ID, Difficulty: model.DifficultyMassive,
		Level: 14.0, Notes: 1000,
	}
	if err := db.Create(&chart).Error; err != nil {
		t.Fatalf("create chart: %v", err)
	}

	runner := NewRunner(db, Params{
		MinEffectiveSamples: 3.0,
		ProximitySigma:      20.0,
		VolumeFullAt:        50,
		PriorStrength:       5.0,
		MaxDeviation:        1.5,
		MinScore:            500000,
		TukeyK:              4.685,
	}, RunnerConfig{ChartBatchSize: 10, PlayerBatchSize: 50})
	report, err := runner.Run(ctx)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	assert.Equal(t, 1, report.ChartsTotal)
	assert.Equal(t, 1, report.ChartsEmpty)

	var updated model.Chart
	if err := db.First(&updated, chart.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	assert.Nil(t, updated.FittingLevel)

	var stat model.ChartStatistic
	if err := db.Where("chart_id = ?", chart.ID).First(&stat).Error; err != nil {
		t.Fatalf("stat: %v", err)
	}
	assert.Nil(t, stat.FittingLevel)
	assert.Equal(t, 0, stat.SampleCount)
}

// seedBestRecord inserts one PlayRecord + one BestPlayRecord pointing at it,
// with a rating precomputed via SingleRating so skill computation works.
func seedBestRecord(t *testing.T, db *gorm.DB, username string, chartID int, score int, level float64) {
	t.Helper()
	s := score
	pr := model.PlayRecord{
		PlayRecordBase: model.PlayRecordBase{
			ChartID: chartID,
			Score:   &s,
		},
		Username: username,
		Rating:   rating.SingleRating(level, score),
	}
	if err := db.Create(&pr).Error; err != nil {
		t.Fatalf("create play record: %v", err)
	}
	bpr := model.BestPlayRecord{
		Username: username, ChartID: chartID, PlayRecordID: pr.ID,
	}
	if err := db.Create(&bpr).Error; err != nil {
		t.Fatalf("create best play record: %v", err)
	}
}
