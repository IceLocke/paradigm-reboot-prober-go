package service

import (
	"context"
	"paradigm-reboot-prober-go/internal/model"
	"paradigm-reboot-prober-go/internal/repository"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecordService(t *testing.T) {
	db := setupTestDB(t)
	recordRepo := repository.NewRecordRepository(db)
	songRepo := repository.NewSongRepository(db)
	recordService := NewRecordService(recordRepo, songRepo)
	ctx := context.Background()

	// Setup Song and Chart
	song := &model.Song{
		SongBase: model.SongBase{WikiID: "song_1", Title: "Test Song"},
		Charts: []model.Chart{
			{Difficulty: model.DifficultyMassive, Level: 15.0},
		},
	}
	createdSong, _ := songRepo.CreateSong(song)
	chartID := createdSong.Charts[0].ID

	t.Run("CreateRecords", func(t *testing.T) {
		records := []model.PlayRecordBase{
			{
				ChartID: chartID,
				Score:   intPtr(1000000),
			},
		}
		savedRecords, err := recordService.CreateRecords(ctx, "testuser", records, false)
		assert.NoError(t, err)
		assert.Len(t, savedRecords, 1)
		assert.Equal(t, 1000000, *savedRecords[0].Score)
		assert.Equal(t, "testuser", savedRecords[0].Username)
	})

	t.Run("GetAllRecords", func(t *testing.T) {
		records, err := recordService.GetAllRecords(ctx, "testuser", 10, 0, "score", "desc", model.RecordFilter{})
		assert.NoError(t, err)
		assert.NotEmpty(t, records)
		assert.Equal(t, 1000000, *records[0].Score)
	})

	t.Run("GetBest50Records", func(t *testing.T) {
		records, err := recordService.GetBest50Records(ctx, "testuser", 0, model.RecordFilter{})
		assert.NoError(t, err)
		assert.NotEmpty(t, records)
	})

	t.Run("GetBestRecords", func(t *testing.T) {
		records, err := recordService.GetBestRecords(ctx, "testuser", 10, 0, "score", "desc", model.RecordFilter{})
		assert.NoError(t, err)
		assert.NotEmpty(t, records)
	})

	t.Run("GetAllChartsWithBestScores", func(t *testing.T) {
		charts, err := recordService.GetAllChartsWithBestScores(ctx, "testuser", model.RecordFilter{})
		assert.NoError(t, err)
		assert.NotEmpty(t, charts)
		assert.Equal(t, 1000000, charts[0].Score)
	})

	t.Run("CountRecords", func(t *testing.T) {
		bestCount, err := recordService.CountBestRecords(ctx, "testuser", model.RecordFilter{})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), bestCount)

		allCount, err := recordService.CountAllRecords(ctx, "testuser", model.RecordFilter{})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), allCount)
	})
}

