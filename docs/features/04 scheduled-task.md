# 异步任务与定时任务管理系统

## 一、功能概述

为通用后台管理 skeleton 构建完整的**异步任务队列**和**定时任务调度**系统，支持客户端通过 API 创建和管理动态定时任务，提供可靠的任务执行、重试、死信和可观测能力。基于 Gin + GORM + PostgreSQL + Redis + Asynq 技术栈，实现与现有 Handler → Biz → Store → Model 分层和 Wire 依赖注入方式的无缝集成。

## 二、背景

当前系统已有 PostgreSQL、Redis、JWT、RBAC、OpenTelemetry 等基础设施，但缺乏以下能力：

- **异步任务队列**：耗时操作（如数据导出、报表生成）仍在 HTTP 请求链路中同步执行，影响接口响应速度
- **定时任务调度**：无法支持系统内置周期任务和客户端动态定时任务
- **任务可观测**：缺乏任务执行状态的追踪、日志、指标和告警能力

随着业务发展，需要将耗时操作从 HTTP 请求链路中解耦，并通过队列实现削峰、重试和死信处理。

## 三、目标

### 3.1 核心目标

| 目标 | 说明 |
|------|------|
| 可靠性 | 异步任务至少执行一次，调度过程具备幂等和去重能力 |
| 可恢复 | 服务重启后可从数据库恢复客户端动态任务 |
| 可观测 | 接入现有 slog、OpenTelemetry、Prometheus 指标体系 |
| 可扩展 | 支持任务类型注册、payload 校验、重试、死信、优先级队列 |
| 低侵入 | 保持现有 Handler → Biz → Store → Model 分层和 Wire 注入方式 |
| 安全可控 | 客户端任务必须经过权限、任务类型白名单、配额和 payload schema 校验 |

### 3.2 设计原则

#### 3.2.1 PostgreSQL 是客户端动态任务的唯一事实源

客户端创建的定时任务必须持久化在 PostgreSQL 中。内存中的 cron 调度器只作为运行时缓存，不能作为事实源。

| 要求 | 说明 |
|------|------|
| 数据库优先 | 创建、更新、删除、启停任务时，数据库状态优先 |
| 启动加载 | 服务启动时必须从数据库加载所有启用任务 |
| 定期对账 | 运行中需要定期 reconcile 数据库和内存调度器状态 |
| 可恢复 | 调度器注册失败不应让数据库状态不可恢复 |
| 状态重建 | 内存调度器状态丢失后，可以通过数据库重建 |

#### 3.2.2 区分"调度成功"和"任务执行成功"

定时任务触发后，调度器只负责把任务投递到 Asynq。**投递成功不等于业务执行成功**。

| 层级 | 含义 | 典型状态 |
|------|------|----------|
| 调度层 | Cron tick 是否成功创建执行记录并投递 Asynq | pending、enqueued、enqueue_failed、skipped |
| 执行层 | Worker 是否真正处理完成业务任务 | pending、running、succeeded、failed、retrying、dead |

执行历史中必须保存 Asynq task ID 或业务 execution ID，Worker 处理完成后再回写真实执行结果。

#### 3.2.3 所有任务按 at-least-once 设计

Asynq、Redis 分布式锁和进程重启都无法天然保证 exactly-once。业务任务必须接受 at-least-once 语义。

| 要求 | 说明 |
|------|------|
| 稳定执行ID | 每次调度生成稳定的 `executionID` |
| 唯一约束 | 执行记录对 `(scheduledTaskID, scheduledAt)` 建唯一约束 |
| Worker 幂等 | Worker 以 `executionID` 或业务幂等键做幂等处理 |
| 重复跳过 | 调度器重复触发时应复用或跳过已有执行记录 |
| 副作用幂等 | 外部副作用操作必须由业务方保证幂等 |

#### 3.2.4 客户端任务不能直接执行任意内部任务

