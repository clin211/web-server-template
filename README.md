# gin-enterprise-template

`gin-enterprise-template` 是一个基于 Go 1.26.9、Gin、GORM、JWT 与 OpenTelemetry 的模块化单体后端模板。API、Worker、Scheduler 在同一仓库独立运行，共享业务层；提供用户认证、基础 RBAC、持久化任务与周期报表、可靠事件和显式数据库迁移。

## 功能特性

- Gin REST API
- PostgreSQL + GORM 数据访问
- JWT Access Token / Refresh Token 认证、token version 撤销和 Redis 单次刷新
- Redis 跨实例登录限流与登录安全审计
- Transactional Outbox、Kafka、Inbox 和持久化 DLQ
- 用户管理：创建、查询、更新、删除、列表、修改密码
- PostgreSQL 任务租约、心跳、fencing、重试、取消和执行历史
- 持久化 Cron 调度、有限补跑、多实例协调与计划管理
- 基础 RBAC、资源所有权授权和事务管理审计
- 异步报表及事件通知、受控持久死信重放
- Goose SQL 迁移与校验值验证、同一不可变镜像质量验证和发布
- OpenTelemetry + Prometheus `/metrics`
- Protobuf / OpenAPI / Wire / Makefile 工程化支持

## 项目架构

项目采用模块化单体架构，API、Worker 和 Scheduler 独立部署，复用同一仓库中的业务与存储代码。总览图按应用进程、共享内核和数据服务分为三层；共享内核表示代码复用，各进程分别创建自己的依赖实例和连接池。

```mermaid
%%{init: {"theme": "base", "themeVariables": {"fontFamily": "system-ui, sans-serif", "fontSize": "15px", "lineColor": "#8193A6", "primaryTextColor": "#243B53", "clusterBkg": "#F8FAFC", "clusterBorder": "#DCE4ED", "edgeLabelBackground": "#FFFFFF"}, "flowchart": {"curve": "basis", "nodeSpacing": 40, "rankSpacing": 48, "padding": 20, "wrappingWidth": 320}}}%%
flowchart TB
    Client(["客户端 · HTTP REST"])

    subgraph Processes["01　应用进程"]
        direction LR
        API("<b>API Server</b><br/>:5558 · 请求处理与认证")
        Worker("<b>Worker</b><br/>:5560 · 任务执行与消息处理")
        Scheduler("<b>Scheduler</b><br/>:5561 · 周期调度与清理")
    end

    subgraph Modules["02　共享内核"]
        direction TB
        Core("<b>业务与后台能力</b><br/>用户 · RBAC · 任务 · 计划 · 报表<br/>Job Registry · Outbox · Inbox · DLQ")
        Adapters("<b>基础设施适配</b><br/>Store · GORM · SQL · Model<br/>Redis Token Store · Kafka Client")
        Core --> Adapters
    end

    subgraph Infrastructure["03　数据服务"]
        direction LR
        PG[("<b>PostgreSQL</b><br/>业务 · 任务 · 审计 · 事件")]
        Redis[("<b>Redis</b><br/>令牌状态 · 跨实例限流")]
        Kafka["<b>Kafka / Redpanda</b><br/>业务事件 · 死信 topic"]
    end

    Client --> API
    API --> Core
    Worker --> Core
    Scheduler --> Core
    Adapters --> PG
    Adapters --> Redis
    Adapters <-->|按需启用| Kafka

    classDef client fill:#284B63,stroke:#284B63,color:#FFFFFF,stroke-width:1.5px;
    classDef process fill:#EDF4FF,stroke:#9DB6D5,color:#253F60,stroke-width:1.5px;
    classDef module fill:#EAF5F0,stroke:#91BAA9,color:#234C40,stroke-width:1.5px;
    classDef data fill:#FFF6E7,stroke:#D6BB87,color:#65502A,stroke-width:1.5px;

    class Client client;
    class API,Worker,Scheduler process;
    class Core,Adapters module;
    class PG,Redis,Kafka data;
    style Processes fill:#F6F9FE,stroke:#D4DFED,stroke-width:1px,color:#526980;
    style Modules fill:#F5FAF7,stroke:#D3E4DB,stroke-width:1px,color:#526F61;
    style Infrastructure fill:#FFFBF4,stroke:#E9DDC6,stroke-width:1px,color:#806D49;
```

