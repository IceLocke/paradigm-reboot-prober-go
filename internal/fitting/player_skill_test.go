package fitting

import (
	"context"
	"fmt"
	"testing"
	"time"

	"paradigm-reboot-prober-go/internal/model"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// helper: create a minimal user row — collectPlayerSkills only joins on the
// play_records/best_play_records tables, so the user row itself is there just
// to satisfy the username string space and any future FK constraints.
func seedUser(t *testing.T, db *gorm.DB, username string) {
	t.Helper()
	u := model.User{
		UserBase: model.UserBase{
			Username: username, Email: username + "@e.com", Nickname: username,
			UploadToken: "tok-" + username, IsActive: true,
		},
		EncodedPassword: "x",
	}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("create user %s: %v", username, err)
	}
}

// seedRatedPlay inserts a PlayRecord with an explicit rating, plus (optionally)
// a BestPlayRecord pointing at it. We set rating directly rather than running
// it through the rating formula so tests can prescribe the exact skill values
// collectPlayerSkills will average.
func seedRatedPlay(t *testing.T, db *gorm.DB, username string, chartID, ratingInt int, makeBest bool) int {
	t.Helper()
	score := 1_005_000
	pr := model.PlayRecord{
		PlayRecordBase: model.PlayRecordBase{ChartID: chartID, Score: &score},
		Username:       username,
		Rating:         ratingInt,
	}
	if err := db.Create(&pr).Error; err != nil {
		t.Fatalf("create play record: %v", err)
	}
	if makeBest {
		bpr := model.BestPlayRecord{Username: username, ChartID: chartID, PlayRecordID: pr.ID}
		if err := db.Create(&bpr).Error; err != nil {
			t.Fatalf("create best play record: %v", err)
		}
	}
	return pr.ID
}

func newTestRunner(db *gorm.DB, cfg RunnerConfig) *Runner {
	return NewRunner(db, Params{
		MinEffectiveSamples: 3.0,
		SkillTopK:           50, // legacy cap; keep existing test assertions stable
		ProximitySigma:      20.0,
		VolumeFullAt:        5,
		PriorStrength:       1.0,
		MaxDeviation:        1.5,
		MinScore:            500000,
		TukeyK:              4.685,
		MinPlayerRecords:    1,
	}, cfg)
}

// TestCollectPlayerSkills_MultiPage verifies that with PlayerBatchSize smaller
// than the total number of users, keyset pagination still visits every user
// exactly once and produces the right per-user skill averages.
func TestCollectPlayerSkills_MultiPage(t *testing.T) {
	db := setupTestDB(t)
	// Need N charts because best_play_records is UNIQUE(username, chart_id).
	charts := seedCharts(t, db, 3)

	// 12 users, each with 3 best records (one per chart), at distinct ratings.
	const users = 12
	for i := 0; i < users; i++ {
		u := fmt.Sprintf("player%02d", i)
		seedUser(t, db, u)
		// Ratings: 14500+i·10, 14600+i·10, 14700+i·10
		for j, chartID := range charts {
			r := 14500 + j*100 + i*10
			seedRatedPlay(t, db, u, chartID, r, true)
		}
	}

	r := newTestRunner(db, RunnerConfig{PlayerBatchSize: 4}) // forces 3 pages
	skills, err := r.collectPlayerSkills(context.Background())
	assert.NoError(t, err)
	assert.Len(t, skills, users, "every user must be visited exactly once")

	for i := 0; i < users; i++ {
		u := fmt.Sprintf("player%02d", i)
		s, ok := skills[u]
		if !assert.True(t, ok, "user %s missing", u) {
			continue
		}
		assert.Equal(t, 3, s.NumRecords)
		wantAvg := float64(14500+14600+14700)/300.0 + float64(i)*0.1
		assert.InDelta(t, wantAvg, s.AvgRating, 1e-6)
	}
}

// TestCollectPlayerSkills_Top50Cap ensures that when a user has more than 50
// best records (across distinct charts), only the top 50 by rating contribute
// to AvgRating but NumRecords reflects the full count.
func TestCollectPlayerSkills_Top50Cap(t *testing.T) {
	db := setupTestDB(t)
	charts := seedCharts(t, db, 60)
	seedUser(t, db, "whale")
	// 60 records with ratings 10100..10700 (increments of 10), one per chart.
	for i, chartID := range charts {
		r := 10100 + i*10
		seedRatedPlay(t, db, "whale", chartID, r, true)
	}

	r := newTestRunner(db, RunnerConfig{PlayerBatchSize: 100})
	skills, err := r.collectPlayerSkills(context.Background())
	assert.NoError(t, err)
	s := skills["whale"]
	assert.Equal(t, 60, s.NumRecords, "NumRecords covers every best record")
	// Top 50 of the 60 = ratings 10200..10690 inclusive, step 10 (50 items).
	// Mean = (10200 + 10690) / 2 = 10445 → /100 = 104.45
	assert.InDelta(t, 104.45, s.AvgRating, 1e-6, "top-50 cap in effect")
}

