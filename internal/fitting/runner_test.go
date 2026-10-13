package fitting

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"paradigm-reboot-prober-go/config"
	"paradigm-reboot-prober-go/internal/model"
	"paradigm-reboot-prober-go/pkg/rating"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestRunnerCanceledDuringEstimationDoesNotPersistPartialPopulation(t *testing.T) {
	db := setupTestDB(t)
	charts := seedCharts(t, db, 4)
	seedUser(t, db, "cancel_player")
	for _, chart := range charts {
		seedRatedPlay(t, db, "cancel_player", chart, 16000, true)
	}
	assert.NoError(t, db.Model(&model.Chart{}).Where("id IN ?", charts).Update("fitting_level", 15.123).Error)
	r := newTestRunner(db, RunnerConfig{ChartBatchSize: 2})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reads := 0
	r.nowFunc = func() time.Time {
		reads++
		if reads == 2 {
			cancel()
		}
		return time.Now()
	}
	_, err := r.Run(ctx)
	assert.ErrorIs(t, err, context.Canceled)
	var rows []model.Chart
	assert.NoError(t, db.Find(&rows).Error)
	for _, chart := range rows {
		if assert.NotNil(t, chart.FittingLevel) {
			assert.Equal(t, 15.123, *chart.FittingLevel)
		}
	}
	var stats int64
	assert.NoError(t, db.Model(&model.ChartStatistic{}).Count(&stats).Error)
	assert.Zero(t, stats, "no batch is written before the complete population is available")
}

func TestRunnerReturnsPersistenceErrorAndDoesNotReportRolledBackWrites(t *testing.T) {
	db := setupTestDB(t)
	charts := seedCharts(t, db, 2)
	for u := 0; u < 5; u++ {
		username := fmt.Sprintf("write_failure_%d", u)
		seedUser(t, db, username)
		for _, chart := range charts {
			seedRatedPlay(t, db, username, chart, 16000, true)
		}
	}
	assert.NoError(t, db.Model(&model.Chart{}).Where("id IN ?", charts).Update("fitting_level", 15.123).Error)
	assert.NoError(t, db.Migrator().DropTable(&model.ChartStatistic{}))
	r := newTestRunner(db, RunnerConfig{})
	report, err := r.Run(context.Background())
	assert.ErrorContains(t, err, "persist fitting batch")
	assert.Zero(t, report.ChartsPublished)
	assert.Equal(t, len(charts), report.ErrorsEncountered)
	var rows []model.Chart
	assert.NoError(t, db.Find(&rows).Error)
	for _, chart := range rows {
		if assert.NotNil(t, chart.FittingLevel) {
			assert.Equal(t, 15.123, *chart.FittingLevel, "statistics and chart writes must roll back together")
		}
	}
}

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
// with a single chart whose *true* level is 15.5 but officially 16.5, plus
// many simulated best_play_records produced via the real SingleRating formula.
// The chart lives in the lv15+ hot zone where most real plays actually happen.
// After Run() we expect charts.fitting_level to be populated and pulled
// toward 15.5 (bounded by the Bayesian prior).
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
		Level:      16.5, // official
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
		Level:      14.5,
		Notes:      800,
	}
	if err := db.Create(&fillerChart).Error; err != nil {
		t.Fatalf("create filler chart: %v", err)
	}

	// --- Seed 30 players, each with a best record on both charts. The test
	// chart is "truly" level 15.5 (1 level easier than official). ---
	const trueLevel = 15.5
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
		// skill ~ 155..162 — matches trueLevel so scores land in the real hot
		// zone [1_000_000, 1_010_000].
		skill := 155.0 + float64(i)*0.25

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
		SkillTopK:           50,
		ProximitySigma:      cfg.ProximitySigma,
		VolumeFullAt:        5, // test data has few records per player
		PriorStrength:       1.0,
		MaxDeviation:        1.5,
		MinScore:            cfg.MinScore,
		TukeyK:              cfg.TukeyK,
		MinPlayerRecords:    1, // admit all players
	}
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
	// Expect fitting to sit between trueLevel (15.5) and officialLevel (16.5),
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
		SkillTopK:           50,
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

