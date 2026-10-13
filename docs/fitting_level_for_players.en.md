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
4. Downweight outliers and pull weak or noisy evidence toward the official
   level, keeping each chart within its deviation limits.
5. After computing all estimates, balance their total using one common
   offset and reapply every chart's limits. Each eligible chart counts once,
   so popularity does not give a chart a larger share of the total.

Votes never directly override an individual chart. Official levels provide
the scale, scores provide observed performance, and votes provide an
experience-based comparison.

The published population has the same total fitting and official levels.
This does not require every chart to differ, or equal counts of increases
and decreases. Charts with insufficient evidence still display `—`.

Matching totals is a scale choice: we assume official levels are correct
on average for the published charts. Scores do not prove this assumption,
and balancing can hide a systematic average error in the official levels.

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

The default recalculation interval is six hours.
