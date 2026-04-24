# Fitting Level Calculation

*Available in: **English** · [中文](./fitting_level.zh.md)*

*Plain-language guide for players: [English](./fitting_level_for_players.en.md) · [中文](./fitting_level_for_players.zh.md)*

> **Scope.** This document specifies how the fitting-calculator microservice
> (`cmd/fitting`) derives each chart's `fitting_level` from the observed
> `best_play_records`. It is intentionally self-contained: readers do not need
> to consult the probe server's code (`cmd/server`) to reproduce the math.
>
> The probe server (the "查分器") **does not** compute fitting levels and does
> not read any config under `fitting.*` at runtime. See `AGENTS.md → 保持查分
> 器本体的单纯性` for the underlying design principle.

## 1. Problem statement

Each `charts` row stores:

- `level` — the **official** difficulty constant published by the game/chart
  authors, a real number of the form `{integer}.{tenth}` (e.g. `14.5`).
- `fitting_level` — a nullable refined estimate that we compute offline from
  player data.

The objective is: given a chart $c$ with official level $L_c$ and the
distribution of best scores $\{s_{p,c}\}$ from players $p \in P_c$, produce a
posterior point estimate $\hat{L}_c$ (`fitting_level`) that:

1. respects the **official level** as an informative prior;
2. adapts to the **observed score distribution**, robust to outliers and
   small-sample charts;
3. weights players by their own skill, so a chart estimated from players
   whose ability is near $L_c$ is more reliable;
4. tolerates heterogeneous player quality (few vs many records, central vs
   peripheral players).

## 2. Notation

| Symbol                   | Meaning                                                                                   |
|--------------------------|-------------------------------------------------------------------------------------------|
| $L_c$                    | Official chart level (float, from `charts.level`).                                         |
| $\hat{L}_c$              | Computed fitting level (float, written to `charts.fitting_level`).                         |
| $s_{p,c}$                | Best score of player $p$ on chart $c$ (integer, 0–1 010 000).                              |
| $r_{p,c}$                | Single-chart rating assigned to $(p,c)$ under the official level; see `pkg/rating/rating.go`. |
| $B_p$                    | Player $p$'s float **B50 mean rating**: mean of their top-$K$ single-chart ratings, $K=\min(|\text{best}_p|, 50)$. |
| $n_p$                    | Total number of best records belonging to player $p$.                                       |
| $\hat{\delta}_{p,c}$     | Level inferred from $(s_{p,c}, B_p)$; see §4.1.                                            |
| $w^{\text{prox}}_{p,c}$  | Proximity weight.                                                                         |
| $w^{\text{vol}}_p$       | Volume weight.                                                                            |
| $w^{\text{rob}}_{p,c}$   | Robustness (Tukey biweight) factor.                                                        |
| $w_{p,c}$                | Final composite weight $w^{\text{prox}}\cdot w^{\text{vol}}\cdot w^{\text{rob}}$.           |
| $N^{\text{eff}}_c$       | Kish effective sample size for chart $c$.                                                 |
| $\kappa$                 | Prior strength (Bayesian shrinkage coefficient), `config.fitting.prior_strength`.          |
| $\Delta_{\max}$          | Hard cap on $|\hat{L}_c - L_c|$, `config.fitting.max_deviation`.                           |
| $\sigma_{\text{prox}}$   | Proximity Gaussian bandwidth, in **rating units**, `config.fitting.proximity_sigma`.        |
| $V_{\text{full}}$        | Record count at which the volume weight saturates to 1, `config.fitting.volume_full_at`.   |
| $k$                      | Tukey biweight tuning constant, `config.fitting.tukey_k` (default 4.685).                   |
| $s_{\min}$               | Minimum score considered, `config.fitting.min_score`.                                      |

## 3. Rating formula (reference)

The rating for a single play is defined by the piecewise function
$\mathrm{Rating}(L, s)$ in `pkg/rating/rating.go`. Letting
$b = \lfloor\max(s, 1\,010\,000)\rfloor$,

