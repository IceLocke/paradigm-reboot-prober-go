package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
