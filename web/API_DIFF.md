# API v1 → v2 差异对照表

> 本文档整理了旧版前端（`web/legacy/`，基于 `/api/v1`）与新版后端 API（`/api/v2`，基于 `docs/swagger.json`）之间的差异，供后续审阅。

---

## 1. 基础变更

| 项目 | v1 (Legacy) | v2 (New) |
|------|-------------|----------|
| Base URL | `/api/v1` | `/api/v2` |
| Host | `api.prp.icel.site` | `api.prp.icel.site` |

---

## 2. 数据模型重命名

这是最大的 breaking change，贯穿所有 API：

| v1 字段/模型 | v2 字段/模型 | 说明 |
|-------------|-------------|------|
| `song_level_id` | `id` (on Chart/ChartInfo/ChartInfoSimple objects; `chart_id` retained as FK) | 谱面唯一 ID |
| `song_levels` (数组) | `charts` (数组) | 歌曲的谱面列表 |
| `SongLevel` | `Chart` / `ChartInfo` / `ChartInfoSimple` | 谱面模型 |
| `difficulty_id` (number: 1,2,3,4) | `difficulty` (string: `"detected"`, `"invaded"`, `"massive"`, `"reboot"`) | 难度表示方式从数字改为字符串枚举 |
| `song_level.title` | `chart.title` (在 `ChartInfo`/`ChartInfoSimple` 中扁平化) | 谱面关联的曲名 |
| `song_level.b15` | `chart.b15` | 新旧版本标记 |

### 影响范围
- 所有涉及谱面 ID 的 API 请求和响应
- 前端所有引用 `song_level_id` 的地方需改为 `id`（主键）或 `chart_id`（外键）
- 难度判断从 `difficulty_id === 1/2/3/4` 改为 `difficulty === 'detected'/'invaded'/'massive'/'reboot'`

---

## 3. 用户相关 API

### 3.1 登录 — 无变化
```
POST /user/login
Content-Type: application/x-www-form-urlencoded
```
请求和响应格式不变。

### 3.2 注册 — 无变化
```
POST /user/register
Content-Type: application/json
```
请求和响应格式不变。

### 3.3 更新用户信息 — HTTP Method 变更

| | v1 | v2 |
|---|---|---|
| Method | `PATCH` | `PUT` |
| Path | `/user/me` | `/user/me` |

### 3.4 新增：修改密码 ⭐
```
PUT /user/me/password
```
v1 中 `ChangePasswordForm.vue` 是空的 stub，v2 正式提供了此接口。

请求体：
```json
{
  "old_password": "string",
  "new_password": "string (min 6)"
}
```

### 3.5 新增：管理员重置密码 ⭐
```
POST /user/reset-password  (Admin only)
```
v1 中不存在此功能。

请求体：
```json
{
  "username": "string",
  "new_password": "string (min 6)"
}
```

---

## 4. 歌曲相关 API

### 4.1 获取所有谱面

| | v1 | v2 |
|---|---|---|
| Path | `GET /songs` | `GET /songs` |
| 响应类型 | `SongLevel[]` (扁平列表) | `ChartInfo[]` (扁平列表) |
| 难度字段 | `difficulty_id: number` | `difficulty: string` |
| 谱面 ID 字段 | `song_level_id` | `id` |

### 4.2 获取单曲信息 — 新增 `src` 参数

| | v1 | v2 |
|---|---|---|
| Path | `GET /songs/:song_id` | `GET /songs/:song_id?src=prp\|wiki` |
| 子谱面字段 | `song_levels: SongLevel[]` | `charts: Chart[]` |

v2 支持通过 `src=wiki` 按 wiki_id 查找歌曲。

### 4.3 更新歌曲 — HTTP Method 变更

| | v1 | v2 |
|---|---|---|
| Method | `PATCH` | `PUT` |
| 请求体中谱面 ID | `song_level_id` | `song_id` (required) |

---

## 5. 成绩记录 API — 变化最大

### 5.1 获取成绩 — scope 参数扩展

