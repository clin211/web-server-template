# 后台排障和恢复

## 健康与监控

liveness `/healthz` 只表示进程可响应；readiness `/readyz` API 检查 PostgreSQL、Redis 及启用的消息依赖，Worker 检查 PostgreSQL 和启用的 Kafka/异步运行时，Scheduler 只检查 PostgreSQL。积压不是 liveness/readiness 条件，单独报警。遥测后端不参与 readiness；导出有界队列、超时和有界退出，故障可能丢失遥测，不回滚业务事务。批处理日志 exporter 超时五秒。

OTel Prometheus endpoint 由 `--otel.output-mode=classic` 或 `hybrid` 启用（hybrid 同时导出 Trace），参见 `configs/configs.container.yaml`。`configs/monitoring/prometheus.yaml`、alerts.yaml、dashboard.json 提供抓取、规则及 Grafana 导入。数据库 gauge 缓存每十五秒刷新，有两秒预算；抓取不执行 SQL，刷新失败保留上次快照并通过 queue_metrics_last_success_timestamp_seconds 报警。多个实例采集同一数据库总量，请使用 max，不 sum。

可查询：task_queue_count/oldest_wait、task_running_count、task_state_count{status}、task_execution_total{result}、task_execution_duration_seconds、task_lease_lost_total、task_lease_takeovers、task_attempt_outcomes、schedule_due_count/oldest_due、schedule_trigger_delay_seconds、schedule_trigger_outcomes、outbox_pending/oldest/dead、dlq_pending/dead/replay、dlq_replay_results{result}、runtime_available_count{role}。execution_total 是本地 Handler 执行计数，取消提交和权限撤销最终结果以 task_state_count/TaskAttempt 为准。历史 gauge 随九十天清理降低，不能当单调累计 counter。schedule delay 是过去一小时最大触发延迟。

Consumer lag：配置 danielqsj/kafka-exporter 在相同 broker 网络，Prometheus 使用 kafka_consumergroup_lag；exporter 不可用单独告警。应用 Consumer 保持串行 Fetch→处理→持久化DLQ或成功→同步Commit；每组增加实例按分区扩容，不在分区内并行处理并跨越未成功 offset。现有 local lag 不能替代权威 broker/group exporter。

task_id/event_id/request_id/trace_id 仅进入日志或 Trace。Task 保存 W3C traceparent，Worker 建 task.execute span；报表 Outbox 携带 trace_id。计划任务没有上游 HTTP，使用新的执行 Trace。不要把 ID、owner、worker 或任意事件字符串变成指标标签。

## 故障处理

| 故障 | 行为和诊断 |
| --- | --- |
| DB 不可达 | API失败关闭；Worker不领取/不续租/不确认结果，恢复后过期租约可接管；Scheduler游标未提交不会推进 |
| Kafka 发布失败 | Outbox 保留意图，按退避重试；达到最大预算 dead 保留，告警后修复配置再审计恢复 |
| 消费业务失败 | Inbox 回滚业务副作用并持久化 attempts，永久/耗尽进入持久DLQ；offset只在可靠记录后提交 |
| offset commit 失败 | Consumer 尝试有界重试后退出，readiness失败；重启重投由 Inbox/business unique 去重 |
| Worker 宕机 | lease到期才能接管，尝试编号和token递增；查询 attempt 的 lease_expired 和后续结果 |
| 超时/取消 | 错误分类 timeout/interrupted/permanent/canceled；协作取消，无自动撤销外部副作用 |
| Scheduler 停机 | 恢复按 skip/catch_up 窗口数量上限，查询 trigger 的 skip原因及配置version |
| Redis 不可达 | API认证/登录或重放限流失败关闭；Worker/Scheduler任务本身不依赖Redis |
| OTel 不可达 | 有界导出失败，业务继续；查看 stderr/本地日志，恢复遥测后追踪缺口不能自动补齐 |

查任务：受控 GET task/attempts，关联 Trace 和 request_id；错误摘要只提供稳定代码，不暴露凭据或任意数据库/HTTP body。确认可重试原因后调用 POST retry +新的幂等键；自动重试使用同实例新 attempt，人工重试新实例保留原历史。取消 running 等待心跳及 Handler退出，必要时停止失联进程等待租约，不能手改token让旧执行者提交。

查死信：GET dead-letters 只含诊断元数据；先修复实际 Handler/数据或依赖问题。POST replay 使用新的幂等键，当前只支持 report.generated.v1、九十天内、合法参数。会创建新 event_id，保留旧 dead_letter 和 terminal Inbox，以原业务 task_id/result_id 控制重复通知。event_replay + audit_log 记录操作者和意图；用新 event_id 查询 Inbox completed/dead_lettered 来判断实际结果。重放成功受理 202 不等于消费成功。批次最多20，返回每条结果，每请求计入Redis 10/min；不要自动循环无限重放。

Outbox dead 没有通用“无条件重新发布”公开接口。受控恢复应在维护窗口复核事件类型、聚合业务状态、授权和副作用幂等，在审批脚本事务中写审计与新事件，原 dead 记录保留；模板默认不重置原记录。

## 顺序、保留和扩展

相同 Kafka key 只用于分区，不承诺数据库意图的全链路顺序：Outbox SKIP LOCKED、多 Relay、发布失败重试仍可反序。报表完成通知不依赖顺序。未来严格顺序业务应在聚合事务中生成单调 version，Consumer以业务版本约束拒绝旧版本、延期缺口并对账；必须同时审查领取、发布、重试和分区提交路径，不能仅增加 key。

Scheduler 每小时分批清理，每表最多500：结果七天；Task/Attempt、trigger、通知、已发布事件、terminal Inbox、已发布或耗尽DLQ及replay九十天；audit一年；旧heartbeat一天。未完成任务、未发布Outbox、活动租约不清理；DLQ重放关联和人工重试父任务保留外键依赖。选择 Task九十天是为了覆盖死信重放验证窗口。高负载清理可能追不上增长，需要容量告警、调整周期和归档。业务法规要求更长审计时先配置归档，不直接缩短保留期。

P2 仅建议：租户/SSO、外部对象存储、资源授权委托、聚合顺序/version、审计归档、按分区并行Consumer；不引入网格、配置中心、跨存储事务或工作流引擎。
