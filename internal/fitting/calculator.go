package fitting

import (
	"math"
	"sort"
)

// Sample represents one (player, score) data point for a single chart.
//
// PlayerSkill is the player's float average B50 single-chart rating (i.e.
// sum(top rating ints) / (K * 100), for the top K records where K ≤ 50). It
// acts as the "target rating" fed to InverseLevel to obtain an implied level
// for this (player, score) pair.
//
// PlayerRecords is the player's total best_play_records count — used by the
// volume weight so that newcomers with very few records contribute less.
type Sample struct {
	Username      string
	Score         int
	PlayerSkill   float64
	PlayerRecords int
}

// Params bundles all tunable knobs exposed through config.fitting.*. See
// docs/fitting_level.en.md (English) / docs/fitting_level.zh.md (中文) for
// the full derivation.
type Params struct {
	MinEffectiveSamples float64 // minimum N_eff to publish a FittingLevel
	ProximitySigma      float64 // σ of the Gaussian proximity weight, in rating units
	VolumeFullAt        int     // saturation point of the volume weight (records)
	PriorStrength       float64 // κ in the Bayesian shrinkage toward the official level
	DeviationPenalty    float64 // λ; extra prior weight when sample-mean deviates from official (0 disables)
	MaxDeviation        float64 // hard cap on |FittingLevel − Level|
	MinScore            int     // discard samples with score below this
	TukeyK              float64 // tuning constant for the Tukey biweight robustness step
	MinPlayerRecords    int     // drop samples from players with fewer than this many best_play_records (0 = no filter)
}

// Result is the output of ComputeFitting. FittingLevel is nil when the chart
// did not accumulate enough effective samples. The remaining fields are
// populated in every case (with zero values when the sample set was empty),
// so callers can persist them into chart_statistics for post-hoc analysis.
type Result struct {
	FittingLevel        *float64
	SampleCount         int
	EffectiveSampleSize float64
	WeightedMean        float64
	WeightedMedian      float64
	StdDev              float64
	MAD                 float64
}

