package controller

import (
	"encoding/json"
	"fmt"
	"paradigm-reboot-prober-go/config"
	"paradigm-reboot-prober-go/internal/model"
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
	for _, scope := range []string{"all", "all-charts"} {
		w := performRequest(r, "GET", "/records/fituser?scope="+scope+"&sort_by=fitting_rating", nil, nil)
		assert.Equal(t, 400, w.Code)
	}
	w := performRequest(r, "GET", "/records/privateuser?scope=b50&sort_by=fitting_rating", nil, nil)
	assert.Equal(t, 403, w.Code)
}