API 通过 Google Wire 装配依赖，Worker/Scheduler 由 `internal/job/runtime` 显式构造和管理生命周期。三个常驻进程启动时只读校验 Goose 版本，数据库迁移由发布步骤单独执行。

| 配套能力 | 入口 | 职责 |
| --- | --- | --- |
| 管理员初始化 | `scripts/bootstrap-admin.sh` | 为已有活跃账号首次授予管理员角色，事务记录审计 |
| 数据库迁移 | 官方 Goose CLI | 显式执行 `migrations/*.sql`，维护数据库版本历史 |
| 指标监控 | Prometheus | 抓取 API、Worker、Scheduler 的 `/metrics` |
| 遥测导出 | OTel Collector | 按输出模式接收 Trace、Metric、Log |

任务队列以 PostgreSQL 为权威数据源；Kafka 承载业务事件。图中消息链路按推荐部署放在 Worker 中，需显式开启 `outbox.enabled` 和 `consumer.enabled`；API 默认关闭这两项，也保留按配置运行消息链路的能力。Prometheus 指标通过 `classic/hybrid` 模式暴露，OTLP 导出由 `otel.output-mode` 决定。

### 报表任务与事件链路

下图展示启用 Outbox 和 Consumer 后的处理流程。任务创建来源包括 API 提交和 Scheduler 调度；任务执行与事件消费均采用至少一次语义。

```mermaid
%%{init: {"theme": "base", "themeVariables": {"fontFamily": "system-ui, sans-serif", "fontSize": "14px", "actorBkg": "#EDF4FF", "actorBorder": "#9DB6D5", "actorTextColor": "#253F60", "actorLineColor": "#CCD6E0", "signalColor": "#60788E", "signalTextColor": "#334E68", "noteBkgColor": "#FFF6E7", "noteBorderColor": "#D6BB87", "noteTextColor": "#65502A", "labelBoxBkgColor": "#F4F7FB", "labelBoxBorderColor": "#D4DFED", "labelTextColor": "#526980"}, "sequence": {"mirrorActors": false, "actorMargin": 24, "messageMargin": 24, "boxMargin": 12, "noteMargin": 12}}}%%
sequenceDiagram
    autonumber
    participant Entry as API / Scheduler
    participant PG as PostgreSQL
    participant Worker as Worker
    participant Relay as Relay
    participant Kafka as Kafka
    participant Consumer as Consumer

    rect rgb(237, 244, 255)
        Note over Entry,PG: 01 · 创建任务
        alt API 提交
            Entry->>PG: 幂等写入 Task 与审计
            PG-->>Entry: 提交成功，返回 task_id
        else 周期计划到期
            Entry->>PG: 锁定计划，创建 Task 并推进游标
        end
    end

    rect rgb(234, 245, 240)
        Note over PG,Worker: 02 · 执行与结果提交
        Worker->>PG: 领取任务，建立租约与 fencing token
        PG-->>Worker: 参数快照与执行上下文
        Worker->>PG: 查询用户资料与任务统计
        Worker->>Worker: 生成报表（事务外）
        Note over PG,Worker: 执行期间短事务续租，提交前校验租约与权限
        Worker->>PG: 同一事务提交结果、任务终态与 Outbox
    end

    rect rgb(255, 246, 231)
        Note over PG,Consumer: 03 · 可靠事件与通知
        Relay->>PG: 领取 Outbox 事件
        PG-->>Relay: report.generated.v1
        Relay->>Kafka: 同步发布
        Kafka-->>Relay: Broker 确认
        Relay->>PG: 标记事件已发布
        Kafka-->>Consumer: 获取事件
        Consumer->>PG: Inbox 幂等协调，事务写入通知与完成状态
        Consumer->>Kafka: 提交消费 offset
    end
```

