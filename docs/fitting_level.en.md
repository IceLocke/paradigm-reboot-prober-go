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
| $B_p$                    | Player $p$'s float **top-K mean rating**: mean of their top-$K$ single-chart ratings, $K=\min(|\text{best}_p|, \texttt{skill\_top\_k})$, default 20; $K\ge1$. |
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
$\mathrm{Rating}(L, s)$ in `pkg/rating/rating.go`. First cap the score at
$s\le1\,010\,000$, then compute:

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

### 4.0 Current default: chart-balanced residual calibration

The defaults are `skill_top_k: 20`, `calibration_enabled: true`,
`calibration_noise_penalty: 1`, and
`balance_total: true`. A player's top-K mean is a performance ceiling,
not an unbiased target for every best record. Treating it as such explains
much of the old upward bias. Section 4.1 now supplies a **raw observation**;
the following correction precedes section 4.3.

Let $I(s,B)$ be the inverse rating and $g=B/10-L_c$ the ability gap. For each
chart, bin samples by $k=\lfloor g/0.5\rfloor$. Using exactly the preweights
in section 4.2, compute the weighted median $m_{c,k}$ of $I(s,B)-L_c$ in
each cell with at least three eligible records. Perfect scores
($s\ge1\,010\,000$) are censored observations and are excluded from both
calibration training and calibrated inference.

For a target chart $c$ and cell $k$, exclude the entire target chart. Only
peer cells with $|L_j-L_c|\le1$ and $|k'-k|\le2$ contribute, with weights

$$
a_{j,k'}=\exp\left[-\frac12\left(\left(\frac{L_j-L_c}{0.5}\right)^2+(k'-k)^2\right)\right].
$$

