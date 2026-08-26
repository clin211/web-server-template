# 生产级能力建设方案

## 1. 文档目的

本文定义 `gin-enterprise-template` 从 API skeleton 演进为生产级服务模板时，需要补齐的三项基础能力：

1. 统一、可靠且可关联的日志体系；
2. 基于 PostgreSQL Transactional Outbox 和 Kafka 的异步事件体系；
3. 规格驱动、测试驱动并由 CI 强制验证的 Vibe Coding 工程体系。

本文既是架构决策，也是后续实现和验收的事实来源。实现应遵循 YAGNI，只建设当前文档明确列出的基础能力，不预置未被业务验证的工作流引擎、调度平台或复杂事件框架。

### 1.1 当前实施状态

截至 2026-08-25，阶段 0、阶段 1 的应用侧闭环和阶段 2 的异步消费基础设施已落地：

- HTTP 安全日志、context 关联字段和 OTel 限时 flush 已实现；
- `configs/otel-collector.production.yaml` 提供生产后端、持久队列和失败重试模板；
- 通用 Outbox 已具备同事务写入、同步 Kafka producer、租约 Relay、attempt fencing 和带抖动指数退避能力；当前没有业务事件生产者，Relay 默认关闭；
- Consumer 已实现手动 offset 提交、可取消处理、重试、显式 handler 注册入口和优雅停止；没有业务 handler 时 fail-closed；
- Inbox 已实现基于 `(consumer_name, event_id)` 和源消息位置的持久幂等、租约及 fencing token；
- DLQ 已实现数据库优先落库、独立 topic Relay 发布、有限重试终态和失败元数据保留；
- Outbox、Consumer 和 DLQ 已接入低基数指标，Kafka 集群 lag 由外部 Kafka exporter 负责；
- 已提供 PostgreSQL/Kafka/Redis 集成测试、独立 Compose 环境、readiness 和启动 smoke test；
- 配置文件、环境变量和命令行优先级已闭合，显式配置文件错误会阻止启动；
- 登录限流、安全审计、refresh token 单次消费和 token version 撤销已接入 Redis/PostgreSQL；
- 规格模板、ADR、PR 模板和增量 CI 质量门禁已建立；
- 当前语义仍是 at-least-once，不能把实现描述为端到端 exactly-once；具体业务事件 handler 和故障注入仍属于后续工作。

## 2. 实施前基线与主要风险

### 2.1 日志与可观测性

仓库已经包含 `log/slog`、Zap、OpenTelemetry Logs、Traces、Metrics 和多组第三方日志适配器，但尚未形成单一日志入口：

- 业务代码主要调用全局 `slog`；
- `pkg/log` 仍维护另一套全局 Zap logger；
- Store、GORM、Resty 等组件存在多组不同的适配层；
- OTel 使用批处理导出，但服务退出时使用已经取消的 signal context 执行 Shutdown，可能无法完成 flush；
- HTTP 调试日志会读取并记录完整请求和响应 body，登录密码、token 和个人信息存在泄露风险；
- Collector 当前只将遥测数据输出到自身日志，没有持久化后端；
- 应用日志和不可丢失的安全审计记录尚未明确区分。

### 2.2 异步能力

仓库已有 `kafka-go` 和 Kafka 配置对象，但尚未接入应用配置、Wire、业务事务和服务生命周期：

- Kafka Writer 默认异步发送，调用者无法可靠感知发送失败；
- 没有解决数据库提交成功但消息发送失败的双写一致性问题；
- 没有事件 envelope、幂等消费、重试、死信和重放协议；
- 没有 worker readiness、优雅停止和消息积压指标；
- `audit_log` 和 `user_login_log` 是业务审计数据，不应通过普通应用日志替代。

### 2.3 Vibe Coding

仓库已有 `AGENTS.md`、项目宪法和本地技能，但主要约束仍停留在文本说明：

- GitHub Actions 只有 release 分支部署，没有 Pull Request 质量门禁；
- 全局覆盖率阈值只有 1%，无法阻止关键逻辑缺少测试；
- 缺少统一的需求规格、架构决策和 PR 模板；
- 缺少生成代码一致性、漏洞、凭据和迁移校验；
- 多个工具专属说明可能产生规则漂移；
- AI 是否生成代码不能替代代码所有者的审查责任。

## 3. 总体架构原则

### 3.1 依赖方向

现有业务依赖方向保持不变：

```text
handler -> biz -> store -> model
```

日志作为横切能力由依赖注入提供。事件由 Biz 在业务事务中写入 Outbox，Store 不主动发布消息，Handler 不直接启动 goroutine 或调用 Kafka。

### 3.2 可靠性语义

