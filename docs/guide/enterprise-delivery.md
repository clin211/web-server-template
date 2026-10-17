# 企业模板改造交付记录

实施工作区：`/private/tmp/web-server-enterprise`，分支 `feat/persistent-tasks`。首次代码审查基线为 `2ecca79507539fd8b028edf0d7b1c13abc4bd59d`；收尾核验 `main` 已更新为 `8c2adc768ac7987f67d8e0977edea8b5a8eb19b6`，仅删除项目级 `.claude`，实施分支已快进并保留该删除。初次交付时变更保留在隔离工作区，尚未提交或推送。2026-10-15 按用户要求，将全部改动整合到当前项目 `/Users/forest/code/backend/Go/web-server-template` 的 main 工作区；未创建提交、未推送或发布。整合前当前工作区干净，两个工作区 HEAD 均为 `8c2adc7`，原有文件备份及逐文件 SHA256 清单保存在 `/private/tmp/enterprise-current-project-before-integration.tar.gz` 和 `/private/tmp/enterprise-transfer-manifest.json`。

## 现状核验与改造矩阵

“依赖”只表示 go.mod 引入；“实现”表示存在执行路径；“接入”表示业务调用该路径；“验证”表示本次实际执行测试；生产发布需另外完成远程质量流水线、环境配置与运维演练。

| 能力 | 原有实现及状态 | 原有未完成部分 | 代码证据 | 本次改造方式 | 优先级 |
|---|---|---|---|---|---|
| API 与装配 | Gin/GORM/PostgreSQL/Redis/JWT/Wire 已实现且用户业务已接入；OTel 已实现 | 只有 API 进程 | `internal/apiserver/wire.go`、`server.go`、`cmd/` | 共用业务和配置，增加 Worker、Scheduler、迁移及首管理员入口 | P0 |
| 事务事件 | Outbox/Kafka/Inbox/持久化 DLQ 已实现，有基础测试 | 未注册实际业务 Handler，可靠事件提交未形成示例闭环 | `internal/messaging/`、`internal/apiserver/asyncruntime.go` | 报表结果与 report.generated.v1 同事务，Inbox 内写幂等通知，独立 Worker 运行链路 | P1 |
| 消费顺序 | 手动 offset 提交、串行消费已实现 | 不能将 Kafka key 当作全链路顺序保证 | `internal/messaging/consumer/consumer.go`、`task_kafka_integration_test.go` | 保留串行安全提交；真实发布/业务/提交失败与重启恢复测试，文档明确顺序边界 | P1 |
| 异步任务 | 无模型、执行或业务接入 | 状态、历史、恢复、幂等、取消缺失 | `store/task.go`、`internal/job/worker.go` | PostgreSQL 短事务领取、租约/心跳/fencing、类型版本注册、有限重试/并发/退出 | P0 |
| 定时计划 | 当前代码无计划模块 | 持久化、多实例协调及停机补跑缺失 | `store/schedule.go`、`internal/job/schedule.go` | 计划行锁、任务/时点唯一约束、原子推进游标，24 小时/100 次补跑上限，allow/forbid | P0 |
| 报表示例 | 无业务接入 | 202、结果、周期、权限边界缺失 | `biz/v1/task/service.go`、`handler/task.go` | 自有数据白名单 CSV，事务创建任务，共享数据库结果，事件通知与计划复用 | P0 |
| RBAC/审计 | JWT 认证；用户资源默认仅本人；原用户操作已有审计 | 角色和管理员能力未实现，管理接口缺失 | `store/authorization.go`、`validation/user.go`、`0002_tasks.sql` | 实时授权查询、固定权限目录/自定义角色、受控首管理员、可靠审计与最后管理员保护 | P0 |
| DLQ 管理 | 持久化与发布已实现 | 管理、重放、Inbox 终态兼容未闭环 | `biz/v1/task/service.go`、`task_http_integration_test.go` | 最小权限、限流/批次上限、新事件身份重放、原业务唯一约束、审计和保留窗口 | P1 |
| 遥测 | OTel/Prometheus/slog 和部分消息指标已实现 | 缺任务、计划、积压、Worker、告警与操作手册 | `pkg/metrics/tasks.go`、`configs/monitoring/` | 有限标签、15 秒数据库缓存、状态/尝试/重放指标，实际 Kafka exporter 配置、仪表盘和 12 条告警 | P1 |
| 迁移 | schema.sql 与历史手动 SQL；没有生产版本 ledger | 无并发防护、校验、明确基线升级 | `migrations/`、`store/store.go` | 官方 Goose CLI、单一版本历史、只读启动版本检查；同库发布串行执行 | P1 |
| 质量/发布 | 已有格式、vet、race、覆盖率、lint、生成、漏洞和 secret 门禁 | 发布另行构建；未绑定相同测试产物 | `.github/workflows/quality.yml`、`deploy.yml` | 保留原门禁和 20% 阈值，真实依赖测试与镜像冒烟，按 quality run/commit 发布同一 artifact，记录 digest | P1 |

