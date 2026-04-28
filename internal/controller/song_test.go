package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"paradigm-reboot-prober-go/internal/model"
	"paradigm-reboot-prober-go/internal/model/request"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSongController(t *testing.T) {
	env := setupEnv(t)
	r := gin.Default()

	r.GET("/songs", env.songCtrl.GetAllCharts)
	r.GET("/songs/:song_id", env.songCtrl.GetSingleSongInfo)

	// Seed data
	song := model.Song{
		SongBase: model.SongBase{
			Title:  "Test Song",
			Artist: "Test Artist",
		},
		Charts: []model.Chart{
			{Difficulty: model.DifficultyMassive, Level: 10},
		},
	}
	env.db.Create(&song)

	t.Run("GetAllCharts Success", func(t *testing.T) {
		w := performRequest(r, "GET", "/songs", nil, nil)
		assert.Equal(t, http.StatusOK, w.Code)
		// Note: GetAllCharts returns []model.ChartInfo, not []model.Song
		// But for simplicity in test, we just check if it's not empty
		assert.NotEmpty(t, w.Body.String())
		assert.NotEmpty(t, w.Header().Get("ETag"))
		assert.Equal(t, "public, max-age=3600", w.Header().Get("Cache-Control"))
	})

	t.Run("GetAllCharts 304 Not Modified", func(t *testing.T) {
		// First request to get the ETag
		w1 := performRequest(r, "GET", "/songs", nil, nil)
		assert.Equal(t, http.StatusOK, w1.Code)
		etag := w1.Header().Get("ETag")
		assert.NotEmpty(t, etag)

		// Second request with matching If-None-Match
		w2 := performRequest(r, "GET", "/songs", nil, map[string]string{"If-None-Match": etag})
		assert.Equal(t, http.StatusNotModified, w2.Code)
		assert.Empty(t, w2.Body.String())
	})

	t.Run("GetSingleSongInfo Success", func(t *testing.T) {
		w := performRequest(r, "GET", "/songs/1", nil, nil)
		assert.Equal(t, http.StatusOK, w.Code)
		var s model.Song
		err := json.Unmarshal(w.Body.Bytes(), &s)
		assert.NoError(t, err)
		assert.Equal(t, "Test Song", s.Title)
	})

	t.Run("GetSingleSongInfo Not Found", func(t *testing.T) {
		w := performRequest(r, "GET", "/songs/999", nil, nil)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestSongController_CreateSong(t *testing.T) {
	gin.SetMode(gin.TestMode)
	env := setupEnv(t)

	r := gin.Default()
	r.POST("/songs", env.songCtrl.CreateSong)

	reqBody := request.CreateSongRequest{
		SongBase: model.SongBase{
			WikiID:  "test_song",
			Title:   "Test Song",
			Artist:  "Test Artist",
			Genre:   "Pop",
			Version: "1.0.0",
		},
		Charts: []model.ChartInput{
			{
				Difficulty: model.DifficultyMassive,
				Level:      13.5,
				Notes:      1000,
			},
		},
	}
	body, _ := json.Marshal(reqBody)
	w := performRequest(r, "POST", "/songs", bytes.NewBuffer(body), map[string]string{"Content-Type": "application/json"})

	assert.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var charts []model.ChartInfo
	err := json.Unmarshal(w.Body.Bytes(), &charts)
	assert.NoError(t, err)
	assert.Len(t, charts, 1)
	assert.Equal(t, "Test Song", charts[0].Title)
	assert.Equal(t, model.DifficultyMassive, charts[0].Difficulty)
}

func TestSongController_UpdateSong(t *testing.T) {
	gin.SetMode(gin.TestMode)
	env := setupEnv(t)

	// Seed a song
	song := model.Song{
		SongBase: model.SongBase{
			WikiID:  "update_test",
			Title:   "Original Song",
			Artist:  "Original Artist",
			Genre:   "Rock",
			Version: "1.0.0",
		},
		Charts: []model.Chart{
			{Difficulty: model.DifficultyMassive, Level: 10, Notes: 500},
		},
	}
	env.db.Create(&song)

	r := gin.Default()
	r.PUT("/songs", env.songCtrl.UpdateSong)

	reqBody := request.UpdateSongRequest{
		ID: song.ID,
		SongBase: model.SongBase{
			WikiID:  "update_test",
			Title:   "Updated Song",
			Artist:  "Updated Artist",
			Genre:   "Rock",
			Version: "2.0.0",
		},
		Charts: []model.ChartInput{
			{
				Difficulty: model.DifficultyMassive,
				Level:      14.0,
				Notes:      800,
			},
		},
	}
	body, _ := json.Marshal(reqBody)
	w := performRequest(r, "PUT", "/songs", bytes.NewBuffer(body), map[string]string{"Content-Type": "application/json"})

	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var charts []model.ChartInfo
	err := json.Unmarshal(w.Body.Bytes(), &charts)
	assert.NoError(t, err)
	assert.Len(t, charts, 1)
	assert.Equal(t, "Updated Song", charts[0].Title)
	assert.Equal(t, 14.0, charts[0].Level)
}
