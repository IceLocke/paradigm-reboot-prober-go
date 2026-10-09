package fitting

import (
	"math"
	"sort"
)

// BalanceReport describes the unweighted published population. Charts without
// an estimate do not contribute to either total and remain nil.
type BalanceReport struct {
	Charts      int
	OfficialSum float64
	FittingSum  float64
	Offset      float64
}

// BalanceFittingTotals projects the complete population onto the official
// total while respecting each chart's deviation cap and valid level range.
// Each chart counts once, irrespective of popularity. The bounded projection
// minimizes squared changes to the independent estimates:
// delta_c = clamp(raw_delta_c - offset, lower_c, upper_c), sum(delta_c) = 0.
//
// This is an optional scale anchor, not an empirical conservation law: assume
// the official mean is appropriate for the published population and suppress
// shared estimator drift. It can hide a real population-wide rating bias;
// disable Params.BalanceTotal to inspect independent estimates. See the
// rationale and example in docs/fitting_level.en.md, section 4.0.
//
// Call once before persistence, never once per database batch. New pointers
// are assigned to map entries so copies of input Results are not mutated.
func BalanceFittingTotals(official map[int]float64, results map[int]Result, params Params) BalanceReport {
	type item struct {
		id                   int
		level, delta, lo, hi float64
	}
	ids := make([]int, 0, len(results))
	for id, result := range results {
		level, ok := official[id]
		if ok && result.FittingLevel != nil && isFinite(*result.FittingLevel) && isFinite(level) && level >= MinInferredLevel && level <= MaxInferredLevel {
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)
	items := make([]item, 0, len(ids))
	report := BalanceReport{Charts: len(ids)}
	left, right := math.Inf(1), math.Inf(-1)
	for _, id := range ids {
		level := official[id]
		delta := *results[id].FittingLevel - level
		lo, hi := MinInferredLevel-level, MaxInferredLevel-level
		if cap := effectiveMaxDeviation(params, level); cap > 0 {
			lo, hi = math.Max(lo, -cap), math.Min(hi, cap)
		}
		items = append(items, item{id, level, delta, lo, hi})
		left, right = math.Min(left, delta-hi), math.Max(right, delta-lo)
		report.OfficialSum += level
		report.FittingSum += level + delta
	}
	if len(items) == 0 || !params.CalibrationEnabled || !params.BalanceTotal {
		return report
	}
	// Zero is feasible for every chart because its official level is in the
	// valid range. Bisection remains valid with asymmetric bounds at the ends
	// of that range. The final deviation caps apply after balancing as well.
	for range 64 {
		mid := left + (right-left)/2
		var sum float64
		for _, c := range items {
			sum += math.Max(c.lo, math.Min(c.hi, c.delta-mid))
		}
		if sum > 0 {
			left = mid
		} else {
			right = mid
		}
	}
	report.Offset = left + (right-left)/2
	report.FittingSum = 0
	for _, c := range items {
		value := c.level + math.Max(c.lo, math.Min(c.hi, c.delta-report.Offset))
		result := results[c.id]
		result.FittingLevel = &value
		results[c.id] = result
		report.FittingSum += value
	}
	return report
}