第三方客户端只能使用平台显式开放的任务类型。每个可开放任务都必须在任务注册中心声明：

| 字段 | 说明 |
|------|------|
| taskType | 对外暴露的任务类型 |
| queue | 默认队列及允许队列范围 |
| permission | 创建、触发该任务所需权限 |
| minInterval | 最小调度间隔，防止高频 cron 滥用 |
| maxPayloadBytes | payload 最大大小 |
| payloadSchema | payload 结构校验规则 |
| timeout | Worker 最大执行时间 |
| retryPolicy | 重试策略 |

## 四、技术方案

### 4.1 技术选型

#### 4.1.1 异步任务队列：Asynq

推荐使用 Asynq 作为异步任务队列，原因如下：

| 能力 | 说明 |
|------|------|
| Redis 依赖 | 项目已有 Redis，可以复用基础设施 |
| 重试机制 | 支持失败重试、超时、死信队列 |
| 优先级 | 支持多队列权重和严格优先级 |
| 延迟任务 | 支持延迟入队和未来时间调度 |
| 可观测 | 可结合日志、指标、asynqmon 做任务观测 |

Asynq 负责异步执行，不负责客户端 cron 定义的持久化管理。

#### 4.1.2 定时任务调度：robfig/cron/v3

推荐使用 `robfig/cron/v3` 作为进程内 cron 解析和触发器。

本方案默认只支持 **标准 5 字段 cron 表达式**：

```text
分钟 小时 日 月 星期
```

| 示例 | 说明 |
|------|------|
| `0 3 * * *` | 每天 03:00 |
| `*/30 * * * *` | 每 30 分钟 |

> **说明**：秒级 cron 暂不作为默认能力。如果未来确实需要秒级调度，必须统一调整 cron parser、API 校验规则、配置示例、最小调度间隔和限流配额策略。

### 4.2 总体架构

```
                         ┌────────────────────┐
                         │  HTTP API / Gin    │
                         └─────────┬──────────┘
                                   │
                                   ▼
                         ┌────────────────────┐
                         │ Biz / Validation   │
                         └──────┬───────┬─────┘
                                │       │
                                │       ▼
                                │  ┌──────────────┐
                                │  │ Job Producer │
                                │  └──────┬───────┘
                                │         │
                                ▼         ▼
                        ┌──────────┐  ┌──────────┐
                        │PostgreSQL│  │  Redis   │
                        │任务定义/历史│  │  Asynq   │
                        └────┬─────┘  └────┬─────┘
                             │             │
                 启动加载/定期对账       │
                             │             ▼
                             │      ┌──────────────┐
                             └─────▶│ Scheduler    │
                                    │ Cron Trigger  │
                                    └──────┬───────┘
                                           │ enqueue
                                           ▼
                                    ┌──────────────┐
                                    │ Asynq Worker │
                                    └──────────────┘
```

**运行角色说明**：

| 角色 | 职责 | 是否可多实例 |
|------|------|--------------|
| apiserver | 提供 REST API，管理任务定义，手动触发任务 | 可以 |
| scheduler | 加载系统任务和客户端动态任务，按 cron 投递 Asynq | 可以，但必须分布式去重 |
| worker | 消费 Asynq 队列并执行业务任务 | 可以 |

> **说明**：可部署在同一二进制内，也可通过配置控制是否启用。

### 4.3 目录结构

建议目录结构如下：

