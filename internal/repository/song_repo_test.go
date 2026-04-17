package repository

import (
	"paradigm-reboot-prober-go/internal/model"
	"paradigm-reboot-prober-go/pkg/rating"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSongRepository_CreateSong(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSongRepository(db)

	t.Run("Create Song With Charts", func(t *testing.T) {
		song := &model.Song{
			SongBase: model.SongBase{
				WikiID: "song_1",
				Title:  "Test Song",
				Artist: "Test Artist",
			},
			Charts: []model.Chart{
				{
					Difficulty: model.DifficultyMassive,
					Level:      15.5,
					Notes:      1000,
				},
			},
		}

		created, err := repo.CreateSong(song)
		assert.NoError(t, err)
		assert.NotZero(t, created.ID)
		assert.Len(t, created.Charts, 1)
		assert.Equal(t, model.DifficultyMassive, created.Charts[0].Difficulty)
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
		Charts: []model.Chart{
			{Difficulty: model.DifficultyDetected, Level: 5.0, Notes: 100},
			{Difficulty: model.DifficultyInvaded, Level: 10.0, Notes: 500},
		},
	}
	_, err := repo.CreateSong(song)
	assert.NoError(t, err)

	t.Run("Update Song Metadata and Charts", func(t *testing.T) {
		updatedSong := &model.Song{
			SongBase: model.SongBase{
				WikiID: "song_update",
				Title:  "New Title", // Changed
			},
			Charts: []model.Chart{
				{Difficulty: model.DifficultyDetected, Level: 6.0, Notes: 120},  // Changed
				{Difficulty: model.DifficultyMassive, Level: 15.0, Notes: 1000}, // New
				// Invaded is missing -> should remain or be removed?
				// Looking at implementation:
				// It iterates over updatedSong.Charts.
				// If exists in DB, update.
				// If not exists in DB (but in update), create.
				// It does NOT delete charts missing from update.
			},
		}

		result, err := repo.UpdateSong(song.ID, updatedSong)
		assert.NoError(t, err)
		assert.Equal(t, "New Title", result.Title)

		// Verify charts
		// We need to fetch fresh from DB to be sure
		var freshSong model.Song
		db.Preload("Charts").First(&freshSong, song.ID)

		assert.Len(t, freshSong.Charts, 2) // Detected (updated), Massive (new); Invaded deleted (not in request)

		for _, l := range freshSong.Charts {
			if l.Difficulty == model.DifficultyDetected {
				assert.Equal(t, 6.0, l.Level)
			}
			if l.Difficulty == model.DifficultyMassive {
				assert.Equal(t, 15.0, l.Level)
			}
		}
	})
}

// TestSongRepository_UpdateSong_ReAddSoftDeletedDifficulty verifies that after a
// chart is removed (soft-deleted) via UpdateSong, a subsequent UpdateSong can
// add back a chart with the same difficulty without hitting a UNIQUE constraint
// violation. This relies on the partial unique index on (song_id, difficulty)
// scoped to `WHERE deleted_at IS NULL`.
func TestSongRepository_UpdateSong_ReAddSoftDeletedDifficulty(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSongRepository(db)

	song := &model.Song{
		SongBase: model.SongBase{WikiID: "soft_delete_readd", Title: "T"},
		Charts: []model.Chart{
			{Difficulty: model.DifficultyMassive, Level: 10.0, Notes: 500},
		},
	}
	_, err := repo.CreateSong(song)
	assert.NoError(t, err)

	// Step 1: remove the Massive chart via UpdateSong (soft delete).
	_, err = repo.UpdateSong(song.ID, &model.Song{
		SongBase: model.SongBase{WikiID: "soft_delete_readd", Title: "T"},
		Charts:   []model.Chart{},
	})
	assert.NoError(t, err)

	// The chart should still exist in the DB but be soft-deleted.
	var softDeletedCount int64
	db.Unscoped().Model(&model.Chart{}).
		Where("song_id = ? AND difficulty = ? AND deleted_at IS NOT NULL", song.ID, model.DifficultyMassive).
		Count(&softDeletedCount)
	assert.Equal(t, int64(1), softDeletedCount, "old chart should be soft-deleted, not hard-deleted")

	// Step 2: add the Massive difficulty back. This must not conflict with
	// the soft-deleted row on the (song_id, difficulty) unique index.
	_, err = repo.UpdateSong(song.ID, &model.Song{
		SongBase: model.SongBase{WikiID: "soft_delete_readd", Title: "T"},
		Charts: []model.Chart{
			{Difficulty: model.DifficultyMassive, Level: 12.5, Notes: 600},
		},
	})
	assert.NoError(t, err)

	// Verify the fresh (non-deleted) chart has the new values.
	var freshSong model.Song
	db.Preload("Charts").First(&freshSong, song.ID)
	assert.Len(t, freshSong.Charts, 1)
	assert.Equal(t, model.DifficultyMassive, freshSong.Charts[0].Difficulty)
	assert.Equal(t, 12.5, freshSong.Charts[0].Level)
}

func TestSongRepository_GetSong(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSongRepository(db)

	song := &model.Song{
		SongBase: model.SongBase{WikiID: "find_me", Title: "Find Me"},
	}
	_, err := repo.CreateSong(song)
	assert.NoError(t, err)

	t.Run("Get By ID", func(t *testing.T) {
		found, err := repo.GetSongByID(song.ID)
		assert.NoError(t, err)
		assert.Equal(t, "Find Me", found.Title)
	})

	t.Run("Get By WikiID", func(t *testing.T) {
		found, err := repo.GetSongByWikiID("find_me")
		assert.NoError(t, err)
		assert.Equal(t, "Find Me", found.Title)
	})
}

func TestSongRepository_GetChartByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSongRepository(db)

	song := &model.Song{
		SongBase: model.SongBase{WikiID: "chart_test", Title: "Chart Test"},
		Charts: []model.Chart{
			{Difficulty: model.DifficultyMassive, Level: 15.0, Notes: 1000},
		},
	}
	created, err := repo.CreateSong(song)
	assert.NoError(t, err)
	chartID := created.Charts[0].ID

	t.Run("Found", func(t *testing.T) {
		chart, err := repo.GetChartByID(chartID)
		assert.NoError(t, err)
		assert.NotNil(t, chart)
		assert.Equal(t, model.DifficultyMassive, chart.Difficulty)
		assert.NotNil(t, chart.Song)
		assert.Equal(t, "Chart Test", chart.Song.Title)
	})

	t.Run("Not Found", func(t *testing.T) {
		chart, err := repo.GetChartByID(99999)
		assert.NoError(t, err)
		assert.Nil(t, chart)
	})
}

func TestSongRepository_GetChartByWikiIDAndDifficulty(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSongRepository(db)

	song := &model.Song{
		SongBase: model.SongBase{WikiID: "felys", Title: "Felys"},
		Charts: []model.Chart{
			{Difficulty: model.DifficultyDetected, Level: 5.0, Notes: 200},
			{Difficulty: model.DifficultyMassive, Level: 15.0, Notes: 1000},
		},
	}
	_, err := repo.CreateSong(song)
	assert.NoError(t, err)

	t.Run("Found", func(t *testing.T) {
		chart, err := repo.GetChartByWikiIDAndDifficulty("felys", model.DifficultyMassive)
		assert.NoError(t, err)
		assert.NotNil(t, chart)
		assert.Equal(t, model.DifficultyMassive, chart.Difficulty)
		assert.Equal(t, 15.0, chart.Level)
		assert.NotNil(t, chart.Song)
		assert.Equal(t, "Felys", chart.Song.Title)
	})

	t.Run("Wrong Difficulty", func(t *testing.T) {
		chart, err := repo.GetChartByWikiIDAndDifficulty("felys", model.DifficultyReboot)
		assert.NoError(t, err)
		assert.Nil(t, chart)
	})

	t.Run("Wrong WikiID", func(t *testing.T) {
		chart, err := repo.GetChartByWikiIDAndDifficulty("nonexistent", model.DifficultyMassive)
		assert.NoError(t, err)
		assert.Nil(t, chart)
	})
}

func TestSongRepository_UpdateSong_RecalculatesRatings(t *testing.T) {
	db := setupTestDB(t)
	songRepo := NewSongRepository(db)
	recordRepo := NewRecordRepository(db)

	// Create song with chart at level 15.0
	song := &model.Song{
		SongBase: model.SongBase{WikiID: "lvl_change", Title: "Level Change Song"},
		Charts: []model.Chart{
			{Difficulty: model.DifficultyMassive, Level: 15.0, Notes: 1000},
		},
	}
	created, err := songRepo.CreateSong(song)
	assert.NoError(t, err)
	chartID := created.Charts[0].ID

	// Create play records
	_, err = recordRepo.CreateRecord(&model.PlayRecord{
		PlayRecordBase: model.PlayRecordBase{ChartID: chartID, Score: intPtr(1000000)},
		Username:       "user_lvlchg",
	}, false)
	assert.NoError(t, err)
	_, err = recordRepo.CreateRecord(&model.PlayRecord{
		PlayRecordBase: model.PlayRecordBase{ChartID: chartID, Score: intPtr(1005000)},
		Username:       "user_lvlchg",
	}, false)
	assert.NoError(t, err)

	t.Run("Level change triggers rating recalculation", func(t *testing.T) {
		newLevel := 16.0
		updatedSong := &model.Song{
			SongBase: model.SongBase{WikiID: "lvl_change", Title: "Level Change Song"},
			Charts: []model.Chart{
				{Difficulty: model.DifficultyMassive, Level: newLevel, Notes: 1000},
			},
		}
		_, err := songRepo.UpdateSong(created.ID, updatedSong)
		assert.NoError(t, err)

		// Verify ratings updated to new level
		var records []model.PlayRecord
		db.Where("chart_id = ?", chartID).Find(&records)
		assert.Len(t, records, 2)
		for _, r := range records {
			expected := rating.SingleRating(newLevel, *r.Score)
			assert.Equal(t, expected, r.Rating, "score=%d should have rating=%d", *r.Score, expected)
		}
	})

	t.Run("No level change does not recalculate", func(t *testing.T) {
		// Capture current ratings
		var before []model.PlayRecord
		db.Where("chart_id = ?", chartID).Order("id").Find(&before)

		// Update with same level but different notes
		updatedSong := &model.Song{
			SongBase: model.SongBase{WikiID: "lvl_change", Title: "Level Change Song V2"},
			Charts: []model.Chart{
				{Difficulty: model.DifficultyMassive, Level: 16.0, Notes: 1200},
			},
		}
		_, err := songRepo.UpdateSong(created.ID, updatedSong)
		assert.NoError(t, err)

		var after []model.PlayRecord
		db.Where("chart_id = ?", chartID).Order("id").Find(&after)
		assert.Len(t, after, len(before))
		for i := range before {
			assert.Equal(t, before[i].Rating, after[i].Rating, "rating should not change when level is unchanged")
		}
	})
}