客户端通过 API 实时授权后查询任务、下载结果和获取通知。任务完成提交通过租约和 fencing token 拒绝过期执行者写回；业务数据与 Outbox 共用事务，消费侧使用 Inbox 和业务唯一约束处理重复投递。消费重试耗尽或遇到永久错误时，先持久化 DLQ，再由独立 Relay 发布到死信 topic，管理员可通过受控 API 重放。

## 目录结构

```text
gin-enterprise-template/
├── .github/workflows/                      # 质量门禁与已测试镜像发布
├── api/openapi/                            # 生成的 OpenAPI 文档
├── cmd/
│   ├── gin-enterprise-template-apiserver/  # API 服务入口与公共命令配置
│   ├── gin-enterprise-template-worker/     # 任务执行与可选消息链路入口
│   ├── gin-enterprise-template-scheduler/  # 周期调度与清理入口
│   └── gen-gorm-model/                     # GORM 模型生成工具
├── configs/
│   ├── configs.yaml                        # 本地运行配置
│   ├── configs.container.yaml              # 容器运行配置基线
│   ├── schema.sql                          # 数据库结构参考快照
│   ├── monitoring/                         # Prometheus 抓取、告警与 Grafana 面板
│   ├── otel-collector.yaml                 # 开发遥测采集配置
│   └── otel-collector.production.yaml      # 生产遥测采集配置
├── docs/
│   ├── adr/                                # 架构决策记录
│   └── guide/                              # 开发、迁移、交付与运维指南
├── internal/
│   ├── apiserver/
│   │   ├── handler/                        # HTTP 请求处理与路由
│   │   ├── biz/                            # 用户、任务、计划、报表与管理用例
│   │   ├── store/                          # 数据访问、事务、租约与授权
│   │   ├── model/                          # 生成模型及手写任务、计划模型
│   │   ├── pkg/                            # 转换、校验与业务指标
│   │   ├── wire.go                         # Wire 依赖注入声明
│   │   └── wire_gen.go                     # 生成的依赖装配代码
│   ├── job/                                # Handler 注册、执行、重试与 Cron 解析
│   │   └── runtime/                        # Worker/Scheduler 装配、健康检查与生命周期
│   ├── messaging/
│   │   ├── outbox/                         # 事务事件、Kafka 发布与 Relay
│   │   └── consumer/                       # Kafka 消费、Inbox 幂等与持久化 DLQ
│   └── pkg/                                # 内部公共能力：上下文、错误、中间件等
│       └── dbschema/                       # 只读 Goose 版本检查与生成的版本列表
├── migrations/                             # 官方 Goose 执行的版本化 SQL
├── pkg/                                    # 可复用公共包：认证、配置、日志、服务等
│   └── api/apiserver/v1/                   # Proto 定义及生成代码
├── scripts/                                # 初始化、生成一致性与冒烟等辅助脚本
│   └── bootstrap-admin.sh                  # 通过 psql 首次授予管理员角色
├── third_party/                            # 第三方 Proto 依赖
├── docker-compose.env.yml                  # PostgreSQL、Redis、Redpanda 等本地依赖
├── docker-compose.app.yml                  # API、Worker、Scheduler 与运维配套部署
├── Dockerfile                              # 包含三个应用入口及 Goose 的镜像
├── Makefile                                # 生成、构建、测试与质量检查命令
├── AGENTS.md                               # 项目代理协作与开发规范
└── README.md                               # 项目概览与快速开始
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

数据库统一使用官方 Goose CLI 执行 `migrations/` 中的 SQL，版本历史由 `goose_db_version` 管理。`make deps` 安装固定版本 Goose 到 `_output/tools/`。`configs/schema.sql` 是参考快照，不导入运行数据库。见[迁移和交付](docs/guide/migrations-and-release.md)。

### 3. 启动本地依赖服务（PostgreSQL、Redis、Redpanda 和 OTel Collector）

```bash
docker compose -f docker-compose.env.yml up -d
export PATH="$PWD/_output/tools:$PATH"
export GOOSE_DRIVER=postgres
export GOOSE_DBSTRING='postgres://postgres:dev-only-postgres@127.0.0.1:54321/enterprise_template?sslmode=disable'
goose -dir migrations up
```

### 4. 构建并运行服务

```bash
make build

