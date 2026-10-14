# Web Console User 业务实施计划

> 状态：User 领域前端实施基线
> 适用范围：认证、注册与账号自助功能
> 文档版本：v1.0
> 最后更新：2026-08-29
> 后端基线：`2ecca79`

## 1. 文档职责与依赖

本文只定义 User 领域的产品范围、页面、接口映射、字段契约、缓存语义、垂直切片和验收证据。工程基建遵循[前端平台架构](./web-console-frontend-architecture.md)，Go API、BFF、Cookie 和 CSRF 遵循[后端接入规范](./web-console-backend-integration.md)。

实施依赖顺序：

```text
前端平台 F0-F4 完成
  + Go API 接入层准入完成
    -> User 业务 feature
```

User feature 不修改平台依赖方向、HTTP transport 或 session port。如果业务要求与平台契约冲突，先记录 ADR 或调整接入层，不能在页面组件中建立旁路。

## 2. 后端能力与业务边界

### 2.1 已落地 API

| 能力 | 方法 | 路径 | 鉴权 | 当前语义 |
| --- | --- | --- | --- | --- |
| 注册 | `POST` | `/v1/users` | 公开 | 创建新账号 |
| 登录 | `POST` | `/v1/auth/login` | 公开 | 返回 access/refresh token 和 RFC3339 `expireAt` |
| 刷新令牌 | `PUT` | `/v1/auth/refresh-token` | refresh token | Redis 单次消费并轮换 token |
| 当前用户 | `GET` | `/v1/auth/info` | access token | 返回当前账号脱敏资料 |
| 退出登录 | `POST` | `/v1/auth/logout` | access token | 递增 `token_version`，撤销当前账号全部旧 token |
| 用户详情 | `GET` | `/v1/users/{userID}` | access token | 只能读取本人 |
| 更新用户 | `PUT` | `/v1/users/{userID}` | access token | 只能更新本人，只更新显式提交字段 |
| 修改密码 | `PUT` | `/v1/users/{userID}/change-password` | access token | 只能修改本人，成功后旧 token 全部失效 |
| 删除用户 | `DELETE` | `/v1/users/{userID}` | access token | 只能删除本人 |
| 用户列表 | `GET` | `/v1/users?page_token=&page_size=` | access token | 路由和业务已实现，但 validation 当前固定拒绝 |

认证中间件接受 `Bearer <access-token>`。登录与刷新只返回 token，当前用户资料通过 `/v1/auth/info` 获取。

### 2.2 当前产品范围

首期交付：

- 登录、会话刷新和退出；
- 公开注册；
- 当前账号资料查看和修改；
- 修改密码；
- 注销本人账号。

首期不交付：

- 全量用户列表和跨用户操作；
- 管理员创建用户；
- 角色、权限、菜单、任务或博客管理；
- 前端模拟的管理员身份、假数据或客户端授权。

当前没有角色、权限和用户角色模型。详情、更新、删除与改密由后端校验 `targetUserID == currentUserID`。用户列表不是前端可用能力，只有后端管理员身份和授权契约落地后才进入后续阶段。

`POST /v1/users` 是公开注册，不能包装成管理员创建用户。当前用户模型中的 `postCount` 只是只读数据，不代表已经存在博客管理 API。

## 3. Feature 与路由设计

### 3.1 Feature 边界

```text
features/
├── session/
│   ├── api/
│   │   ├── contracts.ts
│   │   ├── queries.ts
│   │   └── server.ts
│   ├── components/
│   └── index.ts
├── registration/
│   ├── api/
│   ├── forms/
│   └── index.ts
├── account/
│   ├── api/
│   ├── forms/
│   ├── components/
│   └── index.ts
└── user-admin/                  # 仅 U6 条件满足后新增
```

- `session` 持有登录、刷新、退出和当前身份 query；
- `registration` 持有公开注册 schema 和 mutation；
- `account` 持有资料更新、改密和注销账号；
- routes 负责组合 feature。`account` 不深层导入 `session`，资料页由 route loader 获取当前身份并传入 account 表单；
- `user-admin` 在后端 RBAC 就绪前不存在于生产 route tree。

