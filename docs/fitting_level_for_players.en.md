# What is a fitting level?

[中文](./fitting_level_for_players.zh.md) · [Full formulas](./fitting_level.en.md)

A fitting level is an official-level anchor adjusted using player scores.
It can be slightly higher or lower than the official level. It is a
reference, not a replacement for personal experience or official ratings.

## How is it calculated?

1. Estimate player ability using the mean rating of their best 20 charts.
2. Compare similarly skilled players on charts with nearby official levels.
   Their best 20 performances often reflect strengths and repeated practice;
   expecting the same performance on every chart exaggerates difficulty.
3. Estimate this ordinary performance gap from other charts, then measure
   whether the target is harder or easier than its peers. The target chart
   is excluded from its own reference population.
4. Downweight outliers and pull weak evidence toward the official level.
   Subjective votes help validate direction and calibrate the size of the
   adjustment, preventing small differences from being exaggerated.

Votes never directly override an individual chart. Official levels provide
the scale, scores provide observed performance, and votes provide an
experience-based comparison.

## Reading the result

- Higher than official: relatively harder to achieve comparable performance.
- Lower than official: relatively easier to achieve comparable performance.
- `—`: insufficient eligible scores or comparable peer charts, not zero difficulty.
- Small difference: the evidence only supports a small adjustment. There is
  no universal threshold that proves a chart is misrated.

Perfect scores only show that the player reached the score ceiling; they
cannot identify exact difficulty and are excluded from calibrated fitting.
Practice, chart-style strengths and self-selection still affect results.
Disagreement with votes does not prove either estimate is correct.

The default recalculation interval is six hours. Historical replay shows
substantially less upward bias, but some charts still get the direction
wrong. See the [evaluation and counterexamples](./fitting_calibration_evaluation.zh.md).