$$
\mathrm{Rating}(L, s) =
\begin{cases}
10L + 7 + 3\left(\dfrac{s - 1\,009\,000}{1000}\right)^{1.35}, & s \ge 1\,009\,000,\\[8pt]
10\left(L + \dfrac{2(s - 1\,000\,000)}{30\,000}\right),       & 1\,000\,000 \le s < 1\,009\,000,\\[8pt]
B(s) + 10\left(L\left(\dfrac{s}{10^{6}}\right)^{1.5} - 0.9\right), & 0 \le s < 1\,000\,000,
\end{cases}
$$

where $B(s)$ is the bonus step function

$$
B(s) = 3\mathbf{1}\{s \ge 900\,000\} + \sum_{t\in\{930,950,970,980,990\}\times 10^3} \mathbf{1}\{s \ge t\},
$$

clamped to $\max(\mathrm{Rating}, 0)$. The persisted column
`play_records.rating` is $\lfloor 100\cdot\mathrm{Rating} + \varepsilon\rfloor$.

## 4. Algorithm

### 4.1 Per-sample inferred level

For each best record $(p, c)$ we invert $\mathrm{Rating}$ in $L$, treating
$B_p$ as a target. Because $\mathrm{Rating}$ is linear in $L$ within each of
its three branches, a closed-form inverse exists:

$$
\hat{\delta}_{p,c} =
\begin{cases}
\dfrac{B_p - 7 - 3\left((s - 1\,009\,000)/1000\right)^{1.35}}{10}, & s \ge 1\,009\,000,\\[8pt]
\dfrac{B_p}{10} - \dfrac{2(s - 1\,000\,000)}{30\,000},             & 1\,000\,000 \le s < 1\,009\,000,\\[8pt]
\dfrac{B_p - B(s) + 9}{10\,(s/10^{6})^{1.5}},                       & 0 < s < 1\,000\,000,
\end{cases}
$$

undefined at $s = 0$. We reject the sample if $\hat{\delta}_{p,c} \notin
[0.1, 20.0]$ — the usable level range of the game.

Intuitively $\hat{\delta}_{p,c}$ answers "what level *would* make this player's
observed score exactly match their typical B50 rating". If the chart is
actually easier than its official level, players systematically score above
their skill target, driving $\hat{\delta}_{p,c}$ below $L_c$.

### 4.2 Pre-weighting

**Proximity weight (asymmetric Gaussian with hard cutoff).** Players whose
skill $B_p$ is close to $10\cdot L_c$ play near the intended difficulty
band; their score distribution is most informative about $L_c$. Players far
from the band are either ignoring the chart (too hard for them) or are
overwhelmingly saturating it at SSS (too easy) — in both cases the inverter
$\operatorname{Inv}(\cdot)$ degenerates toward "echoing the player's own
skill" rather than reporting a chart signal, so such samples should carry
little weight.

A naive symmetric Gaussian weights both sides the same, which is almost
right — but not quite. For AP-saturated high-skill players the inversion is
especially degenerate (a clipped score $\ge$ AP boundary provides no upper
bound on how hard the chart might actually be for them), and empirically this
pushed estimates of mid-level charts upward. We therefore widen the penalty
on the over-skilled side only:

$$
\sigma_{\text{eff}}(\Delta_p) =
\begin{cases}
\sigma_{\text{prox}}, & \Delta_p := B_p - 10L_c \le 0,\\[3pt]
\alpha\cdot\sigma_{\text{prox}}, & \Delta_p > 0,
\end{cases}
\qquad
\alpha \in (0,1],\ \alpha = \text{high\_skill\_sigma\_ratio}\ (\text{default } 0.2).
$$

$$
w^{\text{prox}}_{p,c} = \exp\!\left(-\dfrac{\Delta_p^{2}}{2\,\sigma_{\text{eff}}(\Delta_p)^{2}}\right).
$$

The Gaussian only decays, so we also impose a **$2.5\sigma_{\text{eff}}$**
hard cutoff — samples beyond that band are dropped entirely. This is
critical for a robust Kish effective sample size $N^{\text{eff}}$: without a
hard cutoff, a large mass of tiny-weight over-skilled samples could still
inflate $\sum w$ enough to pass the `min_samples` gate.