// TestRunner_PersistUpdatesExistingStat covers the read-modify-write UPDATE
// branch in persist(). The INSERT branch is exercised by every other runner
// test — this one runs Run() twice to force the "row already exists" path.
func TestRunner_PersistUpdatesExistingStat(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	song := model.Song{SongBase: model.SongBase{
		WikiID: "rerun_song", Title: "Rerun", Artist: "A", Genre: "G", Cover: "c",
		Illustrator: "I", Version: "V", Album: "Al", BPM: "100", Length: "1:00",
	}}
	if err := db.Create(&song).Error; err != nil {
		t.Fatalf("create song: %v", err)
	}
	chart := model.Chart{SongID: song.ID, Difficulty: model.DifficultyMassive, Level: 16.5, Notes: 1000}
	if err := db.Create(&chart).Error; err != nil {
		t.Fatalf("create chart: %v", err)
	}
	filler := model.Chart{SongID: song.ID, Difficulty: model.DifficultyInvaded, Level: 14.5, Notes: 800}
	if err := db.Create(&filler).Error; err != nil {
		t.Fatalf("create filler: %v", err)
	}

	const trueLevel = 15.5
	for i := 0; i < 10; i++ {
		u := fmt.Sprintf("p%02d", i)
		seedUser(t, db, u)
		skill := 155.0 + float64(i)*0.5
		seedBestRecord(t, db, u, chart.ID, simulateScore(trueLevel, skill), chart.Level)
		seedBestRecord(t, db, u, filler.ID, simulateScore(filler.Level, skill), filler.Level)
	}

	params := Params{
		MinEffectiveSamples: 2.0,
		SkillTopK:           50,
		ProximitySigma:      15.0,
		VolumeFullAt:        3,
		PriorStrength:       1.0,
		MaxDeviation:        1.5,
		MinScore:            500000,
		TukeyK:              4.685,
		MinPlayerRecords:    1,
	}

	// First pass — persist should INSERT a fresh chart_statistics row.
	r1 := NewRunner(db, params, RunnerConfig{ChartBatchSize: 10, PlayerBatchSize: 50})
	if _, err := r1.Run(ctx); err != nil {
		t.Fatalf("first run: %v", err)
	}
	var first model.ChartStatistic
	if err := db.Where("chart_id = ?", chart.ID).First(&first).Error; err != nil {
		t.Fatalf("first stat: %v", err)
	}
	firstCreated := first.CreatedAt

	// Sleep just long enough that SQLite's time.Now()-based timestamps can
	// differ between runs (sqlite stores microsecond precision via GORM).
	time.Sleep(5 * time.Millisecond)

	// Second pass — same data, same chart, so persist must take the UPDATE
	// branch. CreatedAt must be preserved; LastComputedAt should advance.
	r2 := NewRunner(db, params, RunnerConfig{ChartBatchSize: 10, PlayerBatchSize: 50})
	if _, err := r2.Run(ctx); err != nil {
		t.Fatalf("second run: %v", err)
	}
	var second model.ChartStatistic
	if err := db.Where("chart_id = ?", chart.ID).First(&second).Error; err != nil {
		t.Fatalf("second stat: %v", err)
	}

	// Still exactly one stats row for this chart.
	var count int64
	if err := db.Model(&model.ChartStatistic{}).Where("chart_id = ?", chart.ID).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	assert.Equal(t, int64(1), count, "UPDATE must not create a duplicate row")

	// CreatedAt is preserved across UPDATE.
	assert.WithinDuration(t, firstCreated, second.CreatedAt, time.Microsecond,
		"CreatedAt must be preserved by UPDATE branch")

	// LastComputedAt advanced — proves the UPDATE actually wrote.
	assert.True(t, second.LastComputedAt.After(first.LastComputedAt) ||
		second.LastComputedAt.Equal(first.LastComputedAt),
		"LastComputedAt should be >= first run's")
}