func TestRecordService_PerSongAndChartMethods(t *testing.T) {
	db := setupTestDB(t)
	recordRepo := repository.NewRecordRepository(db)
	songRepo := repository.NewSongRepository(db)
	recordService := NewRecordService(recordRepo, songRepo)
	ctx := context.Background()

	// Setup Song with 2 Charts
	song := &model.Song{
		SongBase: model.SongBase{WikiID: "persong_1", Title: "PerSong Test Song"},
		Charts: []model.Chart{
			{Difficulty: model.DifficultyMassive, Level: 15.0},
			{Difficulty: model.DifficultyInvaded, Level: 12.0},
		},
	}
	createdSong, err := songRepo.CreateSong(song)
	assert.NoError(t, err)
	chart1ID := createdSong.Charts[0].ID // massive
	chart2ID := createdSong.Charts[1].ID // invaded
	songID := createdSong.ID

	// Seed records for chart1 (massive): 3 records with different scores
	_, err = recordService.CreateRecords(ctx, "testuser", []model.PlayRecordBase{
		{ChartID: chart1ID, Score: intPtr(900000)},
		{ChartID: chart1ID, Score: intPtr(950000)},
		{ChartID: chart1ID, Score: intPtr(1000000)},
	}, false)
	assert.NoError(t, err)

	// Seed records for chart2 (invaded): 2 records with different scores
	_, err = recordService.CreateRecords(ctx, "testuser", []model.PlayRecordBase{
		{ChartID: chart2ID, Score: intPtr(800000)},
		{ChartID: chart2ID, Score: intPtr(850000)},
	}, false)
	assert.NoError(t, err)

	t.Run("GetBestRecordsBySong", func(t *testing.T) {
		records, err := recordService.GetBestRecordsBySong(ctx, "testuser", songID)
		assert.NoError(t, err)
		assert.Len(t, records, 2)
		// Records are ordered by rating desc; check that each chart has its best score
		scores := map[int]int{}
		for _, r := range records {
			scores[r.ChartID] = *r.Score
		}
		assert.Equal(t, 1000000, scores[chart1ID])
		assert.Equal(t, 850000, scores[chart2ID])
	})

	t.Run("GetAllRecordsBySong", func(t *testing.T) {
		records, err := recordService.GetAllRecordsBySong(ctx, "testuser", songID, 10, 0, "score", "desc")
		assert.NoError(t, err)
		assert.Len(t, records, 5)
		// First record should have the highest score
		assert.Equal(t, 1000000, *records[0].Score)
	})

	t.Run("GetAllRecordsBySong_Pagination", func(t *testing.T) {
		records, err := recordService.GetAllRecordsBySong(ctx, "testuser", songID, 2, 0, "score", "desc")
		assert.NoError(t, err)
		assert.Len(t, records, 2)

		records2, err := recordService.GetAllRecordsBySong(ctx, "testuser", songID, 2, 1, "score", "desc")
		assert.NoError(t, err)
		assert.Len(t, records2, 2)

		// Pages should not overlap
		assert.NotEqual(t, records[0].ID, records2[0].ID)
	})

	t.Run("CountAllRecordsBySong", func(t *testing.T) {
		count, err := recordService.CountAllRecordsBySong(ctx, "testuser", songID)
		assert.NoError(t, err)
		assert.Equal(t, int64(5), count)
	})

	t.Run("GetBestRecordByChart", func(t *testing.T) {
		record, err := recordService.GetBestRecordByChart(ctx, "testuser", chart1ID)
		assert.NoError(t, err)
		assert.NotNil(t, record)
		assert.Equal(t, 1000000, *record.Score)
		assert.Equal(t, chart1ID, record.ChartID)

		record2, err := recordService.GetBestRecordByChart(ctx, "testuser", chart2ID)
		assert.NoError(t, err)
		assert.NotNil(t, record2)
		assert.Equal(t, 850000, *record2.Score)
		assert.Equal(t, chart2ID, record2.ChartID)
	})

	t.Run("GetBestRecordByChart_NotFound", func(t *testing.T) {
		record, err := recordService.GetBestRecordByChart(ctx, "testuser", 99999)
		assert.NoError(t, err)
		assert.Nil(t, record)
	})

	t.Run("GetAllRecordsByChart", func(t *testing.T) {
		records, err := recordService.GetAllRecordsByChart(ctx, "testuser", chart1ID, 10, 0, "score", "desc")
		assert.NoError(t, err)
		assert.Len(t, records, 3)
		assert.Equal(t, 1000000, *records[0].Score)

		records2, err := recordService.GetAllRecordsByChart(ctx, "testuser", chart2ID, 10, 0, "score", "desc")
		assert.NoError(t, err)
		assert.Len(t, records2, 2)
		assert.Equal(t, 850000, *records2[0].Score)
	})

	t.Run("GetAllRecordsByChart_Pagination", func(t *testing.T) {
		records, err := recordService.GetAllRecordsByChart(ctx, "testuser", chart1ID, 2, 0, "score", "desc")
		assert.NoError(t, err)
		assert.Len(t, records, 2)

		records2, err := recordService.GetAllRecordsByChart(ctx, "testuser", chart1ID, 2, 1, "score", "desc")
		assert.NoError(t, err)
		assert.Len(t, records2, 1)
	})

	t.Run("CountAllRecordsByChart", func(t *testing.T) {
		count, err := recordService.CountAllRecordsByChart(ctx, "testuser", chart1ID)
		assert.NoError(t, err)
		assert.Equal(t, int64(3), count)

		count2, err := recordService.CountAllRecordsByChart(ctx, "testuser", chart2ID)
		assert.NoError(t, err)
		assert.Equal(t, int64(2), count2)
	})
}
