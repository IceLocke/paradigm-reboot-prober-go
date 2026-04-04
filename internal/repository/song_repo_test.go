package repository

import (
	"paradigm-reboot-prober-go/internal/model"
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
		assert.NotZero(t, created.SongID)
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

		result, err := repo.UpdateSong(song.SongID, updatedSong)
		assert.NoError(t, err)
		assert.Equal(t, "New Title", result.Title)

		// Verify charts
		// We need to fetch fresh from DB to be sure
		var freshSong model.Song
		db.Preload("Charts").First(&freshSong, song.SongID)

		assert.Len(t, freshSong.Charts, 3) // Detected (updated), Invaded (untouched), Massive (new)

		for _, l := range freshSong.Charts {
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
	_, err := repo.CreateSong(song)
	assert.NoError(t, err)

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
	chartID := created.Charts[0].ChartID

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
