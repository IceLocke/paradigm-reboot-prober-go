package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"paradigm-reboot-prober-go/config"
	"paradigm-reboot-prober-go/internal/model"
	"paradigm-reboot-prober-go/internal/model/request"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordController_FittingB50(t *testing.T) {
	env := setupEnv(t)
	config.GlobalConfig.Game.B35Limit = 1
	config.GlobalConfig.Game.B15Limit = 1
	t.Cleanup(config.InitDefaults)
	r := gin.New()
	r.GET("/records/:username", env.recordCtrl.GetPlayRecords)
	r.POST("/records/:username", env.recordCtrl.UploadRecords)
	require.NoError(t, env.db.Create(&model.User{UserBase: model.UserBase{
		Username: "fituser", Nickname: "Fitting User", AnonymousProbe: true, UploadToken: "fittoken",
	}}).Error)
	require.NoError(t, env.db.Create(&model.User{UserBase: model.UserBase{Username: "privateuser"}}).Error)
	for _, b15 := range []bool{false, true} {
		level := 16.0
		if b15 {
			level++
		}
		song := model.Song{
			SongBase: model.SongBase{WikiID: fmt.Sprintf("fitting_%t", b15), B15: b15},
			Charts: []model.Chart{
				{Difficulty: model.DifficultyMassive, Level: level, FittingLevel: float64Ptr(level - 0.3)},
				{Difficulty: model.DifficultyReboot, Level: level - 0.1, FittingLevel: float64Ptr(level + 0.2)},
			},
		}
		require.NoError(t, env.db.Create(&song).Error)
		for _, chart := range song.Charts {
			uploadTestRecord(r, "fituser", "fittoken", chart.ID, 1000000)
		}
	}

	for _, tc := range []struct {
		query       string
		total       int
		firstRating int
		firstFitted int
	}{
		{"scope=b50", 2, 16000, 15700},
		{"scope=b50&sort_by=score", 2, 16000, 15700},
		{"scope=best&sort_by=invalid&page_size=1", 4, 17000, 16700},
		{"scope=best&sort_by=invalid&page_size=1&order=invalid", 4, 15900, 16200},
		{"scope=b50&sort_by=fitting_rating", 2, 15900, 16200},
		{"scope=b50&sort_by=fitting_rating&underflow=999", 4, 15900, 16200},
		{"scope=b50&sort_by=fitting_rating&underflow=-1", 2, 15900, 16200},
		{"scope=best&sort_by=fitting_rating&page_size=1", 4, 16900, 17200},
		{"scope=best&sort_by=fitting_rating&page_size=1&page_index=2", 4, 17000, 16700},
		{"scope=best&sort_by=fitting_rating&page_size=1&order=asc", 4, 16000, 15700},
		{"scope=b50&sort_by=fitting_rating&b15=true", 1, 16900, 17200},
	} {
		t.Run(tc.query, func(t *testing.T) {
			w := performRequest(r, "GET", "/records/fituser?"+tc.query, nil, nil)
			require.Equal(t, 200, w.Code, w.Body.String())
			var response model.PlayRecordResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
			assert.Equal(t, "Fitting User", response.Nickname)
			assert.Equal(t, tc.total, response.Total)
			require.NotEmpty(t, response.Records)
			assert.Equal(t, tc.firstRating, response.Records[0].Rating)
			assert.Equal(t, tc.firstFitted, response.Records[0].FittingRating)
		})
	}
	t.Run("replacement can lower the current best record", func(t *testing.T) {
		w := performRequest(r, "GET", "/records/fituser?scope=b50&sort_by=fitting_rating", nil, nil)
		require.Equal(t, 200, w.Code, w.Body.String())
		var before model.PlayRecordResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &before))
		require.NotEmpty(t, before.Records)
		previous := before.Records[0]

		body, err := json.Marshal(request.BatchCreatePlayRecordRequest{
			UploadToken: "fittoken",
			IsReplace:   true,
			PlayRecords: []model.PlayRecordBase{{ChartID: previous.Chart.ID, Score: intPtr(900000)}},
		})
		require.NoError(t, err)
		w = performRequest(r, "POST", "/records/fituser", bytes.NewBuffer(body), nil)
		require.Equal(t, 201, w.Code, w.Body.String())

		for _, sortBy := range []model.RecordSort{model.RecordSortRating, model.RecordSortFittingRating} {
			w = performRequest(r, "GET", "/records/fituser?scope=best&sort_by="+string(sortBy), nil, nil)
			require.Equal(t, 200, w.Code, w.Body.String())
			var best model.PlayRecordResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &best))
			require.Equal(t, 4, best.Total)
			found := false
			for _, record := range best.Records {
				assert.NotEqual(t, previous.ID, record.ID, "the historical high score is no longer the current best")
				if record.Chart.ID == previous.Chart.ID {
					found = true
					assert.Equal(t, 900000, record.Score)
					assert.Less(t, record.Rating, previous.Rating)
					assert.Less(t, record.FittingRating, previous.FittingRating)
				}
			}
			assert.True(t, found, "the lower replacement is the chart's current best record")

			w = performRequest(r, "GET", "/records/fituser?scope=b50&sort_by="+string(sortBy), nil, nil)
			require.Equal(t, 200, w.Code, w.Body.String())
			var b50 model.PlayRecordResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &b50))
			require.Len(t, b50.Records, 2)
			assert.Equal(t, 16000, b50.Records[0].Rating)
			assert.NotEqual(t, previous.Chart.ID, b50.Records[0].Chart.ID, "B50 must rerank the current best records after replacement")
		}

		w = performRequest(r, "GET", "/records/fituser?scope=all", nil, nil)
		require.Equal(t, 200, w.Code, w.Body.String())
		var all model.PlayRecordResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &all))
		assert.Equal(t, 5, all.Total, "replacement retains historical attempts")
		assert.Contains(t, all.Records, previous)
	})
	for _, scope := range []string{"all", "all-charts", "invalid"} {
		w := performRequest(r, "GET", "/records/fituser?scope="+scope+"&sort_by=fitting_rating", nil, nil)
		assert.Equal(t, 400, w.Code)
	}
	w := performRequest(r, "GET", "/records/fituser?scope=invalid", nil, nil)
	assert.Equal(t, 400, w.Code)
	w = performRequest(r, "GET", "/records/privateuser?scope=b50&sort_by=fitting_rating", nil, nil)
	assert.Equal(t, 403, w.Code)
}
