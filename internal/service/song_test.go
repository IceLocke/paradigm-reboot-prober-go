package service

import (
	"fmt"
	"paradigm-reboot-prober-go/internal/model"
	"paradigm-reboot-prober-go/internal/model/request"
	"paradigm-reboot-prober-go/internal/repository"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSongService(t *testing.T) {
	db := setupTestDB(t)
	songRepo := repository.NewSongRepository(db)
	songService := NewSongService(songRepo)

	t.Run("CreateSong", func(t *testing.T) {
		req := &request.CreateSongRequest{
			SongBase: model.SongBase{
				WikiID: "test_song",
				Title:  "Test Song",
				Artist: "Test Artist",
			},
			Charts: []model.ChartInput{
				{
					Difficulty:  model.DifficultyMassive,
					Level:       15.0,
					LevelDesign: "Designer",
					Notes:       1000,
				},
			},
		}
		charts, err := songService.CreateSong(req)
		assert.NoError(t, err)
		assert.Len(t, charts, 1)
		assert.Equal(t, "Test Song", charts[0].Title)
		assert.Equal(t, 15.0, charts[0].Level)
	})

	t.Run("GetSingleSong", func(t *testing.T) {
		song, err := songService.GetSingleSongByWikiID("test_song")
		assert.NoError(t, err)
		assert.NotNil(t, song)
		assert.Equal(t, "Test Song", song.Title)

		songByID, err := songService.GetSingleSong(song.ID, "prp")
		assert.NoError(t, err)
		assert.NotNil(t, songByID)
		assert.Equal(t, song.ID, songByID.ID)
	})

	t.Run("GetAllCharts", func(t *testing.T) {
		charts, err := songService.GetAllCharts()
		assert.NoError(t, err)
		assert.NotEmpty(t, charts)
		assert.Equal(t, "Test Song", charts[0].Title)
	})

	t.Run("UpdateSong", func(t *testing.T) {
		song, _ := songService.GetSingleSongByWikiID("test_song")
		req := &request.UpdateSongRequest{
			ID: song.ID,
			SongBase: model.SongBase{
				WikiID: "test_song",
				Title:  "Updated Song",
				Artist: "Test Artist",
			},
			Charts: []model.ChartInput{
				{
					Difficulty:  model.DifficultyMassive,
					Level:       15.5,
					LevelDesign: "New Designer",
					Notes:       1100,
				},
			},
		}
		charts, err := songService.UpdateSong(req)
		assert.NoError(t, err)
		assert.Len(t, charts, 1)
		assert.Equal(t, "Updated Song", charts[0].Title)
		assert.Equal(t, 15.5, charts[0].Level)
	})
}

func TestSongService_ResolveSongID(t *testing.T) {
	db := setupTestDB(t)
	songRepo := repository.NewSongRepository(db)
	songService := NewSongService(songRepo)

	// Create a song
	req := &request.CreateSongRequest{
		SongBase: model.SongBase{
			WikiID: "resolve_song",
			Title:  "Resolve Song",
			Artist: "Artist",
		},
		Charts: []model.ChartInput{
			{Difficulty: model.DifficultyMassive, Level: 15.0, Notes: 1000},
		},
	}
	_, err := songService.CreateSong(req)
	assert.NoError(t, err)

	song, _ := songService.GetSingleSongByWikiID("resolve_song")

	t.Run("Resolve by numeric ID", func(t *testing.T) {
		id, err := songService.ResolveSongID(fmt.Sprintf("%d", song.ID))
		assert.NoError(t, err)
		assert.Equal(t, song.ID, id)
	})

	t.Run("Resolve by wiki_id", func(t *testing.T) {
		id, err := songService.ResolveSongID("resolve_song")
		assert.NoError(t, err)
		assert.Equal(t, song.ID, id)
	})

	t.Run("Resolve numeric ID not found", func(t *testing.T) {
		_, err := songService.ResolveSongID("99999")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Resolve wiki_id not found", func(t *testing.T) {
		_, err := songService.ResolveSongID("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestSongService_ResolveChartID(t *testing.T) {
	db := setupTestDB(t)
	songRepo := repository.NewSongRepository(db)
	songService := NewSongService(songRepo)

	// Create a song with multiple charts
	req := &request.CreateSongRequest{
		SongBase: model.SongBase{
			WikiID: "resolve_chart",
			Title:  "Resolve Chart Song",
			Artist: "Artist",
		},
		Charts: []model.ChartInput{
			{Difficulty: model.DifficultyDetected, Level: 5.0, Notes: 200},
			{Difficulty: model.DifficultyMassive, Level: 15.0, Notes: 1000},
		},
	}
	charts, err := songService.CreateSong(req)
	assert.NoError(t, err)

	var massiveChartID int
	for _, c := range charts {
		if c.Difficulty == model.DifficultyMassive {
			massiveChartID = c.ID
		}
	}

	t.Run("Resolve by numeric chart ID", func(t *testing.T) {
		id, err := songService.ResolveChartID(fmt.Sprintf("%d", massiveChartID))
		assert.NoError(t, err)
		assert.Equal(t, massiveChartID, id)
	})

	t.Run("Resolve by wiki_id:difficulty", func(t *testing.T) {
		id, err := songService.ResolveChartID("resolve_chart:massive")
		assert.NoError(t, err)
		assert.Equal(t, massiveChartID, id)
	})

	t.Run("Resolve numeric chart ID not found", func(t *testing.T) {
		_, err := songService.ResolveChartID("99999")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Resolve wiki_id:difficulty not found", func(t *testing.T) {
		_, err := songService.ResolveChartID("resolve_chart:reboot")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Resolve invalid difficulty", func(t *testing.T) {
		_, err := songService.ResolveChartID("resolve_chart:easy")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid difficulty")
	})

	t.Run("Resolve missing colon", func(t *testing.T) {
		_, err := songService.ResolveChartID("nocolon")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid chart address")
	})

	t.Run("Resolve empty wiki_id", func(t *testing.T) {
		_, err := songService.ResolveChartID(":massive")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty wiki_id")
	})
}

func TestCompareVersion(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"2.0.0", "1.0.0", 1},
		{"1.0.0", "2.0.0", -1},
		// Semantic vs lexicographic: "2.11.0" > "2.2.0" numerically
		{"2.11.0", "2.2.0", 1},
		{"2.2.0", "2.11.0", -1},
		// Different lengths
		{"1.0", "1.0.0", 0},
		{"1.1", "1.0.1", 1},
		// Single segment
		{"3", "2", 1},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_vs_%s", tt.a, tt.b), func(t *testing.T) {
			got := compareVersion(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetAllCharts_DefaultSortOrder(t *testing.T) {
	db := setupTestDB(t)
	songRepo := repository.NewSongRepository(db)
	songService := NewSongService(songRepo)

	// Create songs in deliberately wrong order with mixed difficulties.
	// Expected final order: version DESC, then difficulty DESC (reboot > massive > invaded > detected).
	songs := []request.CreateSongRequest{
		{
			SongBase: model.SongBase{WikiID: "s1", Title: "Old Song", Artist: "A", Version: "1.0.0"},
			Charts: []model.ChartInput{
				{Difficulty: model.DifficultyMassive, Level: 10, Notes: 500},
				{Difficulty: model.DifficultyDetected, Level: 5, Notes: 200},
			},
		},
		{
			SongBase: model.SongBase{WikiID: "s2", Title: "Newest Song", Artist: "A", Version: "2.11.0"},
			Charts: []model.ChartInput{
				{Difficulty: model.DifficultyDetected, Level: 6, Notes: 300},
				{Difficulty: model.DifficultyReboot, Level: 15, Notes: 1200},
				{Difficulty: model.DifficultyInvaded, Level: 9, Notes: 600},
			},
		},
		{
			SongBase: model.SongBase{WikiID: "s3", Title: "Middle Song", Artist: "A", Version: "2.2.0"},
			Charts: []model.ChartInput{
				{Difficulty: model.DifficultyInvaded, Level: 8, Notes: 400},
				{Difficulty: model.DifficultyReboot, Level: 14, Notes: 1100},
			},
		},
	}

	for i := range songs {
		_, err := songService.CreateSong(&songs[i])
		require.NoError(t, err)
	}

	charts, err := songService.GetAllCharts()
	require.NoError(t, err)
	require.Len(t, charts, 7) // 2 + 3 + 2

	// Expected order:
	//   version 2.11.0: Reboot, Invaded, Detected
	//   version 2.2.0:  Reboot, Invaded
	//   version 1.0.0:  Massive, Detected
	expected := []struct {
		title      string
		version    string
		difficulty model.Difficulty
	}{
		{"Newest Song", "2.11.0", model.DifficultyReboot},
		{"Newest Song", "2.11.0", model.DifficultyInvaded},
		{"Newest Song", "2.11.0", model.DifficultyDetected},
		{"Middle Song", "2.2.0", model.DifficultyReboot},
		{"Middle Song", "2.2.0", model.DifficultyInvaded},
		{"Old Song", "1.0.0", model.DifficultyMassive},
		{"Old Song", "1.0.0", model.DifficultyDetected},
	}

	for i, e := range expected {
		assert.Equal(t, e.title, charts[i].Title, "index %d title", i)
		assert.Equal(t, e.version, charts[i].Version, "index %d version", i)
		assert.Equal(t, e.difficulty, charts[i].Difficulty, "index %d difficulty", i)
	}
}
