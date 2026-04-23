# 拟合定数的计算

*语言版本:[English](./fitting_level.en.md) · **中文***

*面向玩家的人话版说明:[中文](./fitting_level_for_players.zh.md) · [English](./fitting_level_for_players.en.md)*

> **适用范围。** 本文档说明拟合定数微服务(`cmd/fitting`)如何从 `best_play_records`
> 中推导每张谱面的 `fitting_level`。文档力求自包含:读者无需查阅查分器本体
> (`cmd/server`)的源码即可复现全部数学推导。
>
> 查分器本体**不会**计算拟合定数,运行时也**不会**读取 `fitting.*` 下的任何
> 配置。相关设计原则见 `AGENTS.md → 保持查分器本体的单纯性`。

## 1. 问题描述

`charts` 表中每一行包含:

- `level` — 由游戏/谱师公布的**官方**难度定数,取形如 `{整数}.{十分位}` 的实数(例如 `14.5`)。
- `fitting_level` — 一个可空的精修估计值,由我们离线根据玩家数据计算得到。

目标是:给定一张官方定数为 $L_c$ 的谱面 $c$,以及玩家集合 $P_c$ 中每位玩家
$p$ 对该谱面的最佳成绩分布 $\{s_{p,c}\}$,给出一个后验点估计 $\hat{L}_c$
(写入 `charts.fitting_level`),使之满足:

1. 将**官方定数**作为先验信息予以尊重;
2. 能够根据**实际成绩分布**做出调整,并对离群样本和小样本谱面具有鲁棒性;
3. 按玩家自身实力加权,使"实力接近 $L_c$"的玩家贡献更大的可信度;
4. 对玩家质量的异质性(记录数多/少、实力中心/边缘)具备容忍能力。

## 2. 记号

| 符号                         | 含义                                                                                   |
|------------------------------|----------------------------------------------------------------------------------------|
| $L_c$                        | 谱面 $c$ 的官方定数(浮点,来自 `charts.level`)。                                        |
| $\hat{L}_c$                  | 计算得到的拟合定数(浮点,写入 `charts.fitting_level`)。                                |
| $s_{p,c}$                    | 玩家 $p$ 在谱面 $c$ 上的最佳成绩(整数,范围 0–1 010 000)。                              |
| $r_{p,c}$                    | 按官方定数计算的单曲 rating,见 `pkg/rating/rating.go`。                                 |
| $B_p$                        | 玩家 $p$ 的**浮点 B50 均值 rating**:其前 $K$ 个最高单曲 rating 的算术平均,$K = \min(\|\text{best}_p\|, 50)$。 |
| $n_p$                        | 玩家 $p$ 的最佳记录总数。                                                               |
| $\hat{\delta}_{p,c}$         | 根据 $(s_{p,c}, B_p)$ 反推出的单样本定数;见 §4.1。                                      |
| $w^{\text{prox}}_{p,c}$      | 邻近权重(玩家实力离 $10L_c$ 越近越高,远超 2.5σ 直接丢弃)。                          |
| $w^{\text{vol}}_p$           | 数据量权重(玩家记录越多越高,存在饱和点)。                                              |
| $w^{\text{rob}}_{p,c}$       | 鲁棒权重(Tukey 双权,远离中心的样本收到惩罚)。                                          |
| $w_{p,c}$                    | 合成权重 $w^{\text{prox}}\cdot w^{\text{vol}}\cdot w^{\text{rob}}$。                    |
| $N^{\text{eff}}_c$           | 谱面 $c$ 的 Kish 有效样本量。                                                           |
| $\kappa$                     | 先验强度(贝叶斯收缩系数),`config.fitting.prior_strength`。                            |
| $\lambda$                    | 偏差惩罚系数,对偏离官方定数的少样本动态加强 $\kappa$;`config.fitting.deviation_penalty`。 |
| $\alpha$                     | over-skilled 一侧的 σ 缩放比(不对称邻近权重),`config.fitting.high_skill_sigma_ratio`。 |
| $\Delta_{\max}$              | $\|\hat{L}_c - L_c\|$ 的硬上限上端,`config.fitting.max_deviation`。                    |
| $\Delta_{\min}$              | cap 在低定数端的值,`config.fitting.max_deviation_low`;≤0 时关闭斗形坡道回退到平顶 $\Delta_{\max}$。 |
| $L_{\text{low}},L_{\text{high}}$ | cap 斗形坡道的两端点(默认 12.0 / 17.0),`config.fitting.max_deviation_{low_at, high_at}`。 |
| $\sigma_{\text{prox}}$       | 邻近权重高斯带宽(rating 单位),`config.fitting.proximity_sigma`。                      |
| $V_{\text{full}}$            | 数据量权重饱和到 1 所需的记录数,`config.fitting.volume_full_at`。                       |
| $k$                          | Tukey 双权调节常数,`config.fitting.tukey_k`(默认 4.685)。                             |
| $s_{\min}$                   | 参与计算的最小成绩阈值,`config.fitting.min_score`。                                    |

