package fitting

import (
	"math"
	"testing"

	"paradigm-reboot-prober-go/pkg/rating"

	"github.com/stretchr/testify/assert"
)

// TestInverseLevel_RoundTrip confirms that for a wide grid of (level, score)
// pairs, InverseLevel recovers the original level given the float rating that
// SingleRating would have produced for that level.
func TestInverseLevel_RoundTrip(t *testing.T) {
	levels := []float64{1.0, 5.5, 10.0, 13.2, 14.8, 15.5, 16.4, 17.3, 18.0}
	scores := []int{
		// Branch 3: < 1_000_000
		500000, 700000, 800000, 900000, 930000, 950000, 970000, 980000, 990000, 999999,
		// Branch 2: 1_000_000 .. 1_008_999
		1000000, 1002500, 1005000, 1008999,
		// Branch 1: >= 1_009_000
		1009000, 1009500, 1010000,
	}

	for _, L := range levels {
		for _, s := range scores {
			ratingInt := rating.SingleRating(L, s)
			if ratingInt == 0 {
				// Rating was clamped to 0 → cannot recover L uniquely.
				continue
			}
			target := float64(ratingInt) / 100.0
			got, ok := InverseLevel(s, target)
			if !ok {
				// A round-trip should succeed for in-range levels. Fail loudly.
				t.Errorf("round-trip failed for L=%.2f score=%d target=%.4f", L, s, target)
				continue
			}
			// Tolerance: SingleRating truncates to int (×100) and uses EPS=2e-5,
			// so we allow ~0.05 level error in branch 3 (steepest), ~0.01 elsewhere.
			assert.InDelta(t, L, got, 0.05,
				"round-trip L=%.2f score=%d target=%.4f → got=%.4f", L, s, target, got)
		}
	}
}

func TestInverseLevel_ZeroScore(t *testing.T) {
	_, ok := InverseLevel(0, 50.0)
	assert.False(t, ok, "score=0 should always yield ok=false")
}

func TestInverseLevel_NegativeScore(t *testing.T) {
	_, ok := InverseLevel(-1, 50.0)
	assert.False(t, ok, "negative score should yield ok=false")
}

func TestInverseLevel_ClampedRange(t *testing.T) {
	// A target that implies a level absurdly above the usable range
	// (e.g. target=500 at a mediocre score) should be rejected.
	_, ok := InverseLevel(995000, 500.0)
	assert.False(t, ok, "target way outside MaxInferredLevel should yield ok=false")

	// A target that implies a level far below MinInferredLevel should also
	// be rejected.
	_, ok = InverseLevel(995000, -50.0)
	assert.False(t, ok, "target way below MinInferredLevel should yield ok=false")
}

func TestInverseLevel_ScoreCappedAt1010000(t *testing.T) {
	// SingleRating caps score at 1010000, so inversion should treat 1020000
	// identically to 1010000.
	a, okA := InverseLevel(1010000, 120.0)
	b, okB := InverseLevel(1020000, 120.0)
	assert.True(t, okA)
	assert.True(t, okB)
	assert.InDelta(t, a, b, 1e-9)
}

// sanity: the inverted level should depend monotonically on target rating
// (holding score fixed). Higher target → higher required level.
func TestInverseLevel_MonotoneInTarget(t *testing.T) {
	score := 1005000
	prev := math.Inf(-1)
	for target := 100.0; target <= 170.0; target += 5.0 {
		got, ok := InverseLevel(score, target)
		if !ok {
			continue
		}
		assert.Greater(t, got, prev,
			"InverseLevel should be monotone increasing in target; got=%.4f prev=%.4f at target=%.2f",
			got, prev, target)
		prev = got
	}
}
