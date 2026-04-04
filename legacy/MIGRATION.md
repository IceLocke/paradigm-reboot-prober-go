# Legacy → V2 数据库迁移指南

本文档描述如何将旧版 (Python/FastAPI) 数据库迁移到新版 (Go/Gin) schema。

## 迁移概览

### Schema 变更摘要

| 变更项 | 旧版 (Legacy) | 新版 (V2) |
|--------|--------------|-----------|
| 难度存储 | 独立 `difficulties` 查找表 (FK) | 内联 `varchar(20)` 字段于 `charts` 表 |
| 谱面表名 | `song_levels` (PK: `song_level_id`) | `charts` (PK: `chart_id`) |
| play_records 谱面外键 | `song_level_id` | `chart_id` |
| Rating 类型 | `double precision` (如 16.4) | `integer` (如 1640, ×100) |
| best_play_records | 仅 `best_record_id` + `play_record_id` | 新增 `username` + `chart_id` + 组合唯一索引 |
| songs.wiki_id | 可为 NULL | NOT NULL (必填) |
| best50_trend 表 | 存在 | 归档为 `_legacy_best50_trend` |
| upload_token 索引 | 无唯一索引 | 新增唯一索引 |

### 迁移步骤

1. **预检查** — 验证旧表存在、新表不存在、难度名称合法
2. **归档 `best50_trend`** — 重命名为 `_legacy_best50_trend`（新 schema 无此表）
3. **修复 `songs.wiki_id`** — 为 NULL 值生成占位符 (`__song_<id>`)，添加 NOT NULL 约束
4. **创建 `charts` 表** — 从 `song_levels` JOIN `difficulties` 迁移，内联难度名称
5. **迁移 `play_records`** — 列重命名 `song_level_id` → `chart_id`，rating 从浮点转整数
6. **迁移 `best_play_records`** — 新增 `username` + `chart_id` 列，创建组合唯一索引
7. **创建新索引** — `upload_token` 唯一索引等
8. **清理旧表** — 删除 `song_levels`、`difficulties` 及相关序列
9. **验证** — 检查数据完整性、孤立记录

## 前置条件

- PostgreSQL 14+
- **务必备份数据库！**
- 迁移脚本在单个事务中运行，任何错误将自动回滚

```bash
# 备份数据库
pg_dump -h <host> -U <user> -d <dbname> > backup_$(date +%Y%m%d_%H%M%S).sql
```

## 执行方式

### 方式一：直接使用 psql

```bash
psql -h <host> -U <user> -d <dbname> -f legacy/migration.sql
```

### 方式二：使用 Go CLI 工具

```bash
# 使用默认配置文件 (config/config.yaml)
go run cmd/migrate/main.go

# 指定配置文件
go run cmd/migrate/main.go -config /path/to/config.yaml

# 指定 SQL 文件
go run cmd/migrate/main.go -sql-file /path/to/migration.sql

# 预览模式（只打印 SQL，不执行）
go run cmd/migrate/main.go -dry-run
```

Go CLI 工具会自动读取项目配置文件中的数据库连接信息。

### 方式三：通过环境变量

```bash
DB_TYPE=postgres DB_HOST=localhost DB_PORT=5432 \
DB_USER=postgres DB_PASSWORD=secret DB_NAME=prober DB_SSLMODE=disable \
go run cmd/migrate/main.go
```

## 迁移后操作

1. **启动 Go 服务器** — GORM `AutoMigrate` 会自动应用剩余的微调（如列类型精确匹配）

   ```bash
   go run cmd/server/main.go
   ```

2. **验证应用** — 测试核心功能是否正常：
   - 用户登录
   - 查看歌曲列表
   - 查询成绩记录
   - B50 计算

3. **清理归档表**（可选） — 确认迁移成功后，可删除归档表：

   ```sql
   DROP TABLE IF EXISTS _legacy_best50_trend;
   ```

4. **检查 wiki_id 占位符** — 搜索并更新由迁移自动生成的占位符值：

   ```sql
   SELECT song_id, wiki_id, title FROM songs WHERE wiki_id LIKE '__song_%';
   ```

## 数据转换细节

### Rating 转换

旧版存储 `rating` 为 `double precision`，但实际值已经是 ×100 的整数（如 `16546.0` 代表 rating 165.46）。
新版存储 `rating` 为 `integer`（如 `16546`），语义相同，仅类型不同。

转换公式（与 Go 代码 `pkg/rating/rating.go` 中的 EPS 一致）：

```sql
CAST(FLOOR(rating * 100 + 0.00002) AS INTEGER)
```

### 难度映射

旧版 `difficulties` 表中的 `name` 值直接转为小写字符串：

| difficulty_id | name (legacy) | difficulty (new) |
|--------------|---------------|-----------------|
| 1 | Detected / detected | `detected` |
| 2 | Invaded / invaded | `invaded` |
| 3 | Massive / massive | `massive` |
| 4 | Reboot / reboot | `reboot` |

### ID 保留策略

- `song_levels.song_level_id` → `charts.chart_id`：**ID 值保持不变**
- `play_records.play_record_id`：**ID 值保持不变**
- `best_play_records.best_record_id`：**ID 值保持不变**

这确保了所有跨表引用在迁移后依然有效。

## 回滚

如果迁移失败，事务会自动回滚，无需手动干预。

如果迁移已成功但需要回滚，请从备份恢复：

```bash
psql -h <host> -U <user> -d <dbname> < backup_YYYYMMDD_HHMMSS.sql
```

## 故障排除

| 错误 | 原因 | 解决方案 |
|------|------|---------|
| `Legacy table "song_levels" not found` | 数据库不是旧版 schema | 确认连接到正确的数据库 |
| `Table "charts" already exists` | 迁移已执行过 | 如需重新迁移，从备份恢复后重试 |
| `Unknown difficulty names found` | difficulties 表中有未知名称 | 检查并修正 difficulties 表数据 |
| `Migration only supports PostgreSQL` | 配置了非 PostgreSQL 数据库 | 迁移仅支持 PostgreSQL |
