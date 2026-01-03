package controller

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
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
		SongLevels: []model.SongLevel{
			{Difficulty: model.DifficultyMassive, Level: 10},
		},
	}
	env.db.Create(&song)

	t.Run("UploadRecords Success", func(t *testing.T) {
		reqBody := request.BatchCreatePlayRecordRequest{
			UploadToken: "testtoken",
			PlayRecords: []model.PlayRecordBase{
				{
					SongLevelID: 1,
					Score:       1000000,
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

	t.Run("UploadCSV Success", func(t *testing.T) {
		// Create a dummy CSV file
		csvContent := "song_name,difficulty,score,clear\nTest Song,Massive,1000000,Pure Memory"

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", "test.csv")
		assert.NoError(t, err)
		_, err = part.Write([]byte(csvContent))
		assert.NoError(t, err)
		_ = writer.Close()

		r.POST("/upload/csv", func(c *gin.Context) {
			c.Set("username", "testuser")
			env.uploadCtrl.UploadCSV(c)
		})

		w := performRequest(r, "POST", "/upload/csv", body, map[string]string{"Content-Type": writer.FormDataContentType()})

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), ".csv")
	})
}
