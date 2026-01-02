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

	r.POST("/records", env.recordCtrl.UploadRecords)
	r.GET("/b50", env.recordCtrl.GetB50)

	// Seed data
	user := model.User{
		UserBase: model.UserBase{
			Username:    "testuser",
			UploadToken: "testtoken",
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
		w := performRequest(r, "POST", "/records", bytes.NewBuffer(body), map[string]string{"Content-Type": "application/json"})

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "1000000")
	})

	t.Run("GetB50 Success", func(t *testing.T) {
		w := performRequest(r, "GET", "/b50?username=testuser", nil, nil)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "testuser")
	})

	t.Run("UploadCSV Success", func(t *testing.T) {
		// Create a dummy CSV file
		csvContent := "song_name,difficulty,score,clear\nTest Song,Massive,1000000,Pure Memory"

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "test.csv")
		part.Write([]byte(csvContent))
		writer.Close()

		r.POST("/upload/csv", func(c *gin.Context) {
			c.Set("username", "testuser")
			env.uploadCtrl.UploadCSV(c)
		})

		w := performRequest(r, "POST", "/upload/csv", body, map[string]string{"Content-Type": writer.FormDataContentType()})

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), ".csv")
	})
}