- HTTP 请求与数据库事务：同步返回明确结果；
- 应用日志：允许批处理和有限丢失，不得反向阻塞核心业务；
- 审计日志：作为业务数据可靠持久化；
- 业务事件：采用 at-least-once 投递；
- 消费者：通过 `event_id` 实现幂等，不宣称无法证明的 exactly-once；
- 所有后台 goroutine 必须受 `context.Context` 管理，并能在超时内停止。

## 4. 日志体系方案

### 4.1 目标链路

```text
业务代码 / 中间件 / 数据库适配器
                |
                v
             log/slog
                |
                v
     JSON stdout 或 OTel Batch Processor
                |
                v
        OpenTelemetry Collector
                |
                v
        Loki / Elasticsearch / 云日志
```

容器环境默认采用 JSON stdout，由 Collector Agent、DaemonSet 或平台日志代理采集。应用不负责文件轮转，也不在请求路径同步写远端日志服务。

### 4.2 单一入口

- 新代码统一使用 `log/slog`；
- 组件接收 `*slog.Logger` 或最小消费方接口，不新增全局可变 logger；
- 第三方组件通过 adapter 接入同一个 `slog.Handler`；
- `pkg/log` 的 Zap 全局入口进入弃用路径，待调用方迁移完成后移除；
- logger 级别、格式、输出目标只允许一份配置来源。

### 4.3 字段协议

所有生产日志使用结构化字段，字段名保持稳定：

| 分类 | 字段 |
| --- | --- |
| 服务 | `service.name`、`service.version`、`deployment.environment`、`service.instance.id` |
| 事件 | `event.name`、`severity`、`timestamp` |
| 关联 | `trace_id`、`span_id`、`request_id`、`user_id` |
| HTTP | `http.request.method`、`http.route`、`http.response.status_code`、`duration_ms` |
| 错误 | `error.type`、`error.message` |
| 异步 | `messaging.system`、`messaging.destination`、`messaging.message.id`、`messaging.operation` |

业务日志记录事件结果和必要标识，不记录完整对象、密码、JWT、Authorization、Cookie、数据库连接串或个人敏感信息。

### 4.4 HTTP 日志安全

- 默认不记录请求和响应 body；
- 调试模式也只允许白名单路由和白名单字段；
- `password`、`token`、`secret`、`authorization`、`cookie` 等字段必须递归脱敏；
- body 捕获设置最大字节数，避免放大内存使用；
- 使用 Gin 路由模板而不是原始 URL，避免高基数字段；
- 根据 HTTP 状态选择级别：正常请求为 Info，客户端错误按策略为 Info/Warn，服务端错误为 Error。

### 4.5 关闭与故障策略

- OTel provider 使用独立的 shutdown timeout，不复用已取消的 signal context；
- Shutdown、文件关闭和 exporter flush 错误必须显式聚合并记录；
- 日志后端不可用时，应用继续服务并暴露 exporter failure/drop 指标；
- Error 日志不做概率采样；高频成功访问日志可在 Collector 侧采样；
- 审计记录不经过应用日志采样链路。

### 4.6 验收标准

- 同一次 HTTP 请求可通过 `trace_id`、`request_id` 关联日志和 trace；
- 登录和刷新 token 请求的日志中不出现密码、token 或 secret；
- 正常退出时批处理数据在限定时间内 flush；
- Collector 不可用不会导致请求线程无限阻塞；
- 日志字段具有单元测试，关闭流程具有错误和超时测试。

## 5. 异步事件方案

### 5.1 目标链路

```text
HTTP / gRPC
    |
    v
Biz 业务事务
    |-- 更新业务表
    `-- 写入 outbox_event
             |
             v
        Outbox Relay
             |
             v
            Kafka
             |
             v
          Consumer
             |
             |-- inbox_event 幂等占位
             |-- 执行业务处理
             `-- 成功后提交 offset
```

### 5.2 Outbox 数据模型

`outbox_event` 至少包含：

| 字段 | 说明 |
| --- | --- |
| `id` | 内部自增主键 |
| `event_id` | 对外不可枚举的消息唯一标识，唯一索引 |
| `event_type` | 带版本的事件类型，例如 `example.created.v1` |
| `aggregate_type` | 聚合类型 |
| `aggregate_id` | 聚合业务标识 |
| `payload` | JSONB 事件载荷 |
| `trace_id` | 产生事件时的 trace 标识 |
| `status` | `pending`、`processing`、`published`、`failed`、`dead` |
| `attempts` | 已尝试次数 |
| `available_at` | 下次可处理时间 |
| `published_at` | 成功发布时间 |
| `last_error` | 最后一次错误摘要 |
| `created_at`、`updated_at` | 生命周期时间 |

Relay 使用短事务领取批次，支持多个实例通过 `FOR UPDATE SKIP LOCKED` 协作。网络发布期间不长期持有数据库锁；领取状态需要租约或超时恢复机制，避免进程崩溃后事件永久停留在 `processing`。