## 3. Rating 公式(参考)

单次成绩的 rating 由 `pkg/rating/rating.go` 中的分段函数 $\mathrm{Rating}(L, s)$
定义。令 $b = \lfloor\max(s, 1\,010\,000)\rfloor$,则

$$
\mathrm{Rating}(L, s) =
\begin{cases}
10L + 7 + 3\left(\dfrac{s - 1\,009\,000}{1000}\right)^{1.35}, & s \ge 1\,009\,000,\\[8pt]
10\left(L + \dfrac{2(s - 1\,000\,000)}{30\,000}\right),       & 1\,000\,000 \le s < 1\,009\,000,\\[8pt]
B(s) + 10\left(L\left(\dfrac{s}{10^{6}}\right)^{1.5} - 0.9\right), & 0 \le s < 1\,000\,000,
\end{cases}
$$

其中 $B(s)$ 是阶梯奖励函数

$$
B(s) = 3\mathbf{1}\{s \ge 900\,000\} + \sum_{t\in\{930,950,970,980,990\}\times 10^3} \mathbf{1}\{s \ge t\},
$$

并在输出端截断到 $\max(\mathrm{Rating}, 0)$。持久化在数据库中的 `play_records.rating`
列等于 $\lfloor 100\cdot\mathrm{Rating} + \varepsilon\rfloor$。

## 4. 算法

### 4.1 单样本反推定数

对每条最佳记录 $(p, c)$,我们以 $B_p$ 为 rating 目标,反解 $\mathrm{Rating}$ 关于
$L$ 的方程。由于 $\mathrm{Rating}$ 在三个分段内对 $L$ 都是线性的,闭式反函数存在:

$$
\hat{\delta}_{p,c} =
\begin{cases}
\dfrac{B_p - 7 - 3\left((s - 1\,009\,000)/1000\right)^{1.35}}{10}, & s \ge 1\,009\,000,\\[8pt]
\dfrac{B_p}{10} - \dfrac{2(s - 1\,000\,000)}{30\,000},             & 1\,000\,000 \le s < 1\,009\,000,\\[8pt]
\dfrac{B_p - B(s) + 9}{10\,(s/10^{6})^{1.5}},                       & 0 < s < 1\,000\,000,
\end{cases}
$$

在 $s = 0$ 时无定义。若 $\hat{\delta}_{p,c} \notin [0.1, 20.0]$(游戏实际使用的
定数范围之外),则丢弃该样本。

直观理解:$\hat{\delta}_{p,c}$ 回答的是"怎样的谱面定数才能让这位玩家的实际成绩
恰好等于其 B50 平均 rating"。如果一张谱面的实际难度低于官方定数,玩家会普遍
打出高于自己实力目标的分数,从而把 $\hat{\delta}_{p,c}$ 推向小于 $L_c$ 的一侧。

### 4.2 预权重

**邻近权重(不对称 · 硬截断)。** 实力 $B_p$ 接近 $10\cdot L_c$ 的玩家正处于该谱面的“目标难度区间”,他们的成绩分布信息量最大。我们以 rating 单位下、均值为零的**不对称**高斯核打分,并在 2.5·σ 处**硬截断**:

