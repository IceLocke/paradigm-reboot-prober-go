package repository

import (
	"paradigm-reboot-prober-go/internal/model"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSongRepository_CreateSong(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSongRepository(db)

	t.Run("Create Song With Levels", func(t *testing.T) {
		song := &model.Song{
			SongBase: model.SongBase{
				WikiID: "song_1",
				Title:  "Test Song",
				Artist: "Test Artist",
			},
			SongLevels: []model.SongLevel{
				{
					Difficulty: model.DifficultyMassive,
					Level:      15.5,
					Notes:      1000,
				},
			},
		}

		created, err := repo.CreateSong(song)
		assert.NoError(t, err)
		assert.NotZero(t, created.SongID)
		assert.Len(t, created.SongLevels, 1)
		assert.Equal(t, model.DifficultyMassive, created.SongLevels[0].Difficulty)
	})
}

func TestSongRepository_UpdateSong(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSongRepository(db)

	// Setup
	song := &model.Song{
		SongBase: model.SongBase{
			WikiID: "song_update",
			Title:  "Original Title",
		},
		SongLevels: []model.SongLevel{
			{Difficulty: model.DifficultyDetected, Level: 5.0, Notes: 100},
			{Difficulty: model.DifficultyInvaded, Level: 10.0, Notes: 500},
		},
	}
	repo.CreateSong(song)

	t.Run("Update Song Metadata and Levels", func(t *testing.T) {
		updatedSong := &model.Song{
			SongBase: model.SongBase{
				WikiID: "song_update",
				Title:  "New Title", // Changed
			},
			SongLevels: []model.SongLevel{
				{Difficulty: model.DifficultyDetected, Level: 6.0, Notes: 120},  // Changed
				{Difficulty: model.DifficultyMassive, Level: 15.0, Notes: 1000}, // New
				// Invaded is missing -> should remain or be removed?
				// Looking at implementation:
				// It iterates over updatedSong.SongLevels.
				// If exists in DB, update.
				// If not exists in DB (but in update), create.
				// It does NOT delete levels missing from update.
			},
		}

		result, err := repo.UpdateSong(song.SongID, updatedSong)
		assert.NoError(t, err)
		assert.Equal(t, "New Title", result.Title)

		// Verify levels
		// We need to fetch fresh from DB to be sure
		var freshSong model.Song
		db.Preload("SongLevels").First(&freshSong, song.SongID)

		assert.Len(t, freshSong.SongLevels, 3) // Detected (updated), Invaded (untouched), Massive (new)

		for _, l := range freshSong.SongLevels {
			if l.Difficulty == model.DifficultyDetected {
				assert.Equal(t, 6.0, l.Level)
			}
			if l.Difficulty == model.DifficultyInvaded {
				assert.Equal(t, 10.0, l.Level)
			}
			if l.Difficulty == model.DifficultyMassive {
				assert.Equal(t, 15.0, l.Level)
			}
		}
	})
}

func TestSongRepository_GetSong(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSongRepository(db)

	song := &model.Song{
		SongBase: model.SongBase{WikiID: "find_me", Title: "Find Me"},
	}
	repo.CreateSong(song)

	t.Run("Get By ID", func(t *testing.T) {
		found, err := repo.GetSongByID(song.SongID)
		assert.NoError(t, err)
		assert.Equal(t, "Find Me", found.Title)
	})

	t.Run("Get By WikiID", func(t *testing.T) {
		found, err := repo.GetSongByWikiID("find_me")
		assert.NoError(t, err)
		assert.Equal(t, "Find Me", found.Title)
	})
}