```text
internal/
└── apiserver/
    ├── job/
    │   ├── tasks/                  # apiserver 业务任务定义和注册
    │   └── worker/                 # Asynq Worker 生命周期和处理器
    ├── handler/
    │   └── scheduled_task.go       # 客户端任务管理 HTTP Handler
    ├── biz/
    │   └── v1/
    │       └── scheduled_task/     # 客户端任务 Biz 层
    ├── store/
    │   ├── scheduled_task.go
    │   └── scheduled_task_execution.go
    └── pkg/
        ├── conversion/
        │   └── scheduled_task.go
        └── validation/
            └── scheduled_task.go

pkg/
├── job/
│   ├── producer.go                 # Asynq Producer 封装
│   ├── scheduler.go                # Cron 调度抽象
│   ├── registry.go                 # 任务类型注册中心
│   ├── lock.go                     # 分布式调度去重
│   ├── metrics.go                  # 指标定义和注册
│   ├── tracing.go                  # Trace 上下文传播
│   └── provider.go                 # Wire ProviderSet
└── options/
    └── job_options.go              # Job 配置
```

**设计说明**：

- `pkg/job` 不依赖 `internal/apiserver`，只提供通用队列、调度、注册、锁和可观测能力
- 具体业务任务放在 `internal/apiserver/job/tasks` 或 `worker` 下
- 客户端动态任务的 Handler/Biz/Store/Model 保持项目现有资源分层
- Biz 层定义自己消费的接口，避免直接耦合过多底层实现

## 五、数据库设计

### 5.1 scheduled_task（客户端动态任务定义表）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGSERIAL | 内部自增主键，仅用于数据库和游标分页 |
| scheduled_task_id | VARCHAR(64) | 对外暴露的业务 ID，建议 UUID |
| name | VARCHAR(128) | 任务名称 |
| task_type | VARCHAR(100) | 任务类型，必须存在于任务注册中心 |
| payload | JSONB | JSON payload |
| cron_expr | VARCHAR(100) | 标准 5 字段 cron 表达式 |
| queue | VARCHAR(50) | 目标 Asynq 队列 |
| enabled | SMALLINT | 是否启用: 0=否, 1=是 |
| timezone | VARCHAR(50) | cron 解析时区 |
| user_id | VARCHAR(64) | 创建者，用于租户过滤和权限控制 |
| next_run_time | TIMESTAMP | 下次预计触发时间 |
| last_scheduled_at | TIMESTAMP | 上次调度时间 |
| last_execution_id | VARCHAR(64) | 最近一次执行 ID |
| last_error | TEXT | 最近一次错误摘要 |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |
| deleted_at | TIMESTAMP | 软删除 |

**索引建议**：

| 索引 | 说明 |
|------|------|
| `unique(scheduled_task_id)` | 对外业务 ID 唯一 |
| `index(user_id)` | 用户维度查询 |
| `index(enabled, next_run_time)` | 启动加载和 reconcile 查询 |
| `index(task_type)` | 按任务类型过滤 |

### 5.2 scheduled_task_execution（执行历史表）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGSERIAL | 内部自增主键，用于游标分页 |
| execution_id | VARCHAR(64) | 对外暴露的执行 ID，建议 UUID |
| scheduled_task_id | VARCHAR(64) | 对应任务业务 ID |
| user_id | VARCHAR(64) | 冗余创建者，便于执行历史权限过滤 |
| trigger_type | VARCHAR(20) | cron 或 manual |
| scheduled_at | TIMESTAMP | 本次 cron 理论触发时间 |
| enqueued_at | TIMESTAMP | 成功入队时间 |
| asynq_task_id | VARCHAR(200) | Asynq 返回的任务 ID |
| dispatch_status | VARCHAR(20) | 调度状态：pending、enqueued、enqueue_failed、skipped |
| process_status | VARCHAR(20) | 执行状态：pending、running、succeeded、failed、retrying、dead |
| attempt | INT | Worker 当前尝试次数 |
| error_msg | TEXT | 最近一次错误 |
| started_at | TIMESTAMP | Worker 实际开始执行时间 |
| finished_at | TIMESTAMP | Worker 实际完成时间 |
| duration_ms | BIGINT | Worker 执行耗时（毫秒） |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

**索引建议**：