The default $\sigma_{\text{prox}} = 18.5$ corresponds to $\pm 1.85$ level
units of "effective skill" on the under-skilled side and $\pm 0.37$ level
units on the over-skilled side, capturing the realistic audience band of a
chart while heavily discounting over-skilled dabblers.

**Where does $\alpha = 0.2$ come from?** An initial version used a symmetric
Gaussian ($\alpha = 1$) and the middle-level bands (lv13–lv15.5) came out
systematically $+0.5$ to $+0.8$ above the official level. Cross-sweeping on
the production DB (1187 charts, 441\,069 best-play records) yields the
following summary:

| $\alpha$ | global $\mathrm{avg}(\hat\delta)$ | lv13 bias | lv15 bias | lv16 bias | worst band\|dev\| (n$\ge$5) |
|------|--------|----------|----------|----------|----------|
| 0.15 | $-0.020$ | $-0.186$ | $+0.107$ | $-0.078$ | $0.222$  |
| 0.17 | $+0.007$ | $-0.150$ | $+0.142$ | $-0.068$ | $0.206$  |
| **0.20** | **$+0.050$** | **$-0.089$** | **$+0.194$** | **$-0.054$** | **$0.194$** |
| 0.22 | $+0.079$ | $-0.052$ | $+0.228$ | $-0.045$ | $0.228$  |
| 0.30 (old) | $+0.196$ | $+0.134$ | $+0.350$ | $-0.018$ | $0.350$  |

Tightening $\alpha$ below $0.2$ pushes lv11–lv13 too negative; loosening
above $0.2$ lets the lv15+ positive bias creep back. The two directions
meet at $\alpha = 0.20$, which minimises the **worst band-wise $|dev|$**
(0.194). The global bias is closest to zero at $\alpha \approx 0.17$, but
that comes at the cost of $-0.15$ to $-0.21$ in lv11–lv13. We optimise for
the worst band (rather than the global average) because every band is user-
facing.

**σ fine-tune.** With $\alpha = 0.20$ fixed, a follow-up 2-D grid search
over $\sigma_{\text{prox}}$ and `min_samples` on the production database
(1187 charts) yielded:

| $\sigma_{\text{prox}}$ | min_samples | n_pub | global avg | lv11 bias | lv15 bias | worst band\|dev\| |
|-------|-------|--------|----------|-----------|-----------|-------------------|
| 20.0  | 8     | 417    | $+0.050$ | $-0.192$  | $+0.194$  | $0.194$           |
| 19.0  | 5     | 441    | $+0.037$ | $-0.168$  | $+0.177$  | $0.177$           |
| **18.5** | **5** | **437** | **$+0.032$** | **$-0.170$** | **$+0.169$** | **$0.170$** |
| 18.0  | 5     | 436    | $+0.025$ | $-0.194$  | $+0.160$  | $0.194$           |
| 15.0  | 5     | 429    | $-0.006$ | $-0.198$  | $+0.116$  | $0.198$           |

Dropping $\sigma$ from 20 to 18.5 further compresses the lv15+ positive
bias without the reversal seen at $\sigma \le 18$, where lv11 snaps back to
$-0.19$. Relaxing `min_samples` 8 $\to$ 5 adds $\approx 24$ published
charts; MAD and bias are essentially unchanged because §4.4’s
`DeviationPenalty` ($\lambda = 2$) already pulls small-sample charts toward
the official level.

**Volume weight.** Players with very few records have noisier $B_p$
estimates. We apply a linear ramp that saturates at $V_{\text{full}} = 50$
records:

$$
w^{\text{vol}}_p = \min\!\left(1, \dfrac{n_p}{V_{\text{full}}}\right).
$$

**Score-quality weight (opt-in, disabled by default).** A sample from a
player who only barely passed a chart (score just over 1{,}000{,}000) and a
sample from the same player comfortably in the community "high-score"
band ($\ge 1{,}009{,}000$) do not carry equal information. We map the raw
score to an extra weight factor $w^{\text{score}}_{p,c} \in [0, 1]$ via a
three-segment piecewise-linear ramp:

