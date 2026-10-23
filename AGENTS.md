# AGENTS.md

## Project Overview

Paradigm: Reboot Prober is a score-tracking and Best 50 service for the rhythm game Paradigm: Reboot. The backend uses Go 1.25, Gin, and GORM with PostgreSQL or SQLite; the frontend uses Vue 3, TypeScript, Vite, and Naive UI. The Go module is `paradigm-reboot-prober-go`.

Best 50 consists of B35 (old songs, `b15=false`) and B15 (new songs, `b15=true`). The repository also contains a separate fitting-level calculator that publishes `charts.fitting_level` offline.

## Repository Map

| Path | Purpose |
| --- | --- |
| `cmd/server/` | HTTP API entry point |
| `cmd/fitting/` | Offline fitting-level calculator; use package-level commands |
| `cmd/migrate/` | Legacy PostgreSQL migration and verification tools |
| `internal/` | Controllers, services, repositories, models, middleware, and fitting library |
| `pkg/` | Shared authentication and rating logic |
| `config/` | Configuration loading, validation, and example YAML |
| `web/` | Vue frontend; follow `web/AGENTS.md` when working here |
| `docs/` | Generated Swagger files and hand-written design references |
| `legacy/` | Legacy schema and migration resources |

## Setup and Validation

Run backend commands from the repository root. For local setup, copy `config/config.yaml.example` to `config/config.yaml` and replace the default JWT secret. Do not commit the local config or credentials.

```bash
go run ./cmd/server
go test -race ./...
golangci-lint run
```

For frontend changes, run from `web/`:

```bash
pnpm install
pnpm lint
pnpm type-check
pnpm build
```

For fitting work, invoke the whole package rather than `main.go` alone:

```bash
go run ./cmd/fitting --once
go run ./cmd/fitting analyze -chart 51
```

CI also checks Swagger and frontend generated-file drift, frontend build, and Docker Compose health. See `.github/workflows/ci.yml` for the authoritative pipeline.

## Backend Working Rules

- Preserve the controller → service → repository layering and manual dependency wiring in `internal/router/router.go`.
- Controllers validate input and translate errors; services contain business logic; repositories own database operations and record rating calculation.
- Public service methods accept `context.Context`. Pass `c.Request.Context()` from controllers and use `slog.InfoContext`, `WarnContext`, or `ErrorContext` downstream so request fields remain attached.
- Keep comments in English and follow existing Go, GORM, and struct-tag conventions. Validate record sort fields through `model.RecordSort` before using them in SQL.
- GORM entities embed `BaseModel`; preserve soft-delete behavior and the partial unique index on live `(song_id, difficulty)` chart rows.
- Preserve cache invalidation on writes. Cache keys must include every result-changing parameter, and fitting-ranked record queries must remain uncached because the fitting process writes directly to `charts`.
- Add or update tests for behavior changes. Database-backed tests use in-memory SQLite; controller tests use Gin with `httptest`, and assertions use `testify`.

See `docs/architecture.md` before changing request flow, persistence, caching, logging, or authentication.

## Design Principle: 保持查分器本体的单纯性

The probe service (`cmd/server` and its request-path dependencies) receives uploads and answers record/B50 queries. Derived or analytical work that is not required for a live request belongs in a separate `cmd/<name>` binary and `internal/<name>` package.

- `cmd/server` must not import `internal/fitting`, run fitting jobs, read `chart_statistics`, or read `config.fitting.*` at request time.
- The server may synchronously derive `fitting_rating` from the published `charts.fitting_level` using `pkg/rating`. Stored `play_records.rating` remains the official rating.
- `chart_statistics` is owned by the fitting binary and is not exposed through the probe API. Its schema is migrated by `util.InitDB()` so either binary can initialize the database.
- New analytical workloads must not be added to probe controllers, services, or repositories.

The fitting algorithm and operations guide live in `docs/fitting_level.en.md` and `docs/fitting_level.zh.md`; fitting-ranked B50 semantics are in `docs/fitting_b50.md`.

## API Contract Changes

Use `internal/router/router.go` and the generated Swagger document as the API reference instead of duplicating route tables here.

After changing Swagger annotations, regenerate and include the generated backend files:

```bash
swag init -g cmd/server/main.go
```

After any API contract change, regenerate and include the frontend contract files as well:

```bash
cd web
pnpm generate:api
```

This updates `web/src/api/openapi3.json` and `web/src/api/generated.d.ts`. Never hand-edit generated Swagger or OpenAPI outputs.

#### Nullable JSON fields (`*T` without `omitempty`)

For a pointer field serialized as JSON `null`, add `extensions:"x-nullable=true"` so Swagger and the frontend generation pipeline preserve `T | null`:

```go
FittingLevel *float64 `gorm:"column:fitting_level" json:"fitting_level" extensions:"x-nullable=true"`
```

Pointer fields with `omitempty` are optional rather than nullable; do not add `x-nullable` to them.

## Security Rules

- Preserve `anonymous_probe` privacy checks, admin-only song mutations and password resets, and the requirement that uploads use either the owner's JWT or a valid upload token.
- Auth middleware accepts access tokens and legacy tokens without a type claim, but rejects refresh tokens.
- Password hashing uses the configured bcrypt cost, and token lifetimes come from configuration. Never hardcode secrets or assume a fixed token lifetime.

## Detailed References

- Architecture, caching, database, and observability: `docs/architecture.md`
- Configuration keys, defaults, and validation: `config/config.yaml.example` and `config/config.go`
- HTTP API: `docs/swagger.yaml` and `/swagger/index.html` on a running server
- Frontend conventions: `web/AGENTS.md`
- Legacy migration: `legacy/MIGRATION.md`
- Fitting calculator: `docs/fitting_level.en.md` / `docs/fitting_level.zh.md`

Keep these instructions current when commands, architecture, or workflows change. Put detailed, task-specific reference material in the relevant linked document rather than expanding this always-loaded file.
