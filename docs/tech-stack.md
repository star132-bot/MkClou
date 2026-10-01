# MkClou 技术选型文档

- 当前版本：v0.2
- 架构形态：前后端分离 + 后端**模块化单体**（Modular Monolith）

> 每项选型都附有理由和备选方案，用于开发决策和面试答辩。新增或替换技术时，更新本文档并在「变更记录」登记。

---

## 1. 总体架构

```
                ┌──────────────────────────────┐
  买家 / 商家 ──▶│  Nginx（HTTPS、静态资源、反向代理）│
                └───────┬──────────────┬───────┘
                        │              │
              ┌─────────▼───┐   ┌──────▼──────────┐
              │  web         │   │  server          │
              │  Next.js     │──▶│  Go（Gin）REST API│
              │  （SSR/前端） │   └──┬────┬────┬────┘
              └─────────────┘      │    │    │
                       ┌───────────┘    │    └──────────┐
                ┌──────▼─────┐  ┌───────▼──────┐  ┌──────▼──────┐
                │ MySQL 8     │  │ Redis         │  │ 对象存储     │
                │ 业务数据     │  │ 缓存/锁/队列   │  │ RustFS / OSS│
                └────────────┘  └───────┬──────┘  └─────────────┘
                                        │
                                ┌───────▼──────┐
                                │ worker（Asynq）│ 异步任务：超时关单、发邮件、自动交付
                                └──────────────┘
                                        │
                                ┌───────▼──────┐
                                │ 支付宝（沙箱） │
                                └──────────────┘
```

### 为什么是模块化单体，而不是微服务？
- 一个人开发，微服务会带来大量部署、调用链和分布式事务成本，收益很低。
- 按业务模块（用户、店铺、商品、订单、支付、交付）严格分包，模块之间只通过接口调用，**以后可以低成本拆分为微服务**。
- 面试回答要点：“根据团队规模和业务阶段选择架构，而不是追求流行”。

---

## 2. 前端（web）

| 用途 | 选型 | 理由 | 备选 |
|---|---|---|---|
| 框架 | **Next.js 16（App Router，默认 Turbopack）** | 店铺页和商品页需要 SEO 与首屏性能，SSR 天然支持 | Vite + React（纯 SPA，无 SEO） |
| 语言 | **TypeScript** | 类型安全，与后端接口契约对齐 | — |
| 样式 | **Tailwind CSS** | 与设计规范的 Token 一一对应，约束力强 | CSS Modules |
| 组件 | **shadcn/ui（Radix，Nova 预设）**；类名合并使用 shadcn 官方的 `cn` 包 | 源码在项目内、样式完全可控，默认风格符合设计规范 | Ant Design（风格难改，已弃用） |
| 图标 | Lucide | 线性风格统一，shadcn 默认搭配 | — |
| 动画 | Motion | React 生态最成熟的动画库 | CSS 动画 |
| 服务端状态 | **TanStack Query** | 请求缓存、重试、乐观更新、分页 | SWR |
| 客户端状态 | Zustand | 轻量，只存少量全局状态（如登录用户） | Redux Toolkit |
| 表单 | React Hook Form + Zod | 性能好，校验规则可复用 | Formik |
| 提示 | sonner | 设计规范指定 | — |
| 富文本 / 商品描述 | Markdown（react-markdown） | 简单安全，商家易用 | Tiptap |
| 图表（后台） | Recharts | 后台销售统计 | ECharts |
| 接口类型 | 由后端 OpenAPI 文档生成 TS 类型（openapi-typescript） | 前后端类型一致，避免手写出错 | 手写类型 |
| 包管理 | pnpm | 速度快、节省磁盘 | npm |

设计规范见 [design-system.md](./design-system.md)。

---

## 3. 后端（server）

| 用途 | 选型 | 理由 | 备选 |
|---|---|---|---|
| 语言 | **Go（最新稳定版）** | 性能好、并发模型简单、部署只需一个二进制文件 | — |
| Web 框架 | **Gin** | 国内使用最广、资料最多、面试最常见 | Echo、Hertz（字节） |
| ORM | **GORM** | 国内最主流，开发效率高；复杂查询用原生 SQL | sqlc、ent |
| 数据库迁移 | golang-migrate | 表结构变更有版本记录，可回滚 | GORM AutoMigrate（不适合生产） |
| 配置 | Viper | 支持 YAML + 环境变量覆盖 | — |
| 日志 | zap | 高性能结构化日志 | slog（标准库） |
| 参数校验 | validator（Gin 内置） | 结构体标签声明校验规则 | — |
| 认证 | **JWT（Access Token）+ Refresh Token（存 Redis）** | 无状态鉴权，同时支持主动登出和踢下线 | Session |
| 权限 | 自研 RBAC 中间件 | 角色少（买家 / 商家 / 平台管理员），无需引入 Casbin | Casbin |
| 密码加密 | bcrypt | 业界标准 | argon2 |
| 异步任务 | **Asynq**（基于 Redis） | 延时任务（超时关单）、重试、可视化面板 | RabbitMQ、Kafka（规模大时） |
| 支付 | **gopay** 对接支付宝（开发期用沙箱） | 社区成熟，支持支付宝和微信 | 官方 SDK |
| 对象存储 | RustFS（开发）/ 阿里云 OSS 或腾讯云 COS（生产） | S3 兼容接口，开发和生产切换无需改代码 | 本地磁盘 |
| 邮件 | SMTP（go-mail） | 订单通知、交付邮件 | 云邮件推送服务 |
| 接口文档 | **swag**（Swagger / OpenAPI） | 从代码注释生成文档，前端据此生成类型 | Apifox 手写 |
| 限流 | Redis + 令牌桶中间件 | 防刷登录、防刷下单 | — |
| ID 生成 | 主键自增 + 订单号使用雪花算法 | 订单号不暴露业务量，可按时间排序 | UUID |