### 3.2 页面路由

| 路由 | Feature | 页面职责 |
| --- | --- | --- |
| `/login` | `session` | 登录 |
| `/register` | `registration` | 公开注册 |
| `/session-refresh` | `session` | 唯一会话恢复入口 |
| `/` | `session` | 当前账号摘要和账号自助入口 |
| `/settings/profile` | `session + account`，由 route 组合 | 资料查看与更新 |
| `/settings/security` | `account` | 修改密码 |
| `/settings/account` | `account` | 注销账号 |

首期导航只包含“概览、个人资料、安全、账号”。`/admin/users` 在 U6 之前没有路由、导航、loader、预取或后台请求。

### 3.3 路由状态所有权

- `returnTo` 影响重定向目标和安全控制流，必须由 TanStack Router `validateSearch` 解析，再校验为站内绝对路径；
- `returnTo` 拒绝 scheme、host 和 `//` 开头的值；
- `reason`、`registered` 只影响提示展示，可以由 `nuqs` 管理；
- token、用户名、邮箱、手机号和密码不进入 URL；
- 同一个 search key 不能同时属于 Router schema 和 nuqs。

## 4. Console BFF 映射

| Console BFF | 上游 Go API | Credential | CSRF | 成功后的前端动作 |
| --- | --- | --- | --- | --- |
| `GET /api/auth/csrf` | BFF 本地生成 | `anonymous` | 同源、no-store | 写签名 CSRF Cookie 并返回 header token |
| `POST /api/auth/login` | `POST /v1/auth/login` | `anonymous` | Origin + CSRF | 写 token Cookie，刷新 session route |
| `POST /api/auth/refresh` | `PUT /v1/auth/refresh-token` | `refresh` | Origin + CSRF | 原子替换 token Cookie |
| `GET /api/auth/session` | `GET /v1/auth/info` | `access` | 不适用 | 写入 canonical session query |
| `POST /api/auth/logout` | `POST /v1/auth/logout` | `access` | Origin + CSRF | 清 Cookie 和全部 Query cache |
| `POST /api/register` | `POST /v1/users` | `anonymous` | Origin + CSRF | 保持匿名态并跳转登录 |
| `PUT /api/account` | 先 `GET /v1/auth/info`，再 `PUT /v1/users/{userID}` | `access` | Origin + CSRF | 失效 session query |
| `PUT /api/account/password` | 先 `GET /v1/auth/info`，再 `PUT /v1/users/{userID}/change-password` | `access` | Origin + CSRF | 清 Cookie/cache 并重新登录 |
| `DELETE /api/account` | 先 `GET /v1/auth/info`，再 `DELETE /v1/users/{userID}` | `access` | Origin + CSRF | 清 Cookie/cache 并离开受保护区域 |

读取当前账号只有 `GET /api/auth/session` 一条浏览器路径。资料页不得再建立重复的 `GET /api/account` query，避免同一 `/v1/auth/info` 数据拥有两份 cache 和失效语义。

更新、改密和删除的目标 `userID` 始终由 BFF 通过当前会话解析。浏览器提交的目标 ID 和 `Authorization` 必须被忽略。

## 5. 会话与缓存语义

### 5.1 Canonical session query

`session` feature 定义并导出唯一 query factory：

```ts
const sessionKeys = {
  all: ['session'] as const,
  current: () => [...sessionKeys.all, 'current'] as const,
};
```

`sessionKeys.current()` 的数据来自 `/api/auth/session`，同时服务 route guard、首页账号摘要和资料表单初始化。不要再建立 `currentUser` 或 `account.current` 重复 key。

私有 session 数据默认 `staleTime` 为 30 秒，`gcTime` 不超过 5 分钟，不持久化到 Web Storage。

### 5.2 Cache 变化