_output/platforms/$(go env GOOS)/$(go env GOARCH)/gin-enterprise-template-apiserver \
  --config configs/configs.yaml
```

### 5. 首次管理员初始化

需要管理用户或 RBAC 时，使用 [scripts/bootstrap-admin.sh](scripts/bootstrap-admin.sh) 为已有账号首次授予管理员角色。此操作使用 Shell 和 PostgreSQL 客户端，不需要单独的 Admin 应用或二进制。

执行前完成以下准备：

1. 已对目标数据库执行 `goose -dir migrations up`。
2. 已启动 API，通过 `POST /v1/users` 注册正常账号，并取得返回的 `userID`；该账号必须处于活跃状态。
3. 执行机器已安装 Bash 和 `psql`，且 `psql` 在 `PATH` 中。使用包含 `scripts/` 和 `migrations/` 的仓库副本，在仓库根目录执行以下命令。

```bash
# 本地开发容器的连接配置；生产环境替换为实际 PG* 配置。
export PGHOST=127.0.0.1
export PGPORT=54321
export PGDATABASE=enterprise_template
export PGUSER=postgres
export PGPASSWORD=dev-only-postgres

# 将 existing-user-id 替换为注册响应中的实际 userID，不是用户名。
./scripts/bootstrap-admin.sh 'existing-user-id'

# 查看脚本帮助
./scripts/bootstrap-admin.sh --help
```

脚本通过标准 `PG*` 环境变量或 `.pgpass/PGPASSFILE` 连接数据库，不读取 `configs.yaml`，也不会沿用 `APP_POSTGRESQL_*` 或 `GOOSE_DBSTRING`。生产环境可使用 `.pgpass/PGPASSFILE` 提供密码；执行数据库账号须有读取迁移历史和账号、授予角色、写入审计及创建临时表的权限。

脚本校验迁移版本和账号状态，并在同一事务内完成并发加锁、角色授予及审计。成功时输出“首次管理员授权完成，已记录审计。”；任何检查或审计写入失败都会回滚。数据库中已有管理员时拒绝再次初始化，后续角色调整通过受保护的 RBAC API 完成。脚本不创建默认账号或密码，应用镜像仅包含 API、Worker、Scheduler 和 Goose，初始化在运维机器执行。

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

除公开注册外，普通用户资源只允许访问当前账号；用户列表及跨用户管理要求显式 `user.manage`。所有响应均使用 `{code,message,data,reason}` 包络；当前用户资料通过受保护的 `/v1/auth/info` 获取。

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

## 后台能力与运行手册

Worker：`make dev.worker`；事件闭环使用 `--outbox.enabled --consumer.enabled`。Scheduler：`make dev.scheduler`。二者默认分别监听5560/5561，健康、就绪和指标均独立。

首次管理员授权使用 `scripts/bootstrap-admin.sh`，前置条件、连接配置和执行步骤见[首次管理员初始化](#5-首次管理员初始化)。

- [架构 ADR 与 OneX 取舍](docs/adr/0001-persistent-tasks.md)
- [任务状态、API、Cron 和 Handler 开发](docs/guide/persistent-tasks.md)
- [迁移、同一镜像交付、升级和回滚](docs/guide/migrations-and-release.md)
- [指标、告警、死信和故障恢复](docs/guide/background-operations.md)
- [现状矩阵与实施验证](docs/guide/enterprise-delivery.md)

生产部署示例 `docker-compose.app.yml` 使用明确镜像 digest；不要提交真实凭据。`make verify`、`make integration.test` 和 `scripts/image-smoke.sh <image-id>` 分别验证本地质量、真实依赖和镜像闭环。P2 多租户、SSO 和复杂工作流仅记录建议。
