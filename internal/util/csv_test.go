package util

import (
	"os"
	"paradigm-reboot-prober-go/internal/model"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/text/encoding/simplifiedchinese"
)

func TestGenerateCSV(t *testing.T) {
	records := []model.SongLevelWithScore{
		{
			SongLevelID: 1,
			Title:       "Test Song",
			Version:     "1.0",
			Difficulty:  model.DifficultyMassive,
			Level:       10.5,
			Score:       1000000,
		},
	}

	csvStr, err := GenerateCSV(records)
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(csvStr, "\ufeff"))
	assert.Contains(t, csvStr, "song_level_id,title,version,difficulty,level,score")
	assert.Contains(t, csvStr, "1,Test Song,1.0,Massive,10.5,1000000")
}

func TestGenerateEmptyCSVAndGetRecords(t *testing.T) {
	// Setup temporary file
	tmpFile, err := os.CreateTemp("", "test_*.csv")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	songLevels := []model.SongLevel{
		{
			SongLevelID: 1,
			Difficulty:  model.DifficultyMassive,
			Level:       10.5,
			Song: &model.Song{
				SongBase: model.SongBase{
					Title:   "Test Song",
					Version: "1.0",
				},
			},
		},
	}

	// Test GenerateEmptyCSV
	err = GenerateEmptyCSV(tmpFile.Name(), songLevels)
	assert.NoError(t, err)

	// Verify file content
	content, err := os.ReadFile(tmpFile.Name())
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(content), "\ufeff"))
	assert.Contains(t, string(content), "1,Test Song,1.0,Massive,10.5,0")

	// Test GetRecordsFromCSV
	// First, let's modify the file to add a score
	modifiedContent := strings.Replace(string(content), "1,Test Song,1.0,Massive,10.5,0", "1,Test Song,1.0,Massive,10.5,999999", 1)
	err = os.WriteFile(tmpFile.Name(), []byte(modifiedContent), 0644)
	assert.NoError(t, err)

	records, err := GetRecordsFromCSV(tmpFile.Name())
	assert.NoError(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, 1, records[0].SongLevelID)
	assert.Equal(t, 999999, records[0].Score)
}

func TestGetRecordsFromCSV_Malformed(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "malformed_*.csv")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	content := "song_level_id,score\n1,1000000\ninvalid,row\n2,900000"
	tmpFile.WriteString(content)
	tmpFile.Close()

	records, err := GetRecordsFromCSV(tmpFile.Name())
	assert.NoError(t, err)
	assert.Len(t, records, 2)
	assert.Equal(t, 1, records[0].SongLevelID)
	assert.Equal(t, 2, records[1].SongLevelID)
}

func TestGetRecordsFromCSV_GBK(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "gbk_*.csv")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// GBK encoded content: "song_level_id,score\n1,1000000"
	// In GBK, these characters are the same as ASCII, but we can add some Chinese to be sure
	gbkHeader := "song_level_id,score,备注\n"
	gbkData := "1,1000000,测试"

	encoder := simplifiedchinese.GBK.NewEncoder()
	gbkContent, _ := encoder.String(gbkHeader + gbkData)

	err = os.WriteFile(tmpFile.Name(), []byte(gbkContent), 0644)
	assert.NoError(t, err)

	records, err := GetRecordsFromCSV(tmpFile.Name())
	assert.NoError(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, 1, records[0].SongLevelID)
	assert.Equal(t, 1000000, records[0].Score)
}