| 操作 | Cache 行为 |
| --- | --- |
| 登录成功 | 清匿名态错误，预取 `sessionKeys.current()` |
| 注册成功 | 不写 session cache，仍保持匿名态 |
| 更新资料 | 精确失效 `sessionKeys.current()` |
| refresh 成功 | `router.invalidate()`，保留与当前身份一致的数据 |
| logout | `queryClient.clear()` |
| 修改密码成功 | `queryClient.clear()`，不再调用旧 refresh token |
| 注销账号成功 | `queryClient.clear()` |

跨身份边界必须清理全部 Query cache，避免下一个账号读取上一个账号的数据。普通业务 mutation 是否乐观更新由 feature 单独决定；User 自助写操作首期不需要乐观更新。

### 5.3 Refresh 与退出

- `/session-refresh` 是唯一 refresh 协调入口；
- 多个 `401` 共享一个浏览器 single-flight Promise，原请求最多重放一次；
- refresh 再次 `401` 时清 Cookie/cache，并进入 `/login?reason=session-expired`；
- logout Server Route 只要收到请求，无论上游结果如何都清 Cookie；浏览器收到响应后再清 Query cache；
- 上游不可用时返回 `revocationConfirmed:false`，只提示“本地已退出，服务端撤销未确认”；Console BFF 本身不可达时则退出未完成，必须允许重试；
- logout 和修改密码都会递增后端 `token_version`，成功后不能继续使用旧 refresh token。

## 6. User 数据契约

### 6.1 字段约束

| 字段/表单 | 后端当前约束 | 前端实现 |
| --- | --- | --- |
| `username` | 3-20 个字符，只允许字母、数字、下划线 | Zod 长度和 `/^[A-Za-z0-9_]+$/` |
| `password` | 至少 6 个字符，至少一个字母和一个数字 | 登录只检查非空；注册和新密码使用完整 schema |
| `nickname` | UTF-8 字节数 `< 30` | `TextEncoder().encode(value).length < 30` |
| `email` | 非空且符合后端 email 规则 | 注册必填；更新时不把空字符串解释为清空 |
| `phone` | 中国大陆 `^1[3-9]\d{9}$` | 注册必填；更新时空字符串不提交 |
| 资料更新 | 字段 optional，只更新显式 presence | 对初始值与 dirty fields 做 diff，提交最小 payload |
| 修改密码 | `oldPassword`、`newPassword` | 新密码使用密码 schema，成功后清会话 |
| 注销账号 | 后端不要求密码确认 | UI 输入当前用户名二次确认，BFF 仍执行当前用户解析 |

昵称长度当前来自 Go `len(string)`，即 UTF-8 字节数。后端若改为 rune、字符或字素计数，必须在同一业务变更中更新 schema 和契约测试。

### 6.2 展示约束

- `status: 0=活跃, 1=禁用`；
- `gender: 0=未知, 1=男, 2=女`；
- 未知枚举显示稳定 fallback，不导致页面崩溃；
- `createdAt`、`updatedAt`、`lastLoginAt` 是 Unix 秒，进入 UI 前转换为 ISO 或格式化文本；
- 首期只允许编辑 `username`、`nickname`、`email`、`phone`；
- `status`、`gender`、`avatar`、`description`、`postCount` 只读或不展示编辑控件。

## 7. User 错误映射

| 后端语义 | 页面行为 |
| --- | --- |
| `Auth.TokenInvalid`、`Auth.TokenExpired`、HTTP `401` | 进入 session refresh；失败后重新登录 |
| `Auth.Unauthenticated` | 登录表单显示统一凭据错误，不区分账号不存在、密码错误或禁用 |
| `Auth.PermissionDenied`、HTTP `403` | forbidden，不自动重试 |
| `User.AlreadyExists`、HTTP `409` | 注册或资料表单显示唯一性冲突 |
| `User.InvalidUsername`、`User.InvalidPassword`、`Common.InvalidArgument` | 映射 field 或 form root error |
| `User.NotFound`、`Common.NotFound`、HTTP `404` | 当前账号已删除时清会话，否则 not-found |
| `Service.TooManyRequests`、HTTP `429` | 提示稍后重试，遵守可信 `Retry-After` |
| `Infra.*`、`Service.*`、HTTP `5xx` | 通用失败、request ID、允许用户显式重试 |

