package service

import (
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

	// Setup Song and Level
	song := &model.Song{
		SongBase: model.SongBase{WikiID: "song_1", Title: "Test Song"},
		SongLevels: []model.SongLevel{
			{Difficulty: model.DifficultyMassive, Level: 15.0},
		},
	}
	createdSong, _ := songRepo.CreateSong(song)
	levelID := createdSong.SongLevels[0].SongLevelID

	t.Run("CreateRecords", func(t *testing.T) {
		records := []model.PlayRecordBase{
			{
				SongLevelID: levelID,
				Score:       1000000,
			},
		}
		savedRecords, err := recordService.CreateRecords("testuser", records, false)
		assert.NoError(t, err)
		assert.Len(t, savedRecords, 1)
		assert.Equal(t, 1000000, savedRecords[0].Score)
		assert.Equal(t, "testuser", savedRecords[0].Username)
	})

	t.Run("GetAllRecords", func(t *testing.T) {
		records, err := recordService.GetAllRecords("testuser", 10, 0, "score", "desc")
		assert.NoError(t, err)
		assert.NotEmpty(t, records)
		assert.Equal(t, 1000000, records[0].Score)
	})

	t.Run("GetBest50Records", func(t *testing.T) {
		records, err := recordService.GetBest50Records("testuser", 0)
		assert.NoError(t, err)
		assert.NotEmpty(t, records)
	})

	t.Run("GetBestRecords", func(t *testing.T) {
		records, err := recordService.GetBestRecords("testuser", 10, 0, "score", "desc")
		assert.NoError(t, err)
		assert.NotEmpty(t, records)
	})

	t.Run("GetAllLevelsWithBestScores", func(t *testing.T) {
		levels, err := recordService.GetAllLevelsWithBestScores("testuser")
		assert.NoError(t, err)
		assert.NotEmpty(t, levels)
		assert.Equal(t, 1000000, levels[0].Score)
	})

	t.Run("CountRecords", func(t *testing.T) {
		bestCount, err := recordService.CountBestRecords("testuser")
		assert.NoError(t, err)
		assert.Equal(t, int64(1), bestCount)

		allCount, err := recordService.CountAllRecords("testuser")
		assert.NoError(t, err)
		assert.Equal(t, int64(1), allCount)
	})
}
