package repository

import (
	"paradigm-reboot-prober-go/internal/model"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecordRepository_CreateRecord(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRecordRepository(db)
	songRepo := NewSongRepository(db)

	// Setup Song and Chart
	song := &model.Song{
		SongBase: model.SongBase{WikiID: "rec_song", Title: "Record Song"},
		Charts: []model.Chart{
			{Difficulty: model.DifficultyMassive, Level: 15.0, Notes: 1000},
		},
	}
	createdSong, _ := songRepo.CreateSong(song)
	chartID := createdSong.Charts[0].ID

	t.Run("Create New Record", func(t *testing.T) {
		record := &model.PlayRecord{
			PlayRecordBase: model.PlayRecordBase{
				ChartID: chartID,
				Score:   1000000,
			},
			Username: "testuser",
		}
		savedRecord, err := repo.CreateRecord(record, false)
		assert.NoError(t, err)
		assert.NotNil(t, savedRecord)
		assert.Equal(t, 1000000, savedRecord.Score)
		assert.Equal(t, "testuser", savedRecord.Username)
		// Level 15.0, Score 1000000 -> Rating 150.0 -> 15000
		assert.Equal(t, 15000, savedRecord.Rating)

		// Verify Best Record created
		var best model.BestPlayRecord
		err = db.Where("play_record_id = ?", savedRecord.ID).First(&best).Error
		assert.NoError(t, err)
		assert.Equal(t, savedRecord.ID, best.PlayRecordID)
	})

	t.Run("Update Best Record (Higher Score)", func(t *testing.T) {
		// Previous best was 1000000
		record := &model.PlayRecord{
			PlayRecordBase: model.PlayRecordBase{
				ChartID: chartID,
				Score:   1000001,
			},
			Username: "testuser",
		}
		savedRecord, err := repo.CreateRecord(record, false)
		assert.NoError(t, err)

		// Verify Best Record updated
		var best model.BestPlayRecord
		// We need to join to find the best record for this user/level, but here we can just check if the new record ID is in best_play_records table
		// Actually, let's query by play_record_id
		err = db.Where("play_record_id = ?", savedRecord.ID).First(&best).Error
		assert.NoError(t, err)
	})

	t.Run("Do Not Update Best Record (Lower Score)", func(t *testing.T) {
		// Previous best was 1000001
		record := &model.PlayRecord{
			PlayRecordBase: model.PlayRecordBase{
				ChartID: chartID,
				Score:   900000,
			},
			Username: "testuser",
		}
		savedRecord, err := repo.CreateRecord(record, false)
		assert.NoError(t, err)

		// Verify Best Record NOT updated (should still point to previous high score)
		var best model.BestPlayRecord
		err = db.Where("play_record_id = ?", savedRecord.ID).First(&best).Error
		assert.Error(t, err) // Should not find this record as best
		assert.Equal(t, "record not found", err.Error())
	})

	t.Run("Force Update Best Record (isReplaced=true)", func(t *testing.T) {
		// Previous best was 1000001. New score is lower but forced.
		record := &model.PlayRecord{
			PlayRecordBase: model.PlayRecordBase{
				ChartID: chartID,
				Score:   800000,
			},
			Username: "testuser",
		}
		savedRecord, err := repo.CreateRecord(record, true)
		assert.NoError(t, err)

		// Verify Best Record updated
		var best model.BestPlayRecord
		err = db.Where("play_record_id = ?", savedRecord.ID).First(&best).Error
		assert.NoError(t, err)
	})
}

func TestRecordRepository_GetBest50Records(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRecordRepository(db)
	songRepo := NewSongRepository(db)

	// Create B15 Song (New)
	songB15 := &model.Song{
		SongBase: model.SongBase{WikiID: "b15_song", Title: "B15 Song", B15: true},
		Charts: []model.Chart{
			{Difficulty: model.DifficultyMassive, Level: 15.0},
		},
	}
	createdB15, _ := songRepo.CreateSong(songB15)

	// Create Non-B15 Song (Old)
	songOld := &model.Song{
		SongBase: model.SongBase{WikiID: "old_song", Title: "Old Song", B15: false},
		Charts: []model.Chart{
			{Difficulty: model.DifficultyMassive, Level: 15.0},
		},
	}
	createdOld, _ := songRepo.CreateSong(songOld)

	// Create Records
	_, err := repo.CreateRecord(&model.PlayRecord{
		PlayRecordBase: model.PlayRecordBase{ChartID: createdB15.Charts[0].ID, Score: 1000000},
		Username:       "user_b50",
	}, false)
	assert.NoError(t, err)
	_, err = repo.CreateRecord(&model.PlayRecord{
		PlayRecordBase: model.PlayRecordBase{ChartID: createdOld.Charts[0].ID, Score: 1000000},
		Username:       "user_b50",
	}, false)
	assert.NoError(t, err)

	t.Run("Get B50", func(t *testing.T) {
		b35, b15, err := repo.GetBest50Records("user_b50", 0)
		assert.NoError(t, err)
		assert.Len(t, b15, 1)
		assert.Len(t, b35, 1)
		assert.Equal(t, "B15 Song", b15[0].Chart.Song.Title)
		assert.Equal(t, "Old Song", b35[0].Chart.Song.Title)
	})
}

func TestRecordRepository_PerSongQueries(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRecordRepository(db)
	songRepo := NewSongRepository(db)

	// Create two songs with charts
	song1 := &model.Song{
		SongBase: model.SongBase{WikiID: "song_a", Title: "Song A"},
		Charts: []model.Chart{
			{Difficulty: model.DifficultyDetected, Level: 5.0, Notes: 200},
			{Difficulty: model.DifficultyMassive, Level: 15.0, Notes: 1000},
		},
	}
	created1, _ := songRepo.CreateSong(song1)

	song2 := &model.Song{
		SongBase: model.SongBase{WikiID: "song_b", Title: "Song B"},
		Charts: []model.Chart{
			{Difficulty: model.DifficultyMassive, Level: 14.0, Notes: 900},
		},
	}
	created2, _ := songRepo.CreateSong(song2)

	// Create records for user on song1 charts
	_, err := repo.CreateRecord(&model.PlayRecord{
		PlayRecordBase: model.PlayRecordBase{ChartID: created1.Charts[0].ID, Score: 1000000},
		Username:       "user_song",
	}, false)
	assert.NoError(t, err)

	_, err = repo.CreateRecord(&model.PlayRecord{
		PlayRecordBase: model.PlayRecordBase{ChartID: created1.Charts[1].ID, Score: 1005000},
		Username:       "user_song",
	}, false)
	assert.NoError(t, err)

	// A second record on same chart (lower score)
	_, err = repo.CreateRecord(&model.PlayRecord{
		PlayRecordBase: model.PlayRecordBase{ChartID: created1.Charts[1].ID, Score: 900000},
		Username:       "user_song",
	}, false)
	assert.NoError(t, err)

	// Create record for user on song2 chart
	_, err = repo.CreateRecord(&model.PlayRecord{
		PlayRecordBase: model.PlayRecordBase{ChartID: created2.Charts[0].ID, Score: 1000000},
		Username:       "user_song",
	}, false)
	assert.NoError(t, err)

	t.Run("GetBestRecordsBySong", func(t *testing.T) {
		records, err := repo.GetBestRecordsBySong("user_song", created1.ID)
		assert.NoError(t, err)
		assert.Len(t, records, 2) // one per difficulty
		// Should be ordered by rating desc
		assert.True(t, records[0].Rating >= records[1].Rating)
	})

	t.Run("GetBestRecordsBySong - other song", func(t *testing.T) {
		records, err := repo.GetBestRecordsBySong("user_song", created2.ID)
		assert.NoError(t, err)
		assert.Len(t, records, 1)
	})

	t.Run("GetAllRecordsBySong", func(t *testing.T) {
		records, err := repo.GetAllRecordsBySong("user_song", created1.ID, 10, 0, "rating", true)
		assert.NoError(t, err)
		assert.Len(t, records, 3) // 1 on detected + 2 on massive
	})

	t.Run("CountAllRecordsBySong", func(t *testing.T) {
		count, err := repo.CountAllRecordsBySong("user_song", created1.ID)
		assert.NoError(t, err)
		assert.Equal(t, int64(3), count)

		count2, err := repo.CountAllRecordsBySong("user_song", created2.ID)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), count2)
	})

	t.Run("GetBestRecordsBySong - nonexistent user", func(t *testing.T) {
		records, err := repo.GetBestRecordsBySong("nobody", created1.ID)
		assert.NoError(t, err)
		assert.Len(t, records, 0)
	})
}