| | v1 | v2 |
|---|---|---|
| Path | `GET /records/:username` | `GET /records/:username` |
| scope 取值 | `best`, `all` | `b50` (默认), `best`, `all`, `all-charts` |
| 获取 B50 | `GET /records/:username?best=true` | `GET /records/:username?scope=b50` |
| 默认排序 | `sort_by=date` | `sort_by=rating` |
| 新增参数 | — | `underflow` (b50 溢出数量) |

**重要**：v1 使用 `best=true` 参数获取最佳记录，v2 统一使用 `scope` 参数。

### 5.2 获取成绩 — 响应结构变更

v1 响应中的 record 包含 `song_level` 子对象：
```json
{
  "records": [{
    "score": 1000000,
    "rating": 1500,
    "song_level": {
      "song_level_id": 1,
      "title": "Song",
      "difficulty_id": 1,
      "difficulty": "Detected",
      "level": 13.5,
      "b15": false
    }
  }]
}
```

v2 响应中使用 `chart` 子对象（`ChartInfoSimple`）：
```json
{
  "username": "user",
  "total": 50,
  "records": [{
    "id": 1,
    "score": 1000000,
    "rating": 1500,
    "record_time": "2024-01-01T00:00:00Z",
    "chart": {
      "id": 1,
      "song_id": 1,
      "title": "Song",
      "difficulty": "detected",
      "level": 13.5,
      "fitting_level": 13.4,
      "b15": false,
      "cover": "cover.png",
      "version": "1.0",
      "wiki_id": "w1"
    }
  }]
}
```

### 5.3 上传成绩 — 字段重命名

| | v1 | v2 |
|---|---|---|
| 谱面 ID 字段 | `song_level_id` | `chart_id` |
| 新增字段 | — | `upload_token` (支持第三方上传) |

v1 请求体：
```json
{
  "is_replace": false,
  "play_records": [{ "song_level_id": 1, "score": 1000000 }]
}
```

v2 请求体：
```json
{
  "is_replace": false,
  "play_records": [{ "chart_id": 1, "score": 1000000 }],
  "upload_token": "optional-token",
  "csv_filename": "optional-filename"
}
```

---

## 6. Swagger 中缺失的接口

以下接口在 v1 前端中使用，但在 v2 的 Swagger 文档中 **未出现**（可能尚未实现或未添加注解）：

| 功能 | v1 路径 | 状态 |
|------|---------|------|
| 导出 CSV | `GET /records/:username/export/csv` | ❌ 已废除（改为前端本地渲染） |
| 导出 B50 图片 | `GET /records/:username/export/b50` | ❌ 已废除（改为前端本地渲染） |
| B50 趋势数据 | `GET /statistics/:username/b50` | ⚠️ 未在 Swagger 中出现（v1 路径与 v2 可能不同） |

> **说明**：CSV 导出和 B50 图片导出已废除服务端实现，改为前端本地渲染。

---

## 7. 其他差异

| 项目 | v1 | v2 |
|------|-----|-----|
| Rating 存储 | 部分接口返回已除以 100 的值，部分返回原始值（前端需手动 `/100`） | 统一返回 `int` (rating × 100)，前端统一 `/ 100` 显示 |
| 升序参数 | `order=asce` | `order=asce` (不变) |
| 认证头 | `Authorization: Bearer <token>` | `Authorization: Bearer <token>` (不变) |
| 上传 CSV 路径 | `POST /upload/csv` | ❌ 已废除 |
| 上传图片路径 | `POST /upload/img` | ❌ 已废除 |

---

## 8. 前端适配总结

新版前端已完成以下适配：

- ✅ 全部 `song_level_id` → `id`（主键字段）/ `chart_id`（外键引用）
- ✅ 全部 `difficulty_id` (数字) → `difficulty` (字符串)
- ✅ `PATCH` → `PUT`（用户更新、歌曲更新）
- ✅ B50 获取方式从 `best=true` 改为 `scope=b50`
- ✅ 新增修改密码接口支持
- ✅ `upload_token` 第三方上传支持
- ⚠️ CSV 导出、B50 图片导出、B50 趋势图 — 已预留代码，待后端实现后验证
