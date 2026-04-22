# AGENTS.md

## Project Overview

**Paradigm: Reboot Prober** (`paradigm-reboot-prober-go`) is a backend REST API service with a Vue 3 frontend for the rhythm game **Paradigm: Reboot**. It provides score tracking, rating calculation, and Best 50 (B50) computation. The project is a Go rewrite (WIP) of a previous Python implementation.

Core features:
- **User Management**: Registration, JWT authentication, profile updates, upload tokens, password change/reset.
- **Song Management**: CRUD for songs and their difficulty charts (admin-only for create/update).
- **Score/Record Management**: Batch upload play records (JSON), automatic single-chart rating calculation, best record tracking.
- **B50 Calculation**: Best 35 (old songs, `b15=false`) + Best 15 (new songs, `b15=true`) selection.
- **Web Frontend**: Dark-themed Vue 3 + TypeScript + Naive UI single-page application.
- **API Documentation**: Swagger UI auto-generated from code annotations.
- **Fitting Level Microservice** (`cmd/fitting`): Offline calculator that derives `charts.fitting_level` from `best_play_records` on a configurable interval. Kept as a **separate binary** to honour the “保持查分器本体的单纯性” principle (see below). See `docs/fitting_level.en.md` (English) / `docs/fitting_level.zh.md` (中文) for the full math.

## Design Principle: 保持查分器本体的单纯性

The **probe service** (“查分器本体” = `cmd/server` + its direct dependency graph: controllers, services, repositories, middleware, router) must stay **single-purpose**: it receives score uploads, serves record/B50 queries, and nothing else. Any derived/analytical computation that is not strictly required to answer a live API request lives in a **separate binary** under `cmd/` with its own lifecycle and config namespace.

Concretely this means:

- `cmd/server` must **not** import `internal/fitting`, must not call fitting code, and must not read `config.fitting.*` at request time. The only shared artefact is the DB schema migration of `chart_statistics` (added to `util.InitDB()` so whichever binary starts first keeps the schema consistent).
- Fitting results are delivered to the probe service **only via the `charts.fitting_level` column** that the existing `ChartInfo` / `ChartInfoSimple` DTOs already expose. No new HTTP surface, no background goroutine in the server process.
- New analytics tables (currently `chart_statistics`) are owned by the analytical binary and are **not** exposed through any API route. If an endpoint is later needed, expose it from a separate read-only service, not by stapling it onto `cmd/server`.
- When adding a new analytical workload (e.g. player clustering, chart similarity), create a new `cmd/<name>` binary and a new `internal/<name>` package. Do **not** collocate it with controllers/services/repositories of the probe service.

This keeps the probe hot-path small, makes its memory / latency footprint predictable, and lets the analytical jobs be scaled, paused, or replaced independently.

Go module: `paradigm-reboot-prober-go`

Repository: `github.com/IceLocke/paradigm-reboot-prober-go`

## Tech Stack

