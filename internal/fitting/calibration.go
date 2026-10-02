package fitting

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"
)

// Calibration learns the ordinary inverse-rating residual from other charts.
// Cells are balanced by chart, not popularity. The target chart is excluded
// from every prediction, so its own difficulty signal cannot cancel itself.
type Calibration struct {
	charts []calibrationChart
}

type calibrationChart struct {
	id    int
	level float64
	cells map[int]float64
}

const (
	calibrationGapWidth = 0.5 // level units, equivalent to five rating units
	calibrationWidth    = 0.5 // bandwidth on both official level and ability gap
	calibrationMinPeers = 10
)

// AddChart stores robust residuals for each ability-gap cell. Call once per
// chart in ascending ID order. Only a small summary survives the training pass.
func (c *Calibration) AddChart(id int, official float64, samples []Sample, params Params) {
	type cell struct{ values, weights []float64 }
	groups := make(map[int]*cell)
	for _, s := range samples {
		level, w, ok := weightedInference(official, s, params)
		if !ok || s.Score >= 1010000 {
			continue // perfect scores are censored, not exact difficulty observations
		}
		key := int(math.Floor((s.PlayerSkill/10 - official) / calibrationGapWidth))
		if groups[key] == nil {
			groups[key] = &cell{}
		}
		g := groups[key]
		g.values = append(g.values, level-official)
		g.weights = append(g.weights, w)
	}
	chart := calibrationChart{id: id, level: official, cells: make(map[int]float64)}
	for key, g := range groups {
		if len(g.values) >= 3 {
			chart.cells[key] = weightedMedian(g.values, g.weights)
		}
	}
	c.charts = append(c.charts, chart)
}

// Correction estimates the background residual for a target's ability cell.
// Each peer chart contributes one smoothed value and at most unit weight,
// regardless of its play count or the number of occupied ability cells.
func (c *Calibration) Correction(id int, official, skill float64) (float64, bool) {
	key := int(math.Floor((skill/10 - official) / calibrationGapWidth))
	values, weights := make([]float64, 0), make([]float64, 0)
	for _, chart := range c.charts {
		dl := (chart.level - official) / calibrationWidth
		if chart.id == id || math.Abs(dl) > 2 {
			continue
		}
		var sum, sumW, maxW float64
		// Sorted traversal makes floating-point results reproducible.
		for k := key - 2; k <= key+2; k++ {
			v, ok := chart.cells[k]
			if !ok {
				continue
			}
			dg := float64(k-key) * calibrationGapWidth / calibrationWidth
			w := math.Exp(-0.5 * (dl*dl + dg*dg))
			sum += w * v
			sumW += w
			maxW = math.Max(maxW, w)
		}
		if sumW > 0 {
			values = append(values, sum/sumW)
			weights = append(weights, maxW)
		}
	}
	if len(values) < calibrationMinPeers {
		return 0, false
	}
	return weightedMedian(values, weights), true
}

// Apply returns a copy carrying corrections; unsupported cells remain nil and
// are excluded by ComputeFitting in calibrated mode (never silently use B20).
func (c *Calibration) Apply(id int, official float64, samples []Sample) []Sample {
	out := append([]Sample(nil), samples...)
	type prediction struct {
		value float64
		ok    bool
	}
	cache := make(map[int]prediction)
	for i := range out {
		out[i].LevelCorrection = nil
		key := int(math.Floor((out[i].PlayerSkill/10 - official) / calibrationGapWidth))
		p, exists := cache[key]
		if !exists {
			p.value, p.ok = c.Correction(id, official, out[i].PlayerSkill)
			cache[key] = p
		}
		if p.ok {
			value := p.value
			out[i].LevelCorrection = &value
		}
	}
	return out
}

func (r *Runner) collectCalibration(ctx context.Context, charts []chartRow, skills map[string]PlayerSkill) (*Calibration, error) {
	c := &Calibration{}
	for start := 0; start < len(charts); start += r.cfg.ChartBatchSize {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		batch := charts[start:min(start+r.cfg.ChartBatchSize, len(charts))]
		ids := make([]int, len(batch))
		for i, chart := range batch {
			ids[i] = chart.ID
		}
		samples, err := r.fetchBestSamples(ctx, ids, skills)
		if err != nil {
			return nil, fmt.Errorf("calibration samples: %w", err)
		}
		for _, chart := range batch {
			c.AddChart(chart.ID, chart.Level, samples[chart.ID], r.params)
		}
		if r.cfg.BatchPause > 0 && start+len(batch) < len(charts) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(r.cfg.BatchPause):
			}
		}
	}
	return c, nil
}

// ChartAnalysis contains both independent sample statistics and the final
// result after population balancing. It never migrates or writes the database.
type ChartAnalysis struct {
	Samples []Sample
	Result  Result
	Balance BalanceReport
}

// AnalyzeChart replays the same complete population as Run, so the displayed
// target result includes exactly the same global offset.
func (r *Runner) AnalyzeChart(ctx context.Context, id int) (ChartAnalysis, error) {
	out := ChartAnalysis{}
	skills, err := r.collectPlayerSkills(ctx)
	if err != nil {
		return out, err
	}
	charts, err := r.fetchChartsSorted(ctx)
	if err != nil {
		return out, err
	}
	index := sort.Search(len(charts), func(i int) bool { return charts[i].ID >= id })
	if index == len(charts) || charts[index].ID != id {
		return out, fmt.Errorf("chart %d not found", id)
	}
	var calibration *Calibration
	if r.params.CalibrationEnabled {
		calibration, err = r.collectCalibration(ctx, charts, skills)
		if err != nil {
			return out, err
		}
	}

	computed, err := r.computeCharts(ctx, charts, skills, calibration, id)
	if err != nil {
		return out, err
	}
	out.Balance = balanceCharts(charts, computed.results, r.params)
	out.Samples, out.Result = computed.samples, computed.results[id]
	return out, nil
}
