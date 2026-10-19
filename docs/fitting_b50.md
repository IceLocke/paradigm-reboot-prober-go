# 拟合定数 B50 的查询语义

B50 必须先确定排名依据，再从玩家的全部最佳成绩中选取旧曲 B35 和新曲 B15。
例如某成绩按官方定数排第 36，改用拟合定数后排第 30，它就应该进入拟合 B35。
只对已经选出的官方 B50 重算 Rating 会漏掉这类成绩。

PR #119 的前端原先通过分页读取全部最佳成绩完成重选，能够改变入选名单，
但后端 B50 接口仍只提供官方排名，其他 API 客户端无法直接获得拟合 B50。
现在将这套语义统一到后端，页面和图片导出都使用后端选出的结果。

## API

沿用现有 `GET /api/v2/records/{username}`，不新增路由：

| 请求 | 语义 |
| --- | --- |
| `scope=b50` | 默认按官方 `rating` 选 B35 + B15 |
| `scope=b50&sort_by=fitting_rating` | 按拟合 Rating 重新选 B35 + B15 |
| `scope=b50&sort_by=fitting_rating&underflow=5` | 按拟合 Rating 选旧曲前 40、新曲前 20，用于溢出导出 |
| `scope=best&sort_by=fitting_rating&page_size=50&page_index=1` | 不分新旧曲，按拟合 Rating 选全局前 50 |

`sort_by` 表示排名依据，`order=asc/desc` 仍表示排序方向，默认 `desc`。
B50 始终先取最高的若干条，`order=asc` 只反转每个分组内的显示顺序；
`best` 则先按请求方向排完整候选集，再分页。
整数 Rating 相同时用成绩 ID 排序，保证分页和边界选择稳定。
官方和拟合排名共享有效候选查询，跳过已软删除的最佳成绩、游玩记录、谱面或歌曲，计数也使用同一条件。

响应中的 `rating` 保持官方 Rating，新增 `fitting_rating` 返回拟合 Rating，均为整数（×100）。
缺少拟合定数时按官方定数计算；定数为 `0` 不视为缺失。
`chart.level`、`chart.fitting_level` 均保留各自含义。
现有定数范围筛选仍作用于官方 `chart.level`，所有筛选先于排名和截取。

拟合排序仅支持 `b50` 和 `best`；`all`、`all-charts` 请求拟合排序返回 400。
这使完整候选集限于每个玩家每个谱面一条最佳成绩，无需扫描该玩家无限增长的历史成绩。
`b50` 的其他排序键沿用原有行为，使用官方 Rating。

## 数据库和运行边界

`best_play_records` 继续指向每个谱面的最高分成绩。对固定谱面定数，Rating 随分数单调变化，
所以切换官方/拟合定数不需要第二张最佳成绩表，也不需要重建最佳成绩指针。

`play_records.rating` 继续只保存官方 Rating。拟合查询一次读取该玩家所有最佳成绩及其
`charts.fitting_level`，调用现有 `pkg/rating.SingleRating` 计算，完成排序后才做分组截取或分页。
这避免复制公式到 PostgreSQL/SQLite 的不同 SQL 实现，也避免拟合任务每次发布后重写所有历史成绩。
候选数为 N 时，内存为 O(N)，排序为 O(N log N)；若未来单人谱面数量明显增大，可进一步评估堆选取。

拟合 B50 不使用进程内结果缓存，因此独立 fitting 进程发布新定数后，下次拟合查询即可使用。
官方 B50 保留原缓存，升序和降序使用不同缓存键。普通响应里的拟合定数与拟合 Rating
跟随该响应所使用的谱面数据快照，已有缓存读取仍受原 TTL 影响。

本次变更无需数据库迁移，不持久化 `fitting_rating`，不导入 `internal/fitting`，
不访问 `chart_statistics`，不在 API 进程运行拟合任务。