$$
w^{\text{score}}_{p,c} = \begin{cases}
0, & s < s_{\text{floor}},\\[2pt]
w_{\text{good}} \cdot \dfrac{s - s_{\text{floor}}}{s_{\text{good}} - s_{\text{floor}}}, & s_{\text{floor}} \le s < s_{\text{good}},\\[10pt]
w_{\text{good}} + (1 - w_{\text{good}}) \cdot \dfrac{s - s_{\text{good}}}{s_{\text{full}} - s_{\text{good}}}, & s_{\text{good}} \le s < s_{\text{full}},\\[10pt]
1, & s \ge s_{\text{full}}.
\end{cases}
$$

Recommended anchors (if you opt in): $s_{\text{floor}} = 1{,}000{,}000$
(the business-defined "didn't really pass" threshold), $s_{\text{good}} =
1{,}007{,}500$ (the "can play" threshold), $s_{\text{full}} = 1{,}009{,}000$
(the "high-score" threshold), $w_{\text{good}} = 0.6$. **All four knobs
default to `0`**, in which case the factor degenerates to $1$ and this
sub-section is equivalent to disabled. Any non-monotone or partial
configuration (e.g. setting only some of the four) also degenerates to 1.

**What problem does this solve?** About 24% of the raw best-record pool
sits below 1{,}000{,}000 and another 40% sits in the 1{,}000{,}000–1{,}007{,}500
"just scraped by" band. Giving every such sample the same weight as a
stable high-score run introduces survivorship bias in the inferred
$\hat{\delta}$ — we only see the players who “barely made it”, not the
same-skill players who failed and never produced a best record. Damping
these samples at the pre-weight stage discounts that bias.

**Why is it disabled by default?** Under the previous $\alpha = 0.3$
default, enabling this factor ($w_{\text{good}} = 0.6$) visibly improved
the global average — global $\mathrm{avg}(\hat{\delta})$ fell from $+0.20$
to $+0.08$ ($-59\%$), and the lv14–lv15 bands recovered. However, with
the new $\alpha = 0.2$ default, $\alpha$ alone already compresses the mid/
high-level positive bias (lv15 drops from $+0.35$ to $+0.19$); adding
$w^{\text{score}}$ on top pushes lv11–lv13 into a much larger negative
overcorrection (roughly $-0.3$). Production sweeps showed the single
knob $\alpha = 0.2$ achieves worst band $|dev| = 0.194$, beating
$(\alpha = 0.25, w_{\text{good}} = 0.9)$ at $0.29$. Both levers point in
the same direction ("reduce over-trustful samples") and, on this DB,
stacking them over-shoots.

**Interaction with the InverseLevel math (why it is off by default on
this DB).** Under the rating formula, for a fixed $B_p$, an AP sample
(score = 1{,}010{,}000) inverts to $\hat{L} = B_p/10 - 1$, while a score =
1{,}000{,}500 sample inverts to $\hat{L} \approx B_p/10 - 0.033$ — that is,
**high-score samples inherently infer a lower level than low-score samples
do, for the same $B_p$**. Up-weighting high-score samples therefore
nudges the whole $\hat{L}_c$ distribution down, which is exactly why with
$\alpha = 0.2$ already pulling lv15 from $+0.35$ to $+0.19$, enabling
$w^{\text{score}}$ on top sends lv11–lv13 to around $-0.3$. The feature is
still valuable in scenarios where $\alpha$ needs to be loose (e.g. a
regional sub-DB where under-skilled samples dominate); code, tests, docs,
and the config framework are all kept so it can be enabled explicitly. To
opt in, populate all four knobs together (e.g. floor=$1{,}000{,}000$,
good=$1{,}007{,}500$, full=$1{,}009{,}000$, good\_weight=$0.6$).

**Combined pre-weight:** $\tilde{w}_{p,c} = w^{\text{prox}}_{p,c} \cdot
w^{\text{vol}}_p \cdot w^{\text{score}}_{p,c}$.

### 4.3 Robust trimming (Tukey biweight)

Let $\tilde{m}_c$ and $\mathrm{MAD}_c$ be the *weighted* median and *weighted*
median absolute deviation of $\{\hat{\delta}_{p,c}\}$ under pre-weights
$\{\tilde{w}_{p,c}\}$ (ties broken by ascending $\hat{\delta}$):

