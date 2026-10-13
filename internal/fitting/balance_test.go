package fitting

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fittingResult(value float64) Result { return Result{FittingLevel: &value} }

func TestBalanceTotalsRespectsCapsAndUnweightedPopulation(t *testing.T) {
	p := Params{CalibrationEnabled: true, BalanceTotal: true, MaxDeviation: .3}
	levels := map[int]float64{1: 15, 2: 15, 3: 15, 4: 15}
	results := map[int]Result{1: fittingResult(15.3), 2: fittingResult(15.3), 3: fittingResult(14.8), 4: {}}
	original := results[1]
	original.SampleCount, original.WeightedMean = 10000, 15.3
	results[1] = original
	report := BalanceFittingTotals(levels, results, p)
	assert.Equal(t, 3, report.Charts)
	assert.InDelta(t, 45, report.OfficialSum, 1e-10)
	assert.InDelta(t, report.OfficialSum, report.FittingSum, 1e-10)
	assert.InDelta(t, .15, *results[1].FittingLevel-15, 1e-10)
	assert.InDelta(t, -.3, *results[3].FittingLevel-15, 1e-10)
	assert.Nil(t, results[4].FittingLevel)
	assert.Equal(t, 10000, results[1].SampleCount)
	assert.Equal(t, 15.3, results[1].WeightedMean)
	assert.Equal(t, 15.3, *original.FittingLevel, "input pointers are not mutated")
}

func TestBalancePreservesSmallDifferencesAndLevelRange(t *testing.T) {
	p := Params{CalibrationEnabled: true, BalanceTotal: true}
	levels := map[int]float64{1: 15, 2: 15}
	results := map[int]Result{1: fittingResult(15.01), 2: fittingResult(14.99)}
	BalanceFittingTotals(levels, results, p)
	assert.InDelta(t, 15.01, *results[1].FittingLevel, 1e-10)
	assert.InDelta(t, 14.99, *results[2].FittingLevel, 1e-10)
	levels = map[int]float64{1: .1, 2: 20, 3: 15}
	results = map[int]Result{1: fittingResult(-1), 2: fittingResult(23), 3: fittingResult(16)}
	report := BalanceFittingTotals(levels, results, p)
	assert.InDelta(t, report.OfficialSum, report.FittingSum, 1e-10)
	for _, result := range results {
		assert.GreaterOrEqual(t, *result.FittingLevel, MinInferredLevel)
		assert.LessOrEqual(t, *result.FittingLevel, MaxInferredLevel)
	}
}

func TestBalanceDisabledEmptyAndDeterministic(t *testing.T) {
	p := Params{CalibrationEnabled: true, BalanceTotal: true, MaxDeviation: .3}
	assert.Equal(t, BalanceReport{}, BalanceFittingTotals(nil, nil, p))
	levels := map[int]float64{1: 15, 2: 14, 3: 16}
	a, b := map[int]Result{}, map[int]Result{}
	values := map[int]float64{1: 15.3, 2: 13.8, 3: 16.02}
	for _, id := range []int{1, 2, 3} {
		a[id] = fittingResult(values[id])
	}
	for _, id := range []int{3, 2, 1} {
		b[id] = fittingResult(values[id])
	}
	assert.Equal(t, BalanceFittingTotals(levels, a, p), BalanceFittingTotals(levels, b, p))
	assert.Equal(t, a, b)
	for _, disabled := range []Params{{CalibrationEnabled: true}, {BalanceTotal: true}} {
		results := map[int]Result{1: fittingResult(15.3)}
		report := BalanceFittingTotals(levels, results, disabled)
		assert.Equal(t, 15.3, *results[1].FittingLevel)
		assert.Zero(t, report.Offset)
	}
}

func correctedSamples(level, skill float64, n int) []Sample {
	samples := make([]Sample, n)
	for i := range samples {
		correction := 0.0
		samples[i] = Sample{Score: simulateScore(level, skill), PlayerSkill: skill,
			PlayerRecords: 100, LevelCorrection: &correction}
	}
	return samples
}

func TestCalibratedEstimateHasNoAdditionalGain(t *testing.T) {
	p := defaultParams()
	p.CalibrationEnabled = true
	result := ComputeFitting(15, correctedSamples(15.3, 155, 30), p)
	require.NotNil(t, result.FittingLevel)
	expected := (result.EffectiveSampleSize*result.WeightedMean + p.PriorStrength*15) / (result.EffectiveSampleSize + p.PriorStrength)
	assert.InDelta(t, expected, *result.FittingLevel, 1e-10, "no 0.75 multiplier after shrinkage")
	p.MaxDeviation = .1
	capped := ComputeFitting(15, correctedSamples(15.8, 160, 30), p)
	require.NotNil(t, capped.FittingLevel)
	assert.InDelta(t, 15.1, *capped.FittingLevel, 1e-10, "retain the final deviation cap")
}

func TestCalibrationNoiseDampsWeakSignalWithoutCreatingEvidence(t *testing.T) {
	p := defaultParams()
	p.CalibrationEnabled, p.PriorStrength = true, 0
	samples := append(correctedSamples(15.05, 155, 15), correctedSamples(15.3, 155, 15)...)
	independent := ComputeFitting(15, samples, p)
	p.CalibrationNoisePenalty = 1
	attenuated := ComputeFitting(15, samples, p)
	require.NotNil(t, independent.FittingLevel)
	require.NotNil(t, attenuated.FittingLevel)
	assert.Less(t, math.Abs(*attenuated.FittingLevel-15), math.Abs(*independent.FittingLevel-15))
	assert.Nil(t, ComputeFitting(15, nil, p).FittingLevel)
	for _, penalty := range []float64{-1, math.NaN(), math.Inf(1)} {
		p.CalibrationNoisePenalty = penalty
		assert.Nil(t, ComputeFitting(15, samples, p).FittingLevel)
	}
}