## 设计与里程碑

M1 完成实际代码审查、[ADR](../adr/0001-persistent-tasks.md)、模型和两版迁移；M2 完成独立执行器、报表和权限基础；M3 完成持久化计划/恢复；M4 完成管理、RBAC、审计及受控重放；M5 完成遥测、交付、文档和验证。

选择方案 A：任务状态权威源为 PostgreSQL。Kafka 继续传递业务事实，未增加 Asynq 或通用多后端引擎。复用 OneX 的职责分离、显式注册及生命周期思想，未复制其业务/框架代码；参考固定提交和 MIT 核验见 ADR。Handler 不调用 Kafka，也不启动执行 goroutine。

状态为 pending/running/retry_wait/succeeded/failed/canceled。终态不可覆盖；人工重试产生带 retry_of 的新实例。外部副作用不受数据库 fencing 自动保护，新增 Handler 必须另用业务唯一约束、幂等接口或对账。报表结果默认共享 PostgreSQL bytea（最大 10MiB、7 天到期），适用于模板规模；大量导出需要对象存储及生命周期适配。

## 实际验证

以下为迁移入口精简前的历史验证记录，不代表当前官方 CLI 方案的验证结论；当前迁移方式以[迁移和交付](migrations-and-release.md)为准。

验证日期：2026-10-14。使用隔离的真实 PostgreSQL、Redis、Redpanda（Kafka 兼容）容器。故障测试通过同步、伪时钟或更新测试租约推进状态；Cron 镜像冒烟需要等待真实分钟边界。

```bash
GOCACHE=/private/tmp/web-server-enterprise-race-cache \
GOLANGCI_LINT_CACHE=/private/tmp/enterprise-golangci \
TEST_POSTGRES_DSN='postgres://postgres:integration-postgres@127.0.0.1:55432/enterprise_template_test?sslmode=disable' \
TEST_KAFKA_BROKERS=127.0.0.1:29092 TEST_KAFKA_TOPIC=events TEST_KAFKA_DLQ_TOPIC=events.dlq \
TEST_REDIS_ADDR=127.0.0.1:56380 TEST_REDIS_PASSWORD=integration-redis make verify
govulncheck ./...
```

| 已执行验证 | 实际结果/证据 |
|---|---|
| 完整 `make verify` | exit 0；gofmt/Proto 格式、隔离生成 diff、vet、全部 race 测试、增量 lint、五入口构建通过；总覆盖率 27.5%，阈值仍为 20%。日志 `/private/tmp/enterprise-verify-final.log` |
| PostgreSQL 任务/计划/授权/迁移 | `internal/apiserver/store/task_integration_test.go`、当时的自定义迁移集成测试 已实际通过，包含两调度器/两 Worker、旧 token、取消/撤权、补跑边界、checksum 和并发 baseline |
| Kafka 故障恢复 | `TestIntegrationKafkaTaskEventRecovery` 实际通过；真实 broker、失败发布/业务处理/提交及重启，通知副作用为一次；原 Consumer 手动提交断言保留，测试 topic 改为隔离随机 topic |
| Redis 与管理接口 | 原 Redis 安全状态测试及 `TestIntegrationDeadLetterHTTPPolicy` 实际通过；越权403、限流429/审计、依赖缺失503，授权发生在限流之前 |
| 超时/退出/参数/时区 | `internal/job/*test.go` 已实际通过 race，包括 panic、取消、drain、版本拒绝、春季跳过与秋季两次 DST 时点 |
| 敏感数据 | CSV 白名单/公式转义、任务结果内容断言，以及 SQL 凭据日志回归测试已通过；修复 GORM ParamsFilter，保留 OTel 日志原有效断言 |
| 指标故障 | `TestIntegrationQueueMetricsCacheAndLabels` 真实 DB 通过；DB 关闭后刷新失败，scrape 仍读取缓存且没有 task/owner/worker ID 标签 |
| 漏洞 | govulncheck exit 0，0 个可达漏洞；仍提示 1 个 require 模块中的未调用漏洞，不宣称依赖图完全无漏洞。日志 `/private/tmp/enterprise-vuln-final.log` |
| 运维配置 | YAML/仪表盘 JSON 可解析；Prometheus v3.7.3 的 promtool 实际校验配置及 12 条规则成功；Kafka exporter v1.9.0 实际抓取 broker=1、gin-enterprise-template-reports 的 events/partition0 lag=0，证据 `/private/tmp/enterprise-kafka-exporter-metrics.txt` |

