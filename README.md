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
2. 已有受控预置或升级保留的活跃账号，并取得其 `userID`。创建接口现已要求 `user.manage`；空库需先由受控离线流程预置账号（密码存储必须使用 bcrypt 哈希），再执行首次管理员授权，不能通过匿名注册引导。
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

创建用户要求登录及 `user.manage`，不再开放匿名注册。普通用户资源允许本人访问；用户列表及跨用户管理要求显式 `user.manage`，修改密码仅允许本人。所有响应均使用 `{code,message,data,reason}` 包络；当前用户资料及实时有效权限通过受保护的 `/v1/auth/info` 获取，响应 `data` 包含 `user` 和去重后的 `permissions`。用户列表总数不随分页游标变化，只有确实存在后续记录时才返回下一页游标。

角色管理沿用 `rbac.manage`，与 `user.manage` 独立：`GET /v1/roles/{role_id}` 返回角色及已绑定权限；`GET /v1/users/{userID}/roles` 返回角色列表和去重的有效权限。角色与权限目录完整、稳定排序返回。内置角色权限不可修改，撤销角色时保留最后一名活跃管理员；授权写入经过事务、实时鉴权与审计。无需新增迁移。开发启动使用 `make run.apiserver`（或 `make run`），会先构建当前源码再运行；更新代码后需结束旧进程并重新启动，避免新读取接口在旧实例上返回路由 404。前端接入和验证边界见 [角色权限模块](../lin-console/docs/09-rbac-module.md)。


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

## HTTP 中英文提示

HTTP 默认使用英文，通过 `Accept-Language` 选择语言。支持英文 `en`（含 `en-US`、`en-GB`）及简体中文 `zh-Hans`（含 `zh`、`zh-CN`；其他中文区域标签也统一使用简体中文），按 `q` 权重选择第一个受支持的语言，忽略 `q=0`。未提供、无效或没有受支持语言时使用英文。

```bash
# 默认英文
curl -i http://localhost:5558/healthz

# 简体中文，成功提示为“成功”
curl -i -H 'Accept-Language: zh-CN' http://localhost:5558/healthz

# 按权重选择简体中文
curl -i -H 'Accept-Language: en;q=0.5,zh-Hans;q=0.9' http://localhost:5558/missing
```

响应通过 `Content-Language: en` 或 `zh-Hans` 标识所选语言，并追加 `Vary: Accept-Language`，保留跨域请求的 `Vary: Origin`。认证、限流、校验、任务和 404 提示均使用同一请求语言；HTTP 状态码、业务码、`reason` 和业务数据不随语言改变。英文成功提示仍为 `success`。

项目资源位于 `internal/pkg/localization/locales/en.json` 和 `zh-Hans.json`，通过 `go:embed` 打包，部署无需挂载语言文件。增加提示时，在两份资源中添加相同消息键；业务错误默认以 `Reason` 为消息键，需要保留现有 `reason` 并细化提示时使用 `WithMessageID("Validation.EmailInvalid")`。显式 `WithMessage` 保留原文，后续调用 `WithMessageID` 可重新启用翻译。翻译顺序为所选语言、英文、调用方回退文本、消息 ID。

`pkg/i18n` 仅提供通用翻译机制和上下文传递，`New()` 不内置项目文案，其他应用可以通过 `WithFS` 或 `WithFile` 注入自己的资源。`internal/pkg/localization` 负责项目词表、默认英文和语言协商策略，通过 `localization.New()` 构造项目翻译器，由 API Server 注入 HTTP 中间件；公共包不依赖内部资源包。

语言资源应在启动时加载完成，HTTP 运行期间共享只读 Bundle，每个请求独立选择 Localizer；不要在请求处理期间调用 `Add` 或 `AddFS` 修改资源。`localization.DefaultMessage` 为内部错误定义提供兼容的中文默认文案；`pkg/errorsx` 保留自身回退文本，HTTP 在输出时通过请求翻译器翻译。未注入翻译器时，公共包使用调用方或自身的回退文本。`pkg/app/defaults.json` 和 `pkg/errorsx/defaults.json` 只保存各组件自身的兼容回退文本，不包含用户或任务等业务文案。CLI 默认帮助文案由 `pkg/app` 自身维护，不加载项目词表；CLI 语言切换、日志、审计记录和任务载荷不属于本次国际化范围。

### 动态消息参数

语言资源使用 go-i18n 原生 Go 模板语法，参数名采用 Go 的 MixedCaps 风格。例如：

```json
{
  "Validation.NicknameTooLong": "昵称长度必须小于 {{.MaxLength}} 个字符。"
}
```

业务错误携带消息键和参数，在 HTTP 响应时才按照请求语言渲染。校验条件和提示参数必须引用同一业务阈值，不能在翻译中再次写死数字：

```go
const nicknameMaxLength = 30

if utf8.RuneCountInString(nickname) >= nicknameMaxLength {
    return errno.ErrInvalidArgument.
        WithMessageID("Validation.NicknameTooLong").
        WithMessageParams(i18n.Params{"MaxLength": nicknameMaxLength})
}
```

直接翻译可以调用 `translator.T("Validation.NicknameTooLong", i18n.Params{"MaxLength": 45})`；`T`、`E`、`LocalizeT`、`LocalizeE` 的原有无参数调用保持兼容，每次最多提供一份参数表。需要处理渲染错误的调用方使用 `translator.Localize(message, params)`，该方法返回 `(string, error)`；缺失翻译的正常语言回退视为成功，缺少模板参数、模板损坏或参数类型不受支持会返回错误。

`Params` 是平面的参数表，只支持内置字符串、布尔和数值类型；不接受指针、嵌套容器或函数。`WithMessageParams` 替换并复制整份参数表，不合并旧参数，传入 `nil` 清空参数。必须先调用 `WithMessageID`，再调用 `WithMessageParams`：更换消息键会清空旧参数，`WithMessage` 则清空参数并保持显式文案优先。其他错误派生方法同样复制参数，避免请求之间相互影响。

昵称和新密码长度以 Unicode 码点（rune）计数，汉字和单码点 emoji 各计一个；组合字符与多码点 emoji 不按视觉字形合并。创建用户和修改密码时，新密码遵循当前长度及复杂度规则；登录密码与旧密码仅检查非空，由密码哈希比较验证，避免策略调整阻断历史凭据。

所有渲染都经过 go-i18n，使用严格的 `missingkey=error`，并对原生 `index` 补充缺失键检查，不进行二次字符串替换。参数值中的模板符号按普通文本保留；语言资源和显式 `goi18n.Message.Other` 属于受信任的模板，`BizError.Message` 则始终作为普通回退文本保留，请求输入只能作为参数值或普通文本，不能作为模板。参数不进入响应元数据，诊断日志不记录参数值。

`LocalizeT` 等便捷方法遇到渲染错误时记录诊断，尝试默认语言、调用方回退模板，最后返回消息 ID，不输出半成品模板或 `<no value>`。业务错误先尝试指定消息和默认语言；无法渲染时，使用当前请求语言的通用 `reason` 提示，再回退到普通文案，保持状态码、业务码和 `reason` 不变。业务错误的渲染诊断携带请求上下文，供日志关联 request ID 和 trace ID。`localization.DefaultMessage` 也走相同渲染机制，不再直接返回资源中的模板原文。

新增或修改模板时运行相关测试：中英文词表不仅要求消息键相同，还要求占位符集合一致；参数可以在不同语言中按语法需要调整顺序。项目词表使用静态参数名（如 `.MaxLength` 或 `index . "MaxLength"`），以便测试核对两种语言的参数集合；动态索引仍可在通用翻译 API 中使用。
