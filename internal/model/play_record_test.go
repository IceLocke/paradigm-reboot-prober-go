package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func intPtrM(v int) *int { return &v }

func floatPtr(v float64) *float64 { return &v }

func TestRecordFilter_IsEmpty(t *testing.T) {
	t.Run("Empty filter", func(t *testing.T) {
		f := RecordFilter{}
		assert.True(t, f.IsEmpty())
	})

	t.Run("With MinLevel only", func(t *testing.T) {
		f := RecordFilter{MinLevel: floatPtr(10.0)}
		assert.False(t, f.IsEmpty())
	})

	t.Run("With MaxLevel only", func(t *testing.T) {
		f := RecordFilter{MaxLevel: floatPtr(14.0)}
		assert.False(t, f.IsEmpty())
	})

	t.Run("With Difficulties only", func(t *testing.T) {
		f := RecordFilter{Difficulties: []Difficulty{DifficultyMassive}}
		assert.False(t, f.IsEmpty())
	})

	t.Run("With all fields set", func(t *testing.T) {
		f := RecordFilter{
			MinLevel:     floatPtr(10.0),
			MaxLevel:     floatPtr(14.0),
			Difficulties: []Difficulty{DifficultyMassive, DifficultyReboot},
		}
		assert.False(t, f.IsEmpty())
	})

	t.Run("Empty Difficulties slice", func(t *testing.T) {
		f := RecordFilter{Difficulties: []Difficulty{}}
		assert.True(t, f.IsEmpty())
	})
}

func TestToPlayRecordInfo_WithChartAndSong(t *testing.T) {
	now := time.Now()
	score := 995000

	record := &PlayRecord{
		PlayRecordBase: PlayRecordBase{
			ChartID: 10,
			Score:   &score,
		},
		ID:         1,
		RecordTime: now,
		Username:   "testuser",
		Rating:     1350,
		Chart: &Chart{
			ID:           10,
			SongID:       5,
			Difficulty:   DifficultyMassive,
			Level:        13.5,
			FittingLevel: floatPtr(13.7),
			Notes:        800,
			Song: &Song{
				ID: 5,
				SongBase: SongBase{
					WikiID:  "felys",
					Title:   "Felys",
					Artist:  "Silentroom",
					Version: "1.0.0",
					B15:     true,
					Cover:   "felys.jpg",
				},
			},
		},
	}

	info := ToPlayRecordInfo(record)

	assert.Equal(t, 1, info.ID)
	assert.Equal(t, now, info.RecordTime)
	assert.Equal(t, 995000, info.Score)
	assert.Equal(t, 1350, info.Rating)
	assert.Equal(t, 10, info.Chart.ID)
	assert.Equal(t, DifficultyMassive, info.Chart.Difficulty)
	assert.Equal(t, 13.5, info.Chart.Level)
	assert.Equal(t, floatPtr(13.7), info.Chart.FittingLevel)
	assert.Equal(t, "felys", info.Chart.WikiID)
	assert.Equal(t, "Felys", info.Chart.Title)
	assert.Equal(t, "1.0.0", info.Chart.Version)
	assert.Equal(t, true, info.Chart.B15)
	assert.Equal(t, 5, info.Chart.SongID)
	assert.Equal(t, "felys.jpg", info.Chart.Cover)
}

func TestToPlayRecordInfo_WithChartOverride(t *testing.T) {
	now := time.Now()
	score := 990000

	record := &PlayRecord{
		PlayRecordBase: PlayRecordBase{
			ChartID: 20,
			Score:   &score,
		},
		ID:         2,
		RecordTime: now,
		Username:   "testuser",
		Rating:     1200,
		Chart: &Chart{
			ID:         20,
			SongID:     5,
			Difficulty: DifficultyReboot,
			Level:      14.0,
			SongBaseOverride: SongBaseOverride{
				OverrideTitle:   ptr("Felys Reboot"),
				OverrideVersion: ptr("2.0.0"),
			},
			Song: &Song{
				ID: 5,
				SongBase: SongBase{
					WikiID:  "felys",
					Title:   "Felys",
					Artist:  "Silentroom",
					Version: "1.0.0",
					B15:     true,
					Cover:   "felys.jpg",
				},
			},
		},
	}

	info := ToPlayRecordInfo(record)

	// Overridden fields
	assert.Equal(t, "Felys Reboot", info.Chart.Title)
	assert.Equal(t, "2.0.0", info.Chart.Version)
	// Non-overridden fields from song
	assert.Equal(t, "felys.jpg", info.Chart.Cover)
	assert.Equal(t, "felys", info.Chart.WikiID)
}

func TestToPlayRecordInfo_NilChart(t *testing.T) {
	now := time.Now()
	score := 900000

	record := &PlayRecord{
		PlayRecordBase: PlayRecordBase{
			ChartID: 10,
			Score:   &score,
		},
		ID:         3,
		RecordTime: now,
		Username:   "testuser",
		Rating:     1000,
		Chart:      nil,
	}

	info := ToPlayRecordInfo(record)

	assert.Equal(t, 3, info.ID)
	assert.Equal(t, 900000, info.Score)
	assert.Equal(t, 1000, info.Rating)
	// Chart fields should be zero values
	assert.Equal(t, 0, info.Chart.ID)
	assert.Equal(t, "", info.Chart.WikiID)
	assert.Equal(t, "", info.Chart.Title)
}

func TestToPlayRecordInfo_ChartWithoutSong(t *testing.T) {
	now := time.Now()
	score := 950000

	record := &PlayRecord{
		PlayRecordBase: PlayRecordBase{
			ChartID: 10,
			Score:   &score,
		},
		ID:         4,
		RecordTime: now,
		Username:   "testuser",
		Rating:     1100,
		Chart: &Chart{
			ID:         10,
			SongID:     5,
			Difficulty: DifficultyInvaded,
			Level:      11.0,
			Song:       nil,
		},
	}

	info := ToPlayRecordInfo(record)

	assert.Equal(t, 4, info.ID)
	assert.Equal(t, 950000, info.Score)
	// Chart-level fields are populated
	assert.Equal(t, 10, info.Chart.ID)
	assert.Equal(t, DifficultyInvaded, info.Chart.Difficulty)
	assert.Equal(t, 11.0, info.Chart.Level)
	// Song-level fields are zero values since Song is nil
	assert.Equal(t, "", info.Chart.WikiID)
	assert.Equal(t, "", info.Chart.Title)
	assert.Equal(t, 0, info.Chart.SongID)
}
