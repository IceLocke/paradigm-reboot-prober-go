package rating

import (
	"math"
)

var (
	bounds  = []int{900000, 930000, 950000, 970000, 980000, 990000}
	rewards = []float64{3, 1, 1, 1, 1, 1}
)

const EPS = 0.00002

// SingleRating calculates the rating of a single chart.
// level: the float level of the chart. e.g. 16.4
// score: the score of a play record. e.g. 1008900
// return: the (avg) rating.
func SingleRating(level float64, score int) int {
	// Reference: https://www.bilibili.com/read/cv29433852

	// Cap score at 1010000
	if score > 1010000 {
		score = 1010000
	}

	var rating float64

	if score >= 1009000 {
		// rating = level * 10 + 7 + 3 * (((score - 1009000) / 1000) ** 1.35)
		base := float64(score-1009000) / 1000.0
		rating = level*10 + 7 + 3*math.Pow(base, 1.35)
	} else if score >= 1000000 {
		// rating = 10 * (level + 2 * (score - 1000000) / 30000)
		term := float64(score-1000000) / 30000.0
		rating = 10 * (level + 2*term)
	} else {
		// for bound, reward in zip(bounds, rewards):
		//     rating += reward if score >= bound else 0
		for i, bound := range bounds {
			if score >= bound {
				rating += rewards[i]
			}
		}
		// rating += 10 * (level * ((score / 1000000) ** 1.5) - 0.9)
		base := float64(score) / 1000000.0
		rating += 10 * (level*math.Pow(base, 1.5) - 0.9)
	}

	if rating < 0 {
		rating = 0
	}

	// int_rating: int = int(rating * 100 + EPS)
	return int(rating*100 + EPS)
}
