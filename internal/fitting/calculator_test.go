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
//
// Realistic-range sanity check: for a typical target chart (lv15–17.4) and a
// skill in [10·L, 10·L+10], the returned score lands in [1_000_000, 1_010_000].
// That is exactly where the vast majority of real plays sit, so test callers
// should prefer skill ranges near `10*trueLevel..10*trueLevel+10` to stay on
// the common path.
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
	trueLevel := 16.5        // lv15-17.4 is where most charts live
	officialLevel := 16.5

	samples := make([]Sample, 0, 40)
	for i := 0; i < 40; i++ {
		// skill uniform in [165, 175] — matches trueLevel ± 0..10 rating units.
		skill := 165.0 + float64(i)*0.25
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
	trueLevel := 15.5 // chart is easier than its official level suggests
	officialLevel := 16.5

	samples := make([]Sample, 0, 100)
	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 100; i++ {
		skill := 155.0 + rng.Float64()*10.0 // skill ~ [155, 165], matches trueLevel
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
	trueLevel := 16.0
	officialLevel := 16.0

	samples := make([]Sample, 0, 60)
	for i := 0; i < 50; i++ {
		skill := 160.0 + float64(i)*0.15 // [160, 167.5], lands in [1_000_000, 1_010_000]
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
			PlayerSkill:   100.0,     // skill far below chart level
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
	officialLevel := 16.5
	// Only 2 samples — well below MinEffectiveSamples.
	samples := []Sample{
		{Username: "a", Score: 1_005_000, PlayerSkill: 167.0, PlayerRecords: 50},
		{Username: "b", Score: 1_006_000, PlayerSkill: 168.0, PlayerRecords: 50},
	}
	res := ComputeFitting(officialLevel, samples, params)
	assert.Nil(t, res.FittingLevel, "sparse samples should yield nil FittingLevel")
	assert.Equal(t, 2, res.SampleCount)
}

// MaxDeviation hard cap should engage when the inferred level differs wildly
// from the officialLevel (below-official direction).
func TestComputeFitting_DeviationCap(t *testing.T) {
	params := defaultParams()
	params.MaxDeviation = 0.5  // tight cap
	params.PriorStrength = 0.0 // disable shrinkage so the mean survives
	trueLevel := 13.0          // samples imply L ≈ 13
	officialLevel := 17.0      // but chart is officially 17 — huge misclass

	samples := make([]Sample, 0, 60)
	for i := 0; i < 60; i++ {
		skill := 130.0 + float64(i)*0.1 // matches trueLevel, lands in 1_000_000..1_010_000
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

// Companion to TestComputeFitting_DeviationCap: when samples imply a level
// *above* the official one, the positive-direction cap must engage. Exercises
// the `diff > params.MaxDeviation` branch in ComputeFitting.
func TestComputeFitting_DeviationCap_AboveOfficial(t *testing.T) {
	params := defaultParams()
	params.MaxDeviation = 0.4
	params.PriorStrength = 0.0
	trueLevel := 17.0     // samples imply L ≈ 17 — chart is harder than labelled
	officialLevel := 15.0 // but officially only 15

	samples := make([]Sample, 0, 60)
	for i := 0; i < 60; i++ {
		skill := 170.0 + float64(i)*0.1 // [170, 176], matches trueLevel ≈ 17
		score := simulateScore(trueLevel, skill)
		samples = append(samples, Sample{
			Username: "p", Score: score, PlayerSkill: skill, PlayerRecords: 50,
		})
	}
	res := ComputeFitting(officialLevel, samples, params)
	if !assert.NotNil(t, res.FittingLevel) {
		return
	}
	// Cap should pull fit back toward officialLevel from above.
	assert.Greater(t, *res.FittingLevel, officialLevel)
	assert.LessOrEqual(t, *res.FittingLevel, officialLevel+params.MaxDeviation+1e-9)
}

// Round-trip with rating.SingleRating: synthesize samples using the real
// SingleRating formula across all three score branches, then verify that
// ComputeFitting recovers the true level at the high end (lv16.5) of the
// realistic chart range.
func TestComputeFitting_RealRatingRoundTrip(t *testing.T) {
	params := defaultParams()
	params.PriorStrength = 0
	trueLevel := 16.5

	samples := make([]Sample, 0, 80)
	for i := 0; i < 80; i++ {
		// Pick scores across the three branches.
		score := 950_000 + i*750 // 950k..1_010k, crosses both 1_000_000 and 1_009_000 seams
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

// TestComputeFitting_HighScoreCluster models the realistic score distribution:
// the vast majority of players sit in [1_000_000, 1_010_000] with skills
// concentrated in a narrow band matching the chart's true difficulty. The
// chart itself is a typical lv16-17 target — that's where 大部分玩家 spend
// their time, and where fitting accuracy matters most.
func TestComputeFitting_HighScoreCluster(t *testing.T) {
	params := defaultParams()
	params.PriorStrength = 0
	params.MinPlayerRecords = 0

	trueLevel := 16.5
	officialLevel := 16.5

	// 80 players with skill ~ 165..175: matches chart level so simulateScore
	// returns values tightly in [1_000_000, 1_010_000].
	rng := rand.New(rand.NewSource(7))
	samples := make([]Sample, 0, 80)
	for i := 0; i < 80; i++ {
		skill := 165.0 + rng.Float64()*10.0
		score := simulateScore(trueLevel, skill)
		assert.GreaterOrEqual(t, score, 1_000_000, "score must be in realistic high-score range")
		assert.LessOrEqual(t, score, 1_010_000)
		samples = append(samples, Sample{
			Username:      "p",
			Score:         score,
			PlayerSkill:   skill,
			PlayerRecords: 30,
		})
	}

	res := ComputeFitting(officialLevel, samples, params)
	if !assert.NotNil(t, res.FittingLevel, "dense realistic samples should always publish") {
		return
	}
	assert.InDelta(t, trueLevel, *res.FittingLevel, 0.1,
		"high-score cluster should recover trueLevel tightly; got %.4f", *res.FittingLevel)
	assert.Greater(t, res.EffectiveSampleSize, params.MinEffectiveSamples)
}

// TestComputeFitting_HighLevelChart17p4 pins the behaviour at the top of the
// current chart-level ceiling (17.4). The inverter's MaxInferredLevel is 20.0
// so there's still headroom, but samples this high exercise numerical
// conditioning that the mid-range tests don't.
func TestComputeFitting_HighLevelChart17p4(t *testing.T) {
	params := defaultParams()
	params.PriorStrength = 0

	trueLevel := 17.4
	officialLevel := 17.4

	rng := rand.New(rand.NewSource(11))
	samples := make([]Sample, 0, 60)
	for i := 0; i < 60; i++ {
		skill := 174.0 + rng.Float64()*10.0 // [174, 184] — tops out at 10·17.4 + 10
		score := simulateScore(trueLevel, skill)
		assert.GreaterOrEqual(t, score, 1_000_000)
		assert.LessOrEqual(t, score, 1_010_000)
		samples = append(samples, Sample{
			Username:      "p",
			Score:         score,
			PlayerSkill:   skill,
			PlayerRecords: 50,
		})
	}
	res := ComputeFitting(officialLevel, samples, params)
	if !assert.NotNil(t, res.FittingLevel, "top-level chart fitting should still publish") {
		return
	}
	assert.InDelta(t, trueLevel, *res.FittingLevel, 0.15,
		"ceiling-level fit should still track trueLevel tightly")
}

// TestComputeFitting_BranchBoundaries feeds scores exactly on the boundaries
// of the three piecewise rating branches (< 1_000_000, [1_000_000, 1_009_000),
// and [1_009_000, 1_010_000]) to make sure the fitting result does not
// discontinue at the seams — the real distribution clusters *right* at these
// boundaries so any discontinuity would warp real fits.
func TestComputeFitting_BranchBoundaries(t *testing.T) {
	params := defaultParams()
	params.PriorStrength = 0
	trueLevel := 16.0

	boundaries := []int{999_999, 1_000_000, 1_000_001, 1_008_999, 1_009_000, 1_009_500, 1_010_000}
	samples := make([]Sample, 0, len(boundaries)*5)
	for _, s := range boundaries {
		for rep := 0; rep < 5; rep++ {
			skill := float64(rating.SingleRating(trueLevel, s)) / 100.0
			samples = append(samples, Sample{
				Username:      "p",
				Score:         s,
				PlayerSkill:   skill,
				PlayerRecords: 50,
			})
		}
	}
	res := ComputeFitting(trueLevel, samples, params)
	if !assert.NotNil(t, res.FittingLevel, "boundary-only samples should still publish") {
		return
	}
	assert.InDelta(t, trueLevel, *res.FittingLevel, 0.15)
}

// TestComputeFitting_NoSamples: zero samples → Result with nil FittingLevel
// and all stats at their zero values. Exercises the early-return path.
func TestComputeFitting_NoSamples(t *testing.T) {
	res := ComputeFitting(16.5, nil, defaultParams())
	assert.Nil(t, res.FittingLevel)
	assert.Equal(t, 0, res.SampleCount)
	assert.Equal(t, 0.0, res.EffectiveSampleSize)
	assert.Equal(t, 0.0, res.WeightedMean)
	assert.Equal(t, 0.0, res.WeightedMedian)
}

// TestComputeFitting_AllBelowMinScore: every sample is filtered out by the
// MinScore gate → Result looks identical to the zero-samples case.
// Exercises the `if s.Score < params.MinScore { continue }` branch.
func TestComputeFitting_AllBelowMinScore(t *testing.T) {
	params := defaultParams()
	params.MinScore = 999_999 // higher than every sample below
	samples := []Sample{
		{Username: "a", Score: 900_000, PlayerSkill: 150.0, PlayerRecords: 50},
		{Username: "b", Score: 800_000, PlayerSkill: 140.0, PlayerRecords: 50},
		{Username: "c", Score: 500_000, PlayerSkill: 130.0, PlayerRecords: 50},
	}
	res := ComputeFitting(16.5, samples, params)
	assert.Nil(t, res.FittingLevel)
	assert.Equal(t, 0, res.SampleCount, "all samples gated out, raw count stays 0")
}

// TestComputeFitting_AllInverseFails: every sample's (score, skill) pair
// inverts to a level outside [MinInferredLevel, MaxInferredLevel]. Exercises
// the `if !ok { continue }` branch after InverseLevel.
func TestComputeFitting_AllInverseFails(t *testing.T) {
	params := defaultParams()
	params.MinScore = 0
	// A tiny score with a huge skill — implied level blows past MaxInferredLevel.
	samples := []Sample{
		{Username: "a", Score: 100_000, PlayerSkill: 500.0, PlayerRecords: 50},
		{Username: "b", Score: 100_000, PlayerSkill: 500.0, PlayerRecords: 50},
	}
	res := ComputeFitting(16.5, samples, params)
	assert.Nil(t, res.FittingLevel)
	assert.Equal(t, 0, res.SampleCount, "inverter rejected every sample")
}

// TestComputeFitting_ZeroVolumeWeight: players with 0 records combined with
// VolumeFullAt > 0 yield volume weight 0; those samples are dropped *after*
// inversion, so they still contribute to raw SampleCount but cannot influence
// the fit. Exercises the `w <= 0 || NaN` skip.
func TestComputeFitting_ZeroVolumeWeight(t *testing.T) {
	params := defaultParams()
	params.VolumeFullAt = 50
	params.MinEffectiveSamples = 0.5
	params.PriorStrength = 0
	trueLevel := 16.5

	// One real contributor + two zero-volume ghosts.
	samples := []Sample{
		{Username: "real", Score: simulateScore(trueLevel, 168), PlayerSkill: 168, PlayerRecords: 50},
		{Username: "ghost1", Score: 1_005_000, PlayerSkill: 168, PlayerRecords: 0},
		{Username: "ghost2", Score: 1_005_000, PlayerSkill: 168, PlayerRecords: 0},
	}
	res := ComputeFitting(trueLevel, samples, params)
	// All three invert successfully (raw counter increments before the weight
	// check), but only the non-ghost survives the `w <= 0` gate.
	assert.Equal(t, 3, res.SampleCount, "raw count covers all inversion survivors, ghosts included")
	assert.InDelta(t, 1.0, res.EffectiveSampleSize, 1e-6,
		"N_eff must collapse to 1 after ghosts are zeroed out")
}

// weightedMedian exercises.
func TestWeightedMedian_SimpleAndWeighted(t *testing.T) {
	assert.InDelta(t, 3.0, weightedMedian([]float64{1, 2, 3, 4, 5}, []float64{1, 1, 1, 1, 1}), 1e-9)
	// Weight-loaded median: 2 carries 10× the weight, so median should be 2.
	assert.InDelta(t, 2.0, weightedMedian([]float64{1, 2, 3, 4, 5}, []float64{1, 10, 1, 1, 1}), 1e-9)
}

// weightedMedian edge cases: single element and zero total weight fallback.
func TestWeightedMedian_EdgeCases(t *testing.T) {
	// Single element: returns the lone value regardless of weight.
	assert.InDelta(t, 7.5, weightedMedian([]float64{7.5}, []float64{1}), 1e-9)
	assert.InDelta(t, 7.5, weightedMedian([]float64{7.5}, []float64{0}), 1e-9)

	// Zero total weight: fall back to positional median of the sorted values.
	// Sorted: [1, 2, 3, 4, 5], positional median at index n/2 = 2 → value 3.
	assert.InDelta(t, 3.0, weightedMedian([]float64{5, 1, 3, 2, 4}, []float64{0, 0, 0, 0, 0}), 1e-9)
}


// --- DeviationPenalty coverage --------------------------------------------
//
// The penalty targets the exact scenario the user hit in production: an
// official lv14 chart that ended up with a handful of high-skill plays and
// therefore appeared "much harder" than it really is. Without the penalty,
// the static κ=5 shrinkage leaves the fit at (or near) the MaxDeviation cap.
// With the penalty, the prior is inflated by (dev² × nRef/nEff) and the fit
// collapses back toward the official level. The mirror test below verifies
// that DeviationPenalty=0 recovers the original behaviour exactly.

// fiveHighSkillSamples builds 5 identical high-skill samples that, when fed
// to ComputeFitting with *any* officialLevel, all invert to ≈17.0.
func fiveHighSkillSamples() []Sample {
	const trueLevel = 17.0
	const skill = 170.0
	score := simulateScore(trueLevel, skill)
	out := make([]Sample, 5)
	for i := range out {
		out[i] = Sample{
			Username:      string(rune('a' + i)),
			Score:         score,
			PlayerSkill:   skill,
			PlayerRecords: 100, // full volume weight
		}
	}
	return out
}

// TestComputeFitting_DeviationPenalty_PullsSmallSampleTowardOfficial covers
// the core motivation: a scarcely-played lv14 chart whose 5 samples reflect a
// "true" level of ~17. The penalty must override the static MaxDeviation cap
// so the published fit lands close to the official 14.0 rather than at the
// cap edge (15.5).
func TestComputeFitting_DeviationPenalty_PullsSmallSampleTowardOfficial(t *testing.T) {
	params := defaultParams()
	params.MinEffectiveSamples = 3.0
	params.PriorStrength = 5.0
	params.MaxDeviation = 1.5
	params.DeviationPenalty = 2.0

	officialLevel := 14.0
	res := ComputeFitting(officialLevel, fiveHighSkillSamples(), params)

	if !assert.NotNil(t, res.FittingLevel, "should still publish with 5 effective samples") {
		return
	}
	// Expected (see penalty derivation in calculator.go): boost ≈ 22.6,
	// κ_eff ≈ 113, shrunk ≈ 14.13. We assert a generous upper bound to absorb
	// proximity-weight / Tukey-scale jitter while still proving the penalty
	// pulled us well below the static-cap value (15.5).
	assert.Less(t, *res.FittingLevel, 14.5,
		"penalty must pull fit back toward official, beating the MaxDeviation cap")
	assert.Greater(t, *res.FittingLevel, officialLevel-0.01,
		"must not overshoot below the official level")
}

// TestComputeFitting_DeviationPenalty_Disabled_HitsStaticCap verifies that
// DeviationPenalty=0 is a *no-op*: the same data lands exactly at the legacy
// MaxDeviation-capped value. This is a regression guard for users who want
// to keep the old behaviour by zeroing the new parameter.
func TestComputeFitting_DeviationPenalty_Disabled_HitsStaticCap(t *testing.T) {
	params := defaultParams()
	params.MinEffectiveSamples = 3.0
	params.PriorStrength = 5.0
	params.MaxDeviation = 1.5
	params.DeviationPenalty = 0 // explicitly disabled

	officialLevel := 14.0
	res := ComputeFitting(officialLevel, fiveHighSkillSamples(), params)

	if !assert.NotNil(t, res.FittingLevel) {
		return
	}
	// Without the penalty, shrunk = (5·17 + 5·14)/10 = 15.5 exactly matches
	// the 1.5-level cap, so the published fit is pinned to officialLevel+1.5.
	assert.InDelta(t, officialLevel+params.MaxDeviation, *res.FittingLevel, 0.01,
		"disabled penalty ⇒ behaviour must equal the old static-κ + MaxDeviation path")
}

// TestComputeFitting_DeviationPenalty_LowDeviation_NoOp proves that when the
// weighted mean is *close* to the official level, the penalty is negligible
// (≈1× boost) regardless of sample sparsity. This guards against accidental
// over-shrinkage of well-behaved charts.
func TestComputeFitting_DeviationPenalty_LowDeviation_NoOp(t *testing.T) {
	params := defaultParams()
	params.MinEffectiveSamples = 3.0
	params.PriorStrength = 5.0
	params.DeviationPenalty = 2.0

	// Samples that imply ≈14.1 for a chart labelled 14.0 (dev = 0.1).
	officialLevel := 14.0
	const trueLevel = 14.1
	const skill = 141.0
	score := simulateScore(trueLevel, skill)
	samples := make([]Sample, 5)
	for i := range samples {
		samples[i] = Sample{
			Username:      string(rune('a' + i)),
			Score:         score,
			PlayerSkill:   skill,
			PlayerRecords: 100,
		}
	}
	res := ComputeFitting(officialLevel, samples, params)
	if !assert.NotNil(t, res.FittingLevel) {
		return
	}
	// boost ≈ 1 + 2·0.01·(6/5) ≈ 1.024 ⇒ barely touches κ. With κ=5, nEff≈5,
	// the old formula would give (5·14.1+5·14)/10 = 14.05. The new formula
	// yields ≈14.0495 — indistinguishable at our tolerance.
	assert.InDelta(t, 14.05, *res.FittingLevel, 0.02,
		"negligible penalty when dev≈0: must track the old shrinkage result")
}