| 索引 | 说明 |
|------|------|
| `unique(execution_id)` | 执行 ID 唯一 |
| `unique(scheduled_task_id, scheduled_at)` | 防止同一 tick 重复调度 |
| `index(user_id, id)` | 用户维度执行历史分页 |
| `index(scheduled_task_id, id)` | 单任务执行历史分页 |
| `index(dispatch_status)` | 排查调度异常 |
| `index(process_status)` | 排查执行异常 |

## 六、API 设计

### 6.1 API 列表

新增 API 应继续合并到 `pkg/api/apiserver/v1/apiserver.proto` 的 `APIServer` service 中，不单独新增 `ScheduledTaskService`。

建议 RPC：

| RPC | HTTP | 说明 |
|-----|------|------|
| CreateScheduledTask | POST `/v1/scheduled-tasks` | 创建客户端定时任务 |
| UpdateScheduledTask | PUT `/v1/scheduled-tasks/{scheduledTaskID}` | 更新任务定义 |
| DeleteScheduledTask | DELETE `/v1/scheduled-tasks/{scheduledTaskID}` | 删除任务 |
| GetScheduledTask | GET `/v1/scheduled-tasks/{scheduledTaskID}` | 获取任务详情 |
| ListScheduledTasks | GET `/v1/scheduled-tasks` | 查询任务列表 |
| ToggleScheduledTask | PUT `/v1/scheduled-tasks/{scheduledTaskID}/toggle` | 启停任务 |
| TriggerScheduledTask | POST `/v1/scheduled-tasks/{scheduledTaskID}/trigger` | 手动触发一次 |
| ListScheduledTaskExecutions | GET `/v1/scheduled-tasks/{scheduledTaskID}/executions` | 查询执行历史 |

### 6.2 响应约定

| 操作 | 响应建议 |
|------|----------|
| Create | 返回 `scheduledTaskID` |
| Update | 空响应或返回最新任务，需和项目其他资源保持一致 |
| Delete | 空响应 |
| Get | 返回 `ScheduledTask` |
| List | 返回 `totalCount`、`scheduledTasks`、`pageToken` |
| Toggle | 返回最新任务状态 |
| Trigger | 返回 `executionID`，便于客户端追踪 |
| ListExecutions | 返回 `totalCount`、`executions`、`pageToken` |

### 6.3 REST 示例

**创建任务**：

```bash
curl -X POST http://localhost:5555/v1/scheduled-tasks \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "每日数据报表",
    "taskType": "report:generate",
    "payload": {"reportType": "daily", "format": "pdf"},
    "cronExpr": "0 3 * * *",
    "queue": "default",
    "enabled": true,
    "timezone": "Asia/Shanghai"
  }'
```

响应：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "scheduledTaskID": "b2a7d2a4-1f3a-4a35-b4d4-0b7b3d6c8d29"
  }
}
```

**手动触发**：

```bash
curl -X POST http://localhost:5555/v1/scheduled-tasks/${SCHEDULED_TASK_ID}/trigger \
  -H "Authorization: Bearer $TOKEN"
```

响应：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "executionID": "f3d8fbc7-3a4d-4e60-a3d7-c77d8de33d21"
  }
}
```

**查询执行历史**：

```bash
curl "http://localhost:5555/v1/scheduled-tasks/${SCHEDULED_TASK_ID}/executions?page_token=&page_size=20" \
  -H "Authorization: Bearer $TOKEN"
```

> **注意**：响应中应区分 `dispatchStatus` 和 `processStatus`，避免把"入队成功"误判为"业务执行成功"。

## 七、配置设计

配置应使用项目现有 `json` 和 `mapstructure` tag 风格，命名采用 kebab-case。

```yaml
job:
  async:
    enabled: true
    redis:
      reuse-default: true
      addr: "127.0.0.1:6379"
      username: ""
      password: ""
      database: 0
    worker:
      concurrency: 10
      strict-priority: false
      queues:
        critical: 6
        default: 3
        low: 1
    retry:
      max-retry: 3
      timeout: 30s
      deadline: 5m
    dead-letter:
      retention: 168h

  scheduler:
    enabled: true
    timezone: "Asia/Shanghai"
    lock-ttl: 2m
    reconcile-interval: 1m
    min-interval: 1m

  client-task:
    enabled: true
    max-tasks-per-user: 100
    max-payload-bytes: 8192
    allowed-queues:
      - default
      - low
```

