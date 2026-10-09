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
| $B_p$                        | 玩家 $p$ 的**浮点 top-K 均值 rating**:其前 $K$ 个最高单曲 rating 的算术平均,$K = \min(\|\text{best}_p\|, \texttt{skill\_top\_k})$，默认20。 |
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
定义。先将成绩截断至 $s\le1\,010\,000$，然后计算：

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

### 4.0 当前默认：基于同类谱面的残差校准

默认使用 `skill_top_k: 20`、`calibration_enabled: true`、
`calibration_noise_penalty: 1`、`balance_total: true`。
B20 表达玩家最好的发挥，不能直接作为每张谱面的表现目标。
它用于实力分组，§4.1 的反推结果作为**原始观测**，先校准，再进入 §4.3。

记原始反推为 $I(s,B)$，官方定数为 $L_c$，实力差距为 $g=B/10-L_c$。
每个谱面按 $k=\lfloor g/0.5\rfloor$ 分格，用与 §4.2 完全相同的预权重，
计算格内 $I(s,B)-L_c$ 的加权中位数 $m_{c,k}$。一格至少需要 3 条合格成绩；
满分 $s\ge1\,010\,000$ 属于上限截断的观测，在校准训练与推断中均不参与。

预测目标谱面 $c$ 的某个实力格 $k$ 时，**排除整个目标谱面**，只取
$|L_j-L_c|\le1$、$|k'-k|\le2$ 的其他谱面格。权重为：

$$
a_{j,k'}=\exp\left[-\frac12\left(\left(\frac{L_j-L_c}{0.5}\right)^2+(k'-k)^2\right)\right].
$$

