# MkClou

给独立创作者和小商家的虚拟商品店铺 SaaS：钱直接进商家自己的账户，平台不抽成，店铺有品牌感。

## 目录结构

```
.
├── docs/      # 产品与技术文档（立项、竞品、PRD、数据库、接口、设计规范）
├── server/    # 后端：Go + Gin + GORM + Asynq
├── web/       # 前端：Next.js 16 + Tailwind CSS 4 + shadcn/ui
└── deploy/    # 本地依赖服务（Docker Compose）与部署配置
```

文档入口：[立项说明](docs/project-charter.md) · [PRD](docs/prd/README.md) · [数据库设计](docs/database-design.md) · [接口清单](docs/api-list.md) · [设计规范](docs/design-system.md) · [技术选型](docs/tech-stack.md)

## 环境要求

| 工具 | 版本 |
|---|---|
| Go | 1.27+ |
| Node.js | 24+ |
| pnpm | 12+ |
| Docker Desktop | 最新版 |

## 本地启动

### 1. 启动依赖服务（MySQL、Redis、RustFS、Mailpit）

```bash
cp deploy/.env.example deploy/.env   # 首次：修改其中的密码
docker compose -f deploy/docker-compose.yml --env-file deploy/.env up -d
```

| 服务 | 本机地址 |
|---|---|
| MySQL 8.4 | `127.0.0.1:3307` |
| Redis 7.4 | `127.0.0.1:6380` |
| RustFS（S3 API / 控制台） | `127.0.0.1:9000` / `127.0.0.1:9001` |
| Mailpit（SMTP / 查看邮件） | `127.0.0.1:11025` / http://127.0.0.1:8025 |

### 2. 启动后端

```bash
cd server
cp .env.example .env                 # 首次：填入与 deploy/.env 一致的密码
go run ./cmd/migrate up              # 执行数据库迁移
go run ./cmd/api                     # API 服务：http://localhost:18080
go run ./cmd/worker                  # 异步任务（另开一个终端）
```

健康检查：`curl http://localhost:18080/readyz`

> 如果编译时出现 `cannot allocate memory`，说明系统可用内存不足，限制并行编译即可：
> `go env -w GOFLAGS=-p=4`

### 3. 启动前端

```bash
cd web
pnpm install
pnpm dev                             # http://localhost:3000
```

## 常用命令

| 位置 | 命令 | 说明 |
|---|---|---|
| server | `go vet ./... && go test ./...` | 静态检查与测试 |
| server | `go run ./cmd/migrate down 1` | 回滚最近一个迁移 |
| server | `python scripts/e2e/auth_e2e.py` | 账号模块端到端测试（需先启动 API 与 worker） |
| web | `pnpm lint` | ESLint 检查 |
| web | `pnpm exec tsc --noEmit` | 类型检查 |
| web | `pnpm build` | 生产构建 |