**关键约束**：

| 约束 | 说明 |
|------|------|
| Redis 配置 | 不允许在 Worker 或 Producer 中硬编码 Redis 地址、密码、DB |
| 独立配置 | 默认可复用项目 Redis，但生产环境建议为 job 队列单独配置 DB 或独立 Redis |
| 独立开关 | `scheduler.enabled` 和 `async.enabled` 必须可独立控制 |
| 队列权重 | `queues` 建议使用 map 表达权重，避免列表顺序隐含优先级 |
| 配置集成 | 配置结构必须接入 `Config`、`Validate`、`AddFlags`、YAML 示例和 Wire Provider |

## 八、调度器设计

### 8.1 启动加载

Scheduler 启动时必须执行：

| 步骤 | 说明 |
|------|------|
| 1 | 加载配置中的系统任务 |
| 2 | 查询数据库中 `enabled = true` 的客户端任务 |
| 3 | 对每个任务校验 taskType、cron、queue、payload |
| 4 | 注册到内存 cron |
| 5 | 记录启动加载结果和失败原因 |

> **说明**：加载失败的单个客户端任务不应阻止整个服务启动，但必须记录错误并暴露指标。

### 8.2 定期 Reconcile

Scheduler 应定期对账数据库和内存状态：

| 场景 | 处理 |
|------|------|
| DB 新增 enabled 任务 | 注册到内存 cron |
| DB 禁用任务 | 从内存 cron 移除 |
| DB 删除任务 | 从内存 cron 移除 |
| DB cron/payload/queue 变更 | 重新注册任务 |
| 内存存在但 DB 不存在 | 移除内存任务 |

Reconcile 周期由配置控制，例如 `job.scheduler.reconcile-interval`。

### 8.3 分布式去重

多实例 scheduler 下，同一任务同一 tick 需要多层去重：

| 层级 | 机制 | 说明 |
|------|------|------|
| Redis lease | 避免多个实例同时处理同一 tick | lock key 包含 `scheduledTaskID` 和 `scheduledAt` |
| 数据库唯一约束 | 作为最终防线 | `unique(scheduled_task_id, scheduled_at)` |
| Asynq 去重 | 在合理 TTL 内避免重复入队 | - |
| Worker 幂等 | 以 executionID 或业务幂等键避免重复副作用 | - |

**Redis lease 要求**：

| 要求 | 说明 |
|------|------|
| lock key | 包含 `scheduledTaskID` 和 `scheduledAt` |
| lock value | 包含 instanceID 和随机 token |
| 释放方式 | 释放锁必须使用 compare-and-delete，不能直接 `Del` |
| TTL 设置 | 覆盖"创建执行记录 + 入队"耗时即可，不应覆盖 Worker 业务执行时间 |
| 兜底机制 | 入队成功后即使释放锁失败，也必须依赖数据库唯一约束和 Worker 幂等兜底 |

## 九、Worker 与任务注册中心

任务注册中心是 Producer、Scheduler、Worker 的共同契约来源。

每个任务类型至少声明：

| 能力 | 说明 |
|------|------|
| taskType | Asynq task type |
| handler | Worker 处理器 |
| payload validator | payload schema 校验 |
| default queue | 默认队列 |
| allowed queues | 允许队列 |
| timeout | 单次执行超时 |
| retry policy | 重试策略 |
| permission | 客户端创建或触发所需权限 |
| visibility | 是否允许客户端创建动态任务 |

**设计要求**：

