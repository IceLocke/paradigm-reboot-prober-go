package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func ptr(s string) *string { return &s }

func TestWithOverride(t *testing.T) {
	original := SongBase{
		WikiID:      "wiki1",
		Title:       "Original Title",
		Artist:      "Original Artist",
		Genre:       "Pop",
		Cover:       "original.jpg",
		Illustrator: "Illustrator",
		Version:     "1.0.0",
		B15:         false,
		Album:       "Album",
		BPM:         "180",
		Length:      "2:30",
	}

	t.Run("All nil overrides returns original", func(t *testing.T) {
		result := original.WithOverride(SongBaseOverride{})
		assert.Equal(t, original, result)
	})

	t.Run("Override title only", func(t *testing.T) {
		result := original.WithOverride(SongBaseOverride{OverrideTitle: ptr("New Title")})
		assert.Equal(t, "New Title", result.Title)
		assert.Equal(t, "Original Artist", result.Artist)
		assert.Equal(t, "1.0.0", result.Version)
		assert.Equal(t, "original.jpg", result.Cover)
	})

	t.Run("Override version only", func(t *testing.T) {
		result := original.WithOverride(SongBaseOverride{OverrideVersion: ptr("2.0.0")})
		assert.Equal(t, "Original Title", result.Title)
		assert.Equal(t, "2.0.0", result.Version)
	})

	t.Run("Override all four fields", func(t *testing.T) {
		result := original.WithOverride(SongBaseOverride{
			OverrideTitle:   ptr("Alt Title"),
			OverrideArtist:  ptr("Alt Artist"),
			OverrideVersion: ptr("3.0.0"),
			OverrideCover:   ptr("alt.jpg"),
		})
		assert.Equal(t, "Alt Title", result.Title)
		assert.Equal(t, "Alt Artist", result.Artist)
		assert.Equal(t, "3.0.0", result.Version)
		assert.Equal(t, "alt.jpg", result.Cover)
		// Non-overridden fields remain unchanged
		assert.Equal(t, "Pop", result.Genre)
		assert.Equal(t, "Illustrator", result.Illustrator)
		assert.Equal(t, false, result.B15)
	})

	t.Run("Original is not mutated (value semantics)", func(t *testing.T) {
		_ = original.WithOverride(SongBaseOverride{OverrideTitle: ptr("Mutated?")})
		assert.Equal(t, "Original Title", original.Title)
	})
}

func TestValidDifficulty(t *testing.T) {
	t.Run("Valid difficulties", func(t *testing.T) {
		assert.True(t, ValidDifficulty("detected"))
		assert.True(t, ValidDifficulty("invaded"))
		assert.True(t, ValidDifficulty("massive"))
		assert.True(t, ValidDifficulty("reboot"))
	})

	t.Run("Invalid difficulties", func(t *testing.T) {
		assert.False(t, ValidDifficulty(""))
		assert.False(t, ValidDifficulty("easy"))
		assert.False(t, ValidDifficulty("MASSIVE"))
		assert.False(t, ValidDifficulty("Detected"))
		assert.False(t, ValidDifficulty("unknown"))
	})
}