租约必须带 fencing token。当前实现使用领取后递增的 `attempts` 作为 token，`MarkPublished` 和 `MarkFailed` 仅允许更新仍为 `processing` 且 attempt 匹配的记录。旧 worker 在租约过期后的迟到结果返回 `ErrLeaseLost`，不得覆盖新 worker 的最终状态。

### 5.3 事件 Envelope

所有 Kafka 事件采用统一 envelope：

```json
{
  "event_id": "evt_xxx",
  "event_type": "example.created.v1",
  "occurred_at": "2026-08-21T10:00:00Z",
  "producer": "gin-enterprise-template-apiserver",
  "trace_id": "...",
  "aggregate_type": "example",
  "aggregate_id": "example_xxx",
  "payload": {}
}
```

`event_type` 的版本属于兼容性契约。消费者必须忽略未知的可选字段；破坏性 payload 变化通过新事件版本发布。

### 5.4 发布与消费策略

- Kafka Writer 必须以同步方式获取发送结果，并等待全部 ISR 确认；
- Kafka message key 使用 `aggregate_id`，保证同一聚合内的相对顺序；
- Relay 失败使用带随机抖动的指数退避；
- 达到最大次数后标记失败并发送告警，支持显式重放；
- Consumer 在处理成功后手动提交 offset；
- `inbox_event.event_id` 唯一约束作为最终幂等屏障；
- poison message 超过阈值进入 DLQ，DLQ 保留原消息、错误摘要和消费元数据；
- 不在无限循环中使用不可取消的 sleep，轮询等待必须响应 context 取消。

### 5.5 Worker 生命周期与并发

- HTTP Server、Outbox Relay 和 Consumer 都由显式生命周期管理；
- 每个 worker 的 context 来自服务根 context；
- 限制发布和消费并发度，避免数据库或下游过载；
- goroutine 的创建方负责等待其退出；
- channel 由发送方关闭，不通过 recover 掩盖关闭错误；
- 共享状态优先避免；必须共享时使用明确的 mutex 或原子类型；
- 关闭顺序为停止接收新工作、等待在途任务、flush producer、关闭连接；
- 并发代码必须通过 `go test -race`。

### 5.6 可观测性

至少暴露：

- `outbox_pending`（Observable Gauge，待补充 Store 聚合查询后启用）；
- `outbox_oldest_pending_seconds`（Observable Gauge，待补充 Store 聚合查询后启用）；
- `outbox_publish_total{result,event_type}`；
- `outbox_publish_duration_seconds`；
- `consumer_process_total{result,event_type}`；
- `consumer_process_duration_seconds`；
- Kafka consumer lag；
- DLQ 数量和最后失败时间。

Trace context 通过 Kafka headers 传播。日志必须包含 `event_id`、topic、partition、offset 和消费尝试次数。

### 5.7 第一阶段边界

首个垂直切片只建设通用 Outbox 能力，不绑定用户 CRUD 等具体业务事件。业务存在明确消费者和一致性需求时，再定义事件契约并在业务事务中接入；不同时引入 Redis Queue、Asynq、工作流引擎和定时任务平台。确有延迟任务需求时，再基于独立 ADR 选择设施。

### 5.8 验收标准

- 数据库事务回滚时不会留下 Outbox 事件；
- Kafka 短暂不可用时事件保留并可重试；
- 同一事件重复投递不会重复产生业务副作用；
- 两个 Relay 实例不会同时拥有同一事件租约；
- worker 收到取消信号后在超时内停止且无 goroutine 泄漏；
- 单元测试、PostgreSQL 集成测试和 race 测试覆盖核心路径。

## 6. Vibe Coding 工程体系

### 6.1 定位

Vibe Coding 属于研发控制面，不嵌入业务服务运行时。AI 可以辅助分析、生成、测试和审查，但不能绕过规格、测试、代码所有者审批和发布权限。

### 6.2 单一规则源

- `AGENTS.md` 是所有 AI 工具的仓库入口；
- `.claude/constitution.md` 保存不可违背的开发原则；
- 工具专属配置只引用上述入口，不复制完整规则；
- `README.md`、配置文件名、端口和 Makefile 命令必须通过自动检查保持一致；
- 架构变化使用 `docs/architecture/decisions/` 下的 ADR 记录。

### 6.3 规格驱动流程

每个非平凡变更使用 `specs/<feature>/spec.md` 描述：

- 背景与目标；
- 明确的非目标；
- 用户场景和验收条件；
- API、数据和事件契约；
- 安全、隐私、可观测性和回滚要求；
- 测试计划和风险。

交付流程固定为：

```text
需求 -> 规格 -> 实现计划 -> Red -> Green -> Refactor
     -> 本地 verify -> AI 辅助审查 -> 人工审查 -> CI -> 合并
```

