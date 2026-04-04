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
			Charts: []model.ChartInput{
				{
					Difficulty:  model.DifficultyMassive,
					Level:       15.0,
					LevelDesign: "Designer",
					Notes:       1000,
				},
			},
		}
		charts, err := songService.CreateSong(req)
		assert.NoError(t, err)
		assert.Len(t, charts, 1)
		assert.Equal(t, "Test Song", charts[0].Title)
		assert.Equal(t, 15.0, charts[0].Level)
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

	t.Run("GetAllCharts", func(t *testing.T) {
		charts, err := songService.GetAllCharts()
		assert.NoError(t, err)
		assert.NotEmpty(t, charts)
		assert.Equal(t, "Test Song", charts[0].Title)
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
			Charts: []model.ChartInput{
				{
					Difficulty:  model.DifficultyMassive,
					Level:       15.5,
					LevelDesign: "New Designer",
					Notes:       1100,
				},
			},
		}
		charts, err := songService.UpdateSong(req)
		assert.NoError(t, err)
		assert.Len(t, charts, 1)
		assert.Equal(t, "Updated Song", charts[0].Title)
		assert.Equal(t, 15.5, charts[0].Level)
	})
}