同一参考谱面先合成一个值 $v_j=\sum_{k'}a_{j,k'}m_{j,k'}/\sum_{k'}a_{j,k'}$，
其权重为 $u_j=\max_{k'}a_{j,k'}$。背景偏差 $b_{c,k}$ 为这些 $v_j$ 的加权中位数。
这样热门谱面不会仅因成绩更多、占据更多格子而支配基准。
至少需要 **10 张其他谱面**提供支持；不足时丢弃该实力格的目标样本，不回退到未经校准的 B20。

进入 §4.3 的单样本定数变为：

$$
\widetilde\delta_{p,c}=I(s_{p,c},B_p)-b_{c,k(p,c)}.
$$

§4.3–4.5 的中位数、MAD、Tukey、有效样本数与收缩均作用于校准后的值。
记收缩后的偏差为 $r_c$，近似采样方差为 $v_c=\sigma_c^2/N_c^{\mathrm{eff}}$。
先抑制与采样噪声相当的小偏差，再应用最终上限：

$$
d_c^0=\operatorname{clip}\left(r_c\frac{r_c^2}{r_c^2+\tau v_c},\ell_c,u_c\right),
\qquad\tau=1.
$$

分母为零或 $\tau=0$ 时，衰减因子取 1。这是平滑的噪声惩罚，不是置信概率；
它没有涵盖背景校准、玩家练习和选曲习惯的不确定性，也没有设置最小调整幅度。

**为什么要约束总和？** 这是显式的建模假设，不是从数据中验证出的“难度守恒”。
玩家实力来自按官方定数计算的 rating，参考谱面校准衡量的是相对表现差距，
两者都没有提供独立于官方尺度的绝对难度基准。因此选择同一批有结果谱面的官方均值
作为尺度锚点，假设这批官方定数的平均误差为零，再用总和约束消除估计器残留的整体漂移。

成绩数据本身并未证明这个假设。如果官方定数存在系统性的平均误差，约束也会掩盖它；
有结果的谱面集合变化时，公共偏移量可能变化，使独立估计未变的谱面也跟着调整。
设置 `balance_total: false` 可以保留参考谱面校准和噪声衰减，只查看独立估计。
诊断日志会同时显示独立估计和最终结果，便于评估这一步的影响。

例如，三张谱面的官方定数均为 `16`，独立估计分别为 `(16.2, 16.1, 16.0)`，
平均偏差为 `+0.1`。在没有触及上下限时，统一减去 `0.1` 得到 `(16.1, 16.0, 15.9)`，
保留相对差距，并使总和恢复为官方的 `48`。这说明了所选尺度锚点的作用，
不能据此证明第三张谱面在绝对意义上更简单；触及上下限时则需使用下面的投影。

完成**全部谱面**的独立计算后，求一个公共偏移量 $b$：

$$
d_c=\operatorname{clip}(d_c^0-b,\ell_c,u_c),\qquad
\sum_{c\in\mathcal P}d_c=0,\qquad\widehat L_c=L_c+d_c,
$$

其中 $\mathcal P$ 只包含有拟合结果的谱面，
$\ell_c=\max(-\Delta(L_c),0.1-L_c)$，$u_c=\min(\Delta(L_c),20-L_c)$。
用确定顺序的二分法求解这个带边界的最小二乘投影；每张谱面只计一次，不按成绩数量加权。
居中时重新应用上下限，避免原本已达上限的谱面越界。弃算谱面仍为 NULL。
总和约束只执行一次，不能逐数据库批次执行，因此批次大小不会改变定数。
它不要求上调、下调的数量相等，也不要求每首歌都偏离官方。

主观投票可用于验证结果，但不参与生产计算，也不作为某张谱面的目标定数。

计算器分批建立校准格统计，计算完整结果集、平衡总和后再分批写入数据库。
诊断中的均值、中位数、标准差和 MAD 描述收缩、噪声衰减及总和约束前的校准值；
`fitting_level` 保存最终输出。`sample_count` 统计通过最低分数门槛的原始成功反推数，
不是最终参与聚合的样本数。独立估计和最终输出均限制在 `[0.1,20]` 内。

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

这是相对于玩家 top-K 平均 rating 的原始观测。应先减去 §4.0 的同类谱面
背景残差，再做聚合；原始反推本身不能判断谱面是否偏难或偏易。

### 4.2 预权重

**邻近权重(不对称 · 硬截断)。** 玩家实力 $B_p$ 接近 $10L_c$ 时权重较大；
over-skilled 一侧使用更窄的带宽，限制远高于谱面目标实力的玩家对结果的影响。

$$
\sigma_{\text{eff}}(B_p) =
\begin{cases}
\alpha \cdot \sigma_{\text{prox}}, & B_p - 10L_c > 0,\\[4pt]
\sigma_{\text{prox}},               & B_p - 10L_c \le 0,
\end{cases}
\qquad
w^{\text{prox}}_{p,c} = \begin{cases}
\exp\!\left(-\dfrac{(B_p - 10L_c)^2}{2\,\sigma_{\text{eff}}(B_p)^{2}}\right), & \bigl|B_p - 10L_c\bigr| \le 2.5\,\sigma_{\text{eff}}(B_p),\\[8pt]
0\ (\text{样本丢弃,不计入}\ N^{\text{eff}}), & \bigl|B_p - 10L_c\bigr| > 2.5\,\sigma_{\text{eff}}(B_p).
\end{cases}
$$

默认 $\sigma_{\text{prox}} = 18.5$(对应 under-skilled 侧 $\pm 1.85$ 定数单位的实力带宽,over-skilled 侧 $\pm 0.37$),$\alpha = 0.2$。

**数据量权重。** 记录太少的玩家 $B_p$ 估计噪声较大。我们采用线性斜坡,在
$V_{\text{full}} = 50$ 条记录时饱和:

$$
w^{\text{vol}}_p = \min\!\left(1, \dfrac{n_p}{V_{\text{full}}}\right).
$$

**分数质量权重(可选,默认关闭)。** 一个仅仅 barely passed 某谱面(score 刚过 100 万)的样本与一个稳定稳在社区公认的 “高分”区间($\ge 1\,009\,000$)的样本,携带的信息并不等值。我们用三段分段线性斜坡把原始 score 映射到一个额外的权重因子 $w^{\text{score}}_{p,c} \in [0, 1]$:

$$
w^{\text{score}}_{p,c} = \begin{cases}
0, & s < s_{\text{floor}},\\[2pt]
w_{\text{good}} \cdot \dfrac{s - s_{\text{floor}}}{s_{\text{good}} - s_{\text{floor}}}, & s_{\text{floor}} \le s < s_{\text{good}},\\[10pt]
w_{\text{good}} + (1 - w_{\text{good}}) \cdot \dfrac{s - s_{\text{good}}}{s_{\text{full}} - s_{\text{good}}}, & s_{\text{good}} \le s < s_{\text{full}},\\[10pt]
1, & s \ge s_{\text{full}}.
\end{cases}
$$

推荐锚点(如需启用):$s_{\text{floor}} = 1{,}000{,}000$(业务定义的 “没真正过了”阈值),$s_{\text{good}} = 1{,}007{,}500$(“会打”阈值),$s_{\text{full}} = 1{,}009{,}000$(“高分”阈值),$w_{\text{good}} = 0.6$。**本系统默认将四个锚点全置为 0**,此因子退化为 $1$,等价于关闭本子节。`score_floor_at`、`score_good_at`、`score_full_at`、`score_good_weight` 任意一项全部置为 0 或错配时同样退化。

分数质量权重默认关闭；启用时，校准训练和目标推断使用相同的权重。

**时间权重(可选,默认关闭)。** 当 `sample_halflife_days` 为正数 $H$ 时，
对距 `play_records.record_time` 的天数 $a\ge0$ 使用 $w^{\text{age}}=2^{-a/H}$；
缺少时间或时间在未来时取 $a=0$。关闭时权重为 1。

**合成预权重:** $\tilde{w}_{p,c} = w^{\text{prox}}_{p,c} \cdot w^{\text{vol}}_p \cdot w^{\text{score}}_{p,c} \cdot w^{\text{age}}_{p,c}$。

### 4.3 鲁棒裁剪(Tukey 双权)

在预权重 $\{\tilde{w}_{p,c}\}$ 下,令 $\tilde{m}_c$、$\mathrm{MAD}_c$ 分别为
$\{\widetilde{\delta}_{p,c}\}$ 的**加权中位数**与**加权中位数绝对偏差**(同值时按
$\widetilde{\delta}$ 升序断开):

$$
\tilde{m}_c = \operatorname*{wmedian}_{p \in P_c}\widetilde{\delta}_{p,c};
\qquad
\mathrm{MAD}_c = \operatorname*{wmedian}_{p \in P_c}\bigl|\widetilde{\delta}_{p,c} - \tilde{m}_c\bigr|.
$$

对每条样本计算标准化残差

$$
u_{p,c} = \dfrac{\widetilde{\delta}_{p,c} - \tilde{m}_c}{h_c},\qquad
h_c=\begin{cases}
k\cdot\mathrm{MAD}_c, & k\cdot\mathrm{MAD}_c>10^{-9},\\
k\cdot0.01(|L_c|+1), & \text{否则}.
\end{cases}
$$

样本异常集中时使用上述下限保护，避免除零。再应用 Tukey 双权函数

$$
w^{\text{rob}}_{p,c} = \begin{cases}
(1 - u_{p,c}^{2})^{2}, & |u_{p,c}| < 1,\\[4pt]
0,                     & |u_{p,c}| \ge 1.
\end{cases}
$$

校准定数距加权中位数超过鲁棒尺度 $h_c$ 的样本在最终估计中的贡献为零。

### 4.4 聚合

定义最终合成权重 $w_{p,c} = \tilde{w}_{p,c}\cdot w^{\text{rob}}_{p,c}$。

**加权均值**(收缩前的估计量):

$$
\mu_c = \dfrac{\sum_p w_{p,c}\,\widetilde{\delta}_{p,c}}{\sum_p w_{p,c}}.
$$

**Kish 有效样本量**(当前加权方案等价于多少个"理想"无权样本):

$$
N^{\text{eff}}_c = \dfrac{\left(\sum_p w_{p,c}\right)^{2}}{\sum_p w_{p,c}^{2}}.
$$

当 $N^{\text{eff}}_c < \text{min\_samples}$(默认 5)时,**弃算**:
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
\qquad n_{\text{ref}} = \max(1,2\cdot\text{min\_samples}),
$$

当偏差为零或 $N^{\text{eff}}_c \gg n_{\text{ref}}$ 时 boost 接近 1,而当偏差大且样本少时 $\kappa_{\text{eff}}$ 二次放大、反比于 $N^{\text{eff}}_c$,恰好符合“偏得越多越需要证据”的直觉。默认 $\lambda = 2$。将 $\lambda = 0$ 即关闭偏差惩罚。

等价解读:“相信官方定数”的程度相当于 $\kappa_{\text{eff}}$ 个伪样本—— $N^{\text{eff}}_c \gg \kappa_{\text{eff}}$ 时估计值基本等于数据,$N^{\text{eff}}_c \ll \kappa_{\text{eff}}$ 时靠近官方定数;且偏差越大这条天秤越向官方端倾斜。

### 4.6 偏差上限(随定数变化的对数尺度斗形坡道)

作为防止模型错配的最终安全网,我们在收缩后施加硬截断:

$$
\hat{L}_c \leftarrow L_c + \operatorname{clip}\!\bigl(\hat{L}_c - L_c,\ -\Delta(L_c),\ \Delta(L_c)\bigr).
$$

低定数端使用较紧的上限，高定数端允许更大的调整。两个端点间用**对数线性**插值：

$$
\Delta(L) =
\begin{cases}
\Delta_{\min}, & L \le L_{\text{low}},\\[4pt]
\Delta_{\min}\cdot\left(\dfrac{\Delta_{\max}}{\Delta_{\min}}\right)^{t(L)}, & L_{\text{low}} < L < L_{\text{high}},\\[8pt]
\Delta_{\max}, & L \ge L_{\text{high}}.
\end{cases}
\qquad t(L) = \dfrac{L - L_{\text{low}}}{L_{\text{high}} - L_{\text{low}}}.
$$

默认值:$\Delta_{\min} = 0.15$,$\Delta_{\max} = 0.3$,$L_{\text{low}} = 12.0$,$L_{\text{high}} = 17.0$。中间点 $L = 14.5$ 有 $\Delta(14.5) = \sqrt{\Delta_{\min}\cdot\Delta_{\max}} \approx 0.212$。这些是最终输出上限，在噪声衰减后应用，并在全库居中时重新应用。

**退化规则。** 当 $\Delta_{\min} \le 0$,或 $L_{\text{low}}$/$L_{\text{high}}$ 配置不合法(参见 `internal/fitting/calculator.go:effectiveMaxDeviation`),实现静默回退到**平顶** $\Delta(L) \equiv \Delta_{\max}$;启动时的配置校验会拒绝最常见的误配置(见 `AGENTS.md`)。斗形坡道只影响上限的**宽窄**,不影响上限的**对称性**—— 在双向上都采用同一 $\Delta(L_c)$。

上限限制异常大的局部调整，§4.0 的总和约束另行控制整体尺度。α 与 κ 仍影响样本选择和收缩。上限不会要求每张谱面偏离官方。

## 5. 流水线总览

```
calibration := 按 §4.0 首遍建立每谱每实力格的背景统计
对每张官方定数为 L_c 的谱面 c:
    samples := { (p, s_{p,c}) : p ∈ P_c, s_{p,c} ≥ s_min }
    对每条样本:
        δ̂ := InverseRating(s_{p,c}, B_p)                      # §4.1
        若 δ̂ ∉ [0.1, 20.0] 则丢弃
        若启用校准：
            若满分或缺少其他谱面支持则丢弃
            δ̂ -= b_{c,k}
        diff  := B_p - 10·L_c                                  # §4.2
        σ_eff := (diff > 0 ? α·σ_prox : σ_prox)
        若 |diff| > 2.5·σ_eff 则丢弃                           #  ← 硬截断
        w_prox := exp(-diff² / (2·σ_eff²))
        w_vol  := min(1, n_p / V_full)
        w_pre  := w_prox * w_vol * score_quality * sample_age
    m_c  := weighted_median(δ̂; w_pre)                          # §4.3
    MAD  := weighted_median(|δ̂ - m_c|; w_pre)
    对每条样本:
        h := k·MAD 若 k·MAD > 1e-9，否则 k·0.01·(|L_c|+1)
        u := (δ̂ - m_c) / h
        w_rob := (1 - u²)²  若 |u| < 1,否则 0
        w     := w_pre * w_rob
    μ_c     := Σ w·δ̂ / Σ w                                    # §4.4
    N_eff_c := (Σ w)² / Σ w²
    若 N_eff_c < min_samples:
        标记 FittingLevel = NULL(诊断字段照常积累);继续
    dev     := μ_c - L_c                                       # §4.5
    n_ref   := max(1, 2 · min_samples)
    κ_eff   := κ · (1 + λ·dev²·n_ref / N_eff_c)
    L̂_c     := (N_eff_c·μ_c + κ_eff·L_c) / (N_eff_c + κ_eff)
    若启用校准:
        r := L̂_c - L_c; v := σ_c² / N_eff_c
        a := r² / (r² + τ·v)  (分母为零时取 1)
        L̂_c := L_c + a·r
    Δ       := effectiveMaxDeviation(L_c)                      # §4.6
    L̂_c     := L_c + clip(L̂_c - L_c, -Δ, Δ)
    若启用校准: L̂_c := clip(L̂_c, 0.1, 20)
    保存 (c, L̂_c, 诊断字段) 到完整结果集                          # 此时不写库

全部谱面完成计算后，若校准与 balance_total 均启用，按 §4.0 用一个公共偏移量平衡有效谱面的总和。
然后按 chart_batch_size 分批放进短事务统一落库(§7):
    一条 UPDATE ... FROM (VALUES ...) 批量更新 charts.fitting_level
    一条带冲突处理的批量 UPSERT 写入 chart_statistics
        (c, sample_count, N_eff_c, μ_c, m_c, σ_c, MAD, L̂_c, L_c, now)
```

## 6. 超参数(来自 `config.yaml`)

| 键                             | 符号                   | 默认值    | 作用                                                       |
|-------------------------------|------------------------|-----------|-----------------------------------------------------------|
| `fitting.skill_top_k` | K | `20` | 玩家实力所用最高单曲 rating 数。 |
| `fitting.calibration_enabled` | — | `true` | 启用同类谱面校准、满分与无支持样本筛除及噪声衰减。 |
| `fitting.calibration_noise_penalty` | τ | `1.0` | 有限非负的采样噪声惩罚系数；0 关闭。 |
| `fitting.balance_total` | — | `true` | 有效谱面的拟合总和匹配同一批谱面的官方总和；仅用于校准模式。 |
| `fitting.sample_halflife_days` | — | `0` | 可选成绩时间半衰期，0 关闭。 |
| `fitting.enabled`             | —                      | `true`    | 微服务总开关。                                             |
| `fitting.interval`            | —                      | `6h`      | Ticker 周期(Go duration 字符串)。                        |
| `fitting.min_samples`         | min_samples            | `5.0`     | $N^{\text{eff}}$ 低于此值则弃算。                          |
| `fitting.min_player_records`  | —                      | `20`      | 少于此记录数的玩家完全排除。                               |
| `fitting.proximity_sigma`     | $\sigma_{\text{prox}}$ | `18.5`    | 邻近权重高斯带宽(围绕 $10L_c$)。                         |
| `fitting.high_skill_sigma_ratio` | $\alpha$            | `0.2`     | over-skilled 一侧 σ 的缩放比(不对称高斯)。`1.0` 为对称高斯,更小值对大佬玩家折扣更重;样本离中心 2.5·σ 直接丢弃。 |
| `fitting.volume_full_at`      | $V_{\text{full}}$      | `50`      | 数据量权重饱和到 1 的临界记录数。                          |
| `fitting.prior_strength`      | $\kappa$               | `5.0`     | 官方定数的先验强度(收缩基准)。                         |
| `fitting.deviation_penalty`   | $\lambda$              | `2.0`     | 偏差惩罚;让 $\kappa_{\text{eff}} = \kappa(1+\lambda\cdot\text{dev}^2\cdot n_{\text{ref}}/N^{\text{eff}})$。`0` 时回退到静态 $\kappa$。 |
| `fitting.max_deviation`       | $\Delta_{\max}$        | `0.3`     | 高定数端(≥ $L_{\text{high}}$)最终上限;另作斗形坡道关闭时的平顶。 |
| `fitting.max_deviation_low`   | $\Delta_{\min}$        | `0.15`    | 低定数端(≤ $L_{\text{low}}$)最终上限;设为 0 即关闭斗形坡道,回退到平顶 $\Delta_{\max}$。 |
| `fitting.max_deviation_low_at`| $L_{\text{low}}$       | `12.0`    | cap 等于 $\Delta_{\min}$ 的端点;必须小于 $L_{\text{high}}$。 |
| `fitting.max_deviation_high_at`| $L_{\text{high}}$     | `17.0`    | cap 等于 $\Delta_{\max}$ 的端点;两端点之间用对数线性插值 $\Delta(L) = \Delta_{\min}\cdot(\Delta_{\max}/\Delta_{\min})^t$。 |
| `fitting.min_score`           | $s_{\min}$            | `500000`  | 成绩低于此阈值的样本直接丢弃。                             |
| `fitting.score_floor_at`      | $s_{\text{floor}}$    | `0`       | 分数质量权重(可选,默认关闭)。低于此分数的样本分数质量权重为 0(业务上的 “没真正过”线);置为 0 关闭分数权重。 |
| `fitting.score_good_at`       | $s_{\text{good}}$     | `0`       | 分数质量权重达到 $w_{\text{good}}$ 的锚点(“会打” 阈值);启用时建议 `1007500`。 |
| `fitting.score_full_at`       | $s_{\text{full}}$     | `0`       | 分数质量权重饱和到 1.0 的锚点(“高分” 阈值);启用时建议 `1009000`。 |
| `fitting.score_good_weight`   | $w_{\text{good}}$     | `0`       | 在 $s_{\text{good}}$ 处的权重值;启用时须在 $(0, 1)$ 内(典型值 `0.6`)。 |
| `fitting.tukey_k`             | $k$                    | `4.685`   | Tukey 双权调节常数。                                       |
| `fitting.chart_batch_size`    | —                      | `200`     | 每个数据库批次处理的谱面数(控制单次事务规模)。           |
| `fitting.player_batch_size`   | —                      | `500`     | 玩家实力分页时每页用户数(键集分页)。                     |
| `fitting.batch_pause`         | —                      | `50ms`    | 批次之间的暂停时间,用来缓解数据库压力(Go duration)。     |

## 7. 数据库写入与表结构

计算器共写入两处:

1. `charts.fitting_level`(`double precision`,可空)—— 发布的估计值 $\hat{L}_c$,
   弃算时写入 `NULL`。
2. `chart_statistics`(由 `cmd/fitting` 专属拥有)—— 每张谱面一行,主键为
   `chart_id`,保存流水线各阶段的诊断信息:`official_level`、`fitting_level`、
   `sample_count`、`effective_sample_size`、`weighted_mean`、`weighted_median`、
   `std_dev`、`mad`、`last_computed_at`,以及 `BaseModel` 标准时间戳。该表**不
   对外暴露 HTTP 路由**,仅用于内部分析。

为把对在线查分服务的影响降到最低,我们遵循以下策略:

- 玩家实力以 distinct `username` 为键做**键集分页**(每批 `player_batch_size`),
  不使用 OFFSET 扫描。
- 谱面按固定批 `chart_batch_size` 处理,批次间插入 `batch_pause` 短暂休眠。
- 每批谱面使用一个短事务落库:`charts.fitting_level` 由一条
  `UPDATE ... FROM (VALUES ...)` 派生表语句批量更新,`chart_statistics` 由
  一条带冲突处理的批量 upsert 写入。
- 查分服务的缓存不会被主动失效；相关缓存过期或失效后，读取到新写入的值。

## 8. 运维指南

```bash
# 持续模式(默认,遵循 fitting.interval)
go run ./cmd/fitting -config config/config.yaml

# 一次性模式(适合 cron、调试、CI 冒烟测试)
go run ./cmd/fitting --once -config config/config.yaml

# 诊断某张谱面(只读,不写库)
go run ./cmd/fitting analyze -chart 870 -config config/config.yaml
```

两个子命令均遵循共享的 `logging.output` / `logging.format` 设置。
启动日志打印实际生效的关键拟合配置，包括 `skill_top_k`、校准与总量约束开关、
样本门槛，以及 `max_deviation` / `max_deviation_low` 和对应的定数区间端点。
诊断结果以结构化日志输出分数桶、统计量、总体总和及估计值；JSON 日志保留数值类型，
无法提供的定数输出为 null。

`analyze` 使用 `ConnectDB` 查询已有结构，不执行迁移；`run` 和查分服务使用 `InitDB`。
`AutoMigrate` 虽然幂等，但在表、列或索引与模型不一致时仍可能执行 DDL。
跳过迁移可避免诊断改变数据库结构，也允许使用只读数据库账号；诊断前应由其他启动或
迁移流程准备好数据库结构。

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

1. 官方定数同时决定 rating 和校准参照轴。如果某一整组官方定数系统性偏差，局部校准无法识别绝对难度漂移。B20 还包含目标成绩，本方法不是完全独立的玩家实力估计。
2. 最佳成绩包含反复练习和选曲自选择；单个实力数值无法表达偏科。鲁棒聚合不能消除这些系统误差。
3. `sample_halflife_days` 默认 0，旧成绩与新成绩同权；可开启半衰期，但需重新验证结果。
4. 满分、缺少其他谱面支持或目标有效样本不足时弃算。验证时应同时检查发布范围与误差。
5. 多个读取批次不构成一致性快照；持久化期间取消或写入失败可能只提交部分已平衡结果。数据库总和约束要求一次完整成功的运行。估计阶段失败时不会写入任何结果。
