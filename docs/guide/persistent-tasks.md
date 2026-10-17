# 任务、计划和后台服务

## 启动和角色

需要 Go 1.26.9、PostgreSQL、Redis（API 认证/重放限流）；事件闭环还需要 Kafka。OTel 外部后端可选，开发用 `--otel.output-mode=classic`。配置优先级沿用 CLI > APP_* > YAML > 默认值。

```bash
docker compose -f docker-compose.env.yml up -d
make tools.goose
export PATH="$PWD/_output/tools:$PATH"
export GOOSE_DRIVER=postgres
export GOOSE_DBSTRING='postgres://postgres:dev-only-postgres@127.0.0.1:54321/enterprise_template?sslmode=disable'
goose -dir migrations up         # 空库初始化
make build                      # API、Worker、Scheduler 三个入口
export APP_JWT_SECRET="$(openssl rand -hex 32)"
go run ./cmd/gin-enterprise-template-apiserver --config configs/configs.yaml
go run ./cmd/gin-enterprise-template-worker --config configs/configs.yaml --outbox.enabled --consumer.enabled
go run ./cmd/gin-enterprise-template-scheduler --config configs/configs.yaml
```

三个长期进程分别监听 5558、5560、5561。API 保留业务 HTTP/gRPC，Worker 执行任务及既有事件链路，Scheduler 仅调度和清理。各角色共享 biz/store；任务能力统一位于 `internal/job`：根包负责任务注册、执行、重试及 Cron 解析，`runtime` 子包负责 Worker/Scheduler 装配、健康检查和生命周期，避免业务/Store 与运行装配形成循环依赖。Handler 不调用 Kafka，不启动后台 goroutine。Worker/Scheduler 不要求 JWT/Redis 配置；Worker 仅在开启事件链路时要求 Kafka。健康/指标端点应在私有网络。

`tasks`：enabled=true、concurrency=4（1..128）、poll-interval=1s、lease-duration=30s、heartbeat-interval=10s（租约至少为心跳三倍）、drain-timeout=30s、worker-addr=0.0.0.0:5560、scheduler-addr=0.0.0.0:5561。生产根据数据库总连接预算配置各进程连接池；不沿用开发默认 1000 连接。Compose 为每角色设置 cpu/memory，使用 `APP_IMAGE=registry/repo@sha256:...` 启动；同一镜像包含全部角色，可独立扩容。

## 模型和状态

Schedule 保存标识、名称、类型、参数/版本、五字段 Cron、IANA 时区、启停/归档、UTC next_run_at、misfire/concurrency、owner 和配置版本。owner 同时为创建者；修改者在审计记录。Task 保存不可变参数快照、来源、计划/配置版本/时点、状态、优先级、available_at、超时/重试预算、owner/type/key 作用域、结果引用、生命周期、request/trace。TaskAttempt 每次领取新增编号、Worker、租约、token、心跳、起止、结果和错误分类；不会重置原任务历史。

```mermaid
stateDiagram-v2
    pending --> running: 领取
    retry_wait --> running: 到期领取
    running --> running: 租约过期接管（新 attempt/token）
    running --> succeeded: 当前有效租约提交
    running --> retry_wait: 可重试且预算未耗尽
    running --> failed: 永久错误或预算耗尽
    pending --> canceled: 取消
    retry_wait --> canceled: 取消
    running --> canceled: 协作取消并提交/租约恢复
```

succeeded/failed/canceled 为终态。成功不能重试；失败或取消通过人工重试创建新 Task（retry_of 指向原实例），要求新的 Idempotency-Key，原 attempts 保留。相同 owner/type/key + 相同规范化参数返回同一实例，不同参数 409；键至少保留任务历史九十天，清理后不再承诺永久去重。

执行采用数据库时钟，领取和心跳仅短事务。候选先锁 Schedule 再锁 Task；更新、重试、调度使用一致锁顺序。取消 running 只设置 cancel_requested_at，Worker 心跳通知 Handler；最终事务取消优先于结果，取消请求前已发生的外部副作用不能自动撤销。超时或退出中断可重试，永久参数/权限错误直接失败。随机抖动指数退避上限五分钟，最大尝试跨重启保留。Panic 记录固定错误摘要，不记录任意 panic 文本。先停止领取，再保留心跳等待在途任务；超时终止协作上下文，非协作 Handler 最多再等待两秒，停止续租交给接管。

提交、心跳必须同时匹配 running、worker_id、fencing_token 和未过期 lease；旧 Worker 不能写结果。外部 HTTP 请求必须使用带 Context 的客户端与业务幂等键。不要在任务执行期间持有 TX。结果、任务终态和 report.generated.v1 Outbox 事件一次提交；Inbox 事务验证已提交结果并产生按 task_id 唯一的可查询通知。

## 定时可靠性