func TestRecordRepository_PerChartQueries(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRecordRepository(db)
	songRepo := NewSongRepository(db)

	song := &model.Song{
		SongBase: model.SongBase{WikiID: "chart_q", Title: "Chart Query Song"},
		Charts: []model.Chart{
			{Difficulty: model.DifficultyMassive, Level: 15.0, Notes: 1000},
			{Difficulty: model.DifficultyDetected, Level: 5.0, Notes: 200},
		},
	}
	created, _ := songRepo.CreateSong(song)
	massiveChartID := created.Charts[0].ID
	detectedChartID := created.Charts[1].ID

	// Create multiple records on massive chart
	_, _ = repo.CreateRecord(&model.PlayRecord{
		PlayRecordBase: model.PlayRecordBase{ChartID: massiveChartID, Score: 1000000},
		Username:       "user_chart",
	}, false)
	_, _ = repo.CreateRecord(&model.PlayRecord{
		PlayRecordBase: model.PlayRecordBase{ChartID: massiveChartID, Score: 1005000},
		Username:       "user_chart",
	}, false)
	_, _ = repo.CreateRecord(&model.PlayRecord{
		PlayRecordBase: model.PlayRecordBase{ChartID: massiveChartID, Score: 900000},
		Username:       "user_chart",
	}, false)

	// One record on detected chart
	_, _ = repo.CreateRecord(&model.PlayRecord{
		PlayRecordBase: model.PlayRecordBase{ChartID: detectedChartID, Score: 1000000},
		Username:       "user_chart",
	}, false)

	t.Run("GetBestRecordByChart", func(t *testing.T) {
		record, err := repo.GetBestRecordByChart("user_chart", massiveChartID)
		assert.NoError(t, err)
		assert.NotNil(t, record)
		assert.Equal(t, 1005000, record.Score) // best score
		assert.NotNil(t, record.Chart)
		assert.NotNil(t, record.Chart.Song)
	})

	t.Run("GetBestRecordByChart - no record", func(t *testing.T) {
		record, err := repo.GetBestRecordByChart("nobody", massiveChartID)
		assert.NoError(t, err)
		assert.Nil(t, record)
	})

	t.Run("GetAllRecordsByChart", func(t *testing.T) {
		records, err := repo.GetAllRecordsByChart("user_chart", massiveChartID, 10, 0, "score", true)
		assert.NoError(t, err)
		assert.Len(t, records, 3)
		// Should be ordered by score desc
		assert.True(t, records[0].Score >= records[1].Score)
	})

	t.Run("GetAllRecordsByChart - pagination", func(t *testing.T) {
		records, err := repo.GetAllRecordsByChart("user_chart", massiveChartID, 2, 0, "score", true)
		assert.NoError(t, err)
		assert.Len(t, records, 2)

		records2, err := repo.GetAllRecordsByChart("user_chart", massiveChartID, 2, 1, "score", true)
		assert.NoError(t, err)
		assert.Len(t, records2, 1)
	})

	t.Run("CountAllRecordsByChart", func(t *testing.T) {
		count, err := repo.CountAllRecordsByChart("user_chart", massiveChartID)
		assert.NoError(t, err)
		assert.Equal(t, int64(3), count)

		count2, err := repo.CountAllRecordsByChart("user_chart", detectedChartID)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), count2)
	})
}
