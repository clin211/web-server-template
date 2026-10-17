# ADR 0001：PostgreSQL 任务执行与独立后台进程

状态：采用。审查基线：目标仓库 `main@2ecca79507539fd8b028edf0d7b1c13abc4bd59d`；OneX `10119504e12c316569337b0dbca04fb192c7d7ea`。实施分支 `feat/persistent-tasks`，收尾时快进至最新 `main@8c2adc768ac7987f67d8e0977edea8b5a8eb19b6`（仅删除 `.claude`，业务代码未变）。

## 选择

采用方案 A：PostgreSQL 作为任务、计划、执行历史和报表结果的权威数据源，独立 Worker 执行、Scheduler 创建实例，Kafka 继续承载业务事实。API 的默认 Outbox/Consumer 关闭；旧配置仍可运行 API 内的 AsyncRuntime，迁移到 Worker 后应关闭 API 相同后台链路。没有引入 Redis 任务队列、多执行后端或通用工作流。

| 维度 | A：PostgreSQL | B：Asynq + Redis |
| --- | --- | --- |
| 开发维护 | 自行审查领取、租约、恢复；规模限定为本模板需求 | 队列能力成熟，需额外 DB→Redis 派发与任务管理投影 |
| 事务一致性 | 业务修改、任务意图、审计共用已有 TX 上下文 | 必须通过 Outbox 可靠派发，跨存储不能原子提交 |
| 恢复 | DB 时钟、短事务、租约、token、持久化 attempts | 依赖 Redis 持久化、队列恢复及权威状态映射 |
| 吞吐 | 行锁和数据库 I/O 有上限；先测量再扩展 | 队列吞吐优势明显，仍受业务存储限制 |
| 部署 | 不增加新基础设施 | 复用 Redis，但认证与任务持久化容量需隔离规划 |
| 运维 | SQL、指标、审计和受控 API；无管理前端 | 现成管理工具较丰富，版本和 Redis Cluster 兼容须核验 |

本项目已依赖 PostgreSQL、已有事务上下文及事件链路，没有证据要求超高吞吐；保证事务意图和恢复可解释性优先。Asynq 官方仓库当前 v0.26.0 的 v0 API 稳定性和 Redis Cluster 说明需要部署者复核，未将依赖声明当作已接入能力：[官方项目](https://github.com/hibiken/asynq)。

## OneX 取舍

参考 [OneX nightwatch](https://github.com/onexstack/onex/tree/10119504e12c316569337b0dbca04fb192c7d7ea/internal/nightwatch) 的独立后台入口、任务注册、Watcher 生命周期和配置组织。任务注册改为显式类型/参数版本映射，依赖通过结构体构造；调度与执行分离，未复制全局插件体系、框架业务模块或全部状态阶段。

OneX 的数据库锁只能协调实例，不能替代任务唯一约束和 fencing。审查发现其 JobManager.UpdateJob 路径存在重复互斥锁风险；没有移植锁、取消或恢复实现。OneX 为 MIT（[固定提交许可证](https://github.com/onexstack/onex/blob/10119504e12c316569337b0dbca04fb192c7d7ea/LICENSE)），本次未复制其源代码，不引入派生许可证声明。

## 后果与边界

至少一次执行，数据库提交有 fencing；外部副作用仍需要业务幂等键、唯一约束或对账。默认仅示例报表，数据库 bytea 共享存储上限 10 MiB、TTL 七天，适用于多实例模板；大报表扩展对象存储时以 task_id 为幂等对象键，并在提交前重新验证权限。

新 SQL 只在显式迁移入口执行，应用启动校验版本/校验和。基线支持当前六表完整 schema；无法识别的旧部署拒绝自动盖章。初次升级需要暂停旧版后台以建立版本账本；以后以新增迁移、兼容读写和明确发布顺序推进。