五字段 Cron，不支持秒、别名或内嵌 TZ。IANA 时区来自 Go tzdata；robfig/cron v3.0.1 只用于解析/Next 计算。春季不存在的墙钟时点跳过，秋季重复墙钟时间分别对应两个 UTC 时点，可各触发一次。唯一键 schedule_id + UTC scheduled_at；任务创建、触发历史和推进游标同事务。多 Scheduler 通过 SKIP LOCKED 获取最多 32 计划，行锁与配置更新串行化：先获得锁的配置版本确定本次触发；HTTP 修改需当前 version，否则 409。已创建任务保留旧快照。

skip：游标落后超过一分钟，记 misfire_skipped 后推进未来；catch_up：仅过去 24 小时、每计划每轮最多 100 时点，超出记 window_skipped/limit_skipped 并推进未来，绝不无限补跑。forbid：有 pending/running/retry_wait 时跳过触发；领取/人工重试还协调相同计划，禁止两个有效运行租约重叠。过期 Worker 的外部副作用仍需幂等。allow：允许并行。暂停/归档不取消已创建任务；恢复重新计算未来游标，不补暂停期间时点。

## API 和权限

[OpenAPI](../../api/openapi/apiserver.swagger.json) 和 [Proto](../../pkg/api/apiserver/v1/task.proto) 定义完整 REST 契约；新的 TaskService 为 REST 描述，当前 gRPC 业务仍为原 APIServer，不宣称实现新的 gRPC RPC。请求严格 JSON、最大 1 MiB；没有自动猜测参数版本。响应 snake_case 与原用户 camelCase API 并存。

| 接口 | 权限/行为 |
| --- | --- |
| POST /v1/reports | task.submit；days=1..90、Idempotency-Key，202 + task_id |
| GET /v1/tasks、/:taskID、/attempts | task.view；列表 page_token、最多100；all=true 需 .any |
| POST /v1/tasks/:taskID/cancel、/retry | task.cancel/task.retry；重试 202、新实例 |
| GET /v1/tasks/:taskID/result | result.download；所有者或显式 .any，CSV、no-store |
| GET /v1/report-notifications | 自己的完成通知，task.view |
| POST/GET /v1/schedules、GET/PUT /:scheduleID | schedule.manage；修改需 version |
| POST /v1/schedules/:scheduleID/pause、resume、archive；GET /history | schedule.manage；版本控制，最多100历史 |
| GET/POST /v1/roles；PUT /:roleID/permissions；POST /assign、revoke | rbac.manage；固定权限目录，内置角色不可改 |
| GET /v1/permissions、/audits、/dead-letters、/dead-letters/:id | rbac.manage/audit.view/dlq.view；仅白名单字段 |
| POST /v1/dead-letters/:id/replay、/dead-letters/replay | dlq.replay；Redis 10请求/分钟/操作者，单批1..20，幂等键 |

角色 grant 每次从数据库读取，JWT 不携带权威角色。普通用户只拥有自己的任务/计划/结果权限；管理员的 .any 是明确授予能力。后台读取报表只投影 owner 自己的用户资料及任务聚合，CSV 公式前缀转义，排除密码/hash/token/email/phone。完成事务重新锁定活跃 owner 和授权行，权限撤销在终态提交前生效。账号删除不删除审计。最后活跃管理员不能删除/停用。

先完成 Goose 迁移并通过公开用户注册创建正常账号，再在安装了 `psql` 的机器上配置 `PGHOST`、`PGPORT`、`PGDATABASE`、`PGUSER` 和 `.pgpass/PGPASSFILE`（或 `PGPASSWORD`），执行 `./scripts/bootstrap-admin.sh <existing-user-id>`。脚本不读取应用 YAML，只允许首次初始化；迁移版本、活跃账号、并发互斥、角色授予及审计在一次事务中校验和提交，失败整体回滚，后续用审计 RBAC API。没有固定管理员密码。迁移给现有及新注册用户 user 角色，不自动授予 admin。连接示例见 [README](../../README.md#后台能力与运行手册)。

## 新增 Handler

1. 明确任务与事件的边界、参数版本和授权范围，先写行为测试。
2. 在业务包定义结构化参数、纯验证函数及 `func(context.Context,P)(job.Result,error)`，通过 `job.Register` 显式注册，启动 Freeze。
3. Biz 在已有业务 TX 中创建 TaskStore 意图，校验参数、预算、权限和幂等键；Handler 只处理请求/响应。
4. 外部副作用使用 task_id 或业务唯一键去重。错误通过 job.Permanent 区分不可重试。尊重取消、限制结果大小，不输出敏感数据。
5. 新增参数版本同时保留旧版本执行器，待旧 pending/retry/history 保留窗口结束后移除。运行指标标签采用有限注册类型，ID 仅放日志/Trace。

持久化 Task/Schedule 手写模型和显式 SQL 用于行锁/token 操作；既有六表保持生成模型。新增模型不由 `make models` 覆盖。不要使用 AutoMigrate 更新生产 schema。