$$
\sigma_{\text{eff}}(B_p) =
\begin{cases}
\alpha \cdot \sigma_{\text{prox}}, & B_p - 10L_c > 0,\\[4pt]
\sigma_{\text{prox}},               & B_p - 10L_c \le 0,
\end{cases}
\qquad
w^{\text{prox}}_{p,c} = \begin{cases}
\exp\!\left(-\dfrac{(B_p - 10L_c)^2}{2\,\sigma_{\text{eff}}(B_p)^{2}}\right), & \bigl|B_p - 10L_c\bigr| \le 2.5\,\sigma_{\text{eff}}(B_p),\\[8pt]
0\ (\text{样本丢弃,不计入 raw 与}\ N^{\text{eff}}), & \bigl|B_p - 10L_c\bigr| > 2.5\,\sigma_{\text{eff}}(B_p).
\end{cases}
$$

默认 $\sigma_{\text{prox}} = 20$(对应 $\pm 2.0$ 定数单位的实力带宽),$\alpha = 0.3$。

**为什么不对称?** 实力远超 $10L_c$ 的大佬在低定数谱面上几乎必然 AP,此时 §4.1 的单样本反推退化为“回声”玩家自己的 $B_p/10$,而不再度量谱面难度。这类样本会系统性地将 $\hat{\delta}$ 拉高,正是中段偏差的主要来源。在 over-skilled 一侧缩小 σ(乘以 α)将这一福度大幅压缩,但不影响 under-skilled 一侧—— 实力跟不上的玩家仍然提供信息,只是权重自然较低。

**为什么还要硬截断?** Kish 的 $N^{\text{eff}}$ 是尺度无关的(它度量的是权重的相对分布而非绝对大小),仅缩小 σ 不会让这类样本在样本数的估计上消失—— 5 个权重皆为 1% 的样本仍算 $N^{\text{eff}} = 5$。因此我们在 $2.5\,\sigma_{\text{eff}}$ 处直接截断,让 raw / inferred / $N^{\text{eff}}$ 一起下降。结果:一张长期被大佬乱打的低定数谱面会因为样本不足而正确地**弃算**(§4.4),而不是发布一个被大佬实力回声主导的拟合值。

**默认 $\alpha = 0.3$ 的来历。** 最初版本采用对称 σ($\alpha = 1$),考察全局偏差后发现中段(lv13–lv15.5)的 $\hat{L}_c$ 系统性高于官方定数 $+0.5 \sim +0.8$。$\alpha = 0.5$ 缩半后仍剩 $+0.4 \sim +0.7$;进一步缩到 $\alpha = 0.3$ 持续推进,中段偏差降至 $\sim +0.1 \sim +0.4$ 且 lv17+ 高定数端几乎不受影响(高定数谱面本身缺大佬玩家,不对称项几乎不触发)。继续缩到 $\alpha = 0.2$ 会在 lv14 带造成 $-0.36$ 的反向过度修正,故选取 0.3 作为甜蜜点。

**数据量权重。** 记录太少的玩家 $B_p$ 估计噪声较大。我们采用线性斜坡,在
$V_{\text{full}} = 50$ 条记录时饱和:

$$
w^{\text{vol}}_p = \min\!\left(1, \dfrac{n_p}{V_{\text{full}}}\right).
$$

**合成预权重:** $\tilde{w}_{p,c} = w^{\text{prox}}_{p,c} \cdot w^{\text{vol}}_p$。

### 4.3 鲁棒裁剪(Tukey 双权)

在预权重 $\{\tilde{w}_{p,c}\}$ 下,令 $\tilde{m}_c$、$\mathrm{MAD}_c$ 分别为
$\{\hat{\delta}_{p,c}\}$ 的**加权中位数**与**加权中位数绝对偏差**(同值时按
$\hat{\delta}$ 升序断开):