| 要求 | 说明 |
|------|------|
| 统一注册 | Worker 不再使用 switch 分发和注册中心两套机制，统一由注册中心注册处理器 |
| 入队校验 | Producer 入队前通过注册中心校验 taskType 和 payload |
| 可见性控制 | 客户端任务只能选择 `visibility = public` 的任务类型 |
| 内部任务 | 内部系统任务可以使用 `visibility = internal` 的任务类型 |
| 状态回写 | Worker 处理开始、成功、失败时回写 execution 状态 |

## 十、权限、安全与配额

### 10.1 权限控制

客户端任务管理接口需要接入 Casbin/RBAC。

| 操作 | 权限 |
|------|------|
| Create | `scheduled_task:create` |
| Update | `scheduled_task:update`，且只能操作自己的任务，admin 除外 |
| Delete | `scheduled_task:delete`，且只能操作自己的任务，admin 除外 |
| Get/List | `scheduled_task:get` / `scheduled_task:list` |
| Toggle | `scheduled_task:update` |
| Trigger | `scheduled_task:trigger` |
| ListExecutions | `scheduled_task:execution:list` |

> **说明**：此外，具体 taskType 可以声明更细粒度权限。例如创建 `report:generate` 任务需要额外的 `report:generate` 权限。

### 10.2 安全限制

必须限制以下风险：

| 风险 | 控制方式 |
|------|----------|
| 任意 taskType 调用 | 任务注册中心白名单 |
| payload 过大 | maxPayloadBytes |
| payload 结构异常 | schema 校验 |
| 高频 cron 滥用 | minInterval |
| 单用户任务过多 | maxTasksPerUser |
| 队列抢占 | allowedQueues |
| 越权查看执行历史 | user_id 过滤或任务归属校验 |
| 重复触发外部副作用 | executionID 幂等 |

## 十一、Validation 设计

Validation 层只做业务校验，不承担 Gin binding。

**创建任务时校验**：

| 字段 | 规则 |
|------|------|
| name | 非空，长度不超过 128 |
| taskType | 必须存在于任务注册中心，且当前用户有权限 |
| payload | 必须是合法 JSON，并通过该 taskType 的 schema 校验 |
| cronExpr | 标准 5 字段 cron，且不能低于最小调度间隔 |
| queue | 必须在该 taskType 和用户允许的队列范围内 |
| timezone | 必须是合法 IANA timezone |
| enabled | 创建时允许 true/false，默认值需明确 |

**更新任务时校验**：

| 规则 | 说明 |
|------|------|
| 不允许修改字段 | `scheduledTaskID`、`userID` 不允许修改 |
| taskType 变更 | 是否允许修改需要明确，默认建议不允许 |
| 时间刷新 | 修改 cron、timezone、enabled 后必须刷新 `nextRunTime` |
| payload 校验 | 修改 payload 时必须重新做 schema 校验 |

**列表过滤时校验**：

| 规则 | 说明 |
|------|------|
| enabled 过滤 | 必须使用 optional bool |
| pageSize | 使用项目统一分页默认值和最大值 |
| 用户过滤 | 非管理员自动加用户过滤 |

## 十二、错误码设计

错误码应使用项目当前 `errorsx.NewCompat` 风格，并避免复用用户、角色等无关错误码。

建议新增：

| 错误 | HTTP | 说明 |
|------|------|------|
| ScheduledTask.NotFound | 404 | 任务不存在 |
| ScheduledTask.AlreadyExists | 409 | 任务已存在 |
| ScheduledTask.InvalidCronExpr | 400 | cron 表达式非法 |
| ScheduledTask.InvalidPayload | 400 | payload 不符合任务 schema |
| ScheduledTask.TaskTypeNotSupported | 400 | 任务类型不支持或未开放 |
| ScheduledTask.QueueNotAllowed | 400 | 队列不允许 |
| ScheduledTask.QuotaExceeded | 429 | 用户任务数量或频率超限 |
| ScheduledTask.PermissionDenied | 403 | 无权操作该任务或 taskType |

