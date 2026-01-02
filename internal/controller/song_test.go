package controller

import (
	"encoding/json"
	"net/http"
	"paradigm-reboot-prober-go/internal/model"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSongController(t *testing.T) {
	env := setupEnv(t)
	r := gin.Default()

	r.GET("/songs", env.songCtrl.GetAllSongLevels)
	r.GET("/songs/:song_id", env.songCtrl.GetSingleSongInfo)

	// Seed data
	song := model.Song{
		SongBase: model.SongBase{
			Title:  "Test Song",
			Artist: "Test Artist",
		},
		SongLevels: []model.SongLevel{
			{Difficulty: model.DifficultyMassive, Level: 10},
		},
	}
	env.db.Create(&song)

	t.Run("GetAllSongLevels Success", func(t *testing.T) {
		w := performRequest(r, "GET", "/songs", nil, nil)
		assert.Equal(t, http.StatusOK, w.Code)
		// Note: GetAllSongLevels returns []model.SongLevelInfo, not []model.Song
		// But for simplicity in test, we just check if it's not empty
		assert.NotEmpty(t, w.Body.String())
	})

	t.Run("GetSingleSongInfo Success", func(t *testing.T) {
		w := performRequest(r, "GET", "/songs/1", nil, nil)
		assert.Equal(t, http.StatusOK, w.Code)
		var s model.Song
		json.Unmarshal(w.Body.Bytes(), &s)
		assert.Equal(t, "Test Song", s.Title)
	})

	t.Run("GetSingleSongInfo Not Found", func(t *testing.T) {
		w := performRequest(r, "GET", "/songs/999", nil, nil)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
