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
			Nickname:       "Test Nickname",
			UploadToken:    "testtoken",
			AnonymousProbe: true,
		},
	}
	env.db.Create(&user)

	song := model.Song{
		SongBase: model.SongBase{Title: "Test Song"},
		Charts:   []model.Chart{{Difficulty: model.DifficultyMassive, Level: 10}},
	}
	env.db.Create(&song)

	t.Run("UploadRecords Success", func(t *testing.T) {
		reqBody := request.BatchCreatePlayRecordRequest{
			UploadToken: "testtoken",
			PlayRecords: []model.PlayRecordBase{{ChartID: 1, Score: intPtr(1000000)}},
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
		assert.Equal(t, "Test Nickname", resp["nickname"])
	})
}

// uploadTestRecord is a helper that uploads a single record via the API
func uploadTestRecord(r http.Handler, username, token string, chartID, score int) {
	reqBody := request.BatchCreatePlayRecordRequest{
		UploadToken: token,
		PlayRecords: []model.PlayRecordBase{{ChartID: chartID, Score: intPtr(score)}},
	}
	body, _ := json.Marshal(reqBody)
	performRequest(r, "POST", "/records/"+username, bytes.NewBuffer(body), map[string]string{"Content-Type": "application/json"})
}

func TestRecordController_SongAndChartRecords(t *testing.T) {
	env := setupEnv(t)
	r := gin.Default()

	r.GET("/records/:username/song/:song_addr", env.recordCtrl.GetSongRecords)
	r.GET("/records/:username/chart/:chart_addr", env.recordCtrl.GetChartRecords)
	r.POST("/records/:username", env.recordCtrl.UploadRecords)

	// Seed user
	env.db.Create(&model.User{
		UserBase: model.UserBase{
			Username: "recuser", Nickname: "Rec Nickname",
			UploadToken: "rectoken", AnonymousProbe: true,
		},
	})

	// Seed song with two charts
	song := model.Song{
		SongBase: model.SongBase{WikiID: "felys", Title: "Felys"},
		Charts: []model.Chart{
			{Difficulty: model.DifficultyDetected, Level: 5.0, Notes: 200},
			{Difficulty: model.DifficultyMassive, Level: 15.0, Notes: 1000},
		},
	}
	env.db.Create(&song)
	detectedChartID := song.Charts[0].ID
	massiveChartID := song.Charts[1].ID

	// Upload records
	uploadTestRecord(r, "recuser", "rectoken", detectedChartID, 1000000)
	uploadTestRecord(r, "recuser", "rectoken", massiveChartID, 1005000)
	uploadTestRecord(r, "recuser", "rectoken", massiveChartID, 900000)

	// --- Song and Chart record query tests (table-driven) ---
	tests := []struct {
		name       string
		url        string
		wantStatus int
		wantTotal  int    // -1 to skip total check
		checkScore int    // 0 to skip score check
		respType   string // "records" or "error"
	}{
		// Song records
		{"Song by wiki_id scope=best", "/records/recuser/song/felys?scope=best", 200, 2, 0, "records"},
		{"Song by numeric ID scope=best", fmt.Sprintf("/records/recuser/song/%d?scope=best", song.ID), 200, 2, 0, "records"},
		{"Song scope=all", "/records/recuser/song/felys?scope=all", 200, 3, 0, "records"},
		{"Song not found", "/records/recuser/song/nonexistent", 404, -1, 0, "error"},
		{"Song invalid scope", "/records/recuser/song/felys?scope=invalid", 400, -1, 0, "error"},
		// Chart records
		{"Chart by wiki_id:difficulty scope=best", "/records/recuser/chart/felys:massive?scope=best", 200, 1, 1005000, "records"},
		{"Chart by numeric ID scope=best", fmt.Sprintf("/records/recuser/chart/%d?scope=best", massiveChartID), 200, 1, 0, "records"},
		{"Chart scope=all", "/records/recuser/chart/felys:massive?scope=all", 200, 2, 0, "records"},
		{"Chart no records (reboot)", "/records/recuser/chart/felys:reboot?scope=best", 404, -1, 0, "error"},
		{"Chart not found wiki_id", "/records/recuser/chart/nonexistent:massive", 404, -1, 0, "error"},
		{"Chart invalid difficulty", "/records/recuser/chart/felys:easy", 404, -1, 0, "error"},
		{"Chart invalid scope", "/records/recuser/chart/felys:massive?scope=invalid", 400, -1, 0, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := performRequest(r, "GET", tt.url, nil, nil)
			assert.Equal(t, tt.wantStatus, w.Code)

			if tt.respType == "records" && tt.wantTotal >= 0 {
				var resp model.PlayRecordResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, "recuser", resp.Username)
				assert.Equal(t, "Rec Nickname", resp.Nickname)
				assert.Equal(t, tt.wantTotal, resp.Total)
				if tt.checkScore > 0 {
					assert.Equal(t, tt.checkScore, resp.Records[0].Score)
				}
			}
		})
	}
}

