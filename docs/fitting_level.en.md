# Fitting Level Calculation

*Available in: **English** · [中文](./fitting_level.zh.md)*

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

**Proximity weight.** Players whose skill $B_p$ is close to $10\cdot L_c$
play near the intended difficulty band; their score distribution is most
informative. We use a zero-mean Gaussian in rating units:

$$
w^{\text{prox}}_{p,c} = \exp\!\left(-\dfrac{(B_p - 10L_c)^2}{2\sigma_{\text{prox}}^{2}}\right).
$$

The default $\sigma_{\text{prox}} = 20$ corresponds to $\pm 2.0$ level units
of "effective skill", capturing the band a chart's realistic audience spans.

**Volume weight.** Players with very few records have noisier $B_p$
estimates. We apply a linear ramp that saturates at $V_{\text{full}} = 50$
records:

$$
w^{\text{vol}}_p = \min\!\left(1, \dfrac{n_p}{V_{\text{full}}}\right).
$$

**Combined pre-weight:** $\tilde{w}_{p,c} = w^{\text{prox}}_{p,c} \cdot
w^{\text{vol}}_p$.

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

### 4.5 Bayesian shrinkage toward the official level

Treat $L_c$ as a Gaussian prior with strength $\kappa$, and $\mu_c$ as the
likelihood mean with precision $N^{\text{eff}}_c$. The posterior mean is the
precision-weighted combination

$$
\hat{L}_c = \dfrac{N^{\text{eff}}_c\,\mu_c + \kappa\,L_c}{N^{\text{eff}}_c + \kappa}.
$$

Equivalently: the "confidence" of the official level is $\kappa$ pseudo-
samples, so a chart with $N^{\text{eff}}_c \gg \kappa$ effectively follows
the data, while a chart with $N^{\text{eff}}_c \ll \kappa$ stays near the
official value.

### 4.6 Deviation cap

As a final safety net against model mis-specification we enforce

$$
\hat{L}_c \leftarrow L_c + \operatorname{clip}\!\bigl(\hat{L}_c - L_c,\ -\Delta_{\max},\ \Delta_{\max}\bigr).
$$

## 5. Summary pipeline

```
for each chart c with official level L_c:
    samples := { (p, s_{p,c}) : p ∈ P_c, s_{p,c} ≥ s_min }
    for each sample:
        δ̂ := InverseRating(s_{p,c}, B_p)          # §4.1
        if δ̂ ∉ [0.1, 20.0]: drop
        w_prox := exp(-(B_p - 10 L_c)^2 / 2σ_prox^2)  # §4.2
        w_vol  := min(1, n_p / V_full)
        w_pre  := w_prox * w_vol
    m_c  := weighted_median(δ̂; w_pre)               # §4.3
    MAD  := weighted_median(|δ̂ - m_c|; w_pre)
    for each sample:
        u := (δ̂ - m_c) / (k · max(MAD, ε))
        w_rob := (1 - u^2)^2  if |u| < 1 else 0
        w     := w_pre * w_rob
    μ_c     := Σ w·δ̂ / Σ w                          # §4.4
    N_eff_c := (Σ w)^2 / Σ w^2
    if N_eff_c < min_samples:
        publish FittingLevel = NULL; keep stats; continue
    L̂_c := (N_eff_c · μ_c + κ · L_c) / (N_eff_c + κ)  # §4.5
    L̂_c := L_c + clip(L̂_c - L_c, -Δ_max, Δ_max)       # §4.6
    UPDATE charts SET fitting_level = L̂_c WHERE id = c
    UPSERT chart_statistics (c, sample_count, N_eff_c, μ_c, m_c, σ_c, MAD, L̂_c, L_c, now)
```

## 6. Hyperparameters (from `config.yaml`)

| Key                          | Symbol              | Default   | Role                                       |
|------------------------------|---------------------|-----------|--------------------------------------------|
| `fitting.enabled`            | —                   | `true`    | Master switch for the microservice.         |
| `fitting.interval`           | —                   | `6h`      | Ticker period (Go duration).                |
| `fitting.min_samples`        | min_samples         | `8.0`     | $N^{\text{eff}}$ below this → abstain.      |
| `fitting.min_player_records` | —                   | `20`      | Exclude players with fewer best records.     |
| `fitting.proximity_sigma`    | $\sigma_{\text{prox}}$ | `20.0` | Gaussian bandwidth around $10L_c$.          |
| `fitting.volume_full_at`     | $V_{\text{full}}$    | `50`      | Volume weight saturation point.             |
| `fitting.prior_strength`     | $\kappa$            | `5.0`     | Shrinkage toward $L_c$.                     |
| `fitting.max_deviation`      | $\Delta_{\max}$      | `1.5`     | Hard cap on $|\hat{L}_c - L_c|$.            |
| `fitting.min_score`          | $s_{\min}$          | `500000`  | Minimum score admitted.                     |
| `fitting.tukey_k`            | $k$                 | `4.685`   | Biweight tuning constant.                   |
| `fitting.chart_batch_size`   | —                   | `200`     | Charts processed per DB batch.              |
| `fitting.player_batch_size`  | —                   | `500`     | Users fetched per page.                     |
| `fitting.batch_pause`        | —                   | `50ms`    | Sleep between batches (DB load relief).      |

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
go run cmd/fitting/main.go -config config/config.yaml

# One-shot (useful for cron, debugging, CI smoke tests)
go run cmd/fitting/main.go -config config/config.yaml -once
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