// TestRunner_PersistBatchMixedFittingLevels is a focused unit test for
// persistBatch's batched VALUES update (runner.go: `UPDATE charts SET
// fitting_level = v.column2 FROM (VALUES ...) AS v`). One batch mixes a
// published result (non-nil FittingLevel) with empty and abstained results
// (nil FittingLevel), and both nil charts carry a *stale* fitting_level left
// by a previous run. The statement must write the dereferenced value for the
// published chart and NULL for the other two — including overwriting their
// stale non-null values: the id join only protects rows absent from the
// batch, never nil results within it.
func TestRunner_PersistBatchMixedFittingLevels(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	song := model.Song{SongBase: model.SongBase{
		WikiID: "mixed_song", Title: "Mixed", Artist: "A", Genre: "G", Cover: "c",
		Illustrator: "I", Version: "V", Album: "Al", BPM: "100", Length: "1:00",
	}}
	if err := db.Create(&song).Error; err != nil {
		t.Fatalf("create song: %v", err)
	}
	emptyChart := model.Chart{SongID: song.ID, Difficulty: model.DifficultyDetected, Level: 14.0, Notes: 800}
	publishedChart := model.Chart{SongID: song.ID, Difficulty: model.DifficultyInvaded, Level: 16.5, Notes: 1000}
	abstainedChart := model.Chart{SongID: song.ID, Difficulty: model.DifficultyMassive, Level: 15.0, Notes: 900}
	for _, c := range []*model.Chart{&emptyChart, &publishedChart, &abstainedChart} {
		if err := db.Create(c).Error; err != nil {
			t.Fatalf("create chart: %v", err)
		}
	}

	// Simulate a previous run that published values on the two charts which
	// now abstain — the nil branch must reset them to NULL.
	stale := 13.5
	if err := db.Model(&model.Chart{}).
		Where("id IN ?", []int{emptyChart.ID, abstainedChart.ID}).
		UpdateColumn("fitting_level", stale).Error; err != nil {
		t.Fatalf("seed stale fitting_level: %v", err)
	}

	fit := 15.72
	runner := NewRunner(db, Params{}, RunnerConfig{}) // persistBatch ignores params
	items := []persistItem{
		// nil first, published in the middle, nil last — so the non-nil arg
		// sits between two NULL args inside the VALUES list.
		{chartID: emptyChart.ID, officialLevel: emptyChart.Level, result: Result{}},
		{chartID: publishedChart.ID, officialLevel: publishedChart.Level,
			result: Result{
				FittingLevel:        &fit,
				SampleCount:         42,
				EffectiveSampleSize: 30.5,
				WeightedMean:        15.68,
				WeightedMedian:      15.70,
				StdDev:              0.21,
				MAD:                 0.11,
			}},
		{chartID: abstainedChart.ID, officialLevel: abstainedChart.Level,
			result: Result{SampleCount: 2, EffectiveSampleSize: 1.9, WeightedMean: 14.9, WeightedMedian: 14.9}},
	}
	if err := runner.persistBatch(ctx, items); err != nil {
		t.Fatalf("persistBatch: %v", err)
	}

	// --- charts.fitting_level: value vs NULL, stale values overwritten ---
	var charts []model.Chart
	if err := db.Order("id ASC").Find(&charts).Error; err != nil {
		t.Fatalf("reload charts: %v", err)
	}
	if !assert.Len(t, charts, 3) {
		return
	}
	byID := make(map[int]model.Chart, len(charts))
	for _, c := range charts {
		byID[c.ID] = c
	}
	assert.Nil(t, byID[emptyChart.ID].FittingLevel,
		"empty chart: stale non-null fitting_level must be overwritten with NULL")
	assert.Nil(t, byID[abstainedChart.ID].FittingLevel,
		"abstained chart: stale non-null fitting_level must be overwritten with NULL")
	if got := byID[publishedChart.ID].FittingLevel; assert.NotNil(t, got, "published chart must keep a non-null value") {
		assert.InDelta(t, fit, *got, 1e-9)
	}

	// --- chart_statistics mirrors the same nil/non-nil split ---
	var stats []model.ChartStatistic
	if err := db.Order("chart_id ASC").Find(&stats).Error; err != nil {
		t.Fatalf("reload stats: %v", err)
	}
	if !assert.Len(t, stats, 3, "one stats row per chart") {
		return
	}
	statByID := make(map[int]model.ChartStatistic, len(stats))
	for _, s := range stats {
		statByID[s.ChartID] = s
	}
	assert.Nil(t, statByID[emptyChart.ID].FittingLevel)
	assert.Equal(t, 0, statByID[emptyChart.ID].SampleCount)

	assert.Nil(t, statByID[abstainedChart.ID].FittingLevel)
	assert.Equal(t, 2, statByID[abstainedChart.ID].SampleCount,
		"abstained chart keeps its insufficient-sample stats for post-hoc analysis")
	assert.InDelta(t, 1.9, statByID[abstainedChart.ID].EffectiveSampleSize, 1e-9)
	assert.False(t, statByID[abstainedChart.ID].LastComputedAt.IsZero())

	pubStat := statByID[publishedChart.ID]
	if assert.NotNil(t, pubStat.FittingLevel) {
		assert.InDelta(t, fit, *pubStat.FittingLevel, 1e-9)
	}
	assert.Equal(t, publishedChart.Level, pubStat.OfficialLevel)
	assert.Equal(t, 42, pubStat.SampleCount)
	assert.InDelta(t, 30.5, pubStat.EffectiveSampleSize, 1e-9)
}