// TestCollectPlayerSkills_ConfigurableTopK exercises the Params.SkillTopK knob:
// the same seeded records, evaluated with K=5 vs K=60, must yield different
// AvgRating values corresponding to the top-K subset. This is the "B<X>"
// behaviour requested by the fitting microservice's configurable skill proxy.
func TestCollectPlayerSkills_ConfigurableTopK(t *testing.T) {
	db := setupTestDB(t)
	charts := seedCharts(t, db, 10)
	seedUser(t, db, "pro")
	// 10 records with ratings 10000..10090, step 10.
	for i, chartID := range charts {
		seedRatedPlay(t, db, "pro", chartID, 10000+i*10, true)
	}

	// K = 5: take top 5 = ratings 10050, 10060, 10070, 10080, 10090.
	// Mean = 10070 → /100 = 100.70
	rK5 := NewRunner(db, Params{
		MinEffectiveSamples: 3.0, SkillTopK: 5, ProximitySigma: 20.0,
		MinScore: 500000, TukeyK: 4.685, MinPlayerRecords: 1,
	}, RunnerConfig{PlayerBatchSize: 100})
	sk5, err := rK5.collectPlayerSkills(context.Background())
	assert.NoError(t, err)
	assert.InDelta(t, 100.70, sk5["pro"].AvgRating, 1e-6, "K=5 picks top 5")
	assert.Equal(t, 10, sk5["pro"].NumRecords, "NumRecords unchanged by K")

	// K = 60 (larger than population): all 10 records contribute.
	// Mean = 10045 → /100 = 100.45
	rK60 := NewRunner(db, Params{
		MinEffectiveSamples: 3.0, SkillTopK: 60, ProximitySigma: 20.0,
		MinScore: 500000, TukeyK: 4.685, MinPlayerRecords: 1,
	}, RunnerConfig{PlayerBatchSize: 100})
	sk60, err := rK60.collectPlayerSkills(context.Background())
	assert.NoError(t, err)
	assert.InDelta(t, 100.45, sk60["pro"].AvgRating, 1e-6, "K exceeds record count → average over all")

	assert.Greater(t, sk5["pro"].AvgRating, sk60["pro"].AvgRating,
		"smaller K should yield a higher B_p (takes a more selective top slice)")
}

// TestCollectPlayerSkills_DefaultBatchSize exercises the `batch <= 0 → 500`
// fallback when RunnerConfig.PlayerBatchSize is unset. Behaviour must be
// identical to an explicitly-configured batch.
func TestCollectPlayerSkills_DefaultBatchSize(t *testing.T) {
	db := setupTestDB(t)
	chartID := seedSingleChart(t, db)
	seedUser(t, db, "solo")
	seedRatedPlay(t, db, "solo", chartID, 16500, true)

	// NewRunner sets PlayerBatchSize default (500) when cfg value ≤ 0, which
	// means collectPlayerSkills never hits its own fallback branch. Bypass the
	// constructor to drive that branch explicitly.
	r := &Runner{
		db: db,
		params: Params{
			MinEffectiveSamples: 3, SkillTopK: 50, ProximitySigma: 20, VolumeFullAt: 5,
			PriorStrength: 1, MaxDeviation: 1.5, MinScore: 500000, TukeyK: 4.685,
			MinPlayerRecords: 1,
		},
		cfg: RunnerConfig{PlayerBatchSize: 0}, // triggers the inner default
	}
	skills, err := r.collectPlayerSkills(context.Background())
	assert.NoError(t, err)
	assert.Len(t, skills, 1)
	assert.InDelta(t, 165.0, skills["solo"].AvgRating, 1e-6)
}

// TestCollectPlayerSkills_CtxCanceled: a context canceled before the first
// page must surface as ctx.Err() without any DB traffic.
func TestCollectPlayerSkills_CtxCanceled(t *testing.T) {
	db := setupTestDB(t)
	r := newTestRunner(db, RunnerConfig{PlayerBatchSize: 10})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	_, err := r.collectPlayerSkills(ctx)
	assert.ErrorIs(t, err, context.Canceled)
}