First combine each peer chart into one value
$v_j=\sum_{k'}a_{j,k'}m_{j,k'}/\sum_{k'}a_{j,k'}$, with weight
$u_j=\max_{k'}a_{j,k'}$. The weighted median of these chart values is the
background residual $b_{c,k}$. Chart balancing prevents popular charts or
charts occupying many ability cells from dominating the reference.
Require at least **ten other charts**; unsupported target cells abstain,
without silently reverting to the uncalibrated estimator.

Feed $\widetilde\delta_{p,c}=I(s_{p,c},B_p)-b_{c,k(p,c)}$ into the existing
median/MAD, Tukey, effective sample size and prior shrinkage in
sections 4.3–4.5. Let $r_c$ be the post-shrinkage deviation and
$v_c=\sigma_c^2/N_c^{\mathrm{eff}}$ its approximate sampling variance. Before
the final cap, attenuate weak signals:

$$
d_c^0=\operatorname{clip}\left(r_c\frac{r_c^2}{r_c^2+\tau v_c},\ell_c,u_c\right),
\qquad\tau=1.
$$

Use attenuation 1 when the denominator is zero or $\tau=0$. This factor is
a smooth noise penalty, not a confidence probability: it does not capture
uncertainty in peer calibration, practice or player selection. No minimum
deviation is imposed on ordinary charts.

After computing **all** independent estimates, find one offset $b$ such that

$$
d_c=\operatorname{clip}(d_c^0-b,\ell_c,u_c),\qquad
\sum_{c\in\mathcal P}d_c=0,\qquad\widehat L_c=L_c+d_c,
$$

where $\mathcal P$ contains only published charts,
$\ell_c=\max(-\Delta(L_c),0.1-L_c)$ and
$u_c=\min(\Delta(L_c),20-L_c)$. Deterministic bisection solves this bounded
least-squares projection. Each chart contributes once, irrespective of play
count. Reapplying the bounds during centering prevents a chart at its cap
from escaping it. Unsupported charts remain NULL. Balancing runs once per
complete population, so changing database batch sizes cannot change levels.
It does not require equal numbers of upward and downward adjustments.

Votes can be used to validate the results, but never enter the production
calculation or set individual chart targets.

The runner adds a batched training pass, retaining only cell summaries,
then computes the complete result population, balances it, and persists it
in short batch transactions. It remains entirely within `cmd/fitting`.
Statistical means describe corrected levels before shrinkage, attenuation, caps and balancing;
`fitting_level` contains the final output. `sample_count`
counts raw successful inversions above the minimum score, not contributing
players. Both independent and balanced levels stay within `[0.1, 20]`.

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

This is a raw observation relative to the player's top-K mean rating.
Subtract the peer residual from section 4.0 before aggregation; the raw
inversion alone does not establish that a chart is harder or easier.

### 4.2 Pre-weighting

**Proximity weight (asymmetric Gaussian with hard cutoff).** Samples near
$10L_c$ receive more weight. The over-skilled side uses a narrower bandwidth
to limit contributions from players far above the chart's target skill band.

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

Score-quality weighting is disabled by default. When enabled, the same
weight is used in peer calibration and target inference.

**Sample-age weight (optional, disabled by default).** With a positive
`sample_halflife_days` $H$ and age $a\ge0$ in days since
`play_records.record_time`, use $w^{\text{age}}=2^{-a/H}$. Missing or future
timestamps have $a=0$. When disabled, this factor is 1.

**Combined pre-weight:** $\tilde{w}_{p,c} = w^{\text{prox}}_{p,c} \cdot
w^{\text{vol}}_p \cdot w^{\text{score}}_{p,c} \cdot w^{\text{age}}_{p,c}$.

### 4.3 Robust trimming (Tukey biweight)

Let $\tilde{m}_c$ and $\mathrm{MAD}_c$ be the *weighted* median and *weighted*
median absolute deviation of $\{\widetilde{\delta}_{p,c}\}$ under pre-weights
$\{\tilde{w}_{p,c}\}$ (ties broken by ascending $\widetilde{\delta}$):

$$
\tilde{m}_c = \operatorname*{wmedian}_{p \in P_c}\widetilde{\delta}_{p,c};
\qquad
\mathrm{MAD}_c = \operatorname*{wmedian}_{p \in P_c}\bigl|\widetilde{\delta}_{p,c} - \tilde{m}_c\bigr|.
$$

For each sample compute the scaled residual

$$
u_{p,c} = \dfrac{\widetilde{\delta}_{p,c} - \tilde{m}_c}{h_c},\qquad
h_c=\begin{cases}
k\cdot\mathrm{MAD}_c, & k\cdot\mathrm{MAD}_c>10^{-9},\\
k\cdot0.01(|L_c|+1), & \text{otherwise}.
\end{cases}
$$

The second case avoids division by zero for concentrated samples. Apply
the Tukey biweight

$$
w^{\text{rob}}_{p,c} = \begin{cases}
(1 - u_{p,c}^{2})^{2}, & |u_{p,c}| < 1,\\[4pt]
0,                     & |u_{p,c}| \ge 1.
\end{cases}
$$

Samples outside this robust scale contribute zero to the final estimate.

### 4.4 Aggregation

Define the final composite weight $w_{p,c} = \tilde{w}_{p,c}\cdot
w^{\text{rob}}_{p,c}$.

**Weighted mean** (pre-shrinkage estimate):

$$
\mu_c = \dfrac{\sum_p w_{p,c}\,\widetilde{\delta}_{p,c}}{\sum_p w_{p,c}}.
$$

**Kish effective sample size** (how many "ideal" samples the weighting
scheme is equivalent to):

$$
N^{\text{eff}}_c = \dfrac{\left(\sum_p w_{p,c}\right)^{2}}{\sum_p w_{p,c}^{2}}.
$$

We abstain from publishing a fitting level when $N^{\text{eff}}_c <
\text{min\_samples}$ (default 5): the column `charts.fitting_level` is
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
\qquad n_{\text{ref}} = \max(1,2\cdot\text{min\_samples}).
$$

When the deviation is zero or when $N^{\text{eff}}_c \gg n_{\text{ref}}$ the
boost approaches 1; when the deviation is large and
the effective sample is small, $\kappa_{\text{eff}}$ scales quadratically
with the gap and inversely with $N^{\text{eff}}_c$ — matching the intuition
that the further a chart drifts from its official value, the more evidence
we should demand before publishing that drift. The default is $\lambda = 2$.
Setting $\lambda = 0$ disables the deviation penalty.

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

The cap is tighter at lower levels and widens smoothly toward the upper
anchor. Interpolate it **log-linearly** between two anchor points:

$$
\Delta(L) =
\begin{cases}
\Delta_{\min}, & L \le L_{\text{low}},\\[4pt]
\Delta_{\min}\cdot\left(\dfrac{\Delta_{\max}}{\Delta_{\min}}\right)^{t(L)}, & L_{\text{low}} < L < L_{\text{high}},\\[8pt]
\Delta_{\max}, & L \ge L_{\text{high}}.
\end{cases}
\qquad t(L) = \dfrac{L - L_{\text{low}}}{L_{\text{high}} - L_{\text{low}}}.
$$

Defaults: $\Delta_{\min} = 0.15$, $\Delta_{\max} = 0.3$, $L_{\text{low}} =
12.0$, $L_{\text{high}} = 17.0$. The midpoint $L = 14.5$ sits at the
geometric mean $\Delta(14.5) = \sqrt{\Delta_{\min}\cdot\Delta_{\max}}
\approx 0.212$. These are final output limits, applied after noise attenuation and again
during population balancing.

**Degeneration rule.** When $\Delta_{\min} \le 0$ or the anchors
$L_{\text{low}}$/$L_{\text{high}}$ are misconfigured (see
`internal/fitting/calculator.go:effectiveMaxDeviation`), the implementation
silently falls back to a **flat** cap $\Delta(L) \equiv \Delta_{\max}$; the
startup validator rejects the most common misconfigurations (see
`AGENTS.md`). The ramp only controls the **width** of the cap, never its
**symmetry** — the same $\Delta(L_c)$ is used on both sides.

The cap limits unusually large local adjustments; section 4.0 separately
anchors the published total. Alpha and kappa still control eligibility and
shrinkage. The cap does not require every chart to differ from official.

## 5. Summary pipeline

```
calibration := first-pass chart/cell summaries from §4.0
for each chart c with official level L_c:
    samples := { (p, s_{p,c}) : p ∈ P_c, s_{p,c} ≥ s_min }
    for each sample:
        δ̂ := InverseRating(s_{p,c}, B_p)                      # §4.1
        if δ̂ ∉ [0.1, 20.0]: drop
        if calibration enabled:
            if perfect score or unsupported cell: drop
            δ̂ -= b_{c,k}
        diff  := B_p - 10·L_c                                   # §4.2
        σ_eff := (diff > 0 ? α·σ_prox : σ_prox)
        if |diff| > 2.5·σ_eff: drop                              #  ← hard cutoff
        w_prox := exp(-diff² / (2·σ_eff²))
        w_vol  := min(1, n_p / V_full)
        w_pre  := w_prox * w_vol * score_quality * sample_age
    m_c  := weighted_median(δ̂; w_pre)                          # §4.3
    MAD  := weighted_median(|δ̂ - m_c|; w_pre)
    for each sample:
        h := k·MAD if k·MAD > 1e-9, otherwise k·0.01·(|L_c|+1)
        u := (δ̂ - m_c) / h
        w_rob := (1 - u²)²  if |u| < 1 else 0
        w     := w_pre * w_rob
    μ_c     := Σ w·δ̂ / Σ w                                    # §4.4
    N_eff_c := (Σ w)² / Σ w²
    if N_eff_c < min_samples:
        mark FittingLevel = NULL (stats still accumulate); continue
    dev     := μ_c - L_c                                       # §4.5
    n_ref   := max(1, 2 · min_samples)
    κ_eff   := κ · (1 + λ·dev²·n_ref / N_eff_c)
    L̂_c     := (N_eff_c·μ_c + κ_eff·L_c) / (N_eff_c + κ_eff)
    if calibration enabled:
        r := L̂_c - L_c; v := σ_c² / N_eff_c
        a := r² / (r² + τ·v)  (or 1 when denominator is zero)
        L̂_c := L_c + a·r
    Δ       := effectiveMaxDeviation(L_c)                      # §4.6
    L̂_c     := L_c + clip(L̂_c - L_c, -Δ, Δ)
    if calibration enabled: L̂_c := clip(L̂_c, 0.1, 20)
    keep (c, L̂_c, diagnostics) in the complete result population

After all charts have been computed, balance published deltas with a single
bounded offset when calibration and balance_total are enabled (§4.0).
Flush results every chart_batch_size charts in short transactions (§7):
    one UPDATE ... FROM (VALUES ...) bulk-updates charts.fitting_level
    one conflict-aware bulk UPSERT writes chart_statistics
        (c, sample_count, N_eff_c, μ_c, m_c, σ_c, MAD, L̂_c, L_c, now)
```

## 6. Hyperparameters (from `config.yaml`)

| Key                             | Symbol                 | Default   | Role                                                                                   |
|---------------------------------|------------------------|-----------|----------------------------------------------------------------------------------------|
| `fitting.skill_top_k` | K | `20` | Number of highest ratings used for player skill. |
| `fitting.calibration_enabled` | — | `true` | Enable peer correction, exclude perfect/unsupported samples, and attenuate sampling noise. |
| `fitting.calibration_noise_penalty` | τ | `1.0` | Finite nonnegative sampling-noise penalty; zero disables. |
| `fitting.balance_total` | — | `true` | Match the unweighted official total of published charts; calibrated mode only. |
| `fitting.sample_halflife_days` | — | `0` | Optional sample-age half-life; zero disables. |
| `fitting.enabled`               | —                      | `true`    | Master switch for the microservice.                                                    |
| `fitting.interval`              | —                      | `6h`      | Ticker period (Go duration).                                                           |
| `fitting.min_samples`           | min_samples            | `5.0`     | $N^{\text{eff}}$ below this → abstain.                                                 |
| `fitting.min_player_records`    | —                      | `20`      | Exclude players with fewer best records.                                               |
| `fitting.proximity_sigma`       | $\sigma_{\text{prox}}$ | `18.5`    | Gaussian bandwidth around $10L_c$.                                                     |
| `fitting.high_skill_sigma_ratio`| $\alpha$               | `0.2`     | $\sigma$ multiplier on the over-skilled side (asymmetric Gaussian). `1.0` = symmetric, smaller = stronger discount on over-skilled players. Samples outside $2.5\cdot\sigma$ are dropped.  |
| `fitting.volume_full_at`        | $V_{\text{full}}$      | `50`      | Volume weight saturation point.                                                        |
| `fitting.prior_strength`        | $\kappa$               | `5.0`     | Baseline shrinkage strength toward $L_c$.                                              |
| `fitting.deviation_penalty`     | $\lambda$              | `2.0`     | Deviation penalty; sets $\kappa_{\text{eff}} = \kappa(1+\lambda\cdot\text{dev}^2\cdot n_{\text{ref}}/N^{\text{eff}})$. `0` reverts to static $\kappa$. |
| `fitting.max_deviation`         | $\Delta_{\max}$        | `0.3`     | Final cap at the high-level end ($\ge L_{\text{high}}$); also the flat cap when the ramp is disabled. |
| `fitting.max_deviation_low`     | $\Delta_{\min}$        | `0.15`    | Final cap at the low-level end ($\le L_{\text{low}}$); set to `0` to disable the ramp and fall back to flat $\Delta_{\max}$. |
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
2. `chart_statistics` (owned by `cmd/fitting`) — one row per
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
- Each chart batch is persisted in one short transaction: one
  `UPDATE ... FROM (VALUES ...)` derived-table update for
  `charts.fitting_level` and one conflict-aware bulk upsert for
  `chart_statistics`.
- The probe server's caches are not invalidated; persisted values become
  visible when the relevant cached entries expire or are invalidated.

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

1. Official levels anchor both ratings and the reference population. Local calibration cannot identify a shared absolute error across an entire neighborhood. B20 still includes the target performance; this is not a fully independent player skill estimate.
2. Best scores reflect practice and self-selected charts. A scalar skill cannot represent individual chart-style strengths; robust aggregation does not remove these systematic effects.
3. Time decay is available through `sample_halflife_days`, but disabled by default. Enabling it requires revalidation of the estimates.
4. Perfect scores, insufficient peer charts or insufficient target effective samples cause abstention. Coverage can decrease; report it alongside error metrics.
5. Computation reads are not one consistent database snapshot. Cancellation or a write failure can leave only some balanced batches committed; the total constraint holds after a complete successful pass. No writes occur if estimation fails before persistence.
