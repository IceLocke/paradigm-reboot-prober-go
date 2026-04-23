# What is `fitting_level`? — A guide for players

*Available in: [中文](./fitting_level_for_players.zh.md) · **English***

*Technical doc (with formulas): [English](./fitting_level.en.md) · [中文](./fitting_level.zh.md)*

> This document contains **no formulas and assumes no math background**.
> Its goal is to help players understand what the new `fitting_level`
> field in the probe service means, how it is computed, and how to read
> it when it disagrees with the official level.
>
> For the full math and parameter tables, see
> [`docs/fitting_level.en.md`](./fitting_level.en.md).

## TL;DR

- **`fitting_level` is a reference value** — an estimate of a chart's true
  difficulty, inferred from the actual scores of all probed players. It
  is **not the official level, and is not meant to replace it**.
- **Within ±0.3 of the official level** means official and community feel
  agree; a bigger gap is a signal worth noting.
- **Shown as `—` or blank**: not enough data yet. The system **prefers
  silence over a low-confidence number**.
- **Mid-level charts (lv13–15) often show the biggest gaps**, and that is
  normal — mid-level is intrinsically the hardest band to pin down, not
  evidence that the algorithm is wrong.
- It **drifts slowly over time** (recomputed every few hours by default),
  but multiple mechanisms prevent jitter.

## Why compute it at all?

The official level is assigned at design time, based on the designer's
intent and the game's internal difficulty scale. That is often accurate,
but it can drift in practice:

- A chart may ship **underrated** and feel much harder than the label.
- A chart may ship **overrated** and turn out to be a breeze.
- Rating standards may have shifted subtly between game versions.
- Niche chart styles can be lopsided for different player types (a sight-
  reading-heavy chart may crush read-forward players at "the same level").

The official team usually corrects these over time, but the probe
service can offer a live, community-driven reference in the meantime —
helping players calibrate their own sense of a chart's real difficulty.

## What is the probe service actually doing?

Skipping the math, the probe service does exactly four things:

1. **Weight by player skill.** Players whose skill sits close to the
   chart's difficulty give the most informative scores; players way
   above the chart (AP'ing it casually) or way below the chart (can't
   clear it) are downweighted.
2. **Weight by score tier.** A sample where the player only just scraped
   past 1{,}000{,}000 carries less information than a sample where the
   player sits comfortably in the "high-score" band. The probe service
   nudges the former down and keeps the latter at full strength,
   reducing the survivorship bias that comes from counting "barely
   passed" attempts while never seeing the same-skill players who
   didn't pass at all.
3. **Strip out suspicious samples.** A handful of scores that clearly
   deviate from the group (in either direction) are downweighted or
   discarded, so one or two top players — or one or two disastrous
   attempts — cannot swing the whole conclusion.
4. **The fewer the samples, the more we trust the official value.**
   When data is thin, the fitted value is pulled back toward the
   official level; only with enough evidence is it allowed to drift
   noticeably away.

## Frequently asked questions

### Q: Why is the fitting level sometimes shown as "—" or missing?

**A: Because there isn't enough confident data for that chart yet, so
the system chooses to stay silent rather than publish a shaky number.**

The probe service would rather say "I don't know yet" than publish a
low-confidence estimate. To publish a fitting level, a chart has to
meet a quality bar:

- Enough players at the relevant skill band have played it (after all
  the weighting, the "effective sample size" is ≥ 8).
- No single player's score is allowed to dominate the conclusion
  (multiple layers of weighting spread the influence out).

Cold charts, newly released charts, and quirky charts that only a
handful of top players have cleared often fail to meet the bar. That's
not a bug — it's the system's honest way of saying "not confident yet."
As more players touch the chart in future versions, a number will
eventually appear.

### Q: Why do mid-level charts (lv13–15) often show the biggest gaps?

**A: Because mid-level charts are intrinsically the hardest to pin down.**

Two reasons:

1. **The player base on a mid-level chart spans a huge skill range.**
   A lv14 chart is touched by lv12–13 players who grind it out,
   and also by lv16+ players who AP it casually. These two groups
   perceive the "real difficulty" very differently, so the fitted
   value is pulled around by whoever happens to be playing.

2. **The official scale is denser in the middle.** The gameplay gap
   between lv13 and lv14 is much smaller than between lv16 and lv17 —
   the game's difficulty ruler becomes coarser as you go up. So a
   small estimation error on a mid-level chart *looks* like a full-
   level miss, even though in absolute terms the error isn't that large.

The algorithm already does several things specifically for mid-levels
(for example, it discounts scores from players who are far above the
chart's level), but it **cannot be perfect**. A 0.3–0.5 gap between
fitting and official on a mid-level chart is well within normal.

### Q: Is the fitting level "more accurate" than the official level?

**A: No — it's a *different* lens, not necessarily a better one.**

- The **official level** encodes *designer intent* and the game's
  internal difficulty framework.
- The **fitting level** encodes *how the community actually played it*.

Both are useful:

- If you want to **train yourself**, the official level matches the
  game's intended progression path — stick with it.
- If you want an **objective second opinion** on how hard a chart is
  likely to feel for you, the fitting level is a good reference signal.
- If the two numbers disagree noticeably, that's a cue to check
  community discussion for "this chart is overrated / underrated"
  commentary.

### Q: Will a chart's fitting level change as I grind?

**A: Yes, and that's by design.**

The probe service re-runs the calculation periodically (every few hours
by default). Whenever the overall play data shifts — more new players
touching a chart, a wave of new high scores, etc. — the chart's
fitting level can adjust accordingly.

But the system has deliberate mechanisms to **avoid jitter**:

- The official level acts as an "anchor" that pulls the fitting value
  back when samples are thin.
- Old and new samples contribute together; a single update is unlikely
  to swing the estimate.
- Charts showing a large deviation get extra scrutiny — the algorithm
  *demands* more evidence before allowing a bigger move.

So what you normally see is **small, gradual drift**. You won't see a
chart bounce from lv14 to lv16 overnight.

### Q: Why does the fitting level sometimes go *above* the official, and sometimes *below*?

**A: Both directions are legitimate.**

- **Above official**: players tend to struggle on the chart — scores
  skew low, and the inverter reads that as "this chart is harder than
  its label."
- **Below official**: scores skew high (everyone finds it easier than
  expected), or "only top players even try it" selection bias is
  pushing things down. The algorithm partially corrects for the
  selection bias but can't eliminate it entirely.

## Want the math?

If you're curious about the specifics — **how exactly the weighting
works, how outliers are rejected, why σ is asymmetric, why the cap
varies with level** — head to the technical doc:

- [Chinese technical doc](./fitting_level.zh.md)
- [English technical doc](./fitting_level.en.md)

They have the full formulas, parameter tables, pipeline pseudocode,
and design discussion.