### 6.4 本地统一门禁

`make verify` 应成为开发者和 AI 的统一完成定义，至少包含：

- 格式化差异检查；
- `go vet`；
- `go test -race`；
- 覆盖率检查；
- `golangci-lint`；
- 目标二进制构建；
- Proto、OpenAPI、Wire 等生成物一致性检查。

快速测试和完整验证可以拆分，但 PR 合并前必须执行完整门禁。

### 6.5 Pull Request CI

Pull Request 至少执行：

- Go 版本与依赖缓存初始化；
- `make verify`；
- `govulncheck`；
- secret scanning；
- Docker build；
- PostgreSQL/Redis 集成测试和启动 smoke test；
- schema、Proto 和 OpenAPI 兼容性检查。

部署工作流只能依赖已经通过 CI 的不可变构建产物，不能在部署阶段重新生成一份未经验证的二进制。

### 6.6 AI 权限与审查

- AI 不获取生产数据库、云账号、Registry 和部署主机凭据；
- AI 不直接合并受保护分支，也不自动批准自己的变更；
- PR 描述记录需求来源、影响范围、测试证据、生成物和回滚方式；
- 安全、认证、数据库迁移和消息语义变更必须由代码所有者人工审查；
- AI 生成的测试必须验证行为，不能只验证实现细节或无意义地提高覆盖率。

### 6.7 质量目标

- 第一阶段将全局覆盖率门槛从 1% 提升到当前可持续通过的基线，并禁止回退；
- 认证、token、事务、Outbox、幂等和安全脱敏等关键路径目标覆盖率不低于 90%；
- 全局覆盖率逐步提升到 60% 以上，不通过排除业务包伪造结果；
- 主分支保持可构建、可测试、可部署；
- 所有生成物均可重复生成且无未提交差异。

## 7. 分阶段实施计划

### 阶段 0：安全与工程基线

- 移除示例配置中的真实形态凭据并轮换已暴露凭据；
- 禁止 HTTP body 默认日志，增加脱敏和边界测试；
- 增加 `make verify`、PR CI、规格和 PR 模板；
- 修正文档与实际配置文件名、端口不一致问题。

### 阶段 1：日志闭环

- 统一 `slog` 字段和 context 提取；
- 修复 OTel shutdown context 与错误处理；
- 为第三方 logger 提供统一 adapter；
- 配置 Collector 的生产后端、队列、重试和告警；
- 分阶段迁移并弃用重复 logger。

### 阶段 2：异步最小垂直切片

- 新增 Outbox schema、model 和 store；
- 新增事件 envelope 和 publisher 小接口；
- 实现可取消、可重试的 Relay；
- 将 Kafka 配置接入 ServerOptions 和 Wire；
- 通过通用 Outbox Store 集成测试验证事务与租约一致性，不预置无消费者的业务事件；
- 增加幂等消费、DLQ、指标和集成测试。（已完成）

生产 Collector 使用 `configs/otel-collector.production.yaml`，部署时必须提供 `OTEL_BACKEND_ENDPOINT`，可选提供 `OTEL_BACKEND_AUTHORIZATION`，并将 `/var/lib/otelcol/queue` 挂载到持久卷。具体 Loki、Elasticsearch 或云日志后端及告警规则属于部署环境配置，不应把供应商凭据写入仓库。

### 阶段 3：持续治理

- 提升覆盖率门槛；
- 增加架构依赖和 API 兼容性检查；
- 建立日志字段、事件 schema 和 ADR 的版本治理；
- 通过故障注入验证 Collector、Kafka、PostgreSQL 中断和进程退出场景；
- 完善业务事件 handler 注册，并为重试、DLQ Relay 和 offset 提交失败增加故障注入测试；
- 增加 DLQ 告警规则、重放运维流程和关键路径覆盖率报告。

## 8. 明确不做的事项

- 不在 Handler 中通过裸 goroutine 实现可靠异步；
- 不把普通日志当作审计日志；
- 不承诺 Kafka 与 PostgreSQL 的 exactly-once；
- 不同时引入多个消息队列；
- 不让 AI 直接访问生产凭据或执行无审批部署；
- 不为提高覆盖率编写只覆盖 getter、生成代码或无业务断言的测试；
- 不在没有实际业务需求前引入工作流引擎、分布式锁或复杂插件体系。

## 9. 完成定义

三项能力全部完成时，仓库应满足：

- 一个日志 API、一套字段协议、一条可查询和告警的采集链路；
- 一个具备事务一致性、重试、幂等、DLQ、指标和优雅关闭的异步事件链路；
- 一个 AI 和人类共同遵守、由本地命令与 CI 强制执行的开发闭环；
- 无仓库内明文生产凭据；
- `make verify`、构建、集成测试和必要的故障场景验证全部通过。
