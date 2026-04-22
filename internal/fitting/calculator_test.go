package fitting

import (
	"math/rand"
	"testing"

	"paradigm-reboot-prober-go/pkg/rating"

	"github.com/stretchr/testify/assert"
)

func defaultParams() Params {
	return Params{
		MinEffectiveSamples: 3.0,
		ProximitySigma:      20.0,
		VolumeFullAt:        50,
		PriorStrength:       5.0,
		MaxDeviation:        1.5,
		MinScore:            500000,
		TukeyK:              4.685,
	}
}

// simulateScore: given a player of ability `skill` (float rating) playing a
// chart of *true* level L, return a score whose rating.SingleRating(L, score)
// is approximately equal to `skill`. Implemented via bisection so the synthetic
// samples match the actual piecewise rating formula across all three branches.
func simulateScore(trueLevel float64, skill float64) int {
	lo, hi := 0, 1_010_000
	targetInt := int(skill*100 + 0.5)
	for lo < hi {
		mid := (lo + hi) / 2
		if rating.SingleRating(trueLevel, mid) < targetInt {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}

// When every sample agrees with a chart's true level (no noise), the computed
// fitting level should match the true level closely (modulo shrinkage toward
// the `officialLevel` supplied to ComputeFitting).
func TestComputeFitting_NoiseFree(t *testing.T) {
	params := defaultParams()
	params.PriorStrength = 0 // disable shrinkage to compare directly
	trueLevel := 14.0
	officialLevel := 14.0

	samples := make([]Sample, 0, 40)
	for i := 0; i < 40; i++ {
		// skill uniform in [130, 150] rating units
		skill := 130.0 + float64(i)*0.5
		score := simulateScore(trueLevel, skill)
		samples = append(samples, Sample{
			Username:      "p" + string(rune(i)),
			Score:         score,
			PlayerSkill:   skill,
			PlayerRecords: 50,
		})
	}
	res := ComputeFitting(officialLevel, samples, params)
	if !assert.NotNil(t, res.FittingLevel) {
		return
	}
	assert.InDelta(t, trueLevel, *res.FittingLevel, 0.05)
	assert.Greater(t, res.EffectiveSampleSize, params.MinEffectiveSamples)
}

// When the true level is different from the official level, the fitting
// calculation should move toward the true level but still be pulled partly
// toward `officialLevel` by the Bayesian prior.
func TestComputeFitting_DrawsTowardTrueLevel(t *testing.T) {
	params := defaultParams()
	trueLevel := 13.5 // chart is easier than its official level suggests
	officialLevel := 14.5

	samples := make([]Sample, 0, 100)
	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 100; i++ {
		skill := 130.0 + rng.Float64()*20.0 // skill ~ [130, 150]
		// add small random noise to score (±0.03 level worth)
		noise := (rng.Float64() - 0.5) * 1500 // ~ ±750 score points
		score := simulateScore(trueLevel, skill) + int(noise)
		samples = append(samples, Sample{
			Username:      "p",
			Score:         score,
			PlayerSkill:   skill,
			PlayerRecords: 50,
		})
	}
	res := ComputeFitting(officialLevel, samples, params)
	if !assert.NotNil(t, res.FittingLevel) {
		return
	}
	// Should land somewhere between trueLevel and officialLevel, closer to true.
	got := *res.FittingLevel
	assert.Less(t, got, officialLevel, "fitting should drop below official when chart is easy")
	assert.Greater(t, got, trueLevel-0.2, "fitting should not fall below true level by much")
}

// When a handful of outliers are injected (e.g. cheat scores or misreported
// charts), the Tukey biweight step should neutralize them.
func TestComputeFitting_OutlierRobust(t *testing.T) {
	params := defaultParams()
	trueLevel := 14.0
	officialLevel := 14.0

	samples := make([]Sample, 0, 60)
	for i := 0; i < 50; i++ {
		skill := 135.0 + float64(i)*0.3
		score := simulateScore(trueLevel, skill)
		samples = append(samples, Sample{
			Username:      "p",
			Score:         score,
			PlayerSkill:   skill,
			PlayerRecords: 50,
		})
	}
	// Inject 10 outliers: scores much higher than expected (makes chart look
	// absurdly easy) — use max-score plays by low-skill players.
	for i := 0; i < 10; i++ {
		samples = append(samples, Sample{
			Username:      "outlier",
			Score:         1_010_000, // perfect clear, but by weak players
			PlayerSkill:   80.0,      // low skill
			PlayerRecords: 50,
		})
	}
	res := ComputeFitting(officialLevel, samples, params)
	if !assert.NotNil(t, res.FittingLevel) {
		return
	}
	// Despite 10/60 outliers, the fitting should stay close to the true level.
	assert.InDelta(t, trueLevel, *res.FittingLevel, 0.3)
}

// When too few samples are provided, FittingLevel must be nil (abstention)
// even though the stats fields remain populated for transparency.
func TestComputeFitting_MinSamples(t *testing.T) {
	params := defaultParams()
	params.MinEffectiveSamples = 10.0
	officialLevel := 14.0
	// Only 2 samples — well below MinEffectiveSamples.
	samples := []Sample{
		{Username: "a", Score: 1_005_000, PlayerSkill: 140.0, PlayerRecords: 50},
		{Username: "b", Score: 1_006_000, PlayerSkill: 142.0, PlayerRecords: 50},
	}
	res := ComputeFitting(officialLevel, samples, params)
	assert.Nil(t, res.FittingLevel, "sparse samples should yield nil FittingLevel")
	assert.Equal(t, 2, res.SampleCount)
}

// MaxDeviation hard cap should engage when the inferred level differs wildly
// from the officialLevel.
func TestComputeFitting_DeviationCap(t *testing.T) {
	params := defaultParams()
	params.MaxDeviation = 0.5  // tight cap
	params.PriorStrength = 0.0 // disable shrinkage so the mean survives
	trueLevel := 10.0          // samples imply L ≈ 10
	officialLevel := 14.0      // but chart is officially 14

	samples := make([]Sample, 0, 60)
	for i := 0; i < 60; i++ {
		skill := 130.0 + float64(i)*0.2
		score := simulateScore(trueLevel, skill)
		samples = append(samples, Sample{
			Username: "p", Score: score, PlayerSkill: skill, PlayerRecords: 50,
		})
	}
	res := ComputeFitting(officialLevel, samples, params)
	if !assert.NotNil(t, res.FittingLevel) {
		return
	}
	// With no shrinkage, the cap alone limits |fit − official| ≤ 0.5.
	diff := *res.FittingLevel - officialLevel
	if diff < 0 {
		diff = -diff
	}
	assert.LessOrEqual(t, diff, 0.5+1e-9)
}

// Round-trip with rating.SingleRating: synthesize samples using the real
// SingleRating formula, then verify that ComputeFitting recovers the true
// level.
func TestComputeFitting_RealRatingRoundTrip(t *testing.T) {
	params := defaultParams()
	params.PriorStrength = 0
	trueLevel := 15.0

	samples := make([]Sample, 0, 80)
	for i := 0; i < 80; i++ {
		// Pick scores across the three branches.
		score := 950_000 + i*750 // 950k..1_010k
		if score > 1_010_000 {
			score = 1_010_000
		}
		// Skill is exactly what SingleRating(trueLevel, score) would say.
		r := rating.SingleRating(trueLevel, score)
		skill := float64(r) / 100.0
		samples = append(samples, Sample{
			Username:      "p",
			Score:         score,
			PlayerSkill:   skill,
			PlayerRecords: 50,
		})
	}
	res := ComputeFitting(trueLevel, samples, params)
	if !assert.NotNil(t, res.FittingLevel) {
		return
	}
	assert.InDelta(t, trueLevel, *res.FittingLevel, 0.05)
}

// weightedMedian exercise.
func TestWeightedMedian_SimpleAndWeighted(t *testing.T) {
	assert.InDelta(t, 3.0, weightedMedian([]float64{1, 2, 3, 4, 5}, []float64{1, 1, 1, 1, 1}), 1e-9)
	// Weight-loaded median: 2 carries 10× the weight, so median should be 2.
	assert.InDelta(t, 2.0, weightedMedian([]float64{1, 2, 3, 4, 5}, []float64{1, 10, 1, 1, 1}), 1e-9)
}
