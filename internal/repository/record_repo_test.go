package repository

import (
	"paradigm-reboot-prober-go/internal/model"
	"paradigm-reboot-prober-go/pkg/rating"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecordRepository_CreateRecord(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRecordRepository(db)
	songRepo := NewSongRepository(db)

	song := &model.Song{
		SongBase: model.SongBase{WikiID: "rec_song", Title: "Record Song"},
		Charts:   []model.Chart{{Difficulty: model.DifficultyMassive, Level: 15.0, Notes: 1000}},
	}
	createdSong, _ := songRepo.CreateSong(song)
	chartID := createdSong.Charts[0].ID

	tests := []struct {
		name          string
		score         int
		isReplaced    bool
		wantRating    int
		expectNewBest bool
	}{
		{"Create New Record", 1000000, false, 15000, true},
		{"Higher Score Updates Best", 1000001, false, -1, true},
		{"Lower Score Keeps Best", 900000, false, -1, false},
		{"Force Replace Updates Best", 800000, true, -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := &model.PlayRecord{
				PlayRecordBase: model.PlayRecordBase{ChartID: chartID, Score: intPtr(tt.score)},
				Username:       "testuser",
			}
			saved, err := repo.CreateRecord(record, tt.isReplaced)
			assert.NoError(t, err)
			assert.NotNil(t, saved)
			assert.Equal(t, tt.score, *saved.Score)

			if tt.wantRating > 0 {
				assert.Equal(t, tt.wantRating, saved.Rating)
			}

			// Verify whether this record is the best
			var best model.BestPlayRecord
			err = db.Where("play_record_id = ?", saved.ID).First(&best).Error
			if tt.expectNewBest {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestRecordRepository_GetBest50Records(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRecordRepository(db)
	songRepo := NewSongRepository(db)

	songB15, _ := songRepo.CreateSong(&model.Song{
		SongBase: model.SongBase{WikiID: "b15_song", Title: "B15 Song", B15: true},
		Charts:   []model.Chart{{Difficulty: model.DifficultyMassive, Level: 15.0}},
	})
	songOld, _ := songRepo.CreateSong(&model.Song{
		SongBase: model.SongBase{WikiID: "old_song", Title: "Old Song", B15: false},
		Charts:   []model.Chart{{Difficulty: model.DifficultyMassive, Level: 15.0}},
	})

	for _, chartID := range []int{songB15.Charts[0].ID, songOld.Charts[0].ID} {
		_, err := repo.CreateRecord(&model.PlayRecord{
			PlayRecordBase: model.PlayRecordBase{ChartID: chartID, Score: intPtr(1000000)},
			Username:       "user_b50",
		}, false)
		assert.NoError(t, err)
	}

	b35, b15, err := repo.GetBest50Records("user_b50", 0, model.RecordFilter{})
	assert.NoError(t, err)
	assert.Len(t, b15, 1)
	assert.Len(t, b35, 1)
	assert.Equal(t, "B15 Song", b15[0].Chart.Song.Title)
	assert.Equal(t, "Old Song", b35[0].Chart.Song.Title)
}

func TestRecordRepository_PerSongQueries(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRecordRepository(db)
	songRepo := NewSongRepository(db)

	song1, _ := songRepo.CreateSong(&model.Song{
		SongBase: model.SongBase{WikiID: "song_a", Title: "Song A"},
		Charts: []model.Chart{
			{Difficulty: model.DifficultyDetected, Level: 5.0, Notes: 200},
			{Difficulty: model.DifficultyMassive, Level: 15.0, Notes: 1000},
		},
	})
	song2, _ := songRepo.CreateSong(&model.Song{
		SongBase: model.SongBase{WikiID: "song_b", Title: "Song B"},
		Charts:   []model.Chart{{Difficulty: model.DifficultyMassive, Level: 14.0, Notes: 900}},
	})

	// song1: detected×1, massive×2; song2: massive×1
	seedRecords := []struct{ chartID, score int }{
		{song1.Charts[0].ID, 1000000},
		{song1.Charts[1].ID, 1005000},
		{song1.Charts[1].ID, 900000},
		{song2.Charts[0].ID, 1000000},
	}
	for _, s := range seedRecords {
		_, err := repo.CreateRecord(&model.PlayRecord{
			PlayRecordBase: model.PlayRecordBase{ChartID: s.chartID, Score: intPtr(s.score)},
			Username:       "user_song",
		}, false)
		assert.NoError(t, err)
	}

	tests := []struct {
		name      string
		fn        func() (int, error)
		wantCount int
	}{
		{"BestBySong song1", func() (int, error) {
			r, e := repo.GetBestRecordsBySong("user_song", song1.ID)
			return len(r), e
		}, 2},
		{"BestBySong song2", func() (int, error) {
			r, e := repo.GetBestRecordsBySong("user_song", song2.ID)
			return len(r), e
		}, 1},
		{"AllBySong song1", func() (int, error) {
			r, e := repo.GetAllRecordsBySong("user_song", song1.ID, 10, 0, "rating", true)
			return len(r), e
		}, 3},
		{"CountBySong song1", func() (int, error) {
			c, e := repo.CountAllRecordsBySong("user_song", song1.ID)
			return int(c), e
		}, 3},
		{"CountBySong song2", func() (int, error) {
			c, e := repo.CountAllRecordsBySong("user_song", song2.ID)
			return int(c), e
		}, 1},
		{"BestBySong nonexistent user", func() (int, error) {
			r, e := repo.GetBestRecordsBySong("nobody", song1.ID)
			return len(r), e
		}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.fn()
			assert.NoError(t, err)
			assert.Equal(t, tt.wantCount, got)
		})
	}
}

func TestRecordRepository_PerChartQueries(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRecordRepository(db)
	songRepo := NewSongRepository(db)

	song, _ := songRepo.CreateSong(&model.Song{
		SongBase: model.SongBase{WikiID: "chart_q", Title: "Chart Query Song"},
		Charts: []model.Chart{
			{Difficulty: model.DifficultyMassive, Level: 15.0, Notes: 1000},
			{Difficulty: model.DifficultyDetected, Level: 5.0, Notes: 200},
		},
	})
	massiveID := song.Charts[0].ID
	detectedID := song.Charts[1].ID

	// Create records: massive×3, detected×1
	for _, s := range []struct{ chartID, score int }{
		{massiveID, 1000000}, {massiveID, 1005000}, {massiveID, 900000},
		{detectedID, 1000000},
	} {
		_, _ = repo.CreateRecord(&model.PlayRecord{
			PlayRecordBase: model.PlayRecordBase{ChartID: s.chartID, Score: intPtr(s.score)},
			Username:       "user_chart",
		}, false)
	}

	t.Run("GetBestRecordByChart", func(t *testing.T) {
		record, err := repo.GetBestRecordByChart("user_chart", massiveID)
		assert.NoError(t, err)
		assert.NotNil(t, record)
		assert.Equal(t, 1005000, *record.Score)
		assert.NotNil(t, record.Chart)
		assert.NotNil(t, record.Chart.Song)
	})

	t.Run("GetBestRecordByChart no record", func(t *testing.T) {
		record, err := repo.GetBestRecordByChart("nobody", massiveID)
		assert.NoError(t, err)
		assert.Nil(t, record)
	})

	countTests := []struct {
		name      string
		fn        func() (int, error)
		wantCount int
	}{
		{"All massive", func() (int, error) {
			r, e := repo.GetAllRecordsByChart("user_chart", massiveID, 10, 0, "score", true)
			return len(r), e
		}, 3},
		{"Pagination page0", func() (int, error) {
			r, e := repo.GetAllRecordsByChart("user_chart", massiveID, 2, 0, "score", true)
			return len(r), e
		}, 2},
		{"Pagination page1", func() (int, error) {
			r, e := repo.GetAllRecordsByChart("user_chart", massiveID, 2, 1, "score", true)
			return len(r), e
		}, 1},
		{"Count massive", func() (int, error) {
			c, e := repo.CountAllRecordsByChart("user_chart", massiveID)
			return int(c), e
		}, 3},
		{"Count detected", func() (int, error) {
			c, e := repo.CountAllRecordsByChart("user_chart", detectedID)
			return int(c), e
		}, 1},
	}

	for _, tt := range countTests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.fn()
			assert.NoError(t, err)
			assert.Equal(t, tt.wantCount, got)
		})
	}
}

func TestRecalculateRatingsByChart(t *testing.T) {
	db := setupTestDB(t)
	songRepo := NewSongRepository(db)
	recordRepo := NewRecordRepository(db)

	song, err := songRepo.CreateSong(&model.Song{
		SongBase: model.SongBase{WikiID: "recalc_song", Title: "Recalc Song"},
		Charts:   []model.Chart{{Difficulty: model.DifficultyMassive, Level: 15.0, Notes: 1000}},
	})
	assert.NoError(t, err)
	chartID := song.Charts[0].ID

	scores := []int{1000000, 1005000, 900000}
	for _, score := range scores {
		_, err := recordRepo.CreateRecord(&model.PlayRecord{
			PlayRecordBase: model.PlayRecordBase{ChartID: chartID, Score: intPtr(score)},
			Username:       "user_recalc",
		}, false)
		assert.NoError(t, err)
	}

	// Verify initial ratings at level 15.0
	var before []model.PlayRecord
	db.Where("chart_id = ? AND username = ?", chartID, "user_recalc").Find(&before)
	assert.Len(t, before, 3)
	for _, r := range before {
		assert.Equal(t, rating.SingleRating(15.0, *r.Score), r.Rating)
	}

	t.Run("Recalculate with new level", func(t *testing.T) {
		assert.NoError(t, RecalculateRatingsByChart(db, chartID, 16.0))

		var after []model.PlayRecord
		db.Where("chart_id = ? AND username = ?", chartID, "user_recalc").Find(&after)
		assert.Len(t, after, 3)
		for _, r := range after {
			expected := rating.SingleRating(16.0, *r.Score)
			assert.Equal(t, expected, r.Rating, "score=%d", *r.Score)
		}
	})

	t.Run("No records for chart", func(t *testing.T) {
		assert.NoError(t, RecalculateRatingsByChart(db, 99999, 16.0))
	})
}

func TestRecordRepository_RecordFilter(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRecordRepository(db)
	songRepo := NewSongRepository(db)

	// Song 1 (b15=false): detected/10.0, invaded/12.5, massive/14.0
	song1, err := songRepo.CreateSong(&model.Song{
		SongBase: model.SongBase{WikiID: "filter_s1", Title: "Filter Song 1", B15: false},
		Charts: []model.Chart{
			{Difficulty: model.DifficultyDetected, Level: 10.0, Notes: 200},
			{Difficulty: model.DifficultyInvaded, Level: 12.5, Notes: 500},
			{Difficulty: model.DifficultyMassive, Level: 14.0, Notes: 800},
		},
	})
	assert.NoError(t, err)

	// Song 2 (b15=true): massive/13.5, reboot/15.0
	song2, err := songRepo.CreateSong(&model.Song{
		SongBase: model.SongBase{WikiID: "filter_s2", Title: "Filter Song 2", B15: true},
		Charts: []model.Chart{
			{Difficulty: model.DifficultyMassive, Level: 13.5, Notes: 700},
			{Difficulty: model.DifficultyReboot, Level: 15.0, Notes: 1000},
		},
	})
	assert.NoError(t, err)

	// Create one play record per chart
	for _, chart := range append(song1.Charts, song2.Charts...) {
		_, err := repo.CreateRecord(&model.PlayRecord{
			PlayRecordBase: model.PlayRecordBase{ChartID: chart.ID, Score: intPtr(1000000)},
			Username:       "filter_user",
		}, false)
		assert.NoError(t, err)
	}

	tests := []struct {
		name            string
		filter          model.RecordFilter
		wantBestCount   int
		wantAllCount    int
		wantB35Count    int
		wantB15Count    int
		wantChartsCount int
	}{
		{"No filter", model.RecordFilter{}, 5, 5, 3, 2, 5},
		{"MinLevel only", model.RecordFilter{MinLevel: float64Ptr(13.0)}, 3, 3, 1, 2, 3},
		{"MaxLevel only", model.RecordFilter{MaxLevel: float64Ptr(13.0)}, 2, 2, 2, 0, 2},
		{"MinLevel and MaxLevel", model.RecordFilter{MinLevel: float64Ptr(12.5), MaxLevel: float64Ptr(14.0)}, 3, 3, 2, 1, 3},
		{"Single difficulty", model.RecordFilter{Difficulties: []model.Difficulty{model.DifficultyMassive}}, 2, 2, 1, 1, 2},
		{"Multiple difficulties", model.RecordFilter{Difficulties: []model.Difficulty{model.DifficultyMassive, model.DifficultyReboot}}, 3, 3, 1, 2, 3},
		{"Combined level+difficulty", model.RecordFilter{MinLevel: float64Ptr(13.0), Difficulties: []model.Difficulty{model.DifficultyMassive}}, 2, 2, 1, 1, 2},
		{"No matches", model.RecordFilter{MinLevel: float64Ptr(20.0)}, 0, 0, 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bestRecords, err := repo.GetBestRecords("filter_user", 100, 0, "rating", true, tt.filter)
			assert.NoError(t, err)
			assert.Len(t, bestRecords, tt.wantBestCount, "GetBestRecords")

			bestCount, err := repo.CountBestRecords("filter_user", tt.filter)
			assert.NoError(t, err)
			assert.Equal(t, int64(tt.wantBestCount), bestCount, "CountBestRecords")

			allRecords, err := repo.GetAllRecords("filter_user", 100, 0, "rating", true, tt.filter)
			assert.NoError(t, err)
			assert.Len(t, allRecords, tt.wantAllCount, "GetAllRecords")

			allCount, err := repo.CountAllRecords("filter_user", tt.filter)
			assert.NoError(t, err)
			assert.Equal(t, int64(tt.wantAllCount), allCount, "CountAllRecords")

			b35, b15, err := repo.GetBest50Records("filter_user", 0, tt.filter)
			assert.NoError(t, err)
			assert.Len(t, b35, tt.wantB35Count, "B35")
			assert.Len(t, b15, tt.wantB15Count, "B15")

			charts, err := repo.GetAllChartsWithBestScores("filter_user", tt.filter)
			assert.NoError(t, err)
			assert.Len(t, charts, tt.wantChartsCount, "AllChartsWithBestScores")
		})
	}
}