$$
\tilde{m}_c = \operatorname*{wmedian}_{p \in P_c}\hat{\delta}_{p,c};
\qquad
\mathrm{MAD}_c = \operatorname*{wmedian}_{p \in P_c}\bigl|\hat{\delta}_{p,c} - \tilde{m}_c\bigr|.
$$

对每条样本计算标准化残差

$$
u_{p,c} = \dfrac{\hat{\delta}_{p,c} - \tilde{m}_c}{k \cdot \max(\mathrm{MAD}_c, \epsilon)},
$$

其中 $\epsilon$ 为下限保护(实现中取 $(|L_c|+1) \times 1\%$),用于整批样本异常
集中时避免除零。再应用 Tukey 双权函数

$$
w^{\text{rob}}_{p,c} = \begin{cases}
(1 - u_{p,c}^{2})^{2}, & |u_{p,c}| < 1,\\[4pt]
0,                     & |u_{p,c}| \ge 1.
\end{cases}
$$

反推定数距加权中位数超过 $k\cdot\mathrm{MAD}_c$ 的样本在最终估计中的贡献直接
**清零**,直面"记录可信度 / 偏中心"问题——远离共识的样本被完全抑制。

### 4.4 聚合

定义最终合成权重 $w_{p,c} = \tilde{w}_{p,c}\cdot w^{\text{rob}}_{p,c}$。

**加权均值**(收缩前的估计量):

$$
\mu_c = \dfrac{\sum_p w_{p,c}\,\hat{\delta}_{p,c}}{\sum_p w_{p,c}}.
$$

**Kish 有效样本量**(当前加权方案等价于多少个"理想"无权样本):

$$
N^{\text{eff}}_c = \dfrac{\left(\sum_p w_{p,c}\right)^{2}}{\sum_p w_{p,c}^{2}}.
$$

当 $N^{\text{eff}}_c < \text{min\_samples}$(默认 8)时,**弃算**:
`charts.fitting_level` 写入 `NULL`,但 `chart_statistics` 依然写入诊断字段供离线
排查。

### 4.5 向官方定数的贝叶斯收缩(含偏差惩罚)

将 $L_c$ 视作强度为 $\kappa$ 的高斯先验,把 $\mu_c$ 视作精度为 $N^{\text{eff}}_c$
的似然均值,则后验均值为二者的精度加权:

$$
\hat{L}_c = \dfrac{N^{\text{eff}}_c\,\mu_c + \kappa_{\text{eff}}\,L_c}{N^{\text{eff}}_c + \kappa_{\text{eff}}}.
$$

其中 $\kappa_{\text{eff}}$ 是偏差敏感的动态先验强度。将 $\kappa$ 直接代入会有一个问题:在样本稀少的谱面上,几个离群玩家可以把 $\mu_c$ 拉离官方定数好几个 level,但数据量不足以支撑这样的置信。我们因此引入乘法型偏差惩罚,令

$$
\kappa_{\text{eff}} = \kappa\cdot\left(1 + \lambda\,(\mu_c - L_c)^2\cdot\dfrac{n_{\text{ref}}}{N^{\text{eff}}_c}\right),
\qquad n_{\text{ref}} = 2\cdot\text{min\_samples},
$$

当偏差为零或 $N^{\text{eff}}_c \gg n_{\text{ref}}$ 时 boost 退化为 1(无绩效开销),而当偏差大且样本少时 $\kappa_{\text{eff}}$ 二次放大、反比于 $N^{\text{eff}}_c$,恰好符合“偏得越多越需要证据”的直觉。默认 $\lambda = 2$。将 $\lambda = 0$ 即回退到旧的静态- $\kappa$ 行为,所有测试设置 $\lambda = 0$ 进行回归。

等价解读:“相信官方定数”的程度相当于 $\kappa_{\text{eff}}$ 个伪样本—— $N^{\text{eff}}_c \gg \kappa_{\text{eff}}$ 时估计值基本等于数据,$N^{\text{eff}}_c \ll \kappa_{\text{eff}}$ 时靠近官方定数;且偏差越大这条天秤越向官方端倾斜。

### 4.6 偏差上限(随定数变化的对数尺度斗形坡道)

