# Web Server Template

企业级 Go 后端 API 模板集合，基于 Go 1.25+、Gin、GORM、JWT、Casbin 与 OpenTelemetry。

## 模板对比

| 特性 | thin | cron-job |
|------|------|----------|
| RESTful API | ✅ | ✅ |
| JWT 认证 | ✅ | ✅ |
| Casbin RBAC | ✅ | ✅ |
| PostgreSQL + Redis | ✅ | ✅ |
| OpenTelemetry | ✅ | ✅ |
| Prometheus metrics | ✅ | ✅ |
| 定时任务 (robfig/cron) | - | ✅ |
| 异步任务队列 (asynq) | - | ✅ |
| 项目体积 | 轻量 | 完整 |

## gin-enterprise-template-thin

精简模板，适合不需要后台任务处理的场景。

**适用场景**：纯 API 服务、微服务、轻量级后端

```bash
cd gin-enterprise-template-thin
make deps && make build
```

## gin-enterprise-template-cron-job

完整模板，集成定时任务与异步任务队列能力。

**适用场景**：需要后台任务处理、数据同步、定时清理、异步邮件/消息等

```bash
cd gin-enterprise-template-cron-job
make deps && make build
```

## 共享特性

两个模板共享以下设计原则和工程实践：

- **整洁架构**：Handler → Biz → Store → Model 分层
- **依赖注入**：Google Wire
- **可观测性**：OpenTelemetry + Prometheus + 结构化 slog
- **代码规范**：Makefile、Protobuf、golangci-lint、表格驱动测试
- **安全**：JWT secret 校验、配置敏感信息提示

## 快速开始

### 1. 安装依赖

```bash
make deps
make tidy
make protoc
make generate
```

### 2. 配置

编辑 `configs/gin-enterprise-template-apiserver.yaml`：

```yaml
jwt:
  secret: "请替换为至少 32 字符随机字符串"

postgresql:
  addr: 127.0.0.1:5432
  password: "请替换为数据库密码"
```

### 3. 启动依赖服务

```bash
docker compose -f docker-compose.env.yml up -d
```

### 4. 构建运行

```bash
make build BINS=gin-enterprise-template-apiserver

_output/platforms/$(go env GOOS)/$(go env GOARCH)/gin-enterprise-template-apiserver \
  --config configs/gin-enterprise-template-apiserver.yaml
```

服务默认监听：http://localhost:5555

## 常用命令

| 命令 | 说明 |
|------|------|
| `make deps` | 安装开发依赖 |
| `make build` | 构建二进制 |
| `make test` | 单元测试 |
| `make lint` | 静态检查 |
| `make clean` | 清理构建产物 |

## 选择建议

- 只需要 RESTful API，不需要后台任务 → **thin**
- 需要定时任务、异步队列、完整企业功能 → **cron-job**