### 3.1 分层结构

```
handler（HTTP 层：参数解析、校验、返回）
   ↓
service（业务逻辑、事务、状态机）
   ↓
repository（数据访问：MySQL / Redis）
```

- handler 不写业务逻辑，repository 不写业务判断。
- 模块之间只调用对方的 service 接口，不直接访问对方的数据表。

---

## 4. 数据存储

| 组件 | 选型 | 用途 |
|---|---|---|
| 关系数据库 | **MySQL 8** | 业务数据。国内岗位使用率最高，面试必考（索引、事务、锁） |
| 缓存 / 中间件 | **Redis 7** | 缓存、分布式锁、限流、Refresh Token、Asynq 队列 |
| 对象存储 | RustFS / OSS | 商品封面、虚拟商品文件（私有桶 + 签名 URL 下载） |

---

## 5. 测试

| 类型 | 工具 | 目标 |
|---|---|---|
| 后端单元测试 | Go 标准库 testing + testify | 核心 service（订单、支付回调、卡密分配）覆盖率 ≥ 80% |
| 后端集成测试 | testcontainers-go | 用真实 MySQL / Redis 容器测试 repository |
| 接口测试 | Apifox / Postman 集合 | 覆盖所有接口的正常和异常场景 |
| 前端单元测试 | Vitest + Testing Library | 关键组件和工具函数 |
| 端到端测试 | Playwright | 核心流程：注册 → 开店 → 上架 → 购买 → 交付 |
| 压力测试 | k6 | 下单和支付回调接口，产出压测报告 |

---

## 6. 工程化与部署

| 用途 | 选型 |
|---|---|
| 代码仓库 | Git + GitHub（monorepo） |
| 本地环境 | **Docker Compose**（MySQL、Redis、RustFS 一键启动） |
| 代码规范 | 后端 golangci-lint；前端 ESLint + Prettier |
| 提交规范 | Conventional Commits（`feat:` `fix:` `docs:` …） |
| CI | GitHub Actions：lint → 测试 → 构建镜像 |
| 部署 | 云服务器 + Docker Compose + Nginx + Let's Encrypt（HTTPS） |
| 监控（V2） | Prometheus + Grafana；错误追踪 Sentry |

---

## 7. 目录结构（monorepo）

```
MkClou/
├── docs/                 # 所有文档
├── web/                  # 前端 Next.js
│   ├── app/              # 路由页面（营销 / 店铺 / 后台）
│   ├── components/       # 通用组件（含 shadcn/ui）
│   ├── lib/              # 工具函数、API 客户端
│   └── styles/           # 全局样式与设计 Token
├── server/               # 后端 Go
│   ├── cmd/
│   │   ├── api/          # HTTP 服务入口
│   │   └── worker/       # 异步任务入口
│   ├── internal/
│   │   ├── user/         # 用户与认证模块
│   │   ├── shop/         # 店铺模块
│   │   ├── product/      # 商品模块
│   │   ├── order/        # 订单模块
│   │   ├── payment/      # 支付模块
│   │   ├── delivery/     # 交付模块（文件、卡密、链接）
│   │   └── pkg/          # 公共组件（中间件、错误码、响应封装）
│   ├── migrations/       # 数据库迁移脚本
│   └── configs/          # 配置文件
├── deploy/               # Docker Compose、Nginx 配置
└── .github/workflows/    # CI 配置
```

---

## 8. 简历技术亮点对照

| 亮点 | 对应实现 |
|---|---|
| 多租户数据隔离 | 所有业务表带 `shop_id`，中间件注入租户上下文，repository 层强制过滤 |
| 支付回调幂等 | 订单状态机 + 数据库唯一约束 + Redis 分布式锁 |
| 超时自动关单 | Asynq 延时任务 |
| 卡密防超卖 | MySQL 行锁（`SELECT … FOR UPDATE SKIP LOCKED`）或 Redis 原子操作 |
| 文件防盗链 | 私有存储桶 + 有时效的签名下载 URL + 下载次数限制 |
| 接口防刷 | Redis 令牌桶限流 |
| 无感续期登录 | Access Token + Refresh Token 轮换 |
| 工程化 | Docker Compose、CI、测试覆盖率、压测报告 |

---

## 变更记录

| 版本 | 日期 | 修改内容 |
|---|---|---|
| v0.1 | 2026-10-01 | 初始版本：确定前端 Next.js + shadcn/ui，后端 Go + Gin + GORM + MySQL + Redis + Asynq |
| v0.2 | 2026-10-01 | 搭建骨架时调整：MinIO 已停止发布社区版 Docker 镜像，开发环境改用 S3 兼容的 RustFS（Apache-2.0）；记录实际版本：Next.js 16.3、React 19.2、Tailwind CSS 4.3、Go 1.27、Gin 1.12、GORM 1.31、MySQL 8.4、Redis 7.4 |
