# Paradigm: Reboot Prober (Go) 

[![CI/CD Pipeline](https://github.com/IceLocke/paradigm-reboot-prober-go/actions/workflows/ci.yml/badge.svg)](https://github.com/IceLocke/paradigm-reboot-prober-go/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/IceLocke/paradigm-reboot-prober-go)](https://go.dev/)
[![Docker Image](https://img.shields.io/badge/docker-ghcr.io-blue.svg)](https://github.com/IceLocke/paradigm-reboot-prober-go/pkgs/container/paradigm-reboot-prober-go)
[![License](https://img.shields.io/github/license/IceLocke/paradigm-reboot-prober-go)](LICENSE)

A backend service for **Paradigm: Reboot** score tracking, built with Go.

- **Score Management**: Batch upload support, automatic Rating calculation, and best record tracking.
- **B50 Calculation**: Automatically calculates Best 50 (B35 Old + B15 New).
- **Data Export**: Export personal records to CSV.
- **API Documentation**: Integrated Swagger UI.
- **Cloud Native**: Docker and Docker Compose support.

## 🚀 Getting Started

### Local Development

1. **Clone the repo**:

   ```bash
   git clone https://github.com/IceLocke/paradigm-reboot-prober-go.git
   cd paradigm-reboot-prober-go
   ```
2. **Run**:

   ```bash
   go run cmd/server/main.go
   ```

### Using Docker Compose

```bash
docker-compose up -d
```

## 📖 API Documentation

Visit: `http://localhost:8080/swagger/index.html`
