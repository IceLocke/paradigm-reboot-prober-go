package fitting

import (
	"context"
	"fmt"
	"math"
	"testing"

	"paradigm-reboot-prober-go/internal/model"

	"github.com/stretchr/testify/assert"
)

func calibrationSamples(level, residual float64) []Sample {
	// Identical players have an intentionally optimistic B20 ceiling. All
	// ordinary charts share this residual; harder/easier targets differ.
	out := make([]Sample, 30)
	for i := range out {
		out[i] = Sample{PlayerSkill: 160, PlayerRecords: 100,
			Score: simulateScore(level+residual, 160)}
	}
	return out
}

func TestCalibrationRemovesCeilingBiasAndPreservesDirection(t *testing.T) {
	p := defaultParams()
	p.CalibrationEnabled, p.CalibrationScale = true, 0.1
	c := &Calibration{}
	for id := 1; id <= 12; id++ {
		c.AddChart(id, 15, calibrationSamples(15, 0.5), p)
	}
	for _, tc := range []struct{ residual, sign float64 }{{0.5, 0}, {0.8, 1}, {0.2, -1}} {
		s := calibrationSamples(15, tc.residual)
		r := ComputeFitting(15, c.Apply(99, 15, s), p)
		if !assert.NotNil(t, r.FittingLevel) {
			continue
		}
		dev := *r.FittingLevel - 15
		if tc.sign == 0 {
			assert.InDelta(t, 0, dev, 0.001)
		} else {
			assert.Greater(t, dev*tc.sign, 0.01)
		}
		assert.Nil(t, s[0].LevelCorrection, "Apply must not mutate the input")
	}
}

func TestCalibrationExcludesTargetAndBalancesCharts(t *testing.T) {
	p := defaultParams()
	c := &Calibration{}
	for id := 1; id <= 10; id++ {
		c.AddChart(id, 15, calibrationSamples(15, 0.5), p)
	}
	before, ok := c.Correction(99, 15, 160)
	assert.True(t, ok)
	c.AddChart(99, 15, calibrationSamples(15, -0.5), p)
	after, ok := c.Correction(99, 15, 160)
	assert.True(t, ok)
	assert.Equal(t, before, after, "target cannot train its own baseline")
	popular := make([]Sample, 0, 30000)
	for i := 0; i < 1000; i++ {
		popular = append(popular, calibrationSamples(15, -0.5)...)
	}
	c.AddChart(100, 15, popular, p)
	after, ok = c.Correction(99, 15, 160)
	assert.True(t, ok)
	assert.InDelta(t, before, after, 1e-9, "one popular chart cannot dominate ten peers")
	_, ok = c.Correction(99, 10, 100)
	assert.False(t, ok, "do not extrapolate beyond peer support")
}

func TestCalibrationAbstainsWithoutPeersAndAtCeiling(t *testing.T) {
	p := defaultParams()
	p.CalibrationEnabled, p.CalibrationScale = true, 0.1
	c := &Calibration{}
	for id := 1; id <= 9; id++ {
		c.AddChart(id, 15, calibrationSamples(15, 0.5), p)
	}
	s := calibrationSamples(15, 0.5)
	assert.Nil(t, ComputeFitting(15, c.Apply(99, 15, s), p).FittingLevel)
	assert.Nil(t, ComputeFitting(15, s, p).FittingLevel)
	c.AddChart(10, 15, s, p)
	for i := range s {
		s[i].Score = 1010000
	}
	assert.Nil(t, ComputeFitting(15, c.Apply(99, 15, s), p).FittingLevel)
	for _, scale := range []float64{0, -1, 2, math.NaN(), math.Inf(1)} {
		p.CalibrationScale = scale
		assert.Nil(t, ComputeFitting(15, c.Apply(99, 15, calibrationSamples(15, 0.5)), p).FittingLevel)
	}
}

func TestCalibrationHonorsCancellation(t *testing.T) {
	r := NewRunner(setupTestDB(t), defaultParams(), RunnerConfig{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := r.collectCalibration(ctx, []chartRow{{ID: 1, Level: 15}}, nil)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestCalibratedRunnerMatchesDiagnosticAcrossBatches(t *testing.T) {
	db := setupTestDB(t)
	charts := seedCharts(t, db, 12)
	assert.NoError(t, db.Model(&model.Chart{}).Where("id IN ?", charts).Update("level", 15).Error)
	for u := 0; u < 12; u++ {
		username := fmt.Sprintf("cal_player_%d", u)
		seedUser(t, db, username)
		for _, id := range charts {
			seedRatedPlay(t, db, username, id, 16000, true)
		}
	}
	p := defaultParams()
	p.SkillTopK, p.CalibrationEnabled, p.CalibrationScale = 20, true, 0.1
	r := NewRunner(db, p, RunnerConfig{ChartBatchSize: 3, PlayerBatchSize: 4})
	ctx := context.Background()
	samples, err := r.LoadChartSamples(ctx, charts[0])
	assert.NoError(t, err)
	expected := ComputeFitting(15, samples, p)
	if !assert.NotNil(t, expected.FittingLevel) {
		return
	}
	var statsBefore int64
	assert.NoError(t, db.Model(&model.ChartStatistic{}).Count(&statsBefore).Error)
	assert.Zero(t, statsBefore, "diagnostic is read only")
	report, err := r.Run(ctx)
	assert.NoError(t, err)
	assert.Equal(t, len(charts), report.ChartsPublished)
	var chart model.Chart
	assert.NoError(t, db.First(&chart, charts[0]).Error)
	assert.Equal(t, expected.FittingLevel, chart.FittingLevel)
	// Changing batch boundaries cannot change the learned reference population.
	r.cfg.ChartBatchSize = 7
	second, err := r.LoadChartSamples(ctx, charts[0])
	assert.NoError(t, err)
	assert.Equal(t, expected, ComputeFitting(15, second, p))
	// Remove the peer population: an old published value must be cleared.
	assert.NoError(t, db.Where("chart_id IN ?", charts[1:]).Delete(&model.BestPlayRecord{}).Error)
	_, err = r.Run(ctx)
	assert.NoError(t, err)
	chart = model.Chart{}
	assert.NoError(t, db.First(&chart, charts[0]).Error)
	assert.Nil(t, chart.FittingLevel)
}
