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
	ProximitySigma      float64 // σ of the Gaussian proximity weight (skill ≤ 10·Level side), in rating units
	HighSkillSigmaRatio float64 // σ multiplier for skill > 10·Level; <=0 or =1 means symmetric (disabled)
	VolumeFullAt        int     // saturation point of the volume weight (records)
	PriorStrength       float64 // κ in the Bayesian shrinkage toward the official level
	DeviationPenalty    float64 // λ; extra prior weight when sample-mean deviates from official (0 disables)
	MaxDeviation        float64 // hard cap on |FittingLevel − Level| at high levels (also used as the flat cap when the ramp below is disabled)
	MaxDeviationLow     float64 // cap at or below MaxDeviationLowAt; <=0 disables the level-dependent ramp (falls back to flat MaxDeviation)
	MaxDeviationLowAt   float64 // level at which cap = MaxDeviationLow; must be < MaxDeviationHighAt for the ramp to be active
	MaxDeviationHighAt  float64 // level at which cap = MaxDeviation; above this the cap stays at MaxDeviation (clamped)
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
		// proximity weight: Gaussian on (skill − 10·Level) in rating units.
		//
		// Crucially, the σ is **asymmetric**. A player whose skill greatly
		// exceeds 10·Level (i.e. a high-rank player on a low-level chart) is
		// almost certain to hit an AP-tier score, at which point InverseLevel
		// degenerates into simply echoing the player's skill rather than
		// measuring the chart. We therefore shrink σ on that side
		// (σ_high = σ · HighSkillSigmaRatio).
		//
		// Just rescaling σ is not enough on its own — Kish's N_eff is
		// scale-invariant, so 5 identically-weighted samples still count as
		// N_eff=5 even when each carries 1% weight. We therefore also
		// **hard-discard** any sample beyond a 2.5 · σ radius, so raw /
		// inferred / N_eff all drop together. Combined with asymmetric σ,
		// over-skilled samples (diff > 2.5 · σ_high) are dropped entirely,
		// which is what causes chronically mis-played lv14 charts to correctly
		// abstain rather than publish a skill-echoed fit.
		const proximityCutoffSigmas = 2.5
		diff := s.PlayerSkill - 10.0*officialLevel
		sigma := params.ProximitySigma
		if diff > 0 && params.HighSkillSigmaRatio > 0 {
			sigma = sigma * params.HighSkillSigmaRatio
		}
		if math.Abs(diff) > proximityCutoffSigmas*sigma {
			continue
		}
		proximity := math.Exp(-(diff * diff) / (2.0 * sigma * sigma))
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
	capVal := effectiveMaxDeviation(params, officialLevel)
	if capVal > 0 {
		if diff := shrunk - officialLevel; diff > capVal {
			shrunk = officialLevel + capVal
		} else if diff < -capVal {
			shrunk = officialLevel - capVal
		}
	}
	res.FittingLevel = &shrunk
	return res
}

// effectiveMaxDeviation returns the hard cap on |FittingLevel − Level| to
// apply at the given official level, honouring the optional level-dependent
// ramp driven by Params.MaxDeviationLow / MaxDeviationLowAt /
// MaxDeviationHighAt.
//
// Rationale: in this rhythm game (as in most rhythm games) the official
// "定数" axis is logarithmic — going from 15 to 16 is a MUCH bigger
// difficulty jump than going from 12 to 13. A flat cap like ±1.5 therefore
// lets a low-level chart drift across several "real" difficulty tiers
// before hitting the safety net, while being barely noticeable on a
// high-level chart. The ramp encodes this: the cap grows exponentially
// with level, so the permitted deviation is tight near the low end
// (the algorithm must stay close to the official level) and gradually
// loosens as we move up, where the official level is a coarser proxy.
//
// Interpolation is log-linear: with low / high caps c_low and c_high at
// levels L_low and L_high, cap(L) = c_low · (c_high / c_low)^t where
// t = (L − L_low) / (L_high − L_low), clamped to [c_low, c_high]. This is
// the natural shape of an exponential ramp and is easy to read off the
// endpoints — no hidden knobs.
//
// The ramp is deliberately opt-in: if MaxDeviationLow ≤ 0, or the
// endpoints are misconfigured (non-positive LowAt/HighAt, HighAt ≤ LowAt,
// or MaxDeviation ≤ MaxDeviationLow), we return the flat MaxDeviation so
// existing callers and tests keep their original behaviour.
func effectiveMaxDeviation(params Params, level float64) float64 {
	if params.MaxDeviationLow <= 0 ||
		params.MaxDeviationLowAt <= 0 ||
		params.MaxDeviationHighAt <= 0 ||
		params.MaxDeviationHighAt <= params.MaxDeviationLowAt ||
		params.MaxDeviation <= params.MaxDeviationLow {
		return params.MaxDeviation
	}
	if level <= params.MaxDeviationLowAt {
		return params.MaxDeviationLow
	}
	if level >= params.MaxDeviationHighAt {
		return params.MaxDeviation
	}
	t := (level - params.MaxDeviationLowAt) / (params.MaxDeviationHighAt - params.MaxDeviationLowAt)
	return params.MaxDeviationLow * math.Pow(params.MaxDeviation/params.MaxDeviationLow, t)
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