$$
\tilde{m}_c = \operatorname*{wmedian}_{p \in P_c}\hat{\delta}_{p,c};
\qquad
\mathrm{MAD}_c = \operatorname*{wmedian}_{p \in P_c}\bigl|\hat{\delta}_{p,c} - \tilde{m}_c\bigr|.
$$

For each sample compute the scaled residual

$$
u_{p,c} = \dfrac{\hat{\delta}_{p,c} - \tilde{m}_c}{k \cdot \max(\mathrm{MAD}_c, \epsilon)},
$$

with $\epsilon$ a safety floor ($1\%$ of $|L_c|+1$ in the implementation, used
when the entire sample is atypically concentrated). Apply the Tukey biweight

$$
w^{\text{rob}}_{p,c} = \begin{cases}
(1 - u_{p,c}^{2})^{2}, & |u_{p,c}| < 1,\\[4pt]
0,                     & |u_{p,c}| \ge 1.
\end{cases}
$$

Samples whose inferred level exceeds $k\cdot\mathrm{MAD}_c$ from the weighted
median contribute *zero* to the final estimate, directly addressing the user
concern about "records 可信度 / 偏中心" — samples far from the consensus are
suppressed.

### 4.4 Aggregation

Define the final composite weight $w_{p,c} = \tilde{w}_{p,c}\cdot
w^{\text{rob}}_{p,c}$.

**Weighted mean** (pre-shrinkage estimate):

$$
\mu_c = \dfrac{\sum_p w_{p,c}\,\hat{\delta}_{p,c}}{\sum_p w_{p,c}}.
$$

**Kish effective sample size** (how many "ideal" samples the weighting
scheme is equivalent to):

$$
N^{\text{eff}}_c = \dfrac{\left(\sum_p w_{p,c}\right)^{2}}{\sum_p w_{p,c}^{2}}.
$$

We abstain from publishing a fitting level when $N^{\text{eff}}_c <
\text{min\_samples}$ (default 8): the column `charts.fitting_level` is
written as `NULL`, while `chart_statistics` still records the diagnostic
fields for review.

### 4.5 Bayesian shrinkage toward the official level (with deviation penalty)

Treat $L_c$ as a Gaussian prior with strength $\kappa$, and $\mu_c$ as the
likelihood mean with precision $N^{\text{eff}}_c$. The posterior mean is the
precision-weighted combination

$$
\hat{L}_c = \dfrac{N^{\text{eff}}_c\,\mu_c + \kappa_{\text{eff}}\,L_c}{N^{\text{eff}}_c + \kappa_{\text{eff}}}.
$$

where $\kappa_{\text{eff}}$ is a deviation-sensitive dynamic prior strength.
Plugging a plain $\kappa$ into the formula has a failure mode: on
small-sample charts, a few outlier players can drag $\mu_c$ several levels
away from the official value, while there simply isn't enough data to
support that confidence. We therefore introduce a multiplicative deviation
penalty:

$$
\kappa_{\text{eff}} = \kappa\cdot\left(1 + \lambda\,(\mu_c - L_c)^2\cdot\dfrac{n_{\text{ref}}}{N^{\text{eff}}_c}\right),
\qquad n_{\text{ref}} = 2\cdot\text{min\_samples}.
$$

When the deviation is zero or when $N^{\text{eff}}_c \gg n_{\text{ref}}$ the
boost degenerates to 1 (no runtime cost); when the deviation is large and
the effective sample is small, $\kappa_{\text{eff}}$ scales quadratically
with the gap and inversely with $N^{\text{eff}}_c$ — matching the intuition
that the further a chart drifts from its official value, the more evidence
we should demand before publishing that drift. The default is $\lambda = 2$.
Setting $\lambda = 0$ reverts to the old static-$\kappa$ behaviour; all
pre-existing tests pin $\lambda = 0$ for regression coverage.

Equivalent reading: the "confidence" of the official level is
$\kappa_{\text{eff}}$ pseudo-samples — a chart with $N^{\text{eff}}_c \gg
\kappa_{\text{eff}}$ essentially follows the data, a chart with
$N^{\text{eff}}_c \ll \kappa_{\text{eff}}$ stays near the official value,
and larger deviations tilt the balance further toward the official side.

### 4.6 Deviation cap (level-dependent log-linear ramp)