作为防止模型错配的最终安全网,我们在收缩后施加硬截断:

$$
\hat{L}_c \leftarrow L_c + \operatorname{clip}\!\bigl(\hat{L}_c - L_c,\ -\Delta(L_c),\ \Delta(L_c)\bigr).
$$

**为什么随定数变化?** Reboot 的官方定数轴在游戏体验上是大致对数的—— 从 lv12 到 lv13 的难度跳跃远小于从 lv16 到 lv17 的难度跳跃。因此在低定数端我们希望 cap 更紧,不让拟合值跈过“真实难度梯级”;在高定数端则希望 cap 更宽,因为官方步长更粗,需要给算法更多“发挥空间”。我们在两个端点间用**对数线性**插值:

$$
\Delta(L) =
\begin{cases}
\Delta_{\min}, & L \le L_{\text{low}},\\[4pt]
\Delta_{\min}\cdot\left(\dfrac{\Delta_{\max}}{\Delta_{\min}}\right)^{t(L)}, & L_{\text{low}} < L < L_{\text{high}},\\[8pt]
\Delta_{\max}, & L \ge L_{\text{high}}.
\end{cases}
\qquad t(L) = \dfrac{L - L_{\text{low}}}{L_{\text{high}} - L_{\text{low}}}.
$$

默认值:$\Delta_{\min} = 0.6$,$\Delta_{\max} = 1.5$,$L_{\text{low}} = 12.0$,$L_{\text{high}} = 17.0$。从端点对处中间点 $L = 14.5$ 有 $\Delta(14.5) = \sqrt{\Delta_{\min}\cdot\Delta_{\max}} \approx 0.949$,符合“随 level 单调上升且不突变”的期望。

**退化规则。** 当 $\Delta_{\min} \le 0$,或 $L_{\text{low}}$/$L_{\text{high}}$ 配置不合法(参见 `internal/fitting/calculator.go:effectiveMaxDeviation`),实现静默回退到**平顶** $\Delta(L) \equiv \Delta_{\max}$;启动时的配置校验会拒绝最常见的误配置(见 `AGENTS.md`)。斗形坡道只影响上限的**宽窄**,不影响上限的**对称性**—— 在双向上都采用同一 $\Delta(L_c)$。

**防护 · 而非修正。** 在当前数据集上,绝大多数拟合值本就在斗形窗口内,$\Delta(L)$ 很少触发—— 它是拦住偶发灾难性离群值的护栏,不是用来拉近中段整体 bias 的工具。中段 bias 的真正杆杆是 α 与 κ—— 见 §4.2 与 §4.5。

## 5. 流水线总览

```
对每张官方定数为 L_c 的谱面 c:
    samples := { (p, s_{p,c}) : p ∈ P_c, s_{p,c} ≥ s_min }
    对每条样本:
        δ̂ := InverseRating(s_{p,c}, B_p)                      # §4.1
        若 δ̂ ∉ [0.1, 20.0] 则丢弃
        diff  := B_p - 10·L_c                                  # §4.2
        σ_eff := (diff > 0 ? α·σ_prox : σ_prox)
        若 |diff| > 2.5·σ_eff 则丢弃                           #  ← 硬截断
        w_prox := exp(-diff² / (2·σ_eff²))
        w_vol  := min(1, n_p / V_full)
        w_pre  := w_prox * w_vol
    m_c  := weighted_median(δ̂; w_pre)                          # §4.3
    MAD  := weighted_median(|δ̂ - m_c|; w_pre)
    对每条样本:
        u := (δ̂ - m_c) / (k · max(MAD, ε))
        w_rob := (1 - u²)²  若 |u| < 1,否则 0
        w     := w_pre * w_rob
    μ_c     := Σ w·δ̂ / Σ w                                    # §4.4
    N_eff_c := (Σ w)² / Σ w²
    若 N_eff_c < min_samples:
        写入 FittingLevel = NULL;仍写 chart_statistics;继续
    dev     := μ_c - L_c                                       # §4.5
    n_ref   := 2 · min_samples
    κ_eff   := κ · (1 + λ·dev²·n_ref / N_eff_c)
    L̂_c     := (N_eff_c·μ_c + κ_eff·L_c) / (N_eff_c + κ_eff)
    Δ       := effectiveMaxDeviation(L_c)                      # §4.6
    L̂_c     := L_c + clip(L̂_c - L_c, -Δ, Δ)
    UPDATE charts SET fitting_level = L̂_c WHERE id = c
    UPSERT chart_statistics (c, sample_count, N_eff_c, μ_c, m_c, σ_c, MAD, L̂_c, L_c, now)
```

