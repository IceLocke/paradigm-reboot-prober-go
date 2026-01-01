package service

import (
	"paradigm-reboot-prober-go/internal/model"
	"paradigm-reboot-prober-go/internal/model/request"
	"paradigm-reboot-prober-go/internal/repository"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSongService(t *testing.T) {
	db := setupTestDB(t)
	songRepo := repository.NewSongRepository(db)
	songService := NewSongService(songRepo)

	t.Run("CreateSong", func(t *testing.T) {
		req := &request.CreateSongRequest{
			SongBase: model.SongBase{
				WikiID: "test_song",
				Title:  "Test Song",
				Artist: "Test Artist",
			},
			Levels: []model.LevelInfo{
				{
					Difficulty:  model.DifficultyMassive,
					Level:       15.0,
					LevelDesign: "Designer",
					Notes:       1000,
				},
			},
		}
		levels, err := songService.CreateSong(req)
		assert.NoError(t, err)
		assert.Len(t, levels, 1)
		assert.Equal(t, "Test Song", levels[0].Title)
		assert.Equal(t, 15.0, levels[0].Level)
	})

	t.Run("GetSingleSong", func(t *testing.T) {
		song, err := songService.GetSingleSongByWikiID("test_song")
		assert.NoError(t, err)
		assert.NotNil(t, song)
		assert.Equal(t, "Test Song", song.Title)

		songByID, err := songService.GetSingleSong(song.SongID, "prp")
		assert.NoError(t, err)
		assert.NotNil(t, songByID)
		assert.Equal(t, song.SongID, songByID.SongID)
	})

	t.Run("GetAllSongLevels", func(t *testing.T) {
		levels, err := songService.GetAllSongLevels()
		assert.NoError(t, err)
		assert.NotEmpty(t, levels)
		assert.Equal(t, "Test Song", levels[0].Title)
	})

	t.Run("UpdateSong", func(t *testing.T) {
		song, _ := songService.GetSingleSongByWikiID("test_song")
		req := &request.UpdateSongRequest{
			SongID: song.SongID,
			SongBase: model.SongBase{
				WikiID: "test_song",
				Title:  "Updated Song",
				Artist: "Test Artist",
			},
			Levels: []model.LevelInfo{
				{
					Difficulty:  model.DifficultyMassive,
					Level:       15.5,
					LevelDesign: "New Designer",
					Notes:       1100,
				},
			},
		}
		levels, err := songService.UpdateSong(req)
		assert.NoError(t, err)
		assert.Len(t, levels, 1)
		assert.Equal(t, "Updated Song", levels[0].Title)
		assert.Equal(t, 15.5, levels[0].Level)
	})
}