As a final safety net against model mis-specification we enforce a hard
clip on the post-shrinkage estimate:

$$
\hat{L}_c \leftarrow L_c + \operatorname{clip}\!\bigl(\hat{L}_c - L_c,\ -\Delta(L_c),\ \Delta(L_c)\bigr).
$$

**Why level-dependent?** Reboot's official level axis is *perceptually*
roughly logarithmic — the gameplay gap from lv12 to lv13 is much smaller
than the gap from lv16 to lv17. A tight cap at the low end protects
against fitting values leaping across "real" difficulty tiers; a wider cap
at the high end is needed because the official spacing itself is coarser
and the algorithm should have more room to discover surprises. We therefore
interpolate the cap **log-linearly** between two anchor points:

$$
\Delta(L) =
\begin{cases}
\Delta_{\min}, & L \le L_{\text{low}},\\[4pt]
\Delta_{\min}\cdot\left(\dfrac{\Delta_{\max}}{\Delta_{\min}}\right)^{t(L)}, & L_{\text{low}} < L < L_{\text{high}},\\[8pt]
\Delta_{\max}, & L \ge L_{\text{high}}.
\end{cases}
\qquad t(L) = \dfrac{L - L_{\text{low}}}{L_{\text{high}} - L_{\text{low}}}.
$$

Defaults: $\Delta_{\min} = 0.6$, $\Delta_{\max} = 1.5$, $L_{\text{low}} =
12.0$, $L_{\text{high}} = 17.0$. The midpoint $L = 14.5$ sits at the
geometric mean $\Delta(14.5) = \sqrt{\Delta_{\min}\cdot\Delta_{\max}}
\approx 0.949$, consistent with the "monotone, no-jumps" design goal.

**Degeneration rule.** When $\Delta_{\min} \le 0$ or the anchors
$L_{\text{low}}$/$L_{\text{high}}$ are misconfigured (see
`internal/fitting/calculator.go:effectiveMaxDeviation`), the implementation
silently falls back to a **flat** cap $\Delta(L) \equiv \Delta_{\max}$; the
startup validator rejects the most common misconfigurations (see
`AGENTS.md`). The ramp only controls the **width** of the cap, never its
**symmetry** — the same $\Delta(L_c)$ is used on both sides.

**A safety net, not a corrector.** On the current dataset the vast
majority of fitted values already sit inside the trapezoidal window, so
$\Delta(L)$ fires only very rarely — it is a guardrail against the
occasional catastrophic outlier, not a lever for pulling mid-level charts
closer to their official values. The real levers for mid-level bias are
$\alpha$ and $\kappa$ — see §4.2 and §4.5.

## 5. Summary pipeline

```
for each chart c with official level L_c:
    samples := { (p, s_{p,c}) : p ∈ P_c, s_{p,c} ≥ s_min }
    for each sample:
        δ̂ := InverseRating(s_{p,c}, B_p)                      # §4.1
        if δ̂ ∉ [0.1, 20.0]: drop
        diff  := B_p - 10·L_c                                   # §4.2
        σ_eff := (diff > 0 ? α·σ_prox : σ_prox)
        if |diff| > 2.5·σ_eff: drop                              #  ← hard cutoff
        w_prox := exp(-diff² / (2·σ_eff²))
        w_vol  := min(1, n_p / V_full)
        w_pre  := w_prox * w_vol
    m_c  := weighted_median(δ̂; w_pre)                          # §4.3
    MAD  := weighted_median(|δ̂ - m_c|; w_pre)
    for each sample:
        u := (δ̂ - m_c) / (k · max(MAD, ε))
        w_rob := (1 - u²)²  if |u| < 1 else 0
        w     := w_pre * w_rob
    μ_c     := Σ w·δ̂ / Σ w                                    # §4.4
    N_eff_c := (Σ w)² / Σ w²
    if N_eff_c < min_samples:
        publish FittingLevel = NULL; keep stats; continue
    dev     := μ_c - L_c                                       # §4.5
    n_ref   := 2 · min_samples
    κ_eff   := κ · (1 + λ·dev²·n_ref / N_eff_c)
    L̂_c     := (N_eff_c·μ_c + κ_eff·L_c) / (N_eff_c + κ_eff)
    Δ       := effectiveMaxDeviation(L_c)                      # §4.6
    L̂_c     := L_c + clip(L̂_c - L_c, -Δ, Δ)
    UPDATE charts SET fitting_level = L̂_c WHERE id = c
    UPSERT chart_statistics (c, sample_count, N_eff_c, μ_c, m_c, σ_c, MAD, L̂_c, L_c, now)
```

