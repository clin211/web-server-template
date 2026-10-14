# Web Console 后端接入规范

> 状态：Go API 接入实施基线
> 适用范围：`web` TanStack Start BFF 与 `gin-enterprise-template` API
> 文档版本：v1.0
> 最后更新：2026-08-29
> 后端基线：`2ecca79`

## 1. 文档职责

本文定义 Web Console 如何接入当前 Go API，包括契约来源、BFF 映射规则、credential policy、Cookie 会话、CSRF、错误包络、环境变量和契约测试。本文是[前端平台架构](./web-console-frontend-architecture.md)的基础设施适配实现，不定义 User 页面、字段展示或产品交付顺序；这些内容见 [User 业务实施计划](./web-console-user-domain-plan.md)。

依赖方向固定为：

```text
User 或其他业务 feature
  -> integrations/go-api
  -> 前端平台 infrastructure ports
```

平台 infrastructure 不能反向依赖 Go API 的具体 operation，其他业务 feature 也不能直接读取认证 Cookie 或拼接 Bearer header。

## 2. 契约事实来源

发生不一致时按以下顺序判断：

1. `pkg/api/apiserver/v1/*.proto`；
2. `api/openapi/apiserver.swagger.json`；
3. 后端 handler、validation、biz 和测试；
4. 本文；
5. feature 内的 Zod schema 和 TypeScript 类型。

当前 OpenAPI 是 Swagger 2.0 单文件，由 `make protoc` 生成。它已经描述 `BearerAuth`、operation security、成功包络和统一错误包络。后端默认监听配置文件中的 `0.0.0.0:5558`，但该地址只用于本地或服务端内网接入，不能进入浏览器 bundle。

前端不得把本文或手写 TypeScript 类型当成后端契约事实来源。发现 Proto、Swagger 与运行时不一致时 fail closed，先修正契约或建立显式兼容层。

## 3. 接入分层

### 3.1 Infrastructure port

`infrastructure/http/server-client.ts` 提供 credential-neutral 的上游请求能力。调用方必须显式传入策略，而不是让 client 对所有请求默认读取 access token：

```ts
type CredentialPolicy =
  | { kind: 'anonymous' }
  | { kind: 'access' }
  | { kind: 'refresh' };
```

该 client 只负责：

- 接收组合根注入的 `baseUrl`，并由 `infrastructure/env.ts` 提供经校验的服务端配置；
- 只接受相对路径和受允许的 HTTP 方法；
- 根据 credential policy 调用 session adapter 注入凭据；
- 注入受控 `X-Request-ID` 和允许的 trace 关联头；
- 应用总超时和 abort signal；
- 返回原始 `Response` 或 transport result，不理解 User 字段和业务错误码。

`infrastructure/http/browser-client.ts` 只接受同源相对 `/api/*` URL，默认 `credentials: 'include'`，不接受上游 origin 或 `Authorization` 参数。

### 3.2 Go API integration

`integrations/go-api/` 是 Go API 通用适配层，负责：

- 校验 `{code,message,data}` 与 `{code,message,reason}` transport envelope；
- 把 HTTP 状态、`code`、`reason`、request ID 转换为可判别的 `UpstreamApiError`；
- 丢弃未知敏感字段，不透传堆栈和连接信息；
- 暴露构建 feature adapter 所需的最小函数，不建立包含全部业务 operation 的上帝 client。

业务请求和响应 schema 仍由 feature 持有。通用适配层只拥有 envelope 等已证明跨 feature 稳定的协议 schema；分页等协议在契约统一且出现第二个真实消费者前，仍由首个 feature 持有。

### 3.3 Operation manifest

每个 feature 在自己的业务目录提交 manifest，由平台脚本聚合校验。最小结构为：

```ts
type OperationManifest = {
  feature: string;
  operations: Array<{
    name: string;
    method: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
    path: string;
    upstreamSecurity: 'anonymous' | 'bearer';
    credentialPolicy: 'anonymous' | 'access' | 'refresh';
    successRef: string;
    errorRef: string;
  }>;
};
```

`path` 必须是 Swagger 中已存在的相对路径；`method`、`upstreamSecurity` 和成功/失败 `$ref` 必须与 Swagger 匹配。Swagger 只能证明 anonymous 或 Bearer，`credentialPolicy` 中的 access/refresh 区分由 BFF 适配测试证明。`data` 字段的业务 schema 仍在 feature 运行时校验，不放进通用 manifest 以避免业务所有权倒流。

### 3.4 Feature adapter

每个 feature 的 `api/server.ts`：

- 固定上游方法和相对路径；
- 选择 `anonymous`、`access` 或 `refresh` credential policy；
- 使用本 feature 的 Zod schema 校验输入和 `data`；
- 将稳定后端 reason 映射为本 feature 的 domain error；
- 不读取另一个 feature 的内部 schema 或 query key。

相应 `routes/api/<feature>/*` 只处理 HTTP 边界和调用 feature adapter，不复制 schema、缓存或业务规则。

## 4. BFF 安全边界

浏览器只调用同源 TanStack Start Server Route。每条 BFF route 必须：

