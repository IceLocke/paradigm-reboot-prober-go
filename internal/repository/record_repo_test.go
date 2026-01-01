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

	// Setup Song and Level
	song := &model.Song{
		SongBase: model.SongBase{WikiID: "rec_song", Title: "Record Song"},
		SongLevels: []model.SongLevel{
			{Difficulty: model.DifficultyMassive, Level: 15.0, Notes: 1000},
		},
	}
	createdSong, _ := songRepo.CreateSong(song)
	levelID := createdSong.SongLevels[0].SongLevelID

	t.Run("Create New Record", func(t *testing.T) {
		record := &model.PlayRecord{
			PlayRecordBase: model.PlayRecordBase{
				SongLevelID: levelID,
				Score:       1000000,
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
		err = db.Where("play_record_id = ?", savedRecord.PlayRecordID).First(&best).Error
		assert.NoError(t, err)
		assert.Equal(t, savedRecord.PlayRecordID, best.PlayRecordID)
	})

	t.Run("Update Best Record (Higher Score)", func(t *testing.T) {
		// Previous best was 1000000
		record := &model.PlayRecord{
			PlayRecordBase: model.PlayRecordBase{
				SongLevelID: levelID,
				Score:       1000001,
			},
			Username: "testuser",
		}
		savedRecord, err := repo.CreateRecord(record, false)
		assert.NoError(t, err)

		// Verify Best Record updated
		var best model.BestPlayRecord
		// We need to join to find the best record for this user/level, but here we can just check if the new record ID is in best_play_records table
		// Actually, let's query by play_record_id
		err = db.Where("play_record_id = ?", savedRecord.PlayRecordID).First(&best).Error
		assert.NoError(t, err)
	})

	t.Run("Do Not Update Best Record (Lower Score)", func(t *testing.T) {
		// Previous best was 1000001
		record := &model.PlayRecord{
			PlayRecordBase: model.PlayRecordBase{
				SongLevelID: levelID,
				Score:       900000,
			},
			Username: "testuser",
		}
		savedRecord, err := repo.CreateRecord(record, false)
		assert.NoError(t, err)

		// Verify Best Record NOT updated (should still point to previous high score)
		var best model.BestPlayRecord
		err = db.Where("play_record_id = ?", savedRecord.PlayRecordID).First(&best).Error
		assert.Error(t, err) // Should not find this record as best
		assert.Equal(t, "record not found", err.Error())
	})

	t.Run("Force Update Best Record (isReplaced=true)", func(t *testing.T) {
		// Previous best was 1000001. New score is lower but forced.
		record := &model.PlayRecord{
			PlayRecordBase: model.PlayRecordBase{
				SongLevelID: levelID,
				Score:       800000,
			},
			Username: "testuser",
		}
		savedRecord, err := repo.CreateRecord(record, true)
		assert.NoError(t, err)

		// Verify Best Record updated
		var best model.BestPlayRecord
		err = db.Where("play_record_id = ?", savedRecord.PlayRecordID).First(&best).Error
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
		SongLevels: []model.SongLevel{
			{Difficulty: model.DifficultyMassive, Level: 15.0},
		},
	}
	createdB15, _ := songRepo.CreateSong(songB15)

	// Create Non-B15 Song (Old)
	songOld := &model.Song{
		SongBase: model.SongBase{WikiID: "old_song", Title: "Old Song", B15: false},
		SongLevels: []model.SongLevel{
			{Difficulty: model.DifficultyMassive, Level: 15.0},
		},
	}
	createdOld, _ := songRepo.CreateSong(songOld)

	// Create Records
	repo.CreateRecord(&model.PlayRecord{
		PlayRecordBase: model.PlayRecordBase{SongLevelID: createdB15.SongLevels[0].SongLevelID, Score: 1000000},
		Username:       "user_b50",
	}, false)
	repo.CreateRecord(&model.PlayRecord{
		PlayRecordBase: model.PlayRecordBase{SongLevelID: createdOld.SongLevels[0].SongLevelID, Score: 1000000},
		Username:       "user_b50",
	}, false)

	t.Run("Get B50", func(t *testing.T) {
		b35, b15, err := repo.GetBest50Records("user_b50", 0)
		assert.NoError(t, err)
		assert.Len(t, b15, 1)
		assert.Len(t, b35, 1)
		assert.Equal(t, "B15 Song", b15[0].SongLevel.Song.Title)
		assert.Equal(t, "Old Song", b35[0].SongLevel.Song.Title)
	})
}