// TestCollectPlayerSkills_Empty: no best_play_records at all → empty map, nil
// error, no panic. Smoke test for a fresh-DB installation.
func TestCollectPlayerSkills_Empty(t *testing.T) {
	db := setupTestDB(t)
	r := newTestRunner(db, RunnerConfig{PlayerBatchSize: 10})
	skills, err := r.collectPlayerSkills(context.Background())
	assert.NoError(t, err)
	assert.Empty(t, skills)
}

// TestCollectPlayerSkills_BatchPause verifies that a positive BatchPause sleeps
// between pages and yields to ctx.Done(). Uses a very short pause so the test
// stays fast.
func TestCollectPlayerSkills_BatchPause(t *testing.T) {
	db := setupTestDB(t)
	chartID := seedSingleChart(t, db)
	// Two pages of data so at least one pause fires.
	for i := 0; i < 6; i++ {
		u := fmt.Sprintf("u%02d", i)
		seedUser(t, db, u)
		seedRatedPlay(t, db, u, chartID, 15000+i*100, true)
	}

	r := newTestRunner(db, RunnerConfig{
		PlayerBatchSize: 3,                     // forces at least two pages
		BatchPause:      10 * time.Millisecond, // brief pause
	})
	start := time.Now()
	skills, err := r.collectPlayerSkills(context.Background())
	elapsed := time.Since(start)
	assert.NoError(t, err)
	assert.Len(t, skills, 6)
	assert.GreaterOrEqual(t, elapsed, 10*time.Millisecond,
		"BatchPause must actually delay between pages")
}

// TestFetchBestSamples_FilterByMinRecords: samples from players with too few
// best records must be dropped per the MinPlayerRecords guard in Params.
func TestFetchBestSamples_FilterByMinRecords(t *testing.T) {
	db := setupTestDB(t)
	chartID := seedSingleChart(t, db)

	seedUser(t, db, "veteran")
	seedUser(t, db, "rookie")
	seedRatedPlay(t, db, "veteran", chartID, 16500, true)
	seedRatedPlay(t, db, "rookie", chartID, 16500, true)

	r := NewRunner(db, Params{
		MinEffectiveSamples: 3, SkillTopK: 50, ProximitySigma: 20, VolumeFullAt: 5,
		PriorStrength: 1, MaxDeviation: 1.5, MinScore: 500000, TukeyK: 4.685,
		MinPlayerRecords: 10, // high bar — no player qualifies with 1 record
	}, RunnerConfig{PlayerBatchSize: 10, ChartBatchSize: 10})

	skills := map[string]PlayerSkill{
		"veteran": {AvgRating: 165.0, NumRecords: 50}, // passes
		"rookie":  {AvgRating: 165.0, NumRecords: 2},  // filtered out
	}
	got, err := r.fetchBestSamples(context.Background(), []int{chartID}, skills)
	assert.NoError(t, err)
	// Only the veteran's sample should survive.
	if assert.Len(t, got[chartID], 1) {
		assert.Equal(t, "veteran", got[chartID][0].Username)
	}
}

// TestFetchBestSamples_SkillMissing: best records whose author is not in the
// skills map are silently dropped (this happens when NumRecords == 0, which
// shouldn't normally arise but we guard against it anyway).
func TestFetchBestSamples_SkillMissing(t *testing.T) {
	db := setupTestDB(t)
	chartID := seedSingleChart(t, db)

	seedUser(t, db, "ghost")
	seedRatedPlay(t, db, "ghost", chartID, 16500, true)

	r := newTestRunner(db, RunnerConfig{ChartBatchSize: 10})
	// Intentionally empty skills map → every row's Username miss lookup.
	got, err := r.fetchBestSamples(context.Background(), []int{chartID}, map[string]PlayerSkill{})
	assert.NoError(t, err)
	assert.Empty(t, got[chartID], "samples whose player lacks a skill snapshot get dropped")
}