- 显式注册方法和路径，不提供 catch-all proxy；
- 限制请求体大小和 Content-Type；
- 拒绝绝对上游 URL、未知方法和未登记路径；
- 丢弃浏览器传入的 `Authorization`、`Cookie`、`Host` 和 hop-by-hop headers；
- 由服务端重新注入凭据、request ID 和允许的 trace headers；
- 对写请求执行可信 Origin/Referer 与 CSRF 校验；
- 对上游响应执行 envelope 和 feature data schema 校验；
- 使用稳定、安全的 Console 错误响应，不透传上游内部信息。

BFF 不复制 Go 业务授权。隐藏导航和客户端 route guard 只是交互优化，不是授权边界。

## 5. Cookie 会话适配

### 5.1 Cookie 契约

| 环境 | Cookie 名 | 属性 | 内容 |
| --- | --- | --- | --- |
| 生产 | `__Host-console-access` | `HttpOnly; Secure; SameSite=Lax; Path=/` | access token |
| 生产 | `__Host-console-refresh` | `HttpOnly; Secure; SameSite=Lax; Path=/` | refresh token |
| 生产 | `__Host-console-csrf` | 非 HttpOnly、`Secure; SameSite=Lax; Path=/` | 签名 CSRF token |
| 本地 HTTP | `console-access`、`console-refresh`、`console-csrf` | 不带 `Secure`，其余语义一致 | 本地开发专用 |

生产 `__Host-` Cookie 不设置 `Domain`。Cookie 值不得进入浏览器 JavaScript、JSON 响应、URL、日志、trace 或测试快照。

access Cookie 过期时间使用后端返回的 RFC3339 `expireAt`。refresh Cookie 过期时间从服务端收到的 JWT 标准 `exp` claim 读取并做最大期限校验；解析 claim 只为 Cookie 生命周期管理，不等价于验证 JWT 签名，最终认证仍由 Go API 完成。

`CONSOLE_SESSION_SECRET` 只用于签名 CSRF token，不用于签发或替代后端 JWT。

### 5.2 Session port

平台层只依赖下列抽象语义：

```ts
type TokenSet = {
  accessToken: string;
  refreshToken: string;
  accessExpiresAt: string;
  refreshExpiresAt: string;
};

interface SessionPort {
  readAccess(request: Request): string | null;
  readRefresh(request: Request): string | null;
  commit(response: Response, tokens: TokenSet): Response;
  clear(response: Response): Response;
}
```

`integrations/go-api/session-adapter.ts` 实现该 port。原始 token 只在当前服务端请求生命周期内存在，不写入 Start 可序列化 context，不形成进程级可变全局状态。

## 6. CSRF 与 Origin

匿名写操作和已认证写操作都需要 CSRF 防护。流程固定为：

1. `GET /api/auth/csrf` 在 CSRF pair 缺失或需要轮换时生成随机 nonce 和服务端签名；
2. 签名 token 写入非 HttpOnly Cookie，并通过 no-store JSON 返回同一个 token；
3. 浏览器在后续写请求的 `X-CSRF-Token` header 携带该值；
4. BFF 校验签名、Cookie/header 常量时间相等，并验证 `Origin` 或受控 `Referer`；
5. 登录、退出、会话失效或安全事件后轮换或清理 CSRF pair。

不能接受调用方自选的未签名 CSRF Cookie。TanStack Start `createCsrfMiddleware` 保护 Server Function；显式 Server Route 仍执行上述双提交策略，两者不能互相替代。

`CONSOLE_ORIGIN` 是规范化公开 Origin。反向代理必须正确传递可信协议和 host；不能信任来自任意公网客户端的 forwarded headers。

## 7. 请求、刷新与退出语义

### 7.1 普通 access 请求

- `access` policy 从 session adapter 读取 access Cookie；
- 上游返回 `401` 时转换为稳定 `SessionExpired`，普通 Server Route 不自行 refresh；
- 浏览器 session coordinator 导航到唯一刷新入口；
- 私有响应使用 no-store 语义。

### 7.2 Refresh

- refresh route 使用 `refresh` policy 调用后端刷新 operation；
- 成功后原子替换 access 和 refresh Cookie；
- 浏览器维护 refresh single-flight，同一时刻只发送一个 refresh；
- 等待者最多重放原请求一次，再次 `401` 时停止并清理会话；
- 多实例正确性依赖后端 refresh token 单次消费，不能依赖某个 Node 进程内锁；
- refresh token 复用或轮换竞争最终失败时回到重新认证，不无限重试。

### 7.3 Logout

logout Server Route 先尝试调用后端撤销 operation，然后在 `finally` 语义下清理响应中的 Cookie。只要该 Server Route 已收到请求，上游不可用也不影响 Cookie 清理；返回 `revocationConfirmed:false` 表示服务端撤销未确认。

浏览器 session coordinator 在收到 logout 响应后清理 Query cache。如果 Console BFF 本身不可达，浏览器无法删除 HttpOnly Cookie，不能声称已退出；UI 应保持准确失败状态并允许重试。

具体哪些业务操作会使旧 token 失效，由对应业务实施文档定义。

## 8. 响应与错误协议