## 6. Hyperparameters (from `config.yaml`)

| Key                             | Symbol                 | Default   | Role                                                                                   |
|---------------------------------|------------------------|-----------|----------------------------------------------------------------------------------------|
| `fitting.enabled`               | —                      | `true`    | Master switch for the microservice.                                                    |
| `fitting.interval`              | —                      | `6h`      | Ticker period (Go duration).                                                           |
| `fitting.min_samples`           | min_samples            | `5.0`     | $N^{\text{eff}}$ below this → abstain.                                                 |
| `fitting.min_player_records`    | —                      | `20`      | Exclude players with fewer best records.                                               |
| `fitting.proximity_sigma`       | $\sigma_{\text{prox}}$ | `18.5`    | Gaussian bandwidth around $10L_c$.                                                     |
| `fitting.high_skill_sigma_ratio`| $\alpha$               | `0.2`     | $\sigma$ multiplier on the over-skilled side (asymmetric Gaussian). `1.0` = symmetric, smaller = stronger discount on over-skilled players. Samples outside $2.5\cdot\sigma$ are dropped. `0.2` minimises worst band $|dev|$ on prod. |
| `fitting.volume_full_at`        | $V_{\text{full}}$      | `50`      | Volume weight saturation point.                                                        |
| `fitting.prior_strength`        | $\kappa$               | `5.0`     | Baseline shrinkage strength toward $L_c$.                                              |
| `fitting.deviation_penalty`     | $\lambda$              | `2.0`     | Deviation penalty; sets $\kappa_{\text{eff}} = \kappa(1+\lambda\cdot\text{dev}^2\cdot n_{\text{ref}}/N^{\text{eff}})$. `0` reverts to static $\kappa$. |
| `fitting.max_deviation`         | $\Delta_{\max}$        | `1.5`     | Cap at the high-level end ($\ge L_{\text{high}}$); also the flat cap when the ramp is disabled. |
| `fitting.max_deviation_low`     | $\Delta_{\min}$        | `0.6`     | Cap at the low-level end ($\le L_{\text{low}}$); set to `0` to disable the ramp and fall back to flat $\Delta_{\max}$. |
| `fitting.max_deviation_low_at`  | $L_{\text{low}}$       | `12.0`    | Anchor where the cap equals $\Delta_{\min}$; must be less than $L_{\text{high}}$.      |
| `fitting.max_deviation_high_at` | $L_{\text{high}}$      | `17.0`    | Anchor where the cap equals $\Delta_{\max}$; in between, $\Delta(L) = \Delta_{\min}\cdot(\Delta_{\max}/\Delta_{\min})^t$. |
| `fitting.min_score`             | $s_{\min}$             | `500000`  | Minimum score admitted.                                                                |
| `fitting.score_floor_at`        | $s_{\text{floor}}$     | `0`       | Score-quality weight (opt-in, disabled by default). Samples below this score get zero score-quality weight (business "didn't really pass" line); set to `0` to disable the score-quality weight. |
| `fitting.score_good_at`         | $s_{\text{good}}$      | `0`       | Anchor at which the score-quality weight reaches $w_{\text{good}}$ ("can play" threshold); recommended `1007500` when enabled. |
| `fitting.score_full_at`         | $s_{\text{full}}$      | `0`       | Anchor at which the score-quality weight saturates to 1.0 ("high-score" threshold); recommended `1009000` when enabled. |
| `fitting.score_good_weight`     | $w_{\text{good}}$      | `0`       | Weight at $s_{\text{good}}$; must lie in $(0, 1)$ when enabled (typical `0.6`).        |
| `fitting.tukey_k`               | $k$                    | `4.685`   | Biweight tuning constant.                                                              |
| `fitting.chart_batch_size`      | —                      | `200`     | Charts processed per DB batch.                                                         |
| `fitting.player_batch_size`     | —                      | `500`     | Users fetched per page.                                                                |
| `fitting.batch_pause`           | —                      | `50ms`    | Sleep between batches (DB load relief).                                                |