## 十三、可观测性设计

### 13.1 日志

日志字段建议统一包含：

| 字段 | 说明 |
|------|------|
| taskType | 任务类型 |
| queue | Asynq 队列 |
| scheduledTaskID | 客户端任务 ID |
| executionID | 执行 ID |
| asynqTaskID | Asynq 任务 ID |
| userID | 创建者或触发者 |
| triggerType | cron 或 manual |
| scheduledAt | 理论触发时间 |
| traceID | 链路追踪 ID |

### 13.2 指标

建议指标：

| 指标 | 标签 | 说明 |
|------|------|------|
| job_tasks_enqueued_total | taskType、queue、result | 入队次数 |
| job_tasks_processed_total | taskType、queue、status | Worker 处理次数 |
| job_task_duration_seconds | taskType、queue | Worker 执行耗时 |
| job_scheduler_ticks_total | taskType、result | Cron tick 次数 |
| job_scheduler_reconcile_total | result | Reconcile 次数 |
| job_scheduler_registered_tasks | source | 当前注册任务数 |
| job_task_dead_total | taskType、queue | 进入死信的任务数 |

> **说明**：指标注册应在 job 模块初始化阶段统一完成，避免重复注册。

### 13.3 Trace

异步链路需要显式传播 trace context。

| 要求 | 说明 |
|------|------|
| API 触发 | 从请求 context 提取 trace 信息并写入任务元数据 |
| Scheduler 触发 | 创建新的 scheduler span |
| Producer 入队 | 创建 enqueue span |
| Worker 处理 | 从任务元数据恢复 trace context |
| span 关联 | Worker span 需要关联 executionID 和 asynqTaskID |

## 十四、实施计划

### 14.1 分阶段落地

本方案按三阶段落地：

#### 阶段一：异步任务队列

**范围**：

| 范围 | 说明 |
|------|------|
| `pkg/job` | 提供 Producer、任务注册中心、日志和指标封装 |
| `internal/apiserver/job/worker` | 提供 Worker 生命周期和任务处理器注册 |
| 任务注册 | 任务类型必须通过注册中心声明 |
| 入队校验 | Producer 入队前校验 task type、queue、payload |
| 可观测 | Worker 处理时记录 trace、日志、指标 |
| 重试死信 | 支持多队列权重、重试、超时、死信 |

**验收标准**：

| 标准 | 说明 |
|------|------|
| 任务投递 | HTTP/Biz 层可以安全投递异步任务 |
| 任务消费 | Worker 可以消费任务并处理成功、失败、重试 |
| 死信机制 | 任务失败可进入 Asynq 的重试或死信机制 |
| 日志关联 | 日志中能关联 taskType、queue、asynqTaskID、traceID |

#### 阶段二：系统内置定时任务

**范围**：

| 范围 | 说明 |
|------|------|
| 配置声明 | 配置文件声明系统任务名称、cron 表达式、目标 taskType、payload、queue |
| 启动注册 | Scheduler 启动时注册系统任务 |
| 异步投递 | Cron tick 只负责投递 Asynq，不在调度器内执行耗时业务逻辑 |
| 执行ID | 每次 tick 生成 executionID，并用于幂等和观测 |
| 分布式去重 | 多实例 scheduler 通过分布式去重避免重复投递 |

**验收标准**：

| 标准 | 说明 |
|------|------|
| 自动注册 | 系统任务可随服务启动自动注册 |
| 单次投递 | 多实例启动时同一 tick 最多成功投递一次 |
| 失败可观测 | 调度失败和入队失败可观测 |
| 结果关联 | Worker 执行结果与调度结果可以关联 |

#### 阶段三：客户端动态定时任务

**范围**：