// TestFetchBestSamples_AgeDaysPopulated: the (username, score, record_time)
// triple must flow end-to-end into Sample.AgeDays. We pin Runner.now to a
// fixed clock and seed records at known ages (0d, 30d, 365d back) so the
// arithmetic is deterministic. A zero/unset record_time collapses to
// AgeDays=0 (the "fresh" fallback that keeps the decay pipeline safe when
// legacy rows lack a timestamp).
func TestFetchBestSamples_AgeDaysPopulated(t *testing.T) {
	db := setupTestDB(t)
	charts := seedCharts(t, db, 4) // one chart per age bucket (BPR is unique per chart)

	fixedNow := time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC)
	seedUser(t, db, "fresh")
	seedUser(t, db, "month")
	seedUser(t, db, "year")
	seedUser(t, db, "zero")

	seedPlayWithTime(t, db, "fresh", charts[0], 16500, fixedNow)                          // 0d
	seedPlayWithTime(t, db, "month", charts[1], 16500, fixedNow.AddDate(0, 0, -30))       // 30d
	seedPlayWithTime(t, db, "year", charts[2], 16500, fixedNow.AddDate(-1, 0, 0))         // 365d
	seedPlayWithTime(t, db, "zero", charts[3], 16500, time.Time{})                        // unset

	r := newTestRunner(db, RunnerConfig{ChartBatchSize: 10})
	r.nowFunc = func() time.Time { return fixedNow }

	skills := map[string]PlayerSkill{
		"fresh": {AvgRating: 165.0, NumRecords: 50},
		"month": {AvgRating: 165.0, NumRecords: 50},
		"year":  {AvgRating: 165.0, NumRecords: 50},
		"zero":  {AvgRating: 165.0, NumRecords: 50},
	}
	got, err := r.fetchBestSamples(context.Background(), charts, skills)
	assert.NoError(t, err)

	byUser := map[string]float64{}
	for _, chartID := range charts {
		for _, s := range got[chartID] {
			byUser[s.Username] = s.AgeDays
		}
	}
	assert.InDelta(t, 0.0, byUser["fresh"], 1.0, "fresh record ≈ now")
	assert.InDelta(t, 30.0, byUser["month"], 1.0, "month-old record ≈ 30 days")
	assert.InDelta(t, 365.0, byUser["year"], 2.0, "year-old record ≈ 365 days")
	assert.Equal(t, 0.0, byUser["zero"], "zero/unset record_time falls back to fresh (AgeDays=0)")
}

// seedPlayWithTime is like seedRatedPlay but lets callers pin record_time so
// AgeDays in fetchBestSamples becomes deterministic. Always also creates the
// corresponding BestPlayRecord (we never test age-decay plumbing for plays
// that aren't "best", since those don't enter the fitting pipeline).
func seedPlayWithTime(t *testing.T, db *gorm.DB, username string, chartID, ratingInt int, recordTime time.Time) int {
	t.Helper()
	score := 1_005_000
	pr := model.PlayRecord{
		PlayRecordBase: model.PlayRecordBase{ChartID: chartID, Score: &score},
		Username:       username,
		Rating:         ratingInt,
		RecordTime:     recordTime,
	}
	if err := db.Create(&pr).Error; err != nil {
		t.Fatalf("create play record: %v", err)
	}
	bpr := model.BestPlayRecord{Username: username, ChartID: chartID, PlayRecordID: pr.ID}
	if err := db.Create(&bpr).Error; err != nil {
		t.Fatalf("create best play record: %v", err)
	}
	return pr.ID
}

// seedCharts creates `n` charts under a single song and returns their IDs.
// Separate charts are required whenever a test wants multiple best records
// from the same user, because best_play_records has a UNIQUE(username,chart_id)
// constraint.
func seedCharts(t *testing.T, db *gorm.DB, n int) []int {
	t.Helper()
	song := model.Song{SongBase: model.SongBase{
		WikiID: "s1", Title: "S1", Artist: "A", Genre: "G", Cover: "c",
		Illustrator: "I", Version: "V", Album: "Al", BPM: "100", Length: "1:00",
	}}
	if err := db.Create(&song).Error; err != nil {
		t.Fatalf("create song: %v", err)
	}
	diffs := []model.Difficulty{
		model.DifficultyDetected, model.DifficultyInvaded,
		model.DifficultyMassive, model.DifficultyReboot,
	}
	ids := make([]int, n)
	for i := 0; i < n; i++ {
		// After exhausting the four real difficulties, start spinning up more
		// songs so each chart remains uniquely keyed.
		if i > 0 && i%len(diffs) == 0 {
			sibling := song
			sibling.ID = 0
			sibling.WikiID = fmt.Sprintf("s1_%d", i)
			if err := db.Create(&sibling).Error; err != nil {
				t.Fatalf("create sibling song: %v", err)
			}
			song = sibling
		}
		c := model.Chart{
			SongID: song.ID, Difficulty: diffs[i%len(diffs)],
			Level: 16.5, Notes: 1000,
		}
		if err := db.Create(&c).Error; err != nil {
			t.Fatalf("create chart %d: %v", i, err)
		}
		ids[i] = c.ID
	}
	return ids
}

// seedSingleChart creates one Song + one Chart and returns the chart ID.
func seedSingleChart(t *testing.T, db *gorm.DB) int {
	return seedCharts(t, db, 1)[0]
}
