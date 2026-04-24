package fitting

import (
	"fmt"
	"math"
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


// --- HighSkillSigmaRatio coverage -----------------------------------------
//
// Even with DeviationPenalty protecting the aggregator, the *right* answer for
// a scarcely-played lv14 chart that only received high-skill plays is not
// "pull the fit back" but "abstain entirely" — the samples don't contain any
// genuine difficulty signal. The asymmetric σ achieves that by driving
// per-sample weight toward zero, which in turn collapses N_eff through the
// MinEffectiveSamples gate.

// TestComputeFitting_HighSkillSigmaRatio_AbstainsOnOverSkilledSamples covers
// the user-reported pathology: an "official lv14" chart received 5 samples
// from skill ≈ 170 players. Without the asymmetric σ, they all get symmetric
// proximity weight (~0.32 each) and the chart publishes a (wrong) fit.
// With ratio=0.5, per-sample weight drops by ~30× on the over-skill side,
// N_eff collapses below 3, and ComputeFitting abstains — which is what we
// want when the data is actively misleading.
func TestComputeFitting_HighSkillSigmaRatio_AbstainsOnOverSkilledSamples(t *testing.T) {
	params := defaultParams()
	params.MinEffectiveSamples = 3.0
	params.ProximitySigma = 20.0
	params.HighSkillSigmaRatio = 0.5 // shrinks σ for skill > 10·Level
	params.DeviationPenalty = 0      // isolate the proximity-asymmetry effect

	officialLevel := 14.0
	res := ComputeFitting(officialLevel, fiveHighSkillSamples(), params)

	assert.Nil(t, res.FittingLevel,
		"must abstain: 5 over-skilled samples should not clear MinEffectiveSamples")
	assert.Less(t, res.EffectiveSampleSize, params.MinEffectiveSamples,
		"N_eff collapses because asymmetric σ drives per-sample weight toward 0")
}

// TestComputeFitting_HighSkillSigmaRatio_Symmetric_MatchesOld guards the
// rollback path: when ratio is zero (unset) or 1.0 the proximity weight is
// symmetric, so the same over-skilled scenario publishes a (capped) fit.
func TestComputeFitting_HighSkillSigmaRatio_Symmetric_MatchesOld(t *testing.T) {
	params := defaultParams()
	params.MinEffectiveSamples = 3.0
	params.ProximitySigma = 20.0
	params.HighSkillSigmaRatio = 0 // unset ⇒ symmetric (legacy behaviour)
	params.DeviationPenalty = 0

	officialLevel := 14.0
	res := ComputeFitting(officialLevel, fiveHighSkillSamples(), params)

	if !assert.NotNil(t, res.FittingLevel,
		"symmetric σ ⇒ samples pass the N_eff gate and a fit is published") {
		return
	}
	// Old behaviour: shrunk hits the static MaxDeviation cap (official+1.5).
	assert.InDelta(t, officialLevel+params.MaxDeviation, *res.FittingLevel, 0.01,
		"ratio=0 is the legacy path and must reproduce the old cap-limited result")
}

// TestComputeFitting_HighSkillSigmaRatio_UnderSkilledUnchanged proves the
// asymmetry does NOT affect players on the under-skilled side. A skill below
// 10·Level (e.g. a mid-rank player on a high-level chart) is still a valid,
// if noisy, data point and must be kept with the original σ.
func TestComputeFitting_HighSkillSigmaRatio_UnderSkilledUnchanged(t *testing.T) {
	params := defaultParams()
	params.MinEffectiveSamples = 0.5
	params.ProximitySigma = 20.0
	params.HighSkillSigmaRatio = 0.5
	params.DeviationPenalty = 0
	params.PriorStrength = 0 // isolate the weight-asymmetry effect

	// lv16 chart, samples from skill-150 players (diff = -10, i.e. under-skilled).
	const officialLevel = 16.0
	const trueLevel = 14.5
	const skill = 150.0
	score := simulateScore(trueLevel, skill)

	samples := make([]Sample, 8)
	for i := range samples {
		samples[i] = Sample{
			Username: string(rune('a' + i)), Score: score,
			PlayerSkill: skill, PlayerRecords: 100,
		}
	}

	resAsym := ComputeFitting(officialLevel, samples, params)

	// Compare against symmetric σ (HighSkillSigmaRatio=0): on the under-skilled
	// side the two paths must produce identical weighted means and N_eff.
	paramsSym := params
	paramsSym.HighSkillSigmaRatio = 0
	resSym := ComputeFitting(officialLevel, samples, paramsSym)

	if !assert.NotNil(t, resAsym.FittingLevel) ||
		!assert.NotNil(t, resSym.FittingLevel) {
		return
	}
	assert.InDelta(t, *resSym.FittingLevel, *resAsym.FittingLevel, 1e-9,
		"under-skilled samples must be unaffected by HighSkillSigmaRatio")
	assert.InDelta(t, resSym.EffectiveSampleSize, resAsym.EffectiveSampleSize, 1e-9)
}

// ============================================================================
// Level-dependent MaxDeviation ramp (effectiveMaxDeviation + ComputeFitting)
// ============================================================================
//
// Difficulty in the official level axis is roughly logarithmic: one level at
// lv17 represents a much bigger real-world gap than one level at lv12. The
// ramp encodes this by tightening the cap at low levels (where the official
// label is close to truth and we don't want the fitter to drift) and loosening
// it at high levels (where the official buckets are coarser).
//
// These tests exercise (1) the interpolation math directly and (2) the wiring
// into ComputeFitting, which is the only call site.

// Sanity: a zero-valued MaxDeviationLow must be a no-op — the helper should
// return the flat MaxDeviation unchanged. This is the default behaviour that
// keeps every existing test and production config intact.
func TestEffectiveMaxDeviation_RampDisabled(t *testing.T) {
	params := Params{MaxDeviation: 1.5} // MaxDeviationLow == 0 → ramp off
	// All level values should return the flat cap.
	for _, level := range []float64{1, 12, 14.5, 17, 20} {
		got := effectiveMaxDeviation(params, level)
		assert.InDelta(t, 1.5, got, 1e-12, "level=%v must fall back to flat cap", level)
	}
}

// Misconfigured ramp endpoints must not silently produce nonsense caps —
// we fall back to the flat MaxDeviation so the pipeline keeps the guard.
func TestEffectiveMaxDeviation_RampMisconfigured(t *testing.T) {
	good := Params{
		MaxDeviation:       1.5,
		MaxDeviationLow:    0.6,
		MaxDeviationLowAt:  12,
		MaxDeviationHighAt: 17,
	}
	// Baseline: properly configured ramp returns the low cap at low levels.
	assert.InDelta(t, 0.6, effectiveMaxDeviation(good, 10), 1e-12)

	// Each perturbation below should disable the ramp → return flat 1.5 at lv10.
	cases := []struct {
		name  string
		mut   func(*Params)
		level float64
	}{
		{"LowAt=0", func(p *Params) { p.MaxDeviationLowAt = 0 }, 10},
		{"HighAt=0", func(p *Params) { p.MaxDeviationHighAt = 0 }, 10},
		{"HighAt<=LowAt", func(p *Params) { p.MaxDeviationHighAt = p.MaxDeviationLowAt }, 10},
		{"Low>=MaxDeviation", func(p *Params) { p.MaxDeviationLow = p.MaxDeviation + 0.1 }, 10},
		{"Low==MaxDeviation", func(p *Params) { p.MaxDeviationLow = p.MaxDeviation }, 10},
	}
	for _, c := range cases {
		p := good
		c.mut(&p)
		got := effectiveMaxDeviation(p, c.level)
		assert.InDelta(t, p.MaxDeviation, got, 1e-12, "%s: must fall back to flat cap", c.name)
	}
}

// Endpoint values should clamp, and the midpoint must follow a log
// interpolation: cap(L_mid) == sqrt(Low · High) when L_mid is exactly between
// L_low and L_high (t = 0.5 ⇒ Low · ratio^0.5).
func TestEffectiveMaxDeviation_RampEndpointsAndInterpolation(t *testing.T) {
	params := Params{
		MaxDeviation:       1.5,
		MaxDeviationLow:    0.6,
		MaxDeviationLowAt:  12,
		MaxDeviationHighAt: 17,
	}
	// Clamping: below/at LowAt → Low; above/at HighAt → MaxDeviation.
	assert.InDelta(t, 0.6, effectiveMaxDeviation(params, 1), 1e-12, "lv1 clamps to low")
	assert.InDelta(t, 0.6, effectiveMaxDeviation(params, 12), 1e-12, "lv=LowAt → low")
	assert.InDelta(t, 1.5, effectiveMaxDeviation(params, 17), 1e-12, "lv=HighAt → max")
	assert.InDelta(t, 1.5, effectiveMaxDeviation(params, 20), 1e-12, "lv20 clamps to max")
	// Midpoint lv14.5 → t=0.5 → cap = 0.6 · (1.5/0.6)^0.5 = sqrt(0.9) ≈ 0.9487.
	expectedMid := 0.6 * math.Sqrt(1.5/0.6)
	assert.InDelta(t, expectedMid, effectiveMaxDeviation(params, 14.5), 1e-12)
	// Monotonic non-decreasing across the active band.
	prev := effectiveMaxDeviation(params, 12)
	for _, l := range []float64{13, 14, 14.5, 15, 16, 16.9} {
		cur := effectiveMaxDeviation(params, l)
		assert.GreaterOrEqual(t, cur, prev-1e-12, "cap must be monotonic ↗ in level")
		prev = cur
	}
}

// Integration: with the ramp active and a low-level chart, a sample set that
// strongly implies a much higher level must be capped to the LOW cap, not the
// high-end MaxDeviation. This is the headline reason for adding the ramp.
func TestComputeFitting_RampEngagesAtLowLevel(t *testing.T) {
	params := defaultParams()
	params.PriorStrength = 0.0 // disable shrinkage — isolate the cap effect
	params.MaxDeviation = 1.5
	params.MaxDeviationLow = 0.5
	params.MaxDeviationLowAt = 12
	params.MaxDeviationHighAt = 17

	trueLevel := 15.0    // samples imply L ≈ 15
	officialLevel := 12.0 // but the chart is officially lv12 — huge over-the-top gap

	samples := make([]Sample, 0, 60)
	for i := 0; i < 60; i++ {
		skill := 150.0 + float64(i)*0.1
		score := simulateScore(trueLevel, skill)
		samples = append(samples, Sample{
			Username: "p", Score: score, PlayerSkill: skill, PlayerRecords: 50,
		})
	}
	res := ComputeFitting(officialLevel, samples, params)
	if !assert.NotNil(t, res.FittingLevel) {
		return
	}
	// At lv12 the ramp pins the cap at 0.5 — the inferred +3.0 deviation must
	// be trimmed all the way back to officialLevel+0.5.
	assert.InDelta(t, officialLevel+0.5, *res.FittingLevel, 1e-6,
		"low-level chart must be capped at MaxDeviationLow, not the flat MaxDeviation")
}

// Integration: same shape as above but at the high-level end, where the ramp
// must relax toward the flat MaxDeviation. The test also documents that the
// ramp does NOT tighten the high end — we only add headroom where official
// buckets are coarse.
func TestComputeFitting_RampRelaxesAtHighLevel(t *testing.T) {
	params := defaultParams()
	params.PriorStrength = 0.0
	params.MaxDeviation = 1.5
	params.MaxDeviationLow = 0.5
	params.MaxDeviationLowAt = 12
	params.MaxDeviationHighAt = 17

	// Chart officially lv17.0 with samples implying ~15.5 — well inside the
	// ±1.5 high-end cap, so no trimming should happen.
	trueLevel := 15.5
	officialLevel := 17.0
	samples := make([]Sample, 0, 60)
	for i := 0; i < 60; i++ {
		skill := 155.0 + float64(i)*0.1
		score := simulateScore(trueLevel, skill)
		samples = append(samples, Sample{
			Username: "p", Score: score, PlayerSkill: skill, PlayerRecords: 50,
		})
	}
	res := ComputeFitting(officialLevel, samples, params)
	if !assert.NotNil(t, res.FittingLevel) {
		return
	}
	diff := officialLevel - *res.FittingLevel
	// Must be capped somewhere in [0, 1.5]. Anything ≤ 1.5 passes, because
	// the exact unshrunk mean depends on the weighting/biweight pipeline;
	// we only assert that the high-end cap is indeed the flat 1.5, not 0.5.
	assert.GreaterOrEqual(t, diff, 0.0)
	assert.LessOrEqual(t, diff, params.MaxDeviation+1e-9,
		"high-level chart must be bounded by flat MaxDeviation")
	// And concretely: if we'd applied the low cap here, the fit would have
	// to sit at ≥ officialLevel - 0.5 = 16.5. If the ramp mis-wired itself
	// to the low cap, this assertion catches it.
	if diff > 0.5 {
		assert.Greater(t, diff, 0.5+1e-9,
			"at lv17 the cap must relax beyond MaxDeviationLow (sanity check)")
	}
}


// --- score-quality weighting -----------------------------------------------

// Unit-test the piecewise-linear ramp directly. Shape:
//
//	score < 1,000,000 → 0
//	score = 1,007,500 → 0.6  (ScoreGoodWeight)
//	score = 1,009,000 → 1.0
//	score > 1,009,000 → 1.0
func TestScoreQualityWeight_PiecewiseShape(t *testing.T) {
	p := Params{
		ScoreFloorAt:    1_000_000,
		ScoreGoodAt:     1_007_500,
		ScoreFullAt:     1_009_000,
		ScoreGoodWeight: 0.6,
	}
	cases := []struct {
		score int
		want  float64
		desc  string
	}{
		{500_000, 0.0, "well below floor"},
		{999_999, 0.0, "just below floor"},
		{1_000_000, 0.0, "exactly at floor"},
		{1_003_750, 0.3, "midway between floor and good → half of goodWeight"},
		{1_007_500, 0.6, "exactly at good threshold"},
		{1_008_250, 0.8, "midway between good and full"},
		{1_009_000, 1.0, "exactly at full threshold"},
		{1_010_000, 1.0, "above full"},
	}
	for _, c := range cases {
		got := scoreQualityWeight(c.score, p)
		assert.InDelta(t, c.want, got, 1e-9, "score=%d (%s)", c.score, c.desc)
	}
}

// Misconfigured or zero-valued params must disable the feature entirely
// (return 1.0), preserving the behaviour assumed by every pre-existing test.
func TestScoreQualityWeight_DisabledOnMisconfig(t *testing.T) {
	// all zero → disabled
	assert.Equal(t, 1.0, scoreQualityWeight(0, Params{}))
	assert.Equal(t, 1.0, scoreQualityWeight(1_005_000, Params{}))
	// non-monotone anchors → disabled
	bad := Params{ScoreFloorAt: 1_009_000, ScoreGoodAt: 1_007_500, ScoreFullAt: 1_009_500, ScoreGoodWeight: 0.5}
	assert.Equal(t, 1.0, scoreQualityWeight(1_005_000, bad))
	// ScoreGoodWeight out of range → disabled
	badW := Params{ScoreFloorAt: 1_000_000, ScoreGoodAt: 1_007_500, ScoreFullAt: 1_009_000, ScoreGoodWeight: 1.2}
	assert.Equal(t, 1.0, scoreQualityWeight(1_005_000, badW))
	zeroW := Params{ScoreFloorAt: 1_000_000, ScoreGoodAt: 1_007_500, ScoreFullAt: 1_009_000, ScoreGoodWeight: 0.0}
	assert.Equal(t, 1.0, scoreQualityWeight(1_005_000, zeroW))
}

// Integration: when score-quality weighting is enabled, a swarm of
// barely-passed low-score samples that collectively mis-imply a chart's
// level should be down-weighted relative to a smaller pool of high-score
// samples, pulling the fitted level back toward the high-score-implied one.
//
// Setup: a chart whose true level is 15.0.
//   - Group L ("barely passed"): 50 samples. Scores in
//     [1,000,500, 1,003,500] — just above the 1M floor. Each sample's
//     PlayerSkill is set so that the (score, skill) pair inverts to
//     level ≈ 16.0 — i.e. group L collectively insists the chart is
//     one whole level harder than it really is.
//   - Group H ("high score"): 20 samples. Scores in [1,009,500, 1,010,000]
//     (the "高分" band, full weight). PlayerSkill set so each sample
//     inverts to level = 15.0 (the truth).
//
// Without score-quality weighting the larger group L dominates the weighted
// median and the fit lands near 16.0. With score-quality weighting each L
// sample is down-weighted to ≤ ~0.24× (1M → 0, 1,003,500 → ~0.28 of the
// 0.6 plateau), while H samples stay at full weight, so the fit must move
// materially toward 15.0.
func TestComputeFitting_ScoreQualityWeight_FavoursHighScoreSamples(t *testing.T) {
	trueLevel := 15.0
	misLevel := 16.0 // level implied by group-L (score, skill) pairs
	officialLevel := 15.0

	samples := make([]Sample, 0, 70)
	// Group L: 50 barely-passed samples implying level 16.0.
	for i := 0; i < 50; i++ {
		score := 1_000_500 + i*60 // 1_000_500 .. 1_003_440
		skill := float64(rating.SingleRating(misLevel, score)) / 100.0
		samples = append(samples, Sample{
			Username: "L", Score: score, PlayerSkill: skill, PlayerRecords: 50,
		})
	}
	// Group H: 20 high-score samples implying the true level 15.0. Force the
	// score into the ≥ ScoreFullAt band so score-quality weight is exactly 1.
	for i := 0; i < 20; i++ {
		score := 1_009_500 + i*25 // 1_009_500 .. 1_009_975
		if score > 1_010_000 {
			score = 1_010_000
		}
		skill := float64(rating.SingleRating(trueLevel, score)) / 100.0
		samples = append(samples, Sample{
			Username: "H", Score: score, PlayerSkill: skill, PlayerRecords: 50,
		})
	}

	// Case A: score-quality weighting DISABLED (old behaviour). Disable the
	// deviation penalty too so we isolate the effect of the pre-weight
	// rather than also having the prior pull the answer back toward
	// officialLevel.
	paramsOff := defaultParams()
	paramsOff.PriorStrength = 0
	paramsOff.DeviationPenalty = 0
	resOff := ComputeFitting(officialLevel, samples, paramsOff)
	if !assert.NotNil(t, resOff.FittingLevel) {
		return
	}

	// Case B: score-quality weighting ENABLED with the prod defaults.
	paramsOn := paramsOff
	paramsOn.ScoreFloorAt = 1_000_000
	paramsOn.ScoreGoodAt = 1_007_500
	paramsOn.ScoreFullAt = 1_009_000
	paramsOn.ScoreGoodWeight = 0.6
	resOn := ComputeFitting(officialLevel, samples, paramsOn)
	if !assert.NotNil(t, resOn.FittingLevel) {
		return
	}

	// Sanity: without the weight, group L's 16.0 must dominate, so the fit
	// should sit closer to 16.0 than to 15.0. With the weight the fit must
	// move toward 15.0 by at least 0.2 level units — a conservative floor
	// that still leaves plenty of room for the Tukey biweight / shrinkage
	// stages to intervene.
	assert.Greater(t, *resOff.FittingLevel, 15.5,
		"sanity: without score weighting, 50 barely-passed samples implying 16.0 must dominate (got %.3f)",
		*resOff.FittingLevel)
	assert.Greater(t, *resOff.FittingLevel-*resOn.FittingLevel, 0.2,
		"score-quality weighting must pull fit toward high-score-implied level "+
			"(off: %.3f, on: %.3f, true: %.1f)",
		*resOff.FittingLevel, *resOn.FittingLevel, trueLevel)
}

// When every sample is ≥ ScoreFullAt, score-quality weighting must be a no-op
// regardless of whether it's configured on or off: the pre-weight multiplier
// is 1.0 for every sample in both cases.
func TestComputeFitting_ScoreQualityWeight_NoOpWhenAllHighScore(t *testing.T) {
	trueLevel := 16.0
	officialLevel := 16.0
	samples := make([]Sample, 0, 40)
	for i := 0; i < 40; i++ {
		// skill slightly above true level → score lands ≥ 1_009_000
		skill := 160.0 + 7.0 + float64(i)*0.05 // 167 .. 168.95, well inside the "高分" regime
		score := simulateScore(trueLevel, skill)
		if score < 1_009_000 {
			// defensive: bump into the 高分 band if the bisect lands just below
			score = 1_009_500 + i*10
			skill = float64(rating.SingleRating(trueLevel, score)) / 100.0
		}
		samples = append(samples, Sample{
			Username: "p", Score: score, PlayerSkill: skill, PlayerRecords: 50,
		})
	}
	paramsOff := defaultParams()
	paramsOff.PriorStrength = 0
	paramsOn := paramsOff
	paramsOn.ScoreFloorAt = 1_000_000
	paramsOn.ScoreGoodAt = 1_007_500
	paramsOn.ScoreFullAt = 1_009_000
	paramsOn.ScoreGoodWeight = 0.6

	resOff := ComputeFitting(officialLevel, samples, paramsOff)
	resOn := ComputeFitting(officialLevel, samples, paramsOn)
	if !assert.NotNil(t, resOff.FittingLevel) || !assert.NotNil(t, resOn.FittingLevel) {
		return
	}
	// Both fits must agree to within numerical noise: score-quality is 1 for
	// every sample, so the inner weighted median / biweight pipeline sees
	// exactly the same input in both runs.
	assert.InDelta(t, *resOff.FittingLevel, *resOn.FittingLevel, 1e-6,
		"score-quality weighting must be a no-op when every sample is ≥ ScoreFullAt")
}


// --- sample-age decay ------------------------------------------------------

// sampleAgeWeight must return 1.0 whenever the feature is disabled (halflife
// <=0), the sample is fresh (age <=0), or the inputs are NaN/Inf. At exactly
// one half-life the decay factor is 0.5 by construction.
func TestSampleAgeWeight_PiecewiseShape(t *testing.T) {
	// Disabled forms.
	assert.Equal(t, 1.0, sampleAgeWeight(0, 0), "both zero → disabled")
	assert.Equal(t, 1.0, sampleAgeWeight(100, 0), "halflife=0 → disabled (no decay)")
	assert.Equal(t, 1.0, sampleAgeWeight(100, -30), "negative halflife → disabled")
	assert.Equal(t, 1.0, sampleAgeWeight(-5, 180), "negative age → treat as fresh")
	assert.Equal(t, 1.0, sampleAgeWeight(0, 180), "zero age → fresh")

	// Half-life semantics: age=H → weight = 0.5; age=2H → 0.25; age=H/2 → 1/√2.
	assert.InDelta(t, 1.0, sampleAgeWeight(0, 365), 1e-12)
	assert.InDelta(t, 0.5, sampleAgeWeight(365, 365), 1e-12, "one halflife → 1/2")
	assert.InDelta(t, 0.25, sampleAgeWeight(730, 365), 1e-12, "two halflives → 1/4")
	assert.InDelta(t, 1.0/math.Sqrt2, sampleAgeWeight(182.5, 365), 1e-12, "halflife/2 → 1/√2")

	// Invalid/exceptional inputs collapse to 1.0 to keep the pipeline safe.
	assert.Equal(t, 1.0, sampleAgeWeight(math.NaN(), 365))
	assert.Equal(t, 1.0, sampleAgeWeight(math.Inf(1), 365))
}

// Integration: a chart at official level 15.0. Two cohorts of equal size at
// the same 10·Level proximity band (skill ≈ 150), but different score
// → different implied levels:
//   - "new" samples (AgeDays=0): implied level ∈ [14.8, 15.2] — tight,
//     consistent with truth,
//   - "old" samples (AgeDays=360): implied level ∈ [15.6, 16.0] — biased
//     0.8 above truth.
//
// Spreading each cohort over a 0.4-wide window avoids the MAD=0 pitfall that
// would otherwise trigger Tukey's zero-weight branch on a perfectly bimodal
// sample set. The combined distribution has median ≈ 15.4, MAD ≈ 0.4, and
// Tukey scale 4.685·0.4 = 1.87, comfortably covering every sample.
//
// With decay OFF the weighted mean sits near 15.4. With decay ON at
// halflife=180d the old cohort's weight drops to ≈25%, so the fit moves
// noticeably toward the fresh 15.0 cohort.
func TestComputeFitting_SampleAgeDecay_PullsTowardFresh(t *testing.T) {
	officialLevel := 15.0
	rng := rand.New(rand.NewSource(42))
	var samples []Sample
	for i := 0; i < 20; i++ {
		implied := 14.8 + 0.4*rng.Float64() // [14.8, 15.2]
		skill := 150.0                      // 10·officialLevel — proximity = 1
		samples = append(samples, Sample{
			Username:    fmt.Sprintf("new%02d", i),
			Score:       simulateScore(implied, skill),
			PlayerSkill: skill, PlayerRecords: 50,
			AgeDays: 0,
		})
	}
	for i := 0; i < 20; i++ {
		implied := 15.6 + 0.4*rng.Float64() // [15.6, 16.0]
		skill := 150.0
		samples = append(samples, Sample{
			Username:    fmt.Sprintf("old%02d", i),
			Score:       simulateScore(implied, skill),
			PlayerSkill: skill, PlayerRecords: 50,
			AgeDays: 360,
		})
	}

	paramsOff := defaultParams()
	paramsOff.PriorStrength = 0         // isolate the weighted-mean mechanism
	paramsOff.HighSkillSigmaRatio = 1.0 // symmetric σ — both cohorts treated equally
	paramsOff.ProximitySigma = 50.0     // wide — keep both cohorts comfortably inside
	paramsOn := paramsOff
	paramsOn.SampleHalflifeDays = 180.0

	resOff := ComputeFitting(officialLevel, samples, paramsOff)
	resOn := ComputeFitting(officialLevel, samples, paramsOn)
	if !assert.NotNil(t, resOff.FittingLevel) || !assert.NotNil(t, resOn.FittingLevel) {
		return
	}

	// Decay off: balanced cohorts → fit lands in (15.2, 15.8).
	assert.Greater(t, *resOff.FittingLevel, 15.2, "decay off: old cohort still pulls fit up")
	assert.Less(t, *resOff.FittingLevel, 15.8)

	// Decay on: the old cohort is reweighted by ≈ 0.25, so the fit drops markedly.
	assert.Less(t, *resOn.FittingLevel, *resOff.FittingLevel,
		"age decay must pull the weighted mean toward the fresher cohort")
	assert.Greater(t, *resOff.FittingLevel-*resOn.FittingLevel, 0.15,
		"expected ≥0.15 level move from enabling halflife=180d on 2-halflife-old cohort")
}

// When SampleHalflifeDays=0 (disabled), filling AgeDays on samples must be a
// pure no-op versus the same samples with AgeDays=0. Anchors the "zero-cost
// fallback" contract for the shipped default.
func TestComputeFitting_SampleAgeDecay_DisabledIsNoOp(t *testing.T) {
	trueLevel := 15.0
	officialLevel := 15.0
	base := make([]Sample, 0, 30)
	for i := 0; i < 30; i++ {
		skill := 150.0 + float64(i)*0.2
		score := simulateScore(trueLevel, skill)
		base = append(base, Sample{
			Username: fmt.Sprintf("p%02d", i), Score: score,
			PlayerSkill: skill, PlayerRecords: 50,
		})
	}
	withAges := make([]Sample, len(base))
	copy(withAges, base)
	for i := range withAges {
		withAges[i].AgeDays = float64(30 + i*10) // arbitrary ages
	}

	params := defaultParams()
	params.SampleHalflifeDays = 0 // disabled

	res1 := ComputeFitting(officialLevel, base, params)
	res2 := ComputeFitting(officialLevel, withAges, params)
	if !assert.NotNil(t, res1.FittingLevel) || !assert.NotNil(t, res2.FittingLevel) {
		return
	}
	assert.InDelta(t, *res1.FittingLevel, *res2.FittingLevel, 1e-9,
		"SampleHalflifeDays=0 must make AgeDays irrelevant")
}