## 7. Database impact and schema

The calculator writes:

1. `charts.fitting_level` (`double precision`, nullable) — the published
   estimate $\hat{L}_c$ or `NULL` when abstaining.
2. `chart_statistics` (new table, owned by `cmd/fitting`) — one row per
   chart, keyed on `chart_id`, capturing each diagnostic from the pipeline:
   `official_level`, `fitting_level`, `sample_count`, `effective_sample_size`,
   `weighted_mean`, `weighted_median`, `std_dev`, `mad`, `last_computed_at`,
   plus the standard `BaseModel` timestamps. No HTTP endpoint is wired; the
   table is for internal analysis only.

To minimize impact on the live probe service:

- Player skills are built with **keyset pagination** over distinct
  `username`s (batch size `player_batch_size`), never OFFSET-scanning.
- Charts are processed in fixed-size batches (`chart_batch_size`); a short
  `batch_pause` separates batches.
- Each chart's write is its own short transaction — long locks never form.
- The probe server's caches are not invalidated; they refresh naturally via
  TTL after the next upload touches the user.

## 8. Operational guide

```bash
# Continuous mode (default, honours fitting.interval)
go run ./cmd/fitting -config config/config.yaml

# One-shot (useful for cron, debugging, CI smoke tests)
go run ./cmd/fitting --once -config config/config.yaml

# Read-only diagnostic for one chart (does not write the DB)
go run ./cmd/fitting analyze -chart 870 -config config/config.yaml
```

The binary exits cleanly on `SIGINT` / `SIGTERM`. In continuous mode a
transient DB error during one pass is logged but does **not** kill the loop;
the next tick retries.

### Docker / docker-compose

The project's `docker-compose.yaml` ships a `fitting` service behind the
`fitting` Compose profile:

```bash
# Start db + app + fitting together
docker compose --profile fitting up -d

# Or enable permanently (recommended for production)
export COMPOSE_PROFILES=fitting
docker compose up -d

# Tail fitting logs
docker compose --profile fitting logs -f fitting
```

The multi-stage Docker image builds both the `server` and `fitting` binaries;
the `fitting` service selects the fitting entrypoint via
`command: ["./fitting"]`.

#### Running as a true scheduled job (external scheduling)

If you prefer external scheduling (host cron / systemd timer / k8s CronJob)
over the built-in ticker, adapt the `fitting` service in
`docker-compose.yaml`:

```yaml
fitting:
  ...
  command: ["./fitting", "-once"]
  restart: "no"
  profiles: []    # remove profile so `docker compose run` can reach it
```

Then schedule it externally:

```cron
# Run every 6 hours
0 */6 * * * cd /path/to/project && docker compose run --rm fitting >> /var/log/fitting.log 2>&1
```

### Production deployment recommendations

- Run a dedicated replica/process with resource limits separate from the
  main probe server.
- Point `config.database.*` at the primary DB (reads + writes to
  `charts.fitting_level` and `chart_statistics`).
- Keep `fitting.interval` at $\ge 6$ hours in steady state; lower it
  temporarily when onboarding a new batch of charts.
- Monitor `chart_statistics.last_computed_at` for freshness.

## 9. Known limitations

1. **Closed ecosystem.** The player-skill target $B_p$ is derived from the
   same ratings that the official level generates. A systematic bias in the
   official levels propagates weakly into $B_p$. Mitigation: proximity
   weighting constrains the "band" of players contributing, so the bias is
   at most second-order; extensive outlier trimming further suppresses it.
2. **No temporal decay.** Scores are treated equally regardless of
   `record_time`. If the game meta shifts (e.g. judgement changes), old
   records could anchor the estimate against the new reality. Adding a
   recency kernel is straightforward future work.
3. **Chart additions.** Newly added charts with very few plays receive
   `fitting_level = NULL` by design. Abstention is the correct behaviour
   until $N^{\text{eff}}$ crosses `min_samples`.