// ComputeFitting runs the full pipeline for one chart:
//
//  1. Per-sample inverse-rating: given (score, PlayerSkill), solve for the
//     level that makes SingleRating match the player's typical B50 rating.
//  2. Pre-weighting: each sample receives a composite weight equal to
//     proximityWeight × volumeWeight. proximityWeight is a Gaussian centered
//     on 10·Level (so players whose skill matches the chart's official level
//     dominate); volumeWeight ramps linearly up to VolumeFullAt records.
//  3. Robust trimming: compute the weighted median of inferred levels, the
//     weighted MAD, then apply Tukey biweight attenuation so extreme
//     residuals receive zero weight.
//  4. Aggregation: compute weighted mean and Kish effective sample size
//     (N_eff = (Σw)² / Σw²) of the surviving samples.
//  5. Bayesian shrinkage: pull the weighted mean toward the official level
//     with prior strength κ, then cap the deviation at MaxDeviation.
//
// When fewer than MinEffectiveSamples surviving samples remain, FittingLevel
// is left nil — we prefer abstention over publishing a shaky number.
func ComputeFitting(officialLevel float64, samples []Sample, params Params) Result {
	res := Result{}

	// ----- 1. Per-sample inversion + pre-weight -----
	inferred := make([]float64, 0, len(samples))
	prew := make([]float64, 0, len(samples))
	var raw int
	for _, s := range samples {
		if s.Score < params.MinScore {
			continue
		}
		level, ok := InverseLevel(s.Score, s.PlayerSkill)
		if !ok {
			continue
		}
		raw++
		// proximity weight: Gaussian on |skill − 10·Level| in rating units.
		diff := s.PlayerSkill - 10.0*officialLevel
		proximity := math.Exp(-(diff * diff) / (2.0 * params.ProximitySigma * params.ProximitySigma))
		// volume weight: linear ramp to 1.0 at VolumeFullAt records.
		volume := 1.0
		if params.VolumeFullAt > 0 && s.PlayerRecords < params.VolumeFullAt {
			volume = float64(s.PlayerRecords) / float64(params.VolumeFullAt)
		}
		w := proximity * volume
		if w <= 0 || math.IsNaN(w) {
			continue
		}
		inferred = append(inferred, level)
		prew = append(prew, w)
	}
	res.SampleCount = raw

	if len(inferred) == 0 {
		return res
	}

	// ----- 2. Weighted median + MAD -----
	median := weightedMedian(inferred, prew)
	res.WeightedMedian = median
	absDev := make([]float64, len(inferred))
	for i, v := range inferred {
		absDev[i] = math.Abs(v - median)
	}
	mad := weightedMedian(absDev, prew)
	res.MAD = mad

	// ----- 3. Tukey biweight robust weights -----
	final := make([]float64, len(inferred))
	// Denominator for the scaled residual. Use a safe floor when MAD is near
	// zero (samples unusually concentrated), scaled from the official level:
	// 1% of (|Level|+1) is a conservative minimum dispersion.
	scale := params.TukeyK * mad
	if scale <= 1e-9 {
		scale = params.TukeyK * 0.01 * (math.Abs(officialLevel) + 1.0)
	}
	for i := range inferred {
		u := (inferred[i] - median) / scale
		if math.Abs(u) >= 1.0 {
			final[i] = 0
			continue
		}
		biw := 1.0 - u*u
		final[i] = prew[i] * biw * biw
	}

	// ----- 4. Weighted mean + Kish N_eff -----
	var sumW, sumWX, sumW2 float64
	for i, w := range final {
		if w <= 0 {
			continue
		}
		sumW += w
		sumWX += w * inferred[i]
		sumW2 += w * w
	}
	if sumW == 0 {
		return res
	}
	mean := sumWX / sumW
	res.WeightedMean = mean
	nEff := sumW * sumW / sumW2
	res.EffectiveSampleSize = nEff

	var sumWDiff2 float64
	for i, w := range final {
		if w <= 0 {
			continue
		}
		d := inferred[i] - mean
		sumWDiff2 += w * d * d
	}
	res.StdDev = math.Sqrt(sumWDiff2 / sumW)

	if nEff < params.MinEffectiveSamples {
		return res
	}

	// ----- 5. Bayesian shrinkage + deviation cap -----
	//
	// Standard conjugate shrinkage is (nEff·mean + κ·official)/(nEff + κ).
	// With a static κ, a handful of outlier players on a scarcely-played
	// chart can drag the fit several levels away from the official value,
	// which is almost never what we want: the dataset simply isn't large
	// enough to justify that much confidence.
	//
	// We therefore inflate κ dynamically when **both** of these are true:
	//
	//   (a) the weighted mean has drifted far from the official level, and
	//   (b) the effective sample count is small relative to the threshold
	//       we consider "fully trustworthy" (2× MinEffectiveSamples).
	//
	// The multiplicative form  (1 + λ·dev²·nRef/nEff)  is zero-cost when
	// dev≈0 or nEff≫nRef, and grows quadratically in dev / inversely in
	// nEff, which matches the intuition that larger deviations demand more
	// evidence. λ = params.DeviationPenalty; set it to 0 to recover the
	// original static-κ behaviour.
	dev := mean - officialLevel
	nRef := 2.0 * params.MinEffectiveSamples
	if nRef < 1.0 {
		nRef = 1.0
	}
	boost := 1.0
	if params.DeviationPenalty > 0 {
		boost = 1.0 + params.DeviationPenalty*dev*dev*(nRef/nEff)
	}
	kappaEff := params.PriorStrength * boost
	shrunk := (nEff*mean + kappaEff*officialLevel) / (nEff + kappaEff)
	if params.MaxDeviation > 0 {
		if diff := shrunk - officialLevel; diff > params.MaxDeviation {
			shrunk = officialLevel + params.MaxDeviation
		} else if diff < -params.MaxDeviation {
			shrunk = officialLevel - params.MaxDeviation
		}
	}
	res.FittingLevel = &shrunk
	return res
}

// weightedMedian returns the value v such that the cumulative weight of
// samples ≤ v is the first to reach ≥ sumWeights/2. When all weights are
// non-negative and sumWeights > 0, the result is well-defined.
// Values and weights must be the same length and >0 length.
func weightedMedian(values, weights []float64) float64 {
	n := len(values)
	if n == 1 {
		return values[0]
	}
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool {
		return values[idx[a]] < values[idx[b]]
	})
	var total float64
	for _, w := range weights {
		total += w
	}
	if total <= 0 {
		// Fallback: positional median over the sorted order.
		return values[idx[n/2]]
	}
	target := total / 2.0
	var cum float64
	for _, i := range idx {
		cum += weights[i]
		if cum >= target {
			return values[i]
		}
	}
	// Numerical fallthrough; should not happen in practice.
	return values[idx[n-1]]
}
