# gin-enterprise-template 项目代理指南

本文件是本仓库的代理协作入口。对话、计划、技术说明和文档统一使用简体中文；代码、命令、路径和 API 标识保留其原文。实现前先阅读与任务相关的现有代码和测试，保持改动小而明确。

## 规则优先级

1. 系统和用户当前请求优先。
2. 项目级 `.claude` 配置、宪法和技能已于 `8c2adc7` 移除；不根据历史路径假设文件存在。
3. 以本文件、当前代码、`README.md` 和 `Makefile` 为事实来源。
4. 使用当前会话可用且相关的技能；本文件中的开发规范继续适用。

## 项目概况

- 模块：`github.com/clin211/gin-enterprise-template`
- Go：`go 1.26.9`（项目要求 Go 1.25+）
- 类型：Gin 后端 API skeleton
- 当前业务主干：用户登录、刷新令牌、用户 CRUD、修改密码。
- 主要组件：Gin、GORM、PostgreSQL、Redis、JWT、OpenTelemetry、Prometheus、slog、Google Wire、Protobuf/OpenAPI。
- 当前仓库保留用户业务，并新增 PostgreSQL 任务、计划、基础 RBAC、可靠审计和报表事件示例。管理前端、菜单及多租户仍不实现；以当前代码为准。

## 目录职责

```text
cmd/                         应用入口和代码生成工具
internal/apiserver/          业务实现
  handler/                   HTTP/gRPC 处理层
  biz/                       业务逻辑层
  store/                     数据访问层
  model/                     GORM 生成模型
  pkg/conversion/            Model 与 Proto 转换
  pkg/validation/            请求校验
internal/job/                任务注册、执行、重试和 Cron 解析
  runtime/                   Worker/Scheduler 装配、健康检查与生命周期
internal/messaging/          异步消息基础设施
  consumer/                  Kafka 消费、Inbox 幂等和 DLQ 处理
  outbox/                    事务事件契约、Kafka 发布和 Relay
pkg/api/apiserver/v1/        Proto API 定义及生成代码
pkg/                         可复用公共包
migrations/                  Goose SQL 迁移文件（不放 Go 代码）
internal/pkg/dbschema/       只读 Goose 版本检查（不执行迁移）
configs/                     本地配置、数据库 schema、OTel 配置
api/openapi/                 OpenAPI 生成产物
docs/                        项目文档
scripts/                     辅助脚本
third_party/                 第三方 Proto 依赖
```

依赖方向应保持为：

```text
handler -> biz -> store -> model
```

公共能力通过 `pkg/` 或 `internal/pkg/` 提供；依赖注入入口是 `internal/apiserver/wire.go`，生成文件为 `wire_gen.go`。

## 常用命令

```bash
# 依赖与生成
make deps
make tidy
make protoc
make generate
make models

# 格式化、构建、测试、检查
make format
make build BINS=gin-enterprise-template-apiserver
make test
make cover
make lint
go test -bench=. ./...

# 启动本地依赖
docker compose -f docker-compose.env.yml up -d
```

当前实际配置文件是 `configs/configs.yaml`。构建后可使用：

```bash
_output/platforms/$(go env GOOS)/$(go env GOARCH)/gin-enterprise-template-apiserver \
  --config configs/configs.yaml
```

运行前检查 JWT secret、PostgreSQL、Redis 和 OpenTelemetry endpoint；不要把真实凭据提交到仓库。服务默认监听配置文件中的 `http.addr`，当前为 `0.0.0.0:5558`。

## 开发规范

### 宪法要求

- 只实现明确需求，遵循 YAGNI，优先复用标准库和仓库已有能力。
- 新功能或 Bug 修复遵循 Red-Green-Refactor：先写失败测试，再实现，再重构。
- 测试优先采用表格驱动；优先真实依赖或 fake object，避免过度 Mock。
- 所有错误必须显式处理；错误传递使用 `fmt.Errorf("...: %w", err)` 包装，不要用 `_` 丢弃错误。
- 不使用全局变量传递可变状态；依赖通过参数或结构体显式注入。
- 公共符号提供以符号名开头的 GoDoc；注释解释原因，不重复代码表面行为。
- 包、文件、函数遵循单一职责；接口由消费者定义，并保持小而专一。

