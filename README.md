# gin-enterprise-template

`gin-enterprise-template` 是一个基于 Go 1.25+、Gin、GORM、JWT 与 OpenTelemetry 的后端 API skeleton。当前仓库保留的主干能力聚焦在：用户登录、刷新令牌、基础用户 CRUD、修改密码，以及可观测性与基础工程设施。

## 功能特性

- Gin REST API
- PostgreSQL + GORM 数据访问
- JWT Access Token / Refresh Token 认证、token version 撤销和 Redis 单次刷新
- Redis 跨实例登录限流与登录安全审计
- Transactional Outbox、Kafka、Inbox 和持久化 DLQ
- 用户管理：创建、查询、更新、删除、列表、修改密码
- OpenTelemetry + Prometheus `/metrics`
- Protobuf / OpenAPI / Wire / Makefile 工程化支持

## 目录结构

```text
gin-enterprise-template/
├── api/                         # OpenAPI 等生成产物
├── cmd/                         # 应用入口
├── configs/                     # 运行配置
├── docs/                        # 项目文档
├── internal/apiserver/          # apiserver 业务代码
│   ├── biz/
│   ├── handler/
│   ├── model/
│   ├── pkg/
│   └── store/
├── pkg/                         # 可复用公共包与 API 定义
├── scripts/                     # 辅助脚本
├── third_party/                 # 第三方 proto 依赖
├── docker-compose.env.yml       # 本地依赖服务 Compose
├── Dockerfile
├── Makefile
└── README.md
```

## 快速开始

### 1. 安装依赖并生成代码

```bash
make deps
make tidy
make protoc
make generate
```

### 2. 准备运行配置

默认会从用户配置目录、`./configs` 和当前目录查找 `configs.yaml`；也可通过
`--config` 显式指定。显式路径不存在或无法解析时服务会拒绝启动。

开发环境的数据库和 Redis 密码只用于本地容器。JWT secret 不写入仓库，启动前通过环境变量注入：

```bash
export APP_JWT_SECRET="$(openssl rand -hex 32)"
```

生产部署使用 `configs/configs.container.yaml` 作为无凭据基线，并通过 secret manager
或环境变量覆盖 `APP_POSTGRESQL_PASSWORD`、`APP_REDIS_PASSWORD` 和 TLS 配置。仓库不会
创建固定管理员，首个账号应由受控 bootstrap 流程创建。

已有数据库升级时按顺序执行 `configs/migrations/` 下的 SQL；`configs/schema.sql` 仅用于初始化空数据库。

### 3. 启动本地依赖服务（PostgreSQL、Redis、Redpanda 和 OTel Collector）

```bash
docker compose -f docker-compose.env.yml up -d
```

### 4. 构建并运行服务

```bash
make build BINS=gin-enterprise-template-apiserver

_output/platforms/$(go env GOOS)/$(go env GOARCH)/gin-enterprise-template-apiserver \
  --config configs/configs.yaml
```

## API 概览

### 通用接口

| 方法  | 路径            | 说明            |
| ----- | --------------- | --------------- |
| `GET` | `/healthz`      | 健康检查        |
| `GET` | `/readyz`       | PostgreSQL、Redis、Kafka 和异步 worker 就绪检查 |
| `GET` | `/metrics`      | Prometheus 指标 |

### 认证接口

| 方法   | 路径                     | 说明     |
| ------ | ------------------------ | -------- |
| `POST` | `/v1/auth/login`         | 用户登录 |
| `PUT`  | `/v1/auth/refresh-token` | 刷新令牌 |
| `GET`  | `/v1/auth/info`          | 获取当前用户信息 |
| `POST` | `/v1/auth/logout`        | 服务端撤销当前账号令牌 |

### 用户接口

| 方法     | 路径                                 | 说明     |
| -------- | ------------------------------------ | -------- |
| `POST`   | `/v1/users`                          | 创建用户 |
| `GET`    | `/v1/users`                          | 用户列表 |
| `GET`    | `/v1/users/{userID}`                 | 用户详情 |
| `PUT`    | `/v1/users/{userID}`                 | 更新用户 |
| `DELETE` | `/v1/users/{userID}`                 | 删除用户 |
| `PUT`    | `/v1/users/{userID}/change-password` | 修改密码 |

除公开注册外，用户资源只允许访问当前账号；在 RBAC 管理员角色落地前，用户列表接口默认返回 `403`。所有响应均使用 `{code,message,data,reason}` 包络；当前用户资料通过受保护的 `/v1/auth/info` 获取。

## 验证服务

```bash
curl -i http://localhost:5558/healthz
curl -i http://localhost:5558/readyz
curl -i http://localhost:5558/metrics
```

## 常用命令

```bash
make test
make cover
make lint
make verify
make verify.generated
make integration.up integration.test integration.smoke integration.down
go test -bench=. ./...
```

## 说明

用户与认证主干以代码和 OpenAPI 产物为准。
