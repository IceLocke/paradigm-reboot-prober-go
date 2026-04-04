# AGENTS.md

## Project Overview

**Paradigm: Reboot Prober** (`paradigm-reboot-prober-go`) is a backend REST API service for the rhythm game **Paradigm: Reboot**. It provides score tracking, rating calculation, and Best 50 (B50) computation. The project is a Go rewrite (WIP) of a previous implementation.

Core features:
- **User Management**: Registration, JWT authentication, profile updates, upload tokens, password change/reset.
- **Song Management**: CRUD for songs and their difficulty charts (admin-only for create/update).
- **Score/Record Management**: Batch upload play records (JSON), automatic single-chart rating calculation, best record tracking.
- **B50 Calculation**: Best 35 (old songs, `b15=false`) + Best 15 (new songs, `b15=true`) selection.
- **API Documentation**: Swagger UI auto-generated from code annotations.

Repository: `github.com/IceLocke/paradigm-reboot-prober-go`

## Tech Stack

| Component      | Technology                                       |
|----------------|--------------------------------------------------|
| Language       | Go 1.25 (toolchain go1.25.5)                     |
| Web Framework  | [Gin](https://github.com/gin-gonic/gin)          |
| ORM            | [GORM](https://gorm.io/)                         |
| Database       | PostgreSQL (production), SQLite (dev/testing)    |
| Authentication | JWT (HS256) via `golang-jwt/jwt/v5`, bcrypt      |
| API Docs       | Swagger via `swaggo/swag` + `swaggo/gin-swagger` |
| Testing        | `testing` stdlib + `stretchr/testify`            |
| Linting        | golangci-lint v2.6                               |
| CI/CD          | GitHub Actions                                   |
| Container      | Docker (multi-stage Alpine build)                |
| Orchestration  | Docker Compose (app + PostgreSQL 16)             |
| Frontend       | Vue.js (legacy, in `web/legacy/`)                |

## Project Structure

```
.
├── cmd/
│   └── server/
│       └── main.go              # Application entry point, Swagger annotations
├── config/
│   ├── config.go                # Config struct, YAML loading, env var overrides
│   └── config.yaml              # Default configuration file
├── internal/
│   ├── controller/              # HTTP handlers (Gin handlers with Swagger annotations)
│   │   ├── user.go              # Register, Login, GetMe, UpdateMe, RefreshUploadToken, ChangePassword, ResetPassword
│   │   ├── song.go              # GetAllCharts, GetSingleSongInfo, CreateSong, UpdateSong
│   │   └── record.go            # GetPlayRecords, UploadRecords
│   ├── middleware/
│   │   └── auth.go              # AuthMiddleware, OptionalAuthMiddleware, AdminMiddleware
│   ├── model/                   # Data models (GORM entities + DTOs)
│   │   ├── user.go              # User, UserBase, UserInDB, UserPublic
│   │   ├── song.go              # Song, SongBase, Difficulty enum, Chart, ChartInfo, ChartInfoSimple, ChartCSV, ChartWithScore, ChartInput
│   │   ├── play_record.go       # PlayRecord, BestPlayRecord, PlayRecordBase, PlayRecordInfo, PlayRecordResponse, AllChartsResponse, ToPlayRecordInfo()
│   │   ├── auth.go              # Token, UploadToken
│   │   ├── common.go            # Response (generic error/message response)
│   │   └── request/             # Request DTOs
│   │       ├── user.go          # CreateUserRequest, UpdateUserRequest, ChangePasswordRequest, ResetPasswordRequest
│   │       ├── song.go          # CreateSongRequest, UpdateSongRequest
│   │       └── play_record.go   # BatchCreatePlayRecordRequest
│   ├── repository/              # Database access layer (GORM queries)
│   │   ├── user_repo.go
│   │   ├── song_repo.go
│   │   └── record_repo.go       # Includes rating calculation on record creation
│   ├── router/
│   │   └── router.go            # Route definitions, dependency wiring, middleware setup
│   ├── service/                  # Business logic layer
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
├── docs/                         # Auto-generated Swagger docs (do NOT edit manually)
│   ├── docs.go
│   ├── swagger.json
│   └── swagger.yaml
├── web/                          # Frontend assets
│   │   ├── src/                 # Vue components, utils, styles
│   │   └── public/              # Static assets (covers, icons)
│   └── styles/                  # Frontend style documentation
│       ├── vue-style-pattern.md
│       └── rules/               # Style rules (components, naive-ui, responsive, tokens)
├── .github/workflows/
│   └── ci.yml                   # CI/CD pipeline
├── .claude/                      # Claude AI settings
├── config/config.yaml           # Default config
├── Dockerfile                   # Multi-stage Docker build
├── docker-compose.yaml          # App + PostgreSQL compose setup
├── .golangci.yml                # Linter configuration
├── go.mod / go.sum              # Go module files
└── AGENTS.md                    # This file
```

## Architecture

The application follows a **layered architecture**:

```
Request → Router → Middleware → Controller → Service → Repository → Database
```

- **Router** (`internal/router/`): Registers all routes, sets up middleware, wires dependencies (manual DI, no framework).
- **Middleware** (`internal/middleware/`): JWT auth extraction (`AuthMiddleware`, `OptionalAuthMiddleware`), admin role check (`AdminMiddleware(userService)`).
- **Controller** (`internal/controller/`): Handles HTTP request/response, input validation, delegates to services.
- **Service** (`internal/service/`): Business logic. Orchestrates repository calls.
- **Repository** (`internal/repository/`): Direct database operations via GORM. Rating calculation happens here when creating records.
- **Model** (`internal/model/`): GORM entities (with table name overrides) and DTOs. Request-specific DTOs live in `internal/model/request/`.
- **Pkg** (`pkg/`): Reusable, domain-specific packages — `auth` (password hashing, JWT) and `rating` (score-to-rating calculation).

### Key Domain Concepts

- **Song**: A music track with metadata (title, artist, genre, cover, illustrator, version, album, bpm, length, wiki_id, b15 flag).
- **Chart**: A specific difficulty chart (谱面) of a song. Difficulties: `detected`, `invaded`, `massive`, `reboot`. Each chart has a level, optional fitting_level, optional level_design, and notes count.
- **PlayRecord**: A single play attempt with a score, linked to a Chart and User.
- **BestPlayRecord**: Points to the best PlayRecord per user per Chart (unique constraint on username+chart_id). Updated automatically when a higher score is submitted.
- **Rating**: Calculated from chart level and score using a piecewise formula (see `pkg/rating/rating.go`). Stored as `int` (rating × 100).
- **B50**: Best 50 = B35 (top 35 ratings from old songs where `b15=false`) + B15 (top 15 ratings from new songs where `b15=true`).

### Database Tables

| Table               | Model            | Primary Key       |
|---------------------|------------------|-------------------|
| `prober_users`      | `User`           | `user_id`         |
| `songs`             | `Song`           | `song_id`         |
| `charts`            | `Chart`          | `chart_id`        |
| `play_records`      | `PlayRecord`     | `play_record_id`  |
| `best_play_records` | `BestPlayRecord` | `best_record_id`  |

GORM `AutoMigrate` handles schema creation/updates at startup. Foreign key constraints are disabled during migration (`DisableForeignKeyConstraintWhenMigrating: true`).

## Build and Run Commands

### Local Development

```bash
# Run the server (requires config/config.yaml with valid secret_key)
go run cmd/server/main.go

# Build binary
go build -o server ./cmd/server/main.go
```

### Docker

```bash
# Build and run with Docker Compose (app + PostgreSQL)
docker-compose up -d

# Build Docker image only
docker build -t prprober-app .
```

The server listens on port **8080** by default. Health check: `GET /health`.

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
| `internal/repository`    | CRUD for users, songs, records; best record logic      |
| `internal/service`       | User creation/login, song CRUD, record management      |
| `internal/controller`    | HTTP handler integration (register, login, songs, records) |
| `internal/middleware`     | Auth middleware (valid/invalid/expired/missing tokens)  |
| `internal/util`          | CSV generation, parsing (UTF-8, GBK encoding, BOM)     |

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

Configuration is loaded from `config/config.yaml`, with **environment variable overrides** taking precedence.

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
| `auth.jwt_expiration`        | —              | `24h`                                | Access token lifetime (Go duration)      |
| `auth.bcrypt_cost`           | —              | `10`                                 | bcrypt hashing cost (4–31)               |
| `auth.upload_token_length`   | —              | `16`                                 | Upload token bytes (hex output is 2×)    |
| `auth.username_pattern`      | —              | `^[A-Za-z][A-Za-z0-9_]{5,15}$`      | Regex for username validation            |
| `pagination.default_page_size` | —            | `50`                                 | Default page size for list endpoints     |
| `pagination.max_page_size`   | —              | `200`                                | Maximum allowed page size                |
| `game.b35_limit`             | —              | `35`                                 | B35 best record count (old songs)        |
| `game.b15_limit`             | —              | `15`                                 | B15 best record count (new songs)        |

**Startup guard**: The server will `log.Fatal` if `secret_key` is left at the default `"your_secret_key_here"`, or if `jwt_expiration`/`username_pattern` cannot be parsed, or if `bcrypt_cost` is out of range.

## CI/CD Pipeline

Defined in `.github/workflows/ci.yml`. Triggers on push/PR to `master` and `dev` branches.

### Pipeline Stages

1. **Lint**: Runs `golangci-lint` v2.6.
2. **Unit Tests**: Runs `go test -v ./...`.
3. **Swagger Consistency Check**: Regenerates Swagger docs and fails if `docs/` directory has changes (drift detection).
4. **Docker Build & Push** (depends on lint + test + swagger-check):
   - Builds Docker image and pushes to `ghcr.io`.
   - Runs Docker Compose integration test (health check on `/health`).
5. **Deploy** (master branch only, after docker-build): Currently a simulated deployment step. SSH-based deploy is commented out for future use.

### Docker Image Tags

- Branch name (e.g., `master`, `dev`)
- Short SHA (e.g., `sha-abc1234`)
- `latest` (only on `master`)

Registry: `ghcr.io/icelocke/paradigm-reboot-prober-go`

## API Routes

Base path: `/api/v2`

### Public Routes
| Method | Path                  | Handler                        |
|--------|-----------------------|--------------------------------|
| POST   | `/user/register`      | `UserController.Register`      |
| POST   | `/user/login`         | `UserController.Login`         |
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
| GET    | `/health`         | Health check         |
| GET    | `/swagger/*any`   | Swagger UI           |

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
- **Naming**: Standard Go conventions. The `var-naming` revive rule is disabled to allow certain non-standard names (e.g., `ID` suffixes in model fields).
- **Error handling**: Errors are returned up the call chain. Controllers translate errors to appropriate HTTP status codes and `model.Response` JSON.
- **Comments**: English. Controller methods have Swagger godoc annotations.
- **SQL injection prevention**: Sort column names are validated against a whitelist (`allowedSortColumns` in `record_repo.go`).
- **GORM conventions**: Models define explicit `TableName()` methods. Primary keys use custom column names (e.g., `user_id`, `song_id`, `chart_id`).
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
- **SQL injection**: Sort parameters are whitelisted; all other queries use GORM's parameterized queries.
- **Token expiration**: JWT access tokens expire after 24 hours. Default (no duration specified) is 30 minutes.