以下 16 项需求均有已通过行为测试或真实镜像闭环证据：事务回滚无任务/事件；相同键返回原任务/冲突参数拒绝；同一计划时点去重；停机有限补跑；执行租约排他；宕机接管；旧执行者拒绝写回；重启后尝试预算持久化；永久错误终止；超时/取消/drain；重复消费业务唯一约束；消费及提交失败不跳过；DLQ 新身份重放且审计；任务/计划/结果越权拒绝；空库/基线升级；结果和 SQL 日志不暴露凭据。真实重启验证不是停掉所有基础设施的全域灾难恢复演练。

最终本地 Linux/amd64 审查镜像 ID 为 `sha256:9ccc2133e131c04b554539b581e22ef59d7ba1e778c2c8b9eebce8ffba44dd63`，五入口构建成功。同一 image ID 的 migrate/--check、API/Worker/Scheduler readiness、202 提交、幂等/409、跨用户403、共享 CSV 下载、真实 Kafka 通知、周期任务触发并成功执行、归档均通过；日志 `/private/tmp/enterprise-image-final.log`、`/private/tmp/enterprise-image-smoke-final.log`，实际指标抓取 `/private/tmp/enterprise-worker-metrics.txt`。本地辅助脚本为 macOS 配置临时网络和专用数据库，CI 使用仓库内 `scripts/image-smoke.sh` 的 Ubuntu host 网络配置。镜像来自尚未提交的工作区，是本地审查产物，未作为已发布 commit/digest 宣称。

未运行 GitHub 远程 Actions、GHCR 发布、真实生产部署、生产级负载及灾难恢复演练。本地检查不能代替这些验证。

## 兼容性、升级和回滚

原用户/认证 API 保留；新增 REST 请求用 snake_case、分页列表返回非 null items，TaskService Proto/OpenAPI 作为 REST 契约维护，未宣称新增 TaskService gRPC 服务实现。配置新增 tasks.*，示例 API 默认关闭 outbox/consumer，Worker 显式启用。示例事件 reader 与 writer 改为 events、使用专用消费组；既有自定义拓扑必须显式保留原配置。Go/OTel/gRPC/x/* 同步补丁升级见 go.mod。

数据库升级显式执行官方 `goose -dir migrations up`，Compose 不隐式导入 schema。模板不提供旧库接管逻辑，存量数据库按实际结构制定升级计划。管理员通过 `scripts/bootstrap-admin.sh <existing-user-id>` 从现有活跃账号初始化，无固定密码。生产数据库账号/TLS、topic ACL、采集后端及 GitHub release environment 审批由环境维护者配置。

使用测试通过的 commit/digest 运行迁移及三个角色，检查 readyz、报表/通知/计划及积压。回滚先停止领取与调度并 drain，再恢复兼容的已测试 digest/config；首次升级完全回滚需要恢复备份，可能丢失备份之后任务/审计，必须对账。详见 [升级交付指南](migrations-and-release.md) 与 [排障手册](background-operations.md)。

## 边界与后续风险

- P0/P1 代码及本地关键路径已实现；远程 CI/secret scan、发布与生产观测闭环尚未执行，原因是本次未授权推送/发布/生产操作。
- 保留串行 Consumer，吞吐扩容依靠分区和实例，不实现容易错提交的进程内并发；严格聚合顺序仍需按业务引入版本校验。
- 数据库结果适配有明确容量/到期上限；高容量报表、长时间非协作外部调用和非事务副作用需要额外业务设计与演练。
- 首次数据库迁移采用维护窗口；版本校验拒绝旧二进制重新启动到较新 schema，后续滚动变更必须按 expand/contract 兼容窗口安排。
- Outbox 终态 dead 保留排障证据，没有提供忽略类型/授权的无限制重发接口；新增事件 Handler 需同步重放参数验证与业务幂等。
- 多租户、SSO、复杂工作流仅作为 P2 扩展建议，不加入服务网格/配置中心/分布式事务/另一队列。

审查说明见 [代码审查记录](enterprise-code-review.md)。最终格式与生成复核也通过，日志 `/private/tmp/enterprise-final-consistency.log`。

## 迁移入口精简（2026-10-16）

模板直接使用固定版本官方 Goose CLI，移除自定义 migrate 入口、旧库接管、checksum 扩展与相关兼容测试。版本历史统一为 `goose_db_version`，应用只读检查版本。镜像携带官方工具与 SQL，构建 API、Worker、Scheduler 三个应用入口；首次管理员授权使用运维脚本 `scripts/bootstrap-admin.sh`，已移除独立 Admin 二进制。历史自定义迁移的测试、体积和镜像记录不作为此方案的验证证据；当前命令与边界见[迁移和交付](migrations-and-release.md)。
