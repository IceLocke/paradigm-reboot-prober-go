# Paradigm: Reboot Prober (Go)

[![CI/CD Pipeline](https://github.com/IceLocke/paradigm-reboot-prober-go/actions/workflows/ci.yml/badge.svg)](https://github.com/IceLocke/paradigm-reboot-prober-go/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/IceLocke/paradigm-reboot-prober-go)](https://go.dev/)
[![Docker Image](https://img.shields.io/badge/docker-ghcr.io-blue.svg)](https://github.com/IceLocke/paradigm-reboot-prober-go/pkgs/container/paradigm-reboot-prober-go)
[![License](https://img.shields.io/github/license/IceLocke/paradigm-reboot-prober-go)](LICENSE)

这是一个基于 Go 语言开发的 **Paradigm: Reboot** 查分器后端服务，搭配 Vue 3 前端。支持如下特性：

- **用户管理**: 注册、JWT 认证、个人资料更新、上传令牌、密码修改/重置。
- **曲目管理**: 曲目和谱面的增删改查（管理员操作）。
- **成绩管理**: 支持批量上传成绩（JSON），自动计算单曲 Rating 并维护最佳成绩。
- **B50 计算**: 自动筛选 B35 (旧曲) + B15 (新曲) 构成 Best 50。
- **数据导出**: 支持将个人成绩导出为 CSV 文件。
- **前端界面**: 基于 Vue 3 + TypeScript + Naive UI 的暗色主题 Web 界面。
- **API 文档**: 集成 Swagger 文档，方便对接。
- **容器化**: 支持 Docker 一键部署。

## 🚀 快速开始

### 本地运行（后端）

1. **克隆仓库**:

   ```bash
   git clone https://github.com/IceLocke/paradigm-reboot-prober-go.git
   cd paradigm-reboot-prober-go
   ```

2. **配置**: 复制并修改配置文件（务必修改 `secret_key`）:

   ```bash
   cp config/config.yaml.example config/config.yaml
   # 编辑 config/config.yaml，修改 secret_key 和数据库配置
   ```

3. **启动服务**:

   ```bash
   go run cmd/server/main.go
   ```

### 本地运行（前端）

```bash
cd web
pnpm install
pnpm dev
```

前端开发服务器会自动代理 API 请求到后端 `:8080` 端口。

### 使用 Docker Compose

```bash
docker-compose up -d
```

这会启动后端服务和 PostgreSQL 16 数据库。

### 数据库迁移（从旧版迁移）

如果需要从旧版 Python 后端迁移数据：

```bash
go run cmd/migrate/main.go -config config/config.yaml
```

详见 `legacy/MIGRATION.md`。

## 📖 API 文档

访问：`http://localhost:8080/swagger/index.html`

## 🧪 测试

```bash
go test -v ./...
```

测试使用内存 SQLite 数据库，无需外部依赖。

## 📁 项目结构

```
.
├── cmd/
│   ├── server/          # 应用入口
│   └── migrate/         # 数据库迁移工具
├── config/              # 配置文件
├── internal/            # 内部模块 (controller, service, repository, model, middleware, util)
├── pkg/                 # 可复用包 (auth, rating)
├── web/                 # Vue 3 前端
├── docs/                # Swagger 文档（自动生成）
├── legacy/              # 旧版迁移资料 (SQL, OpenAPI 规范)
└── scripts/             # 辅助脚本
```

## 📄 许可证

[MIT](LICENSE)