## 6. 超参数(来自 `config.yaml`)

| 键                             | 符号                   | 默认值    | 作用                                                       |
|-------------------------------|------------------------|-----------|-----------------------------------------------------------|
| `fitting.enabled`             | —                      | `true`    | 微服务总开关。                                             |
| `fitting.interval`            | —                      | `6h`      | Ticker 周期(Go duration 字符串)。                        |
| `fitting.min_samples`         | min_samples            | `8.0`     | $N^{\text{eff}}$ 低于此值则弃算。                          |
| `fitting.min_player_records`  | —                      | `20`      | 少于此记录数的玩家完全排除。                               |
| `fitting.proximity_sigma`     | $\sigma_{\text{prox}}$ | `20.0`    | 邻近权重高斯带宽(围绕 $10L_c$)。                         |
| `fitting.high_skill_sigma_ratio` | $\alpha$            | `0.3`     | over-skilled 一侧 σ 的缩放比(不对称高斯)。`1.0` 为对称高斯,更小值对大佬玩家折扣更重;样本离中心 2.5·σ 直接丢弃。 |
| `fitting.volume_full_at`      | $V_{\text{full}}$      | `50`      | 数据量权重饱和到 1 的临界记录数。                          |
| `fitting.prior_strength`      | $\kappa$               | `5.0`     | 官方定数的先验强度(收缩基准)。                         |
| `fitting.deviation_penalty`   | $\lambda$              | `2.0`     | 偏差惩罚;让 $\kappa_{\text{eff}} = \kappa(1+\lambda\cdot\text{dev}^2\cdot n_{\text{ref}}/N^{\text{eff}})$。`0` 时回退到静态 $\kappa$。 |
| `fitting.max_deviation`       | $\Delta_{\max}$        | `1.5`     | 高定数端(≥ $L_{\text{high}}$)上限;另作斗形坡道关闭时的平顶。 |
| `fitting.max_deviation_low`   | $\Delta_{\min}$        | `0.6`     | 低定数端(≤ $L_{\text{low}}$)上限;设为 0 即关闭斗形坡道,回退到平顶 $\Delta_{\max}$。 |
| `fitting.max_deviation_low_at`| $L_{\text{low}}$       | `12.0`    | cap 等于 $\Delta_{\min}$ 的端点;必须小于 $L_{\text{high}}$。 |
| `fitting.max_deviation_high_at`| $L_{\text{high}}$     | `17.0`    | cap 等于 $\Delta_{\max}$ 的端点;两端点之间用对数线性插值 $\Delta(L) = \Delta_{\min}\cdot(\Delta_{\max}/\Delta_{\min})^t$。 |
| `fitting.min_score`           | $s_{\min}$             | `500000`  | 成绩低于此阈值的样本直接丢弃。                             |
| `fitting.tukey_k`             | $k$                    | `4.685`   | Tukey 双权调节常数。                                       |
| `fitting.chart_batch_size`    | —                      | `200`     | 每个数据库批次处理的谱面数(控制单次事务规模)。           |
| `fitting.player_batch_size`   | —                      | `500`     | 玩家实力分页时每页用户数(键集分页)。                     |
| `fitting.batch_pause`         | —                      | `50ms`    | 批次之间的暂停时间,用来缓解数据库压力(Go duration)。     |

## 7. 数据库写入与表结构

计算器共写入两处:

1. `charts.fitting_level`(`double precision`,可空)—— 发布的估计值 $\hat{L}_c$,
   弃算时写入 `NULL`。