| Component      | Technology                                                    |
|----------------|---------------------------------------------------------------|
| Language       | Go 1.25 (toolchain go1.25.5)                                 |
| Web Framework  | [Gin](https://github.com/gin-gonic/gin) + gin-contrib/cors + gin-contrib/gzip |
| ORM            | [GORM](https://gorm.io/)                                     |
| Database       | PostgreSQL (production), SQLite (dev/testing)                 |
| Authentication | JWT (HS256) via `golang-jwt/jwt/v5`, bcrypt                   |
| API Docs       | Swagger via `swaggo/swag` + `swaggo/gin-swagger`              |
| Testing        | `testing` stdlib + `stretchr/testify`                         |
| Caching        | `jellydator/ttlcache/v3` (in-process, per-repository)          |
| Linting        | golangci-lint v2.6                                            |
| CI/CD          | GitHub Actions                                                |
| Container      | Docker (multi-stage Alpine build)                             |
| Orchestration  | Docker Compose (`db` + `app` by default, `fitting` via `--profile fitting`) |
| Frontend       | Vue 3 + TypeScript + Vite + Naive UI (in `web/`), pako (gzip request body) |
| Frontend Lint  | ESLint 10 + typescript-eslint + eslint-plugin-vue              |

## Project Structure

```
.
├── cmd/
│   ├── server/
│   │   └── main.go              # Application entry point, Swagger annotations
│   ├── fitting/
│   │   └── main.go              # Fitting-level microservice (separate binary)
│   └── migrate/
│       ├── main.go              # Legacy → Go schema migration tool (PostgreSQL)
│       └── verify/
│           └── main.go          # Post-migration verification tool
├── config/
│   ├── config.go                # Config struct, YAML loading, env var overrides
│   ├── config.yaml              # Local configuration file (gitignored for secrets)
│   └── config.yaml.example      # Example configuration (safe to commit)
├── internal/
│   ├── controller/              # HTTP handlers (Gin handlers with Swagger annotations)
│   │   ├── user.go              # Register, Login, GetMe, UpdateMe, RefreshUploadToken, ChangePassword, ResetPassword, RefreshToken
│   │   ├── song.go              # GetAllCharts, GetSingleSongInfo, CreateSong, UpdateSong
│   │   └── record.go            # GetPlayRecords, GetSongRecords, GetChartRecords, UploadRecords
│   ├── fitting/                 # Fitting-level calculator library (used ONLY by cmd/fitting)
│   │   ├── inverter.go          # Closed-form inverse of pkg/rating.SingleRating
│   │   ├── calculator.go        # Weighting + robust aggregation + shrinkage + deviation cap
│   │   ├── player_skill.go      # Per-player B50 mean rating collection (keyset pagination)
│   │   └── runner.go            # Orchestrator: load → batch-process charts → persist
│   ├── logging/                 # Structured logging infrastructure (slog + context)
│   │   ├── context.go           # AppendCtx helper, context key for slog attrs
│   │   ├── handler.go           # ContextHandler wrapping slog.Handler
│   │   └── setup.go             # Global logger initialization (TextHandler / JSONHandler, file/stdout/stderr output)
│   ├── metrics/                 # Prometheus metrics registry, Gin middleware, and /metrics handler
│   │   └── metrics.go           # http_requests_total / _duration / _size / _in_flight, route-template `path` label, prefix-based exclusion
│   ├── middleware/
│   │   ├── auth.go              # AuthMiddleware, OptionalAuthMiddleware, AdminMiddleware
│   │   ├── gzip.go              # GzipResponseMiddleware (compress responses), GzipRequestMiddleware (decompress request bodies)
│   │   ├── logging.go           # RequestIDMiddleware, SlogRequestMiddleware
│   │   └── ratelimit.go         # Per-IP token bucket rate limiter
│   ├── model/                   # Data models (GORM entities + DTOs)
│   │   ├── base.go              # BaseModel (CreatedAt, UpdatedAt, DeletedAt — embedded by all GORM entities)
│   │   ├── user.go              # User, UserBase, UserInDB, UserPublic
│   │   ├── song.go              # Song, SongBase, Difficulty enum, Chart, ChartInfo, ChartInfoSimple, ChartCSV, ChartWithScore, ChartInput
│   │   ├── play_record.go       # PlayRecord, BestPlayRecord, PlayRecordBase, PlayRecordInfo, PlayRecordResponse, AllChartsResponse, ToPlayRecordInfo()
│   │   ├── chart_statistic.go   # ChartStatistic (owned by cmd/fitting; one row per chart)
│   │   ├── auth.go              # Token, UploadToken (access_token + refresh_token)
│   │   ├── common.go            # Response (generic error/message response)
│   │   └── request/             # Request DTOs
│   │       ├── user.go          # CreateUserRequest, UpdateUserRequest, ChangePasswordRequest, ResetPasswordRequest, RefreshTokenRequest
│   │       ├── song.go          # CreateSongRequest, UpdateSongRequest
│   │       └── play_record.go   # BatchCreatePlayRecordRequest
│   ├── repository/              # Database access layer (GORM queries) + in-process cache
│   │   ├── cache.go             # Cache helpers: prefix invalidation, filter key, TTL constants
│   │   ├── user_repo.go
│   │   ├── song_repo.go
│   │   └── record_repo.go       # Includes rating calculation on record creation
│   ├── router/
│   │   └── router.go            # Route definitions, dependency wiring, CORS, middleware setup
│   ├── service/                  # Business logic layer (all methods accept context.Context)
│   │   ├── errors.go            # Sentinel errors (ErrNotFound, ErrForbidden, ErrUnauthorized)
│   │   ├── user.go
│   │   ├── song.go
│   │   └── record.go
│   └── util/
│       ├── database.go          # DB initialization (SQLite/PostgreSQL), auto-migration
│       └── csv.go               # CSV generation, parsing (UTF-8 BOM, GBK decoding), empty CSV template
├── pkg/                          # Shared/reusable packages
│   ├── auth/
│   │   └── auth.go              # Password hashing (bcrypt), JWT generation/validation
│   └── rating/
│       └── rating.go            # Single-chart rating calculation algorithm
├── docs/                         # Auto-generated Swagger docs (do NOT edit manually) + hand-written design docs
│   ├── docs.go
│   ├── swagger.json
│   ├── swagger.yaml
│   ├── fitting_level.en.md      # Mathematical specification of the fitting-level calculator (English)
│   └── fitting_level.zh.md      # Mathematical specification of the fitting-level calculator (中文)
├── web/                          # Vue 3 frontend (has its own AGENTS.md)
│   ├── src/                     # Vue components, stores, utils, styles
│   ├── public/                  # Static assets (Git submodule → prp-resource)
│   ├── dist/                    # Build output
│   ├── styles/                  # Frontend style documentation
│   ├── AGENTS.md                # Frontend-specific agent instructions
│   └── API_DIFF.md              # v1 → v2 API migration reference
├── legacy/                       # Legacy migration resources
│   ├── MIGRATION.md             # Step-by-step migration guide
│   ├── migration.sql            # Schema migration SQL (Python → Go)
│   ├── db_schema.sql            # Legacy database schema
│   ├── db_full.sql              # Legacy full database dump
│   └── openapi.json             # Legacy OpenAPI specification
├── scripts/
│   └── setup-submodules.sh      # Git submodule setup (private repos with GH_TOKEN)
├── .github/workflows/
│   └── ci.yml                   # CI/CD pipeline
├── .claude/                      # Claude AI settings
├── Dockerfile                   # Multi-stage Docker build
├── docker-compose.yaml          # App + PostgreSQL compose setup
├── .golangci.yml                # Linter configuration
├── go.mod / go.sum              # Go module files
└── AGENTS.md                    # This file
```

## Architecture

The application follows a **layered architecture**:

```
Request → Router → RequestID → SlogRequest → Metrics → CORS → Gzip → RateLimit → Auth → Controller → Service → Repository → Database
```

- **Router** (`internal/router/`): Registers all routes, sets up CORS and middleware, wires dependencies (manual DI, no framework). Uses `gin.New()` (not `gin.Default()`) with explicit middleware chain.
- **RequestID** (`internal/middleware/logging.go`): `RequestIDMiddleware` generates a random 8-byte hex request ID (or reuses `X-Request-ID` header), injects it into slog context and response header.
- **SlogRequest** (`internal/middleware/logging.go`): `SlogRequestMiddleware(excludePrefixes)` enriches context with `method`, `path`, `client_ip`; logs a request-completed summary with status, latency, and bytes. WARN for 4xx, ERROR for 5xx. Paths starting with any prefix in `logging.exclude_paths` (e.g. `/healthz`) skip only the final log line — context fields and request ID header are still applied.
- **Metrics** (`internal/metrics/metrics.go`): `metrics.Middleware(excludePrefixes)` records `http_requests_total`, `http_request_duration_seconds`, `http_response_size_bytes` (all labelled by `method`, `path`, `status`) and `http_requests_in_flight`. The `path` label uses Gin's matched route template (`c.FullPath()`, e.g. `/api/v2/records/:username`) to keep cardinality bounded; unmatched routes are reported as `path="unknown"`. Paths matching `metrics.exclude_paths` (default `/healthz`) are not observed. Metrics are NOT served on the main API port — a dedicated `http.Server` started in `cmd/server/main.go` exposes `promhttp.Handler()` on `metrics.addr` (default `:9090`) at `metrics.path` (default `/metrics`).
- **CORS**: Configured via `gin-contrib/cors` — allows all origins, standard methods and headers.
- **Gzip** (`internal/middleware/`): `GzipRequestMiddleware` transparently decompresses `Content-Encoding: gzip` request bodies; `GzipResponseMiddleware` (via `gin-contrib/gzip`) compresses responses when the client sends `Accept-Encoding: gzip`.
- **RateLimit** (`internal/middleware/ratelimit.go`): Per-IP token bucket rate limiter applied to login and registration endpoints.
- **Auth** (`internal/middleware/`): JWT auth extraction (`AuthMiddleware`, `OptionalAuthMiddleware`), admin role check (`AdminMiddleware(userService)`). Both `AuthMiddleware` and `OptionalAuthMiddleware` reject refresh tokens (only access tokens or legacy tokens without a type claim are accepted). Injects `username` into slog context on successful authentication.
- **Controller** (`internal/controller/`): Handles HTTP request/response, input validation, delegates to services. Injects business-specific fields (e.g. `target_user`, `scope`, `song_id`) into context via `logging.AppendCtx` before calling services.
- **Service** (`internal/service/`): Business logic. All public methods accept `context.Context` as their first parameter. Uses `slog.InfoContext`/`WarnContext`/`ErrorContext` for automatic inclusion of upstream context fields (request ID, HTTP metadata, username, business fields). Orchestrates repository calls.
- **Repository** (`internal/repository/`): Direct database operations via GORM. Rating calculation happens here when creating records. Each repository embeds an in-process `go-cache` instance for cache-aside (read-through, invalidate-on-write).
- **Model** (`internal/model/`): GORM entities (with table name overrides) and DTOs. All GORM entities embed `BaseModel` for audit timestamps and soft-delete support. Request-specific DTOs live in `internal/model/request/`.
- **Pkg** (`pkg/`): Reusable, domain-specific packages — `auth` (password hashing, JWT generation for access/refresh tokens, token type extraction) and `rating` (score-to-rating calculation).

### In-Process Caching

The repository layer implements a **cache-aside** pattern using [`jellydator/ttlcache/v3`](https://github.com/jellydator/ttlcache), a generic in-process key-value cache with TTL-based expiration and automatic cleanup. Each repository struct owns its own `*ttlcache.Cache[string, any]` instance — no shared state between repositories, no external dependencies like Redis.

**Cache configuration** (defined in `internal/repository/cache.go`):

| Repository | Default TTL | Cleanup Interval | Rationale |
|------------|-------------|-------------------|-----------|
| `SongRepository` | 10 min | 15 min | Song data changes extremely rarely (admin-only mutations) |
| `UserRepository` | 5 min | 10 min | User data changes on profile update, password change |
| `RecordRepository` | 5 min | 10 min | Record data changes on score upload |

**Cached operations**:

| Repository | Cached Methods | Cache Key Pattern |
|------------|---------------|------------------|
| `UserRepository` | `GetUserByUsername` | `user:{username}` |
| `SongRepository` | `GetAllSongs`, `GetSongByID`, `GetSongByWikiID`, `GetChartByID`, `GetChartByWikiIDAndDifficulty` | `all_songs`, `song:id:{id}`, `song:wiki:{wikiID}`, `chart:id:{id}`, `chart:wiki_diff:{wikiID}:{diff}` |
| `RecordRepository` | `GetBest50Records`, `GetBestRecordsBySong`, `GetBestRecordByChart`, `GetAllChartsWithBestScores` | `{username}:b50:{underflow}:{filterKey}`, `{username}:best_song:{songID}`, `{username}:best_chart:{chartID}`, `{username}:all_charts:{filterKey}` |

**Invalidation rules**:
- Song writes (`CreateSong`, `UpdateSong`) → `cache.Flush()` (flush all song/chart entries).
- User writes (`UpdateUser`) → `cache.Delete("user:" + username)` (targeted).
- Record writes (`CreateRecord`, `BatchCreateRecords`) → `invalidateByPrefix(cache, username + ":")` (per-user prefix scan).
- Returns shallow copies from cache to prevent callers from mutating cached data.
- `nil` results (not found) are never cached.

**Design notes**:
- Constructor signatures are unchanged (`NewUserRepository(db)`, etc.) — cache is created internally. The service layer is completely unaware of caching.
- `UserRepository.WithTransaction` shares the cache reference with the transactional repo so writes inside the TX trigger invalidation.
- Paginated/sorted record queries are NOT cached (low hit rate due to parameter variation).
- When `UpdateSong` triggers `RecalculateRatingsByChart`, the record cache is not directly invalidated (cross-repo). Stale entries expire via TTL. This is acceptable because song updates are extremely rare.

### Key Domain Concepts

- **Song**: A music track with metadata (title, artist, genre, cover, illustrator, version, album, bpm, length, wiki_id, b15 flag).
- **Chart**: A specific difficulty chart (谱面) of a song. Difficulties: `detected`, `invaded`, `massive`, `reboot`. Each chart has a level, optional fitting_level, optional level_design, and notes count. Charts may also carry `SongBaseOverride` fields (`override_title`, `override_artist`, `override_version`, `override_cover`) to override the parent song's metadata — useful when a chart was added in a later version (e.g. Reboot difficulty). `SongBase.WithOverride()` applies non-nil overrides when building API responses.
- **PlayRecord**: A single play attempt with a score, linked to a Chart and User.
- **BestPlayRecord**: Points to the best PlayRecord per user per Chart (unique constraint on username+chart_id). Updated automatically when a higher score is submitted.
- **Token** (`model.Token`): Contains `access_token`, `refresh_token`, and `token_type`.
- **Access Token**: Short-lived JWT with `"type": "access"` claim, used for API authentication.
- **Refresh Token**: Long-lived JWT with `"type": "refresh"` claim, used to obtain new token pairs via `POST /user/refresh`. Auth middleware rejects refresh tokens.
- **Rating**: Calculated from chart level and score using a piecewise formula (see `pkg/rating/rating.go`). Stored as `int` (rating × 100).
- **B50**: Best 50 = B35 (top 35 ratings from old songs where `b15=false`) + B15 (top 15 ratings from new songs where `b15=true`).

### Database Tables

| Table                | Model            | Primary Key       |
|----------------------|------------------|-------------------|
| `prober_users`       | `User`           | `id`              |
| `songs`              | `Song`           | `id`              |
| `charts`             | `Chart`          | `id`              |
| `play_records`       | `PlayRecord`     | `id`              |
| `best_play_records`  | `BestPlayRecord` | `id`              |
| `chart_statistics`   | `ChartStatistic` | `chart_id`        |

`chart_statistics` is owned by the **fitting-calculator microservice** (`cmd/fitting`) and is not read by `cmd/server`. Its schema is migrated by `util.InitDB()` for consistency regardless of which binary starts first. See `docs/fitting_level.en.md` / `docs/fitting_level.zh.md` for its columns and semantics.

All GORM entities embed `BaseModel` (`internal/model/base.go`), which provides `created_at`, `updated_at`, and `deleted_at` columns. GORM automatically manages `created_at`/`updated_at` timestamps and filters soft-deleted rows (`WHERE deleted_at IS NULL`) in all SELECT queries. The `Chart` entity uses a **partial unique index** on `(song_id, difficulty) WHERE deleted_at IS NULL`, so soft-deleted charts do not block re-adding a chart with the same difficulty — this is required because PostgreSQL and SQLite treat `NULL` values in composite UNIQUE indexes as distinct, so naively adding `deleted_at` to the unique index would break uniqueness on live rows instead.

GORM `AutoMigrate` handles schema creation/updates at startup. Foreign key constraints are disabled during migration (`DisableForeignKeyConstraintWhenMigrating: true`).

### Structured Logging

The application uses Go's `log/slog` package with a custom `ContextHandler` for structured, context-aware logging.

**Architecture**: `internal/logging/` provides the foundation:
- **`ContextHandler`** wraps a `slog.Handler` (either `TextHandler` or `JSONHandler`, selected via `logging.format`) and automatically extracts `[]slog.Attr` stored in `context.Context` via `AppendCtx()`. This means any field injected into the context upstream (by middleware or controllers) is automatically included in all downstream log records.
- **`AppendCtx(ctx, attrs...)`** is the primary helper for injecting fields into context.
- **`Setup()`** initializes the global `slog` default logger (called once from `main.go`).

**Context propagation pattern**:
```
Middleware (request_id, method, path, client_ip) → Auth (username) → Controller (target_user, scope, song_id) → Service (slog.*Context auto-enriched)
```

- All service methods accept `context.Context` as their first parameter.
- Controllers pass `c.Request.Context()` (enriched by middleware) to services.
- Services use `slog.InfoContext(ctx, ...)` / `slog.WarnContext(ctx, ...)` / `slog.ErrorContext(ctx, ...)` — fields from context are automatically attached.
- The `request_id` field is shared across all log lines from the same HTTP request, enabling correlation during debugging.

**Log output format**: Configurable via `logging.format`. Default is `text` (slog `TextHandler`, `key=value` format); set to `json` to use `slog.JSONHandler` (one JSON object per line). Destination is configurable via `logging.output` (`stdout` / `stderr` / `file`). Example (text):
```
time=2026-04-05T10:00:00.000+08:00 level=INFO msg="request completed" request_id=a1b2c3d4e5f6g7h8 method=POST path=/api/v2/records/testuser client_ip=203.0.113.42 username=adminuser status=201 latency_ms=87 bytes_out=2048
```
Example (json):
```json
{"time":"2026-04-05T10:00:00.000+08:00","level":"INFO","msg":"request completed","request_id":"a1b2c3d4e5f6g7h8","method":"POST","path":"/api/v2/records/testuser","client_ip":"203.0.113.42","username":"adminuser","status":201,"latency_ms":87,"bytes_out":2048}
```

## Build and Run Commands

### Local Development

```bash
# Run the server (requires config/config.yaml with valid secret_key)
go run cmd/server/main.go

# Build binary
go build -o server ./cmd/server/main.go
```

### Frontend Development

```bash
cd web
pnpm install
pnpm dev          # Dev server with API proxy to :8080
pnpm build        # Production build → web/dist/
pnpm lint         # ESLint
```

### Docker

```bash
# Default: start db + app (no fitting). Matches the CI integration test.
docker compose up -d

# Include the fitting-level microservice alongside app + db
docker compose --profile fitting up -d
# Or enable the profile permanently via the environment variable
COMPOSE_PROFILES=fitting docker compose up -d

# Build Docker image only (produces both ./server and ./fitting inside the image)
docker build -t prprober-app .
```

The image is multi-stage and compiles **both** binaries (`./server` and `./fitting`). The `app` service uses the default `CMD ["./server"]`; the `fitting` service overrides it via `command: ["./fitting"]` and is gated behind the `fitting` Compose profile so `docker compose up` keeps its existing behaviour (db + app only) and the CI integration test remains unchanged.

The server listens on port **8080** by default. Health check: `GET /healthz`.

### Database Migration (Legacy → Go)

```bash
# Run migration from legacy Python schema to Go schema
go run cmd/migrate/main.go -config config/config.yaml

# Dry run (print SQL without executing)
go run cmd/migrate/main.go -config config/config.yaml -dry-run

# Verify migration
go run cmd/migrate/verify/main.go
```

See `legacy/MIGRATION.md` for the full step-by-step guide.

### Fitting-Level Microservice (cmd/fitting)

```bash
# Local: continuous mode (runs every fitting.interval, default 6h) until SIGINT/SIGTERM
go run cmd/fitting/main.go -config config/config.yaml

# Local: one-shot run (useful for cron, smoke tests, backfill)
go run cmd/fitting/main.go -config config/config.yaml -once

# Local: build standalone binary
go build -o fitting ./cmd/fitting/main.go

# Containerised: start alongside db + app via the `fitting` Compose profile
docker compose --profile fitting up -d

# Containerised: tail fitting logs
docker compose --profile fitting logs -f fitting
```

Scheduling: the binary carries its **own internal `time.Ticker`** (period = `FITTING_INTERVAL`, default `6h`), so a single long-lived process is the full scheduler — **no host cron required**. For external scheduling (cron / systemd timer / k8s CronJob) override the service to `command: ["./fitting", "-once"]`, drop the `profiles` entry, set `restart: "no"`, and invoke via `docker compose run --rm fitting`; see the “Docker / docker-compose” subsection of `docs/fitting_level.en.md` / `docs/fitting_level.zh.md` for the exact snippet.

The binary shares `config/config.yaml` and the DB schema with `cmd/server` but
runs as a **separate process** — it never starts an HTTP listener, never
imports `internal/router` or `internal/controller`, and only writes to
`charts.fitting_level` + `chart_statistics`. See `docs/fitting_level.en.md` /
`docs/fitting_level.zh.md` for the algorithm and its “Operational guide”
section for deployment notes.

### Swagger Documentation

```bash
# Install swag CLI
go install github.com/swaggo/swag/cmd/swag@latest

# Regenerate Swagger docs (MUST be committed — CI checks for drift)
swag init -g cmd/server/main.go
```

Swagger UI is available at: `http://localhost:8080/swagger/index.html`

**Important**: After modifying any Swagger annotations (godoc comments on controller methods or in `cmd/server/main.go`), you MUST run `swag init -g cmd/server/main.go` and commit the regenerated files in `docs/`. The CI pipeline checks for Swagger doc consistency and will fail if they are out of date.

## Testing

### Running Tests

```bash
# Run all tests
go test -v ./...

# Run tests for a specific package
go test -v ./internal/service/...
go test -v ./pkg/rating/...
```

### Test Architecture

- Tests use **in-memory SQLite** databases (`file::memory:?cache=shared` or `:memory:`) — no external database required.
- Each test package has a `setup_test.go` that provides `setupTestDB(t)` to create a fresh in-memory DB with all models auto-migrated.
- Controller tests additionally provide `setupEnv(t)` which initializes the full dependency chain (repos → services → controllers) and sets test config values.
- Controller tests use `httptest.NewRecorder()` and `performRequest()` helper to test HTTP handlers directly against Gin's engine.
- Assertion library: `github.com/stretchr/testify/assert`.
- The `config.GlobalConfig.Auth.SecretKey` is set to `"testsecret"` in controller test setup.

### Test Coverage by Layer

| Package                  | Tested Areas                                          |
|--------------------------|-------------------------------------------------------|
| `pkg/auth`               | Password hashing, JWT generation/extraction/expiry     |
| `pkg/rating`             | Rating formula with various score ranges               |
| `internal/repository`    | CRUD for users, songs, records; best record logic; cache consistency (hit/miss/invalidation, cross-user isolation, shallow copy safety, TX rollback) |
| `internal/service`       | User creation/login, song CRUD, record management      |
| `internal/controller`    | HTTP handler integration (register, login, songs, records) |
| `internal/middleware`     | Auth middleware (valid/invalid/expired/missing tokens)  |
| `internal/model`         | Model validation and enum logic                        |
| `internal/util`          | CSV generation, parsing (UTF-8, GBK encoding, BOM)     |
| `internal/fitting`       | Rating inverter round-trip; calculator (noise-free, outlier-robust, shrinkage, deviation cap, sparse); runner end-to-end against in-memory SQLite |


## Linting

```bash
# Run linter (requires golangci-lint v2.6+)
golangci-lint run
```

### Linter Configuration (`.golangci.yml`)

- **Enabled linters**: `misspell`, `revive`
- **Enabled formatters**: `gofmt`
- **Disabled rules**: `var-naming` in revive (allows non-standard Go naming where needed)
- **Exclusions**: Test files are excluded from `errcheck`, `revive`, and `unused` linters
- Tests are not run by the linter (`run.tests: false`)

## Configuration

Configuration is loaded from `config/config.yaml`, with **environment variable overrides** taking precedence. A `config/config.yaml.example` is provided as a safe-to-commit template.

| Config Key                   | Env Var        | Default                              | Description                              |
|------------------------------|----------------|--------------------------------------|------------------------------------------|
| `server.port`                | `SERVER_PORT`  | `:8080`                              | Server listen address                    |
| `database.type`              | `DB_TYPE`      | `sqlite`                             | `sqlite` or `postgres`                   |
| `database.dsn`               | `DB_DSN`       | `prober.db`                          | SQLite file path                         |
| `database.host`              | `DB_HOST`      | —                                    | PostgreSQL host                          |
| `database.port`              | `DB_PORT`      | —                                    | PostgreSQL port                          |
| `database.user`              | `DB_USER`      | —                                    | PostgreSQL user                          |
| `database.password`          | `DB_PASSWORD`  | —                                    | PostgreSQL password                      |
| `database.dbname`            | `DB_NAME`      | —                                    | PostgreSQL database name                 |
| `database.sslmode`           | `DB_SSLMODE`   | —                                    | PostgreSQL SSL mode                      |
| `auth.secret_key`            | `SECRET_KEY`   | `your_secret_key_here`               | JWT signing secret (**must change**)     |
| `auth.jwt_algorithm`         | —              | `HS256`                              | JWT algorithm (hardcoded HS256)          |
| `auth.jwt_expiration`        | —              | `30m`                                | Access token lifetime (Go duration)      |
| `auth.refresh_token_expiration` | —           | `168h`                               | Refresh token lifetime (Go duration)     |
| `auth.bcrypt_cost`           | —              | `10`                                 | bcrypt hashing cost (4–31)               |
| `auth.upload_token_length`   | —              | `16`                                 | Upload token bytes (hex output is 2×)    |
| `auth.username_pattern`      | —              | `^[a-z][a-z0-9_]{5,15}$`            | Regex for username validation            |
| `pagination.default_page_size` | —            | `50`                                 | Default page size for list endpoints     |
| `pagination.max_page_size`   | —              | `200`                                | Maximum allowed page size                |
| `game.b35_limit`             | —              | `35`                                 | B35 best record count (old songs)        |
| `game.b15_limit`             | —              | `15`                                 | B15 best record count (new songs)        |
| `logging.output`             | `LOG_OUTPUT`   | `stdout`                             | Log destination: `stdout`, `stderr`, or `file` |
| `logging.file`               | `LOG_FILE`     | —                                    | File path when `logging.output=file` (appended, parent dirs auto-created) |
| `logging.format`             | `LOG_FORMAT`   | `text`                               | slog handler format: `text` or `json`    |
| `logging.exclude_paths`      | `LOG_EXCLUDE_PATHS` | `["/healthz"]`                  | Path prefixes excluded from request logs (env var is comma-separated) |
| `metrics.enabled`            | `METRICS_ENABLED` | `true`                           | Install the metrics middleware and start the dedicated metrics HTTP server |
| `metrics.addr`               | `METRICS_ADDR`    | `:9090`                          | Listen address for the metrics server (MUST differ from `server.port`)   |
| `metrics.path`               | `METRICS_PATH`    | `/metrics`                       | URL path that serves Prometheus metrics (must start with `/`)            |
| `metrics.exclude_paths`      | `METRICS_EXCLUDE_PATHS` | `["/healthz"]`              | Route-template prefixes excluded from HTTP metrics (env var comma-separated) |
| `fitting.enabled`            | `FITTING_ENABLED` | `true`                         | Master switch for the fitting-calculator microservice (`cmd/fitting`)     |
| `fitting.interval`           | `FITTING_INTERVAL` | `6h`                         | Ticker period for continuous mode (Go duration string, must be > 0)       |
| `fitting.min_samples`        | —               | `8.0`                             | Minimum effective sample size (`N_eff`) to publish `FittingLevel`         |
| `fitting.min_player_records` | —               | `20`                              | Minimum total best records a player must have to contribute samples        |
| `fitting.proximity_sigma`    | —               | `20.0`                            | Gaussian σ in rating units centered on 10×Level (proximity weight)        |
| `fitting.volume_full_at`     | —               | `50`                              | Records count at which a player receives full volume weight (1.0)          |
| `fitting.prior_strength`     | —               | `5.0`                             | κ in Bayesian shrinkage toward the official level                          |
| `fitting.max_deviation`      | —               | `1.5`                             | Hard cap on \|FittingLevel − Level\|                                       |
| `fitting.min_score`          | —               | `500000`                          | Discard samples with score below this threshold                            |
| `fitting.tukey_k`            | —               | `4.685`                           | Tukey biweight tuning constant                                             |
| `fitting.chart_batch_size`   | —               | `200`                             | Charts processed per DB batch (keeps per-tx footprint small)                |
| `fitting.player_batch_size`  | —               | `500`                             | Distinct users fetched per paginated skill-collection query                 |
| `fitting.batch_pause`        | `FITTING_BATCH_PAUSE` | `50ms`                     | Sleep between chart batches to ease DB load (Go duration string)           |

**Startup guard**: The server will `log.Fatal` if `secret_key` is left at the default `"your_secret_key_here"`, or if `jwt_expiration`/`username_pattern` cannot be parsed, or if `bcrypt_cost` is out of range, or if `logging.output` is not one of `stdout`/`stderr`/`file`, or if `logging.output=file` but `logging.file` is empty, or if `logging.format` is not `text` or `json`, or if `metrics.enabled=true` and any of `metrics.addr` is empty / `metrics.path` does not start with `/` / `metrics.addr` equals `server.port`, or if `fitting.interval`/`fitting.batch_pause` cannot be parsed as durations, or if `fitting.interval` is non-positive, or if `fitting.proximity_sigma`/`fitting.tukey_k` is non-positive, or if `fitting.prior_strength`/`fitting.max_deviation` is negative, or if `fitting.chart_batch_size`/`fitting.player_batch_size` is non-positive.

## CI/CD Pipeline

Defined in `.github/workflows/ci.yml`. Triggers on push/PR to `master` and `dev` branches, and on version tags (`v*`).

### Pipeline Stages

1. **Lint & Swagger Check** (`lint`): Runs `golangci-lint` v2.6, regenerates Swagger docs, and fails if `docs/` has uncommitted changes (drift detection). Uses `actions/checkout@v6` and `actions/setup-go@v6`.
2. **Frontend Lint** (`frontend-lint`): Runs `pnpm lint` (ESLint) in `web/` directory on `ubuntu-slim`. Uses pnpm 10 and Node.js 22.
3. **Unit Tests** (`test`): Runs `go test -v ./...`. Depends on `lint` and `frontend-lint`.
4. **Docker Build & Push** (`docker-build`, depends on lint + test + frontend-lint):
   - Builds Docker image and pushes to `ghcr.io`.
   - Runs Docker Compose integration test (health check on `/healthz`).
   - Only runs on push events (not PRs).

### Docker Image Tags

- `debug` (on non-tag pushes, e.g. branch pushes)
- Version tag (e.g., `v1.0.0`, when pushing a Git tag)
- `latest` (only on version tag pushes)

Registry: `ghcr.io/icelocke/paradigm-reboot-prober-go`

## API Routes

Base path: `/api/v2`

### Public Routes
| Method | Path                  | Handler                        |
|--------|-----------------------|--------------------------------|
| POST   | `/user/register`      | `UserController.Register`      |
| POST   | `/user/login`         | `UserController.Login`         |
| POST   | `/user/refresh`       | `UserController.RefreshToken`  |
| GET    | `/songs`              | `SongController.GetAllCharts`  |
| GET    | `/songs/:song_id`     | `SongController.GetSingleSongInfo` |

### Optional Auth Routes (accessible without login, but auth checked for permissions)
| Method | Path                                          | Handler                            |
|--------|-----------------------------------------------|------------------------------------|  
| GET    | `/records/:username`                          | `RecordController.GetPlayRecords`  |
| GET    | `/records/:username/song/:song_addr`          | `RecordController.GetSongRecords`  |
| GET    | `/records/:username/chart/:chart_addr`        | `RecordController.GetChartRecords` |
| POST   | `/records/:username`                          | `RecordController.UploadRecords`   |

### Authenticated Routes (JWT required)
| Method | Path                        | Handler                              |
|--------|-----------------------------|--------------------------------------|
| GET    | `/user/me`                  | `UserController.GetMe`               |
| PUT    | `/user/me`                  | `UserController.UpdateMe`            |
| PUT    | `/user/me/password`         | `UserController.ChangePassword`       |
| POST   | `/user/me/upload-token`     | `UserController.RefreshUploadToken`   |

### Admin Routes (JWT + `is_admin=true`)
| Method | Path                    | Handler                        |
|--------|-------------------------|--------------------------------|
| POST   | `/songs`                | `SongController.CreateSong`    |
| PUT    | `/songs`                | `SongController.UpdateSong`    |
| POST   | `/user/reset-password`  | `UserController.ResetPassword` |

### Non-API Routes
| Method | Path              | Description          |
|--------|-------------------|----------------------|
| GET    | `/healthz`        | Health check         |
| GET    | `/swagger/*any`   | Swagger UI           |

### Observability Routes (separate port)

Served by a dedicated `http.Server` on `metrics.addr` (default `:9090`). This port is intentionally distinct from the main API port so Prometheus scraping is not exposed on the public API surface.

| Method | Path (default) | Description                                        |
|--------|----------------|----------------------------------------------------|
| GET    | `/metrics`     | Prometheus exposition (HTTP RED + Go/process stats) |

Exposed metrics:

| Metric                             | Type      | Labels                     |
|------------------------------------|-----------|----------------------------|
| `http_requests_total`              | Counter   | `method`, `path`, `status` |
| `http_request_duration_seconds`    | Histogram | `method`, `path`, `status` |
| `http_response_size_bytes`         | Histogram | `method`, `path`, `status` |
| `http_requests_in_flight`          | Gauge     | —                          |
| `go_*`, `process_*`                | various   | auto-registered by `client_golang` default registry |

The `path` label is Gin's matched route template (e.g. `/api/v2/records/:username`); requests that match no route are reported as `path="unknown"`. Prefixes in `metrics.exclude_paths` are skipped entirely.

### GetPlayRecords Scopes

The `GET /records/:username` endpoint supports the following `scope` query parameter values:

| Scope        | Description                                              | Pagination |
|--------------|----------------------------------------------------------|------------|
| `b50`        | Best 50 records (B35 + B15), supports `underflow` param  | No         |
| `best`       | All best records (one per chart per user)                 | Yes        |
| `all`        | All play records                                          | Yes        |
| `all-charts` | All charts with user's best score (0 if not played)       | No         |

### Per-Song and Per-Chart Record Queries

**Song address (`song_addr`)**: Numeric `song_id` or `wiki_id` string (e.g. `felys`).

**Chart address (`chart_addr`)**: Numeric `chart_id` or `wiki_id:difficulty` string (e.g. `felys:massive`).
Valid difficulties: `detected`, `invaded`, `massive`, `reboot`.

The `GET /records/:username/song/:song_addr` endpoint supports:

| Scope  | Description                                    | Pagination |
|--------|------------------------------------------------|------------|
| `best` | Best record per difficulty for the song         | No         |
| `all`  | All play records for the song                   | Yes        |

The `GET /records/:username/chart/:chart_addr` endpoint supports:

| Scope  | Description                                    | Pagination |
|--------|------------------------------------------------|------------|
| `best` | Single best record for the chart                | No         |
| `all`  | All play records for the chart                  | Yes        |

## Code Style Guidelines

- **Go version**: 1.25 (set in `go.mod` and CI), toolchain go1.25.5.
- **Formatting**: Enforced by `gofmt` via golangci-lint.
- **Naming**: Standard Go conventions. The `var-naming` revive rule is disabled to allow certain non-standard names where needed.
- **Error handling**: Errors are returned up the call chain. Controllers translate errors to appropriate HTTP status codes and `model.Response` JSON.
- **Comments**: English. Controller methods have Swagger godoc annotations.
- **SQL injection prevention**: Sort column names are validated against a whitelist (`allowedSortColumns` in `record_repo.go`).
- **GORM conventions**: Models define explicit `TableName()` methods. Primary keys use the standard `id` column name.
- **Struct tags**: Models use `gorm`, `json`, `binding`, and `example` tags.
- **Deferred cleanup**: File handles use `defer func() { _ = f.Close() }()` pattern (explicitly ignoring close errors).

## Security Considerations

- **JWT secret**: Must be changed from default before startup (enforced by `log.Fatal` check in `main.go`).
- **Password storage**: bcrypt hashing via `golang.org/x/crypto/bcrypt` with `DefaultCost`.
- **Authorization model**:
  - Record viewing respects `anonymous_probe` user setting; admins can view any user's records.
  - Record uploading requires JWT auth (own records) or a valid `upload_token` (third-party upload).
  - Song creation/update requires admin role.
  - Password reset requires admin role.
- **CORS**: Configured to allow all origins (`AllowAllOrigins: true`) with standard methods and headers.
- **SQL injection**: Sort parameters are whitelisted; all other queries use GORM's parameterized queries.
- **Token expiration**: JWT access tokens expire after 24 hours. Default (no duration specified) is 30 minutes.
