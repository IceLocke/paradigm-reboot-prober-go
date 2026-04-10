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