2. `chart_statistics`(新表,由 `cmd/fitting` 专属拥有)—— 每张谱面一行,主键为
   `chart_id`,保存流水线各阶段的诊断信息:`official_level`、`fitting_level`、
   `sample_count`、`effective_sample_size`、`weighted_mean`、`weighted_median`、
   `std_dev`、`mad`、`last_computed_at`,以及 `BaseModel` 标准时间戳。该表**不
   对外暴露 HTTP 路由**,仅用于内部分析。

为把对在线查分服务的影响降到最低,我们遵循以下策略:

- 玩家实力以 distinct `username` 为键做**键集分页**(每批 `player_batch_size`),
  不使用 OFFSET 扫描。
- 谱面按固定批 `chart_batch_size` 处理,批次间插入 `batch_pause` 短暂休眠。
- 每张谱面的写入独立开启短事务,避免长事务持锁。
- 查分服务的缓存不会被主动失效,依靠 TTL 自然过期;下一次用户上传会顺带刷新到
  新的 `fitting_level`。

## 8. 运维指南

```bash
# 持续模式(默认,遵循 fitting.interval)
go run ./cmd/fitting -config config/config.yaml

# 一次性模式(适合 cron、调试、CI 冒烟测试)
go run ./cmd/fitting --once -config config/config.yaml

# 只时诊断某张谱面(只读,不写库)
go run ./cmd/fitting analyze -chart 870 -config config/config.yaml
```

进程收到 `SIGINT` / `SIGTERM` 时会干净退出。在持续模式下,单次迭代的数据库错误
只会被记录到日志,**不会**导致循环退出——下一次 tick 会自动重试。

### Docker / docker-compose

项目的 `docker-compose.yaml` 在 `fitting` profile 下定义了 `fitting` 服务:

```bash
# 同时启动 db + app + fitting
docker compose --profile fitting up -d

# 或永久启用(推荐生产环境)
export COMPOSE_PROFILES=fitting
docker compose up -d

# 只查看 fitting 日志
docker compose --profile fitting logs -f fitting
```

镜像在构建阶段同时编译 `server` 和 `fitting` 两个二进制,`fitting` 服务通过
`command: ["./fitting"]` 切换入口。

#### 作为真正的定时任务(外部调度)

如果更倾向用外部调度器(host cron / systemd timer / k8s CronJob)而非内置
ticker,调整 `docker-compose.yaml` 中的 `fitting` 服务:

```yaml
fitting:
  ...
  command: ["./fitting", "-once"]
  restart: "no"
  profiles: []    # 移除 profile,使其可被 `docker compose run` 启动
```

再配合 crontab 等调度:

```cron
# 每 6 小时跑一次
0 */6 * * * cd /path/to/project && docker compose run --rm fitting >> /var/log/fitting.log 2>&1
```

### 生产部署建议

- 在独立进程/副本上运行,与主查分服务的资源限制分离。
- 把 `config.database.*` 指向主数据库(需要读写 `charts.fitting_level` 和
  `chart_statistics`)。
- 稳态下 `fitting.interval` 建议 $\ge 6$ 小时;新批次谱面上线时可临时调低。
- 通过 `chart_statistics.last_computed_at` 监控新鲜度。

## 9. 已知局限

1. **闭环生态。** 玩家实力目标 $B_p$ 源于同一个官方定数生成的 rating。若官方
   定数存在系统性偏差,$B_p$ 会弱相关地继承它。缓解措施:邻近权重限制了贡献者
   的"实力带",使这种偏差至多是二阶的;鲁棒裁剪进一步压制离群值。
2. **无时间衰减。** 我们平等对待所有成绩,不论 `record_time`。若游戏玩法发生
   变动(例如判定改动),旧记录可能让估计值与新现实脱节。后续可加入 recency
   kernel,实现简单直接。
3. **新谱冷启动。** 记录极少的新谱会被特意写成 `fitting_level = NULL`。在
   $N^{\text{eff}}$ 跨过 `min_samples` 之前,弃算是正确行为。
