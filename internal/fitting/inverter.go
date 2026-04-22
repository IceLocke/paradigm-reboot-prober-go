package fitting

import (
	"math"
)

// These constants MUST stay identical to pkg/rating/rating.go. They are
// duplicated here deliberately: we invert the SingleRating formula analytically
// and want a single local source of truth for the inversion math. If
// pkg/rating/rating.go changes, update both.
var (
	invBounds  = []int{900000, 930000, 950000, 970000, 980000, 990000}
	invRewards = []float64{3, 1, 1, 1, 1, 1}
)

// MinInferredLevel and MaxInferredLevel bound the output of the inverter.
// Anything outside this range is treated as "inversion failed" (the solver
// does not clamp silently — callers will receive ok=false).
const (
	MinInferredLevel = 0.1
	MaxInferredLevel = 20.0
)

// InverseLevel returns the chart level L such that
//
//	rating.SingleRating(L, score) ≈ targetRating * 100
//
// where targetRating is a *float* rating value (i.e. the human-readable rating,
// not the int×100 form persisted in play_records.rating). The inversion is
// closed-form within each branch of the piecewise SingleRating formula, so
// there is no iterative root-finding involved.
//
// Returns ok=false when:
//   - score = 0 (rating is identically 0 regardless of L, so L is
//     unidentifiable),
//   - the analytically-inverted L falls outside [MinInferredLevel,
//     MaxInferredLevel] (e.g. the target is trivially low, making the chart
//     look absurdly easy/hard),
//   - numerical issues yield a NaN/Inf.
//
// Callers are expected to discard !ok samples rather than clamp them.
func InverseLevel(score int, targetRating float64) (float64, bool) {
	// Mirror SingleRating: cap score at the game's theoretical max.
	if score > 1010000 {
		score = 1010000
	}
	if score <= 0 {
		return 0, false
	}

	var level float64
	switch {
	case score >= 1009000:
		// rating = level*10 + 7 + 3 * base^1.35, base = (score-1009000)/1000
		base := float64(score-1009000) / 1000.0
		level = (targetRating - 7 - 3*math.Pow(base, 1.35)) / 10.0
	case score >= 1000000:
		// rating = 10 * (level + 2 * (score-1000000)/30000)
		term := float64(score-1000000) / 30000.0
		level = targetRating/10.0 - 2*term
	default:
		// rating = bonuses + 10 * (level * (score/1e6)^1.5 - 0.9)
		var bonus float64
		for i, bound := range invBounds {
			if score >= bound {
				bonus += invRewards[i]
			}
		}
		base := float64(score) / 1000000.0
		coef := math.Pow(base, 1.5) // guaranteed > 0 because score > 0
		if coef == 0 || math.IsNaN(coef) || math.IsInf(coef, 0) {
			return 0, false
		}
		level = (targetRating - bonus + 9) / (10.0 * coef)
	}

	if math.IsNaN(level) || math.IsInf(level, 0) {
		return 0, false
	}
	if level < MinInferredLevel || level > MaxInferredLevel {
		return 0, false
	}
	return level, true
}
