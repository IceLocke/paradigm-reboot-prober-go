package rating

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSingleRating(t *testing.T) {
	tests := []struct {
		name     string
		level    float64
		score    int
		expected int
	}{
		{
			name:     "Max Score",
			level:    10.0,
			score:    1010000,
			expected: 11000,
		},
		{
			name:     "Over Max Score (Capped)",
			level:    10.0,
			score:    1020000,
			expected: 11000,
		},
		{
			name:     "High Score (>= 1009000)",
			level:    10.0,
			score:    1009500,
			expected: 10817, // Calculated: 108.17687... * 100
		},
		{
			name:     "Mid Score (>= 1000000)",
			level:    10.0,
			score:    1005000,
			expected: 10333, // Calculated: 103.333... * 100
		},
		{
			name:     "Low Score (< 1000000)",
			level:    10.0,
			score:    995000,
			expected: 9825, // Calculated: 98.25... * 100
		},
		{
			name:     "Very Low Score",
			level:    10.0,
			score:    500000,
			expected: 2635, // Calculated: 26.355... * 100
		},
		{
			name:     "Zero Score",
			level:    10.0,
			score:    0,
			expected: 0,
		},
		{
			name:  "Level 16.4 Score 1008900", // Example from docstring
			level: 16.4,
			score: 1008900,
			// 1000000 <= 1008900 < 1009000
			// rating = 10 * (16.4 + 2 * (8900 / 30000))
			// rating = 10 * (16.4 + 2 * 0.296666...)
			// rating = 10 * (16.4 + 0.593333...)
			// rating = 10 * 16.993333...
			// rating = 169.93333...
			// int_rating = 16993
			expected: 16993,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SingleRating(tt.level, tt.score)
			assert.Equal(t, tt.expected, result)
		})
	}
}