当前成功和失败包络分别为：

```ts
type ApiSuccess<T> = {
  code: 0;
  message: string;
  data: T;
};

type ApiFailure = {
  code: number;
  reason: string;
  message: string;
};
```

适配器同时检查 HTTP 状态、`code` 和 runtime schema，不能把 HTTP `200` 直接视为业务成功。错误分类至少包括：

| 分类 | 输入 | 通用行为 |
| --- | --- | --- |
| `Unauthenticated` | HTTP `401` 或认证 reason | 进入 session 恢复或重新认证 |
| `Forbidden` | HTTP `403` 或权限 reason | 不自动重试，交给 route 呈现 forbidden |
| `InvalidInput` | HTTP `400` 或参数 reason | 由 feature 尝试映射 field/root error |
| `NotFound` | HTTP `404` | 由 feature 决定 not-found 或会话清理 |
| `Conflict` | HTTP `409` | 由 feature 映射冲突语义 |
| `RateLimited` | HTTP `429` | 尊重可信 `Retry-After`，默认不自动 mutation retry |
| `UpstreamUnavailable` | 网络错误、超时、HTTP `5xx` | 安全提示、request ID、允许显式重试 |
| `InvalidEnvelope` | schema/code 不一致 | fail closed，记录脱敏协议错误 |

具体 `reason -> domain error -> 页面行为` 映射归 feature，不进入通用 HTTP client。

## 9. OpenAPI 消费与漂移检查

当前 Swagger 2.0 与 TypeScript 7 工具链不预设一个未经验证的全量代码生成器。首期策略是：

- 通用 envelope 与 feature 最小数据边界使用 Zod；
- `scripts/check-api-contract.mjs` 读取 `api/openapi/apiserver.swagger.json`；
- 每个 feature 提交自己的 operation manifest，声明依赖的 path、method、security、成功/失败 envelope 和关键字段；
- 平台契约工具聚合 manifests，但不硬编码 User 或其他业务 operation；
- 删除或改变必需 operation、security 或 `$ref` 时检查必须失败；
- 运行时 Zod 校验仍保留，不能用静态检查替代不可信网络边界校验。

如果未来引入生成器，必须证明支持当前 Swagger/OpenAPI 版本、TypeScript major、包管理和 envelope 结构，并通过 ADR 替换现有策略。

## 10. 环境变量

| 变量 | 作用 | 是否允许 `VITE_` |
| --- | --- | --- |
| `BACKEND_API_URL` | Go API 服务端内网地址 | 否 |
| `CONSOLE_ORIGIN` | 规范化公开 Origin，用于 Cookie 和 CSRF | 否 |
| `CONSOLE_SESSION_SECRET` | CSRF token 签名密钥，至少 32 字节 | 否 |
| `CONSOLE_REQUEST_TIMEOUT_MS` | BFF 调用上游的总超时，默认 `10000` | 否 |

上述变量由 `infrastructure/env.ts` 在服务端启动时用 Zod 校验，缺失或非法时 fail fast。禁止把后端 URL、JWT secret、service token、数据库/Redis 凭据或 CSRF secret 放入 `VITE_*`。

## 11. 测试与准入

基础接入测试必须覆盖：

- credential-neutral client 对三种 policy 的选择和错误使用；
- 相对路径白名单、敏感 header 丢弃、超时和 abort；
- 成功/失败/非法 envelope；
- Cookie 名称、属性、expiry、原子替换和清理；
- CSRF 签发、签名、轮换、Origin、缺失与不匹配；
- refresh single-flight、最多一次重放和再次 `401` 停止；
- logout Server Route 在上游成功和不可用时都清理 Cookie，并区分 BFF 本身不可达的失败；
- request ID 生成、校验、传递和日志脱敏；
- 浏览器 bundle、日志和快照不包含服务端配置或 token；
- production build 中 Server Route 能写 Cookie，并能访问配置的 Go API。

接入层完成定义：

- infrastructure 不包含 Go API 包络、具体业务 path、schema、query key 或 reason 映射；
- `integrations/go-api` 不引用任何 feature，业务 `data` schema 仍归 feature；
- feature adapter 不能绕过 infrastructure client 直接拼接上游 origin；
- OpenAPI manifest 检查和运行时 Zod 校验均可证明失败；
- Cookie、CSRF、refresh 和 logout 安全场景有分支全覆盖；
- 上游不可用、契约漂移和环境配置缺失均 fail closed；
- `pnpm api:check`、相关 Vitest、build 和 BFF smoke test 全部通过。

## 12. ADR 触发条件

以下变化必须记录 ADR：

- 浏览器直接调用 Go API；
- 更换 Cookie 会话为浏览器持有 token；
- 跨域、多子域或第三方身份提供商改变 Cookie/CSRF 模型；
- 自动 refresh 从浏览器 coordinator 移入任意普通 Server Route；
- 引入通配上游代理；
- 引入全量 OpenAPI 生成器并替换 Zod/manifest 策略；
- 需要按设备或单会话撤销，而后端不再使用当前账号级撤销语义；
- 将 BFF 部署到 Edge runtime 或独立服务。