| 范围 | 说明 |
|------|------|
| 数据模型 | 新增 `scheduled_task` 和 `scheduled_task_execution` 数据模型 |
| 任务 CRUD | 新增任务 CRUD、启停、手动触发、执行历史 API |
| 校验 | 创建和更新任务时校验权限、taskType、payload、cron、配额 |
| 启动加载 | Scheduler 启动时加载数据库中启用的客户端任务 |
| 定期对账 | Scheduler 定期 reconcile 数据库状态和内存注册状态 |
| 状态回写 | Worker 回写执行状态 |

**验收标准**：

| 标准 | 说明 |
|------|------|
| 重启恢复 | 服务重启后已启用的客户端任务可以恢复 |
| 删除暂停 | 删除或暂停任务后不会继续触发 |
| cron 刷新 | 修改 cron 后下一次触发时间正确刷新 |
| 执行历史 | 执行历史能区分调度失败、入队成功、Worker 成功、Worker 失败 |
| 权限过滤 | 非管理员只能管理和查看自己的任务 |

### 14.2 部署与运维建议

| 项目 | 建议 |
|------|------|
| Redis | 生产环境建议 job 队列使用独立 DB 或独立实例 |
| Worker | 可水平扩容，通过 Asynq 和 Redis 协调消费 |
| Scheduler | 可多实例，但必须启用分布式去重和 DB 唯一约束 |
| Cron 频率 | 默认最小 1 分钟，禁止客户端创建秒级或过高频任务 |
| 死信处理 | 配合 asynqmon 或内部管理接口排查 |
| 告警 | 关注入队失败、执行失败率、队列积压、死信数量、reconcile 失败 |
| 数据保留 | 执行历史需要配置保留周期，避免无限增长 |

### 14.3 主要风险与应对

| 风险 | 应对 |
|------|------|
| 多实例重复调度 | Redis lease + DB 唯一约束 + Asynq 去重 + Worker 幂等 |
| 服务重启丢失动态任务 | 启动加载 enabled 任务 + 周期 reconcile |
| 入队成功但业务失败 | 区分 dispatchStatus 和 processStatus，由 Worker 回写执行结果 |
| 客户端滥用高频任务 | minInterval、配额、权限、任务类型白名单 |
| payload 导致 Worker 重试风暴 | 入队前 schema 校验，限制大小，明确 retry 策略 |
| 数据库和调度器状态不一致 | DB 作为事实源，调度器只做运行时缓存 |
| trace 链路断裂 | enqueue 和 worker 显式传播 trace context |

## 十五、验收标准

### 15.1 功能验收

- [ ] 可以创建、编辑、删除客户端定时任务
- [ ] 可以启停和手动触发定时任务
- [ ] 可以查询执行历史（区分调度状态和执行状态）
- [ ] 系统内置任务可随服务启动自动注册
- [ ] 多实例调度时同一 tick 最多成功投递一次
- [ ] 任务失败可进入重试或死信机制
- [ ] Worker 执行结果与调度结果可以关联
- [ ] 非管理员只能管理和查看自己的任务

### 15.2 安全验收

- [ ] 客户端任务必须经过权限校验
- [ ] 只能使用任务类型白名单中的任务
- [ ] payload 必须符合 schema 校验
- [ ] cron 表达式不能低于最小调度间隔
- [ ] 单用户任务数量不能超过配额
- [ ] 队列必须在允许范围内

### 15.3 可观测验收

- [ ] 日志中能关联 taskType、queue、asynqTaskID、traceID
- [ ] 指标可以统计入队次数、处理次数、执行耗时
- [ ] 调度失败和入队失败可观测
- [ ] trace 链路从 API → Scheduler → Producer → Worker 完整贯通

## 十六、参考文档

- 技术方案原文：`@./../异步任务与定时任务技术方案.md`
- 项目 README：`@./../README.md`
- 项目宪法：`@.claude/constitution.md`
- 用户模块业务逻辑：`@./01 user.md`
- RBAC 权限管理系统：`@./02 permision.md`
- 菜单管理系统：`@./03 menu.md`