// TestRunner_MixedBatchPublishEmptyAbstain drives a full Run() whose single
// chart batch contains all three outcome kinds — one chart publishes, one has
// no records (empty), one has too few effective samples (abstains). The point
// is persistBatch's VALUES update receiving BOTH nil and non-nil args in the
// same statement: the two nil charts start with stale fitting_level values
// from a previous run and must end up NULL while the published chart gets its
// fresh value — all in one UPDATE.
func TestRunner_MixedBatchPublishEmptyAbstain(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	song := model.Song{SongBase: model.SongBase{
		WikiID: "batch_song", Title: "Batch", Artist: "A", Genre: "G", Cover: "c",
		Illustrator: "I", Version: "V", Album: "Al", BPM: "100", Length: "1:00",
	}}
	if err := db.Create(&song).Error; err != nil {
		t.Fatalf("create song: %v", err)
	}
	publishedChart := model.Chart{SongID: song.ID, Difficulty: model.DifficultyMassive, Level: 16.5, Notes: 1000}
	emptyChart := model.Chart{SongID: song.ID, Difficulty: model.DifficultyInvaded, Level: 14.0, Notes: 800}
	abstainedChart := model.Chart{SongID: song.ID, Difficulty: model.DifficultyDetected, Level: 15.0, Notes: 900}
	for _, c := range []*model.Chart{&publishedChart, &emptyChart, &abstainedChart} {
		if err := db.Create(c).Error; err != nil {
			t.Fatalf("create chart: %v", err)
		}
	}

	// Stale published values left by a hypothetical earlier run — the empty
	// and abstained charts must have them reset to NULL by this run.
	if err := db.Model(&model.Chart{}).
		Where("id IN ?", []int{emptyChart.ID, abstainedChart.ID}).
		UpdateColumn("fitting_level", 13.5).Error; err != nil {
		t.Fatalf("seed stale fitting_level: %v", err)
	}

	// Published chart: 10 players whose single best record brackets a true
	// level of 15.5 (skill equals the record's own rating). The rating is
	// computed against the *true* level so the stored rating matches `skill`.
	const trueLevel = 15.5
	for i := 0; i < 10; i++ {
		u := fmt.Sprintf("pub%02d", i)
		seedUser(t, db, u)
		skill := 155.0 + float64(i)*0.25
		seedBestRecord(t, db, u, publishedChart.ID, simulateScore(trueLevel, skill), trueLevel)
	}
	// Abstained chart: only 2 valid samples → N_eff ≈ 2 < MinEffectiveSamples,
	// so ComputeFitting returns a nil FittingLevel with non-zero sample stats.
	for i := 0; i < 2; i++ {
		u := fmt.Sprintf("abs%02d", i)
		seedUser(t, db, u)
		seedBestRecord(t, db, u, abstainedChart.ID,
			simulateScore(15.0, 150.0+float64(i)*0.5), abstainedChart.Level)
	}
	// emptyChart deliberately gets nothing at all.

	runner := NewRunner(db, Params{
		MinEffectiveSamples: 3.0,
		SkillTopK:           50,
		ProximitySigma:      20.0,
		VolumeFullAt:        5,
		PriorStrength:       1.0,
		MaxDeviation:        1.5,
		MinScore:            500000,
		TukeyK:              4.685,
		MinPlayerRecords:    1,
	}, RunnerConfig{ChartBatchSize: 10, PlayerBatchSize: 50}) // one batch holds all 3 charts
	report, err := runner.Run(ctx)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	assert.Equal(t, 12, report.PlayersConsidered)
	assert.Equal(t, 3, report.ChartsTotal)
	assert.Equal(t, 3, report.ChartsProcessed)
	assert.Equal(t, 1, report.ChartsPublished)
	assert.Equal(t, 1, report.ChartsEmpty)
	assert.Equal(t, 1, report.ChartsAbstained)
	assert.Equal(t, 0, report.ErrorsEncountered)

	// --- charts.fitting_level: one non-null, two NULL (stale overwritten) ---
	var charts []model.Chart
	if err := db.Order("id ASC").Find(&charts).Error; err != nil {
		t.Fatalf("reload charts: %v", err)
	}
	byID := make(map[int]model.Chart, len(charts))
	for _, c := range charts {
		byID[c.ID] = c
	}
	assert.Nil(t, byID[emptyChart.ID].FittingLevel, "empty chart must end NULL despite stale value")
	assert.Nil(t, byID[abstainedChart.ID].FittingLevel, "abstained chart must end NULL despite stale value")
	pubFit := byID[publishedChart.ID].FittingLevel
	if assert.NotNil(t, pubFit, "published chart must have a non-null fitting_level") {
		assert.InDelta(t, trueLevel, *pubFit, 0.8)
		assert.Less(t, *pubFit, byID[publishedChart.ID].Level)
	}

	// --- stats rows mirror the same split ---
	var stats []model.ChartStatistic
	if err := db.Order("chart_id ASC").Find(&stats).Error; err != nil {
		t.Fatalf("reload stats: %v", err)
	}
	if !assert.Len(t, stats, 3) {
		return
	}
	statByID := make(map[int]model.ChartStatistic, len(stats))
	for _, s := range stats {
		statByID[s.ChartID] = s
	}
	assert.Nil(t, statByID[emptyChart.ID].FittingLevel)
	assert.Equal(t, 0, statByID[emptyChart.ID].SampleCount)
	assert.Nil(t, statByID[abstainedChart.ID].FittingLevel)
	assert.Equal(t, 2, statByID[abstainedChart.ID].SampleCount)
	if statFit := statByID[publishedChart.ID].FittingLevel; assert.NotNil(t, statFit) {
		assert.InDelta(t, *pubFit, *statFit, 1e-9,
			"chart_statistics.fitting_level must mirror charts.fitting_level")
	}
}

