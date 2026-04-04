package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"paradigm-reboot-prober-go/internal/model"
	"paradigm-reboot-prober-go/internal/model/request"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRecordController(t *testing.T) {
	env := setupEnv(t)
	r := gin.Default()

	r.POST("/records/:username", env.recordCtrl.UploadRecords)
	r.GET("/records/:username", env.recordCtrl.GetPlayRecords)

	// Seed data
	user := model.User{
		UserBase: model.UserBase{
			Username:       "testuser",
			UploadToken:    "testtoken",
			AnonymousProbe: true,
		},
	}
	env.db.Create(&user)

	song := model.Song{
		SongBase: model.SongBase{
			Title: "Test Song",
		},
		Charts: []model.Chart{
			{Difficulty: model.DifficultyMassive, Level: 10},
		},
	}
	env.db.Create(&song)

	t.Run("UploadRecords Success", func(t *testing.T) {
		reqBody := request.BatchCreatePlayRecordRequest{
			UploadToken: "testtoken",
			PlayRecords: []model.PlayRecordBase{
				{
					ChartID: 1,
					Score:   1000000,
				},
			},
		}
		body, _ := json.Marshal(reqBody)
		w := performRequest(r, "POST", "/records/testuser", bytes.NewBuffer(body), map[string]string{"Content-Type": "application/json"})

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("GetPlayRecords Success", func(t *testing.T) {
		w := performRequest(r, "GET", "/records/testuser?scope=b50", nil, nil)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "testuser", resp["username"])
	})
}

func TestRecordController_SongAndChartRecords(t *testing.T) {
	env := setupEnv(t)
	r := gin.Default()

	r.GET("/records/:username/song/:song_addr", env.recordCtrl.GetSongRecords)
	r.GET("/records/:username/chart/:chart_addr", env.recordCtrl.GetChartRecords)
	r.POST("/records/:username", env.recordCtrl.UploadRecords)

	// Seed user
	user := model.User{
		UserBase: model.UserBase{
			Username:       "recuser",
			UploadToken:    "rectoken",
			AnonymousProbe: true,
		},
	}
	env.db.Create(&user)

	// Seed songs
	song := model.Song{
		SongBase: model.SongBase{
			WikiID: "felys",
			Title:  "Felys",
		},
		Charts: []model.Chart{
			{Difficulty: model.DifficultyDetected, Level: 5.0, Notes: 200},
			{Difficulty: model.DifficultyMassive, Level: 15.0, Notes: 1000},
		},
	}
	env.db.Create(&song)

	detectedChartID := song.Charts[0].ChartID
	massiveChartID := song.Charts[1].ChartID

	// Upload records via API
	uploadRecords := func(chartID int, score int) {
		reqBody := request.BatchCreatePlayRecordRequest{
			UploadToken: "rectoken",
			PlayRecords: []model.PlayRecordBase{
				{ChartID: chartID, Score: score},
			},
		}
		body, _ := json.Marshal(reqBody)
		performRequest(r, "POST", "/records/recuser", bytes.NewBuffer(body), map[string]string{"Content-Type": "application/json"})
	}

	uploadRecords(detectedChartID, 1000000)
	uploadRecords(massiveChartID, 1005000)
	uploadRecords(massiveChartID, 900000) // lower score, same chart

	// --- Song Records Tests ---

	t.Run("GetSongRecords by wiki_id scope=best", func(t *testing.T) {
		w := performRequest(r, "GET", "/records/recuser/song/felys?scope=best", nil, nil)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp model.PlayRecordResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "recuser", resp.Username)
		assert.Equal(t, 2, resp.Total) // best per difficulty: detected + massive
	})

	t.Run("GetSongRecords by numeric ID scope=best", func(t *testing.T) {
		w := performRequest(r, "GET", fmt.Sprintf("/records/recuser/song/%d?scope=best", song.SongID), nil, nil)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp model.PlayRecordResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, 2, resp.Total)
	})

	t.Run("GetSongRecords scope=all", func(t *testing.T) {
		w := performRequest(r, "GET", "/records/recuser/song/felys?scope=all", nil, nil)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp model.PlayRecordResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, 3, resp.Total) // all 3 records
	})

	t.Run("GetSongRecords not found", func(t *testing.T) {
		w := performRequest(r, "GET", "/records/recuser/song/nonexistent", nil, nil)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("GetSongRecords invalid scope", func(t *testing.T) {
		w := performRequest(r, "GET", "/records/recuser/song/felys?scope=invalid", nil, nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	// --- Chart Records Tests ---

	t.Run("GetChartRecords by wiki_id:difficulty scope=best", func(t *testing.T) {
		w := performRequest(r, "GET", "/records/recuser/chart/felys:massive?scope=best", nil, nil)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp model.PlayRecordResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "recuser", resp.Username)
		assert.Equal(t, 1, resp.Total) // single best record
		assert.Equal(t, 1005000, resp.Records[0].Score)
	})

	t.Run("GetChartRecords by numeric ID scope=best", func(t *testing.T) {
		w := performRequest(r, "GET", fmt.Sprintf("/records/recuser/chart/%d?scope=best", massiveChartID), nil, nil)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp model.PlayRecordResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, 1, resp.Total)
	})

	t.Run("GetChartRecords scope=all", func(t *testing.T) {
		w := performRequest(r, "GET", "/records/recuser/chart/felys:massive?scope=all", nil, nil)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp model.PlayRecordResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, 2, resp.Total) // 2 records on massive chart
	})

	t.Run("GetChartRecords best with no records", func(t *testing.T) {
		w := performRequest(r, "GET", "/records/recuser/chart/felys:reboot?scope=best", nil, nil)
		assert.Equal(t, http.StatusNotFound, w.Code) // reboot chart doesn't exist
	})

	t.Run("GetChartRecords not found wiki_id", func(t *testing.T) {
		w := performRequest(r, "GET", "/records/recuser/chart/nonexistent:massive", nil, nil)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("GetChartRecords invalid difficulty", func(t *testing.T) {
		w := performRequest(r, "GET", "/records/recuser/chart/felys:easy", nil, nil)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("GetChartRecords invalid scope", func(t *testing.T) {
		w := performRequest(r, "GET", "/records/recuser/chart/felys:massive?scope=invalid", nil, nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
