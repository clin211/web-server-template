# gin-enterprise-template

`gin-enterprise-template` 是一个基于 Go 1.25+、Gin、GORM、JWT 与 OpenTelemetry 的后端 API skeleton。当前仓库保留的主干能力聚焦在：用户登录、刷新令牌、基础用户 CRUD、修改密码，以及可观测性与基础工程设施。

## 功能特性

- Gin REST API
- PostgreSQL + GORM 数据访问
- JWT Access Token / Refresh Token 认证
- 用户管理：创建、查询、更新、删除、列表、修改密码
- OpenTelemetry + Prometheus `/metrics`
- `pprof` 诊断入口 `/debug/pprof/`
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

默认配置文件：`configs/gin-enterprise-template-apiserver.yaml`

至少需要更新：

```yaml
jwt:
  secret: "请替换为至少 32 字符随机字符串"

postgresql:
  addr: 127.0.0.1:5432
  database: template
  username: postgres
  password: "请替换为数据库密码"

redis:
  addr: 127.0.0.1:6379
  password: "请替换为 Redis 密码"
```

### 3. 启动本地依赖服务

```bash
docker compose -f docker-compose.env.yml up -d
```

### 4. 构建并运行服务

```bash
make build BINS=gin-enterprise-template-apiserver

_output/platforms/$(go env GOOS)/$(go env GOARCH)/gin-enterprise-template-apiserver \
  --config configs/gin-enterprise-template-apiserver.yaml
```

## API 概览

### 通用接口

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/healthz` | 健康检查 |
| `GET` | `/metrics` | Prometheus 指标 |
| `GET` | `/debug/pprof/` | pprof 入口 |

### 认证接口

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/v1/auth/login` | 用户登录 |
| `PUT` | `/v1/auth/refresh-token` | 刷新令牌 |

### 用户接口

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/v1/users` | 创建用户 |
| `GET` | `/v1/users` | 用户列表 |
| `GET` | `/v1/users/{userID}` | 用户详情 |
| `PUT` | `/v1/users/{userID}` | 更新用户 |
| `DELETE` | `/v1/users/{userID}` | 删除用户 |
| `PUT` | `/v1/users/{userID}/change-password` | 修改密码 |

## 验证服务

```bash
curl -i http://localhost:5555/healthz
curl -i http://localhost:5555/metrics
curl -i http://localhost:5555/debug/pprof/
```

## 常用命令

```bash
make test
make cover
make lint
go test -bench=. ./...
```

## 说明

当前仓库已移除菜单、角色、权限、用户角色、定时任务和管理前端相关实现；用户与认证主干以代码和 OpenAPI 产物为准。