### Go 代码质量

- 提交前运行 `gofmt`；适用时使用 `goimports` 和 `golangci-lint`。
- 遵循 Effective Go：MixedCaps 命名、返回错误而非无理由 panic、接收接口参数并返回具体类型。
- 使用 `log/slog` 结构化日志，按上下文记录 `traceID`、`userID` 等信息；不要记录密码、JWT secret 等敏感信息。
- 涉及 goroutine、channel、锁或共享状态时，说明竞态风险和同步策略，并用 `go test -race` 验证。

## 新增 API/资源的推荐顺序

1. 阅读需求、现有相邻资源、相关测试、`configs/schema.sql` 和 API 定义。
2. 在 `pkg/api/apiserver/v1/` 定义 Proto，并更新 service；运行 `make protoc`。
3. 如有数据库变更，更新 schema 和 GORM 生成配置；运行 `make models`。
4. 按 `model -> store -> conversion/validation -> biz -> handler` 分层实现。
5. Biz 中按动作拆分文件，例如 `create.go`、`update.go`、`delete.go`、`refreshtoken.go`。
6. 更新 Wire 注册和生成代码（必要时运行 `make generate`）。
7. 先运行相关包测试，再运行 `make test`、`make cover` 和必要的 lint/race 检查。

新增资源命名应明确区分：类型名使用大写驼峰，单数和复数使用小写驼峰；非生成文件和目录使用小写字母拼接，不使用特殊字符。

## 数据库约定

- PostgreSQL 表通常采用内部自增 `id` 与对外业务标识的双标识设计；对外 API 不暴露可枚举的内部 ID。
- 时间字段优先使用 `timestamptz`；外键、级联删除、唯一约束和索引命名保持 schema 现有风格。
- 软删除查询使用部分索引等优化时，先确认现有表是否真的存在 `deleted_at` 字段，不要套用模板示例。
- 数据库表、字段和复杂索引添加清晰注释；变更同步更新 `configs/schema.sql` 和必要的清理脚本。

## API 与验证

当前通用入口包括 `/healthz`、`/metrics` 和 `/debug/pprof/`；认证入口包括 `/v1/auth/login`、`/v1/auth/refresh-token`；用户入口以 `README.md` 和 Proto/OpenAPI 产物为准。

修改接口时同步考虑：请求默认值、参数校验、错误码、认证中间件、OpenAPI 生成物、日志和可观测性。

国际化职责分离：`pkg/i18n` 仅提供通用翻译机制；项目词表和语言协商策略放在 `internal/pkg/localization`，由应用启动时装配注入。公共包不得依赖项目内部词表，CLI 默认文案由 `pkg/app` 自身维护。动态文案使用 go-i18n 原生 `{{.MaxLength}}` 模板和 `i18n.Params`，业务通过 `WithMessageID(...).WithMessageParams(...)` 传递阈值；校验和提示参数共用业务常量，不自行实现字符串替换。

## Git 约定

提交消息使用 Conventional Commits：

```text
<type>(<scope>): <subject>
```

常用类型：`feat`、`fix`、`docs`、`refactor`、`test`、`chore`、`build`、`ci`、`perf`。需要提交时包含 `Signed-off-by` 脚注，并在提交前检查工作区，避免提交无关文件或生成物。

## 交付检查清单

- [ ] 已阅读相关现有代码、测试和配置。
- [ ] 需求范围和实现计划符合项目宪法。
- [ ] 新增或修改行为有测试，优先表格驱动。
- [ ] 错误、日志、并发安全和敏感信息处理正确。
- [ ] 已运行适用的格式化、测试、覆盖率、lint 或 race 检查。
- [ ] 生成代码、Proto、OpenAPI、schema 和 Wire 状态保持同步。
- [ ] 最终说明使用中文，并列出验证命令及未运行项目。
