# Paradigm: Reboot Prober (Go)

[![CI/CD Pipeline](https://github.com/IceLocke/paradigm-reboot-prober-go/actions/workflows/ci.yml/badge.svg)](https://github.com/IceLocke/paradigm-reboot-prober-go/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/IceLocke/paradigm-reboot-prober-go/branch/master/graph/badge.svg)](https://codecov.io/gh/IceLocke/paradigm-reboot-prober-go)
[![Go Version](https://img.shields.io/github/go-mod/go-version/IceLocke/paradigm-reboot-prober-go)](https://go.dev/)
[![Docker Image](https://img.shields.io/badge/docker-ghcr.io-blue.svg)](https://github.com/IceLocke/paradigm-reboot-prober-go/pkgs/container/paradigm-reboot-prober-go)
[![License](https://img.shields.io/github/license/IceLocke/paradigm-reboot-prober-go)](LICENSE)

A backend REST API service with a Vue 3 frontend for **Paradigm: Reboot** score tracking, built with Go.

- **User Management**: Registration, JWT authentication, profile updates, upload tokens, password change/reset.
- **Song Management**: CRUD for songs and difficulty charts (admin-only for create/update).
- **Score Management**: Batch upload support, automatic Rating calculation, and best record tracking.
- **B50 Calculation**: Automatically calculates Best 50 (B35 Old + B15 New).
- **Data Export**: Export personal records to CSV.
- **Web Frontend**: Dark-themed UI built with Vue 3 + TypeScript + Naive UI.
- **API Documentation**: Integrated Swagger UI.
- **Cloud Native**: Docker and Docker Compose support.

## 🚀 Getting Started

### Local Development (Backend)

1. **Clone the repo**:

   ```bash
   git clone https://github.com/IceLocke/paradigm-reboot-prober-go.git
   cd paradigm-reboot-prober-go
   ```

2. **Configure**: Copy and edit the config file (make sure to change `secret_key`):

   ```bash
   cp config/config.yaml.example config/config.yaml
   # Edit config/config.yaml, change secret_key and database settings
   ```

3. **Run**:

   ```bash
   go run cmd/server/main.go
   ```

### Local Development (Frontend)

```bash
cd web
pnpm install
pnpm dev
```

The dev server automatically proxies API requests to the backend on port `:8080`.

### Using Docker Compose

```bash
docker-compose up -d
```

This starts the backend service and a PostgreSQL 16 database.

### Database Migration (from legacy)

To migrate data from the legacy Python backend:

```bash
go run cmd/migrate/main.go -config config/config.yaml
```

See `legacy/MIGRATION.md` for details.

## 📖 API Documentation

Visit: `http://localhost:8080/swagger/index.html`

## 🧪 Testing

```bash
go test -v ./...
```

Tests use in-memory SQLite databases — no external dependencies required.

## 📁 Project Structure

```
.
├── cmd/
│   ├── server/          # Application entry point
│   └── migrate/         # Database migration tool
├── config/              # Configuration files
├── internal/            # Internal packages (controller, service, repository, model, middleware, util)
├── pkg/                 # Reusable packages (auth, rating)
├── web/                 # Vue 3 frontend
├── docs/                # Swagger docs (auto-generated)
├── legacy/              # Legacy migration resources (SQL, OpenAPI spec)
└── scripts/             # Helper scripts
```

## 📄 License

[MIT](LICENSE)