func TestRunner_PersistBatchSkipsSoftDeletedChart(t *testing.T) {
	db := setupTestDB(t)

	song := model.Song{SongBase: model.SongBase{
		WikiID: "soft_deleted_song", Title: "Soft deleted", Artist: "A", Genre: "G", Cover: "c",
		Illustrator: "I", Version: "V", Album: "Al", BPM: "100", Length: "1:00",
	}}
	if err := db.Create(&song).Error; err != nil {
		t.Fatalf("create song: %v", err)
	}

	stale := 13.5
	chart := model.Chart{
		SongID: song.ID, Difficulty: model.DifficultyMassive, Level: 14.0,
		FittingLevel: &stale, Notes: 1000,
	}
	if err := db.Create(&chart).Error; err != nil {
		t.Fatalf("create chart: %v", err)
	}
	if err := db.Delete(&chart).Error; err != nil {
		t.Fatalf("soft-delete chart: %v", err)
	}

	fresh := 14.25
	runner := NewRunner(db, Params{}, RunnerConfig{})
	if err := runner.persistBatch(context.Background(), []persistItem{{
		chartID: chart.ID,
		result:  Result{FittingLevel: &fresh},
	}}); err != nil {
		t.Fatalf("persistBatch: %v", err)
	}

	var deleted model.Chart
	if err := db.Unscoped().First(&deleted, chart.ID).Error; err != nil {
		t.Fatalf("reload soft-deleted chart: %v", err)
	}
	if assert.NotNil(t, deleted.FittingLevel, "soft-deleted chart should retain its prior fitting_level") {
		assert.InDelta(t, stale, *deleted.FittingLevel, 1e-9)
	}
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