func TestRecordController_RecordFilter(t *testing.T) {
	env := setupEnv(t)
	r := gin.Default()

	r.POST("/records/:username", env.recordCtrl.UploadRecords)
	r.GET("/records/:username", env.recordCtrl.GetPlayRecords)

	// Seed user
	env.db.Create(&model.User{
		UserBase: model.UserBase{
			Username: "filteruser", Nickname: "Filter User",
			UploadToken: "filtertoken", AnonymousProbe: true,
		},
	})

	// Song 1 (b15=false): detected/10.0, invaded/12.5, massive/14.0
	song1 := model.Song{
		SongBase: model.SongBase{WikiID: "ctrl_fs1", Title: "Ctrl Filter Song 1", B15: false},
		Charts: []model.Chart{
			{Difficulty: model.DifficultyDetected, Level: 10.0, Notes: 200},
			{Difficulty: model.DifficultyInvaded, Level: 12.5, Notes: 500},
			{Difficulty: model.DifficultyMassive, Level: 14.0, Notes: 800},
		},
	}
	env.db.Create(&song1)

	// Song 2 (b15=true): massive/13.5, reboot/15.0
	song2 := model.Song{
		SongBase: model.SongBase{WikiID: "ctrl_fs2", Title: "Ctrl Filter Song 2", B15: true},
		Charts: []model.Chart{
			{Difficulty: model.DifficultyMassive, Level: 13.5, Notes: 700},
			{Difficulty: model.DifficultyReboot, Level: 15.0, Notes: 1000},
		},
	}
	env.db.Create(&song2)

	// Upload records for each chart
	for _, chart := range append(song1.Charts, song2.Charts...) {
		uploadTestRecord(r, "filteruser", "filtertoken", chart.ID, 1000000)
	}

	// Happy-path filter tests (table-driven)
	type recordCheck func(t *testing.T, resp model.PlayRecordResponse)

	recordTests := []struct {
		name       string
		url        string
		wantTotal  int
		checkFn    recordCheck // optional per-record validation
	}{
		{
			"min_level=13.0", "/records/filteruser?scope=best&min_level=13.0", 3,
			func(t *testing.T, resp model.PlayRecordResponse) {
				for _, rec := range resp.Records {
					assert.GreaterOrEqual(t, rec.Chart.Level, 13.0)
				}
			},
		},
		{
			"max_level=13.0", "/records/filteruser?scope=best&max_level=13.0", 2,
			func(t *testing.T, resp model.PlayRecordResponse) {
				for _, rec := range resp.Records {
					assert.LessOrEqual(t, rec.Chart.Level, 13.0)
				}
			},
		},
		{
			"difficulty=massive+reboot", "/records/filteruser?scope=best&difficulty=massive&difficulty=reboot", 3,
			func(t *testing.T, resp model.PlayRecordResponse) {
				for _, rec := range resp.Records {
					assert.Contains(t, []model.Difficulty{model.DifficultyMassive, model.DifficultyReboot}, rec.Chart.Difficulty)
				}
			},
		},
		{
			"combined level+difficulty", "/records/filteruser?scope=best&min_level=12.0&max_level=14.0&difficulty=massive", 2,
			func(t *testing.T, resp model.PlayRecordResponse) {
				for _, rec := range resp.Records {
					assert.Equal(t, model.DifficultyMassive, rec.Chart.Difficulty)
					assert.GreaterOrEqual(t, rec.Chart.Level, 12.0)
					assert.LessOrEqual(t, rec.Chart.Level, 14.0)
				}
			},
		},
		{
			"scope=all difficulty=massive", "/records/filteruser?scope=all&difficulty=massive", 2,
			func(t *testing.T, resp model.PlayRecordResponse) {
				for _, rec := range resp.Records {
					assert.Equal(t, model.DifficultyMassive, rec.Chart.Difficulty)
				}
			},
		},
		{
			"scope=b50 min_level=14.0", "/records/filteruser?scope=b50&min_level=14.0", 2,
			func(t *testing.T, resp model.PlayRecordResponse) {
				for _, rec := range resp.Records {
					assert.GreaterOrEqual(t, rec.Chart.Level, 14.0)
				}
			},
		},
	}

	for _, tt := range recordTests {
		t.Run(tt.name, func(t *testing.T) {
			w := performRequest(r, "GET", tt.url, nil, nil)
			assert.Equal(t, http.StatusOK, w.Code)
			var resp model.PlayRecordResponse
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantTotal, resp.Total)
			assert.Len(t, resp.Records, tt.wantTotal)
			if tt.checkFn != nil {
				tt.checkFn(t, resp)
			}
		})
	}

	// all-charts scope with filter
	t.Run("scope=all-charts difficulty=massive", func(t *testing.T) {
		w := performRequest(r, "GET", "/records/filteruser?scope=all-charts&difficulty=massive", nil, nil)
		assert.Equal(t, http.StatusOK, w.Code)
		var resp model.AllChartsResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Len(t, resp.Charts, 2)
		for _, ch := range resp.Charts {
			assert.Equal(t, model.DifficultyMassive, ch.Difficulty)
		}
	})

	// Error cases (table-driven)
	errorTests := []struct {
		name         string
		url          string
		wantContains string
	}{
		{"invalid difficulty", "/records/filteruser?scope=best&difficulty=invalid_diff", "invalid difficulty"},
		{"invalid min_level", "/records/filteruser?scope=best&min_level=abc", "invalid min_level"},
		{"invalid max_level", "/records/filteruser?scope=best&max_level=xyz", "invalid max_level"},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			w := performRequest(r, "GET", tt.url, nil, nil)
			assert.Equal(t, http.StatusBadRequest, w.Code)
			var resp model.Response
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)
			assert.Contains(t, resp.Error, tt.wantContains)
		})
	}
}
