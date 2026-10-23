# Backend Architecture

This is the detailed companion to the root `AGENTS.md`. Read it when changing request flow, persistence, caching, logging, observability, or authentication. Configuration, API routes, and fitting algorithms have their own canonical references listed at the end.

## Request Flow and Layers

The server uses `gin.New()` with an explicit middleware chain and manual dependency injection in `internal/router/router.go`.

```text
Request
  → request ID → request logging → metrics → CORS
  → request decompression → decompressed body limit → response compression
  → route-specific rate limit / authentication
  → Controller → Service → Repository → Database
```

- **Middleware**: Request IDs come from `X-Request-ID` or a random value and are returned in the response. Request logging adds method, path, and client IP to the context. Login and registration have per-IP rate limits.
- **Authentication**: `AuthMiddleware` requires a valid access token and an active user. `OptionalAuthMiddleware` allows anonymous requests and attaches identity only when valid credentials are present. `AdminMiddleware` checks the admin role.
- **Controllers**: Bind and validate requests, enforce endpoint-level authorization, add business fields to the logging context, and translate service errors to HTTP responses.
- **Services**: Contain business logic and orchestrate repositories. Public methods accept `context.Context` as their first argument.
- **Repositories**: Own GORM queries, transactions, rating calculation during record creation, and repository-local caches.
- **Models**: Define GORM entities and API DTOs. Request DTOs live under `internal/model/request/`.
- **Shared packages**: `pkg/auth` handles passwords and JWTs; `pkg/rating` implements the single-chart rating formula.

## Domain Model and Persistence

- **Song** stores track metadata and the `b15` season flag.
- **Chart** belongs to a song and has difficulty `detected`, `invaded`, `massive`, or `reboot`, plus an official level and optional fitting level. Per-chart `SongBaseOverride` values can replace selected song metadata in API responses.
- **PlayRecord** is one play attempt. Its stored `rating` is calculated from the official chart level and score and is stored as an integer scaled by 100.
- **BestPlayRecord** points to one record per user and chart. It normally advances to a higher score, but replacement uploads may deliberately replace it with a lower score.
- **FittingRating** is derived at query time from `charts.fitting_level`, falling back to the official level when no fitting level is published. It is not persisted in `play_records`.
- **B50** selects B35 from songs with `b15=false` and B15 from songs with `b15=true`, using the configured limits. Fitting-ranked B50 re-ranks the full best-record candidate pool before applying those limits.

The main tables are `prober_users`, `songs`, `charts`, `play_records`, `best_play_records`, and `chart_statistics`.

All GORM entities embed `BaseModel`, which supplies creation/update timestamps and soft deletion. Queries must not accidentally include soft-deleted records through manually joined tables. The chart uniqueness index covers `(song_id, difficulty)` only where `deleted_at IS NULL`; do not replace it with a composite index that includes `deleted_at`, because PostgreSQL and SQLite handle `NULL` differently in unique indexes.

`util.InitDB()` runs GORM AutoMigrate with foreign-key constraint creation disabled during migration. It also migrates `chart_statistics` so the schema is consistent regardless of whether the server or fitting binary starts first. The fitting binary owns that table's contents; the server does not read it.

## Fitting Boundary

The server and fitting calculator share the database and rating package, but have separate processes and lifecycles.

The server must not import `internal/fitting`, schedule fitting work, read fitting configuration on the request path, or query `chart_statistics`. It may read the published `charts.fitting_level` and synchronously calculate fitting ratings with `pkg/rating.SingleRating`.

This distinction allows fitting-ranked record and B50 queries without moving the offline calculator into the API process. New analytical workloads should follow the same separate-binary pattern.

## Repository Caching

Each repository owns an in-process `jellydator/ttlcache/v3` cache. TTL constants and cache-key builders are centralized in `internal/repository/cache.go`.

| Repository | Default TTL |
| --- | --- |
| Songs and charts | 10 minutes |
| Users | 5 minutes |
| Records | 5 minutes |

Important invariants:

- Song and chart writes clear the song repository cache. User writes invalidate the affected username. Record writes invalidate all cache entries prefixed with that username.
- `UserRepository.WithTransaction` shares the cache with the transactional repository so writes inside the transaction still invalidate entries.
- Cache keys must include filters, ranking fields, direction, underflow, and pagination whenever those inputs affect the result.
- Cached values are defensively copied where needed so callers cannot mutate cached slices. Missing (`nil`) results are not cached.
- Fitting-ranked record queries bypass the result cache. The fitting process updates `charts` directly and cannot invalidate the API process's memory cache.
- Song updates can recalculate official ratings without directly invalidating the separate record repository cache. Those entries expire by TTL; keep this trade-off in mind when changing update or recalculation behavior.

## Structured Logging and Metrics

`internal/logging` wraps `log/slog` with a context-aware handler. `logging.AppendCtx(ctx, attrs...)` adds fields upstream; downstream `slog.*Context` calls include them automatically.

The normal propagation path is:

```text
Middleware (request_id, method, path, client_ip)
  → Auth (username)
  → Controller (target_user, scope, song_id, ...)
  → Service logs
```

Log output can be text or JSON and can target stdout, stderr, or a file. Request-log exclusion prefixes suppress the final request-summary line, not context enrichment or the request ID header.

Prometheus HTTP metrics use Gin's matched route template as the `path` label to bound cardinality; unmatched routes use `unknown`. Metrics are served by a separate HTTP server on `metrics.addr`, not on the public API listener. The `/healthz` endpoint remains on the main server.

## Authentication and Authorization

- Passwords use bcrypt with `auth.bcrypt_cost` from configuration.
- Access and refresh JWTs carry different `type` claims. API auth middleware rejects refresh tokens; legacy tokens without a type claim remain accepted.
- Token lifetimes and the JWT signing secret are configurable. Startup rejects the example signing secret rather than running with a known default.
- Record reads respect the target user's `anonymous_probe` setting, with administrator access where allowed.
- Record uploads require either the owner's access token or a valid upload token. Song creation/update and password reset require an administrator.
- CORS currently allows all origins without credentials. Request decompression is followed by a decompressed-body size limit.
- Record sort fields are normalized through `model.RecordSort`; do not interpolate unvalidated sort input into SQL.

## Canonical References

- Configuration: `config/config.yaml.example`, `config/config.go`
- API schema and routes: `docs/swagger.yaml`, `internal/router/router.go`
- Fitting mathematics and operations: `docs/fitting_level.en.md`, `docs/fitting_level.zh.md`
- Fitting-ranked B50 behavior: `docs/fitting_b50.md`
- Frontend architecture and rules: `web/AGENTS.md`
- CI checks and deployment image workflow: `.github/workflows/ci.yml`
- Legacy migration: `legacy/MIGRATION.md`