未知 reason 使用安全通用文案和 request ID，不能默认为某个字段错误，也不能展示 SQL、堆栈、JWT 或内部主机信息。

## 8. 垂直切片计划

```text
U1 会话闭环
  |-> U2 公开注册
  `-> U3 本人资料
        `-> U4 改密与注销
              `-> U5 User 首期发布就绪

平台 F0-F4 + 后端 RBAC/分页契约就绪 -> U6 管理员列表
```

U2 和 U3 可以在 U1 稳定后并行。U4 复用 U3 已建立的当前用户解析和 account adapter。U6 不阻塞账号自助首期发布。

### 8.1 U1：登录、当前会话、刷新与退出

交付：

- 实现 `/login`、`/session-refresh`、`/api/auth/csrf`、login、session、refresh 和 logout routes；
- 实现 session query、受保护布局 guard、`returnTo` 安全解析和 refresh single-flight；
- 登录和刷新只写 HttpOnly Cookie，响应 body 不返回 token；
- 首页使用 canonical session query 展示当前账号摘要。

验证：

- Cookie、CSRF、正确/错误登录、非法 envelope 和上游超时；
- 多个 `401` 只刷新一次，再次 `401` 停止；
- logout 成功和上游不可用时都无 Cookie/cache 残留，Console BFF 不可达时不伪装成功；
- Playwright 覆盖登录、保护路由、access 过期、refresh 失效和退出；
- 浏览器 JS、Web Storage、URL、日志和快照均不出现 token。

### 8.2 U2：公开注册

交付：

- 实现 `/register`、注册 schema、RHF 表单和固定 BFF 映射；
- 注册成功进入 `/login?registered=1`，不自动登录；
- 处理已存在、字段错误、限流、网络失败和重复提交。

验证：

- 表格驱动覆盖用户名、密码、昵称字节数、邮箱和手机号边界；
- Server Route 覆盖 CSRF、请求体上限、reason 映射和不设置认证 Cookie；
- Playwright 覆盖注册成功后登录、已存在账号和无效输入；
- 注册后 session cache 保持匿名态。

### 8.3 U3：本人资料

交付：

- 实现 `/settings/profile` 和 `PUT /api/account`；
- route loader 使用 canonical session query 初始化表单；
- 只提交 changed fields，成功后只失效 session query；
- 浏览器不提交目标用户 ID。

验证：

- diff payload、空值、枚举 fallback 和时间转换；
- 浏览器提供的 `userID`/`Authorization` 被忽略；
- pristine 禁止提交、pending 防重复、field/root error 和成功失效；
- Playwright 验证最小字段更新及重新登录后的持久结果。

### 8.4 U4：修改密码与注销账号

交付：

- 实现 `/settings/security`、`PUT /api/account/password`；
- 改密成功清 Cookie/cache，进入 `/login?reason=password-changed`；
- 实现 `/settings/account`、`DELETE /api/account` 和用户名二次确认；
- 注销成功清会话并离开受保护区域。

验证：

- 旧/新密码、确认值、用户名确认、pending 和错误关联；
- 当前用户解析、错误密码、删除成功、不存在和上游超时；
- 改密后旧 access/refresh 不可用，新密码可登录；
- 注销后受保护页面不可访问。

### 8.5 U5：User 发布就绪

- 汇总 U1-U4 Playwright 主路径和异常路径；
- 验证 375px、768px、1280px、1440px 的布局与交互；
- 验证键盘、焦点、错误关联、对比度和 reduced-motion；
- 执行跨 Origin、CSRF 不匹配、开放重定向、敏感 header 和过大 body 回归；
- 在真实 Go API 隔离环境跑完整注册、登录、刷新、资料、改密、退出和注销链路；
- 证明日志、trace、bundle 和测试产物无 token、密码、Cookie 或完整个人资料。

### 8.6 U6：管理员用户列表（延期）

只有以下条件全部满足才开始：

- 后端存在可识别的管理员身份或权限；
- 管理员调用 `GET /v1/users` 不再被 validation 固定拒绝；
- 普通用户仍返回 `403`；
- Proto、Swagger 和运行时统一分页参数命名；
- 管理员授权和契约测试已经落地。

届时新增 `features/user-admin`、`/admin/users` 和显式 `/api/admin/users`。采用 TanStack Table 手动游标分页；`pageSize`、`pageToken`、`search`、`status`、`sort` 等请求 identity 由 Router search schema 管理，`view`、`density` 等纯 UI 状态才使用 nuqs。

后端没有排序、筛选或跨用户写 operation 时，前端不能用客户端数据模拟这些服务端能力。

## 9. Definition of Ready

开始某个 User 切片前必须满足：

- 前端平台 F0-F4 和后端接入准入已通过；
- 后端基线、Proto、Swagger 和本地配置可获得；
- 对应 path、method、security、envelope 和关键字段已经加入 User operation manifest；
- 成功、业务失败、网络失败、未认证和无权限行为已有验收场景；
- 测试账号和数据可隔离创建、修改和删除；
- 对应 feature 没有绕过平台 dependency boundary；
- U6 还必须满足第 8.6 节全部条件。

## 10. Definition of Done

一个 User 切片只有同时满足以下条件才完成：

- 测试提交体现 Red-Green-Refactor，覆盖成功、业务失败、网络失败和权限边界；
- Proto/OpenAPI、feature schema、BFF 映射、reason 和页面行为一致；
- 业务 schema 与 query key 位于所属 feature，不进入 platform shared/infrastructure；
- 浏览器不持有 Bearer token，所有写操作只经显式 BFF Server Route；
- 页面具备适用的 loading、empty、error、unauthorized、forbidden 和 not-found 状态；
- 表单或表格通过键盘、焦点、错误关联、响应式和 reduced-motion 验证；
- session query 没有重复数据 owner，身份切换后无跨账号缓存；
- `pnpm format:check`、lint、`boundaries:check`、typecheck、api:check、coverage 和 build 通过；
- 达到 U5 或涉及关键会话流程时，相应 Playwright 和真实 Go API E2E 通过；
- 未运行项、已知风险和后续任务在证据包中明确记录。

## 11. 证据包

每个切片在 PR 或 CI artifact 中记录：

```text
切片：Ux / 名称
前端平台基线：<commit>
后端基线：<commit>
OpenAPI SHA-256：<digest>
运行环境：Node <version> / pnpm <version>
门禁：<command> -> PASS|FAIL
测试：<新增测试和关键场景>
覆盖率：lines/statements/branches/functions
E2E：<project、场景、trace/screenshot>
安全检查：<Cookie、CSRF、敏感信息扫描>
未运行项：<原因、风险、责任人或后续任务>
```

截图只能补充视觉证据，不能替代可重复断言。所有测试 token、Cookie、密码和个人资料必须脱敏。

## 12. 业务 ADR 触发条件

| 事项 | 当前默认 | ADR 触发条件 |
| --- | --- | --- |
| 当前账号读取 | canonical `/api/auth/session` query | 需要独立账户详情模型或不同缓存生命周期 |
| logout 撤销粒度 | 后端撤销账号全部旧 token | 需要按设备或单会话退出 |
| logout 上游不可用 | 清本地会话并报告未确认 | 产品要求保证服务端撤销完成 |
| 注册 | 公开注册，不自动登录 | 改为邀请制、管理员创建或注册即登录 |
| 昵称长度 | UTF-8 `< 30 bytes` | 后端改为字符或字素计数 |
| 管理员列表 | U6 延期 | RBAC、管理员身份和列表授权落地 |
| 分页参数 | 运行时 snake_case | Proto、Swagger 或网关统一新的公开契约 |
| 搜索、排序、过滤 | 不提供 | 后端定义参数、索引、稳定排序和游标语义 |

业务 ADR 不能改写平台安全边界；若涉及渲染、BFF、Cookie、runtime 或共享包，同时触发前端平台 ADR。
