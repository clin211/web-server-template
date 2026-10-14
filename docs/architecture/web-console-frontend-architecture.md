# Web Console 前端架构

> 状态：前端平台基建实施基线
> 适用范围：仓库根目录下的 `web` TanStack Start Web Console
> 文档版本：v2.1
> 最后更新：2026-08-31
> 包管理器：`pnpm`

## 1. 文档目的

本文只定义 Web Console 的平台架构：技术选型、运行时边界、目录与依赖方向、路由和状态职责、安全基线、质量门禁以及构建部署约束。目标是在不依赖具体业务资源、页面清单或后端字段的前提下，建立后续 feature 可以复用的工程基座。

本文不定义具体业务范围、API 路径、字段校验、错误码映射、页面信息架构或业务发布顺序。相关内容由以下文档负责：

- [Web Console 后端接入规范](./web-console-backend-integration.md)：Go API、OpenAPI、BFF、认证会话和错误协议适配；
- [Web Console User 业务实施计划](./web-console-user-domain-plan.md)：User 领域页面、接口、字段和垂直切片。

架构与业务文档必须单向依赖：业务实施计划可以引用本文，本文不能以某个业务 feature 的存在作为成立条件。

## 2. 架构目标与非目标

### 2.1 架构目标

- 让新 feature 按统一模板接入路由、数据请求、表单、表格、错误状态和测试，而不修改平台核心结构；
- 把浏览器、TanStack Start 服务端和上游 API 的信任边界显式分开；
- 让路由状态、服务端状态、表单状态和局部 UI 状态各有唯一 owner；
- 让业务 schema、query key 和错误映射由 feature 自己持有，避免全局业务注册表；
- 通过固定工具链、锁文件、CI 和生产构建 smoke test 保证可重复交付；
- 保持单应用内聚，只有出现第二个真实消费者时才抽取共享 package。

### 2.2 非目标

- 不提前建设微前端、插件系统、动态菜单引擎或通用低代码平台；
- 不为尚未出现的消费者建立 `packages/ui`、`packages/api` 或全局状态包；
- 不在前端复制后端授权、业务校验或资源状态机；
- 不在本文固化具体业务路由、字段、Query Key、Table columns 或发布阶段；
- 不因为框架支持 SSR 就默认启用 SSR。

## 3. 架构决策摘要

| 决策 | 结论 | 约束 |
| --- | --- | --- |
| 应用边界 | `web` 是仓库根目录下的独立单包 pnpm 应用 | 只保留一个 `package.json` 和一个 `pnpm-lock.yaml`，不建立 `pnpm-workspace.yaml` 或多包委托脚本 |
| 应用框架 | TanStack Start + TanStack Router + React 19 | 不并列引入第二套路由或全栈框架 |
| 渲染模型 | `defaultSsr: false`，客户端优先 | 按路由开启 SSR 前必须有量化收益、数据边界证明和 ADR |
| 浏览器 API 边界 | 浏览器只调用同源显式 Server Route | 禁止浏览器直接访问上游 API 或持有 Bearer token |
| 服务端 API 边界 | BFF 只做传输、安全和协议适配 | 业务授权仍由上游服务执行；禁止 catch-all 代理 |
| 服务端状态 | TanStack Query | 每个 feature 自持 query key factory 和 query options |
| 路由状态 | TanStack Router | 参与导航、权限、loader 或请求 identity 的参数由 Router schema 管理 |
| 纯 UI URL 状态 | `nuqs` | 只管理不影响控制流和数据请求的展示状态 |
| 表单 | React Hook Form + Zod | 业务输入 schema 归 feature，客户端校验不替代服务端校验 |
| 表格 | TanStack Table | 表格实例、columns、分页和筛选语义归 feature |
| UI | HeroUI + Tailwind CSS v4 + Motion | 共享层只封装稳定 token 和跨 feature 组件 |
| 工程质量 | TypeScript、oxlint、oxfmt、Vitest、Playwright、Husky、lint-staged | 本地快速门禁和 CI 完整门禁职责分开 |

## 4. 技术栈与版本基线

下表是 2026-08-29 初始化工程时通过 npm registry `latest` dist-tag 核验的稳定版本。实际依赖事实来源依次为 `web/package.json` 和 `web/pnpm-lock.yaml`；本文表格是 bootstrap 基线，不替代锁文件。升级必须通过独立 PR、变更记录和完整门禁。

### 4.1 核心栈

| 类别 | 包 | 初始化版本 |
| --- | --- | --- |
| Node.js 运行时 | `node` | `26.8.1` |
| 包管理 | `pnpm` | `11.24.0` |
| 框架 | `@tanstack/react-start` | `1.168.49` |
| 路由 | `@tanstack/react-router` | `1.170.32` |
| 构建工具 | `vite` | `8.2.2` |
| 语言 | `typescript` | `7.0.2` |
| UI 运行时 | `react`、`react-dom` | `19.2.8` |
| 样式 | `tailwindcss`、`@tailwindcss/vite` | `4.3.3` |
| UI 组件 | `@heroui/react` | `3.2.4` |
| 服务端状态 | `@tanstack/react-query` | `5.102.8` |
| 数据表格 | `@tanstack/react-table` | `9.2.3` |
| 表单 | `react-hook-form` | `7.86.0` |
| 表单校验适配 | `@hookform/resolvers` | `5.9.1` |
| URL 状态 | `nuqs` | `2.10.1` |
| Schema 校验 | `zod` | `4.4.3` |
| 动画 | `motion` | `13.1.1` |
| Lint | `oxlint` | `1.80.0` |
| Format | `oxfmt` | `0.65.0` |
| 单元测试 | `vitest` | `4.1.11` |
| Git hooks | `husky` | `9.1.7` |
| staged 检查 | `lint-staged` | `17.4.1` |

### 4.2 配套包

| 用途 | 包 | 初始化版本 |
| --- | --- | --- |
| Node 类型 | `@types/node` | `26.4.0` |
| React 类型 | `@types/react`、`@types/react-dom` | `19.2.18`、`19.2.5` |
| Query 调试 | `@tanstack/react-query-devtools` | `5.102.8` |
| Router 调试 | `@tanstack/react-router-devtools` | `1.167.1` |
| React Vite 集成 | `@vitejs/plugin-react` | `6.1.1` |
| Node 构建适配 | `@tanstack/nitro-v2-vite-plugin` | `1.155.0` |
| Vitest DOM 环境 | `jsdom` | `30.0.1` |
| React 测试 | `@testing-library/react`、`@testing-library/dom` | `16.3.3`、`10.4.1` |
| 用户交互测试 | `@testing-library/user-event` | `14.6.6` |
| DOM 断言 | `@testing-library/jest-dom` | `7.0.1` |
| 覆盖率 | `@vitest/coverage-v8` | `4.1.11` |
| API mock | `msw` | `2.15.0` |
| 锁文件解析 | `yaml` | `2.9.0` |
| 图标 | `lucide-react` | `1.34.0` |
| HeroUI peer 依赖 | `react-aria`、`react-aria-components`、`@react-aria/ssr` | `3.51.0`、`1.20.0`、`3.10.1` |
| 端到端测试 | `@playwright/test` | `1.62.1` |

Node.js 通过 `web/.node-version`、`engines`、`packageManager` 和 CI toolchain 锁定。命令必须在 `web` 目录执行，不建立 workspace，也不混用 npm 或 Yarn。

pnpm 11.24 会把依赖构建授权写入 `pnpm-workspace.yaml`，但本项目明确是单包且禁止该文件。当前 frozen install 固定使用：

```bash
PNPM_CONFIG_DANGEROUSLY_ALLOW_ALL_BUILDS=true pnpm install --frozen-lockfile
```

该环境变量只用于已经过锁文件评审的 CI、容器构建和本地安装；依赖升级必须独立审查其 install script。不得为了记录 `allowBuilds` 静默生成 `pnpm-workspace.yaml`。如果后续 pnpm 提供单包级精确 allowlist，或项目决定收紧为依赖级 allowlist，必须通过工具链变更或 ADR 迁移。

`@tanstack/nitro-v2-vite-plugin` 使用稳定 dist-tag，但 README 仍标记为 Experimental。工程基建准入必须验证生产构建、Node 启动、Server Route、Cookie、静态资源和终止信号；验证失败时记录部署 ADR，不静默切换到 beta/nightly 依赖。

`scripts/check-dependency-policy.mjs` 使用 YAML parser 读取完整 lockfile。直接依赖出现任何预发布版本都会失败；传递依赖默认同样失败，当前只允许已经评审并被锁文件固定的 `h3@2.0.1-rc.20`、`unenv@2.0.0-rc.24` 和 Babel 构建期依赖 `gensync@1.0.0-beta.2`。allowlist 必须写精确版本，依赖升级导致版本或条目变化时重新评审，不能使用包名前缀或通配符放行。

## 5. 单应用与目录架构

### 5.1 单应用根目录

```text
web/
├── package.json
├── pnpm-lock.yaml
├── .node-version
├── Dockerfile
├── vite.config.ts
├── tsconfig.json
├── vitest.config.ts
├── playwright.config.ts
├── e2e/
├── public/
├── scripts/
└── src/
```

`web` 是前端应用的独立根目录，负责版本、安装、锁文件、脚本、构建和部署。它不加入 pnpm workspace，不依赖仓库根级 package.json，也不在父目录寻找第二份 lockfile。`package.json` 的 `name` 固定为 `@gin-enterprise/console`。

### 5.2 Console 目录骨架

```text
web/
├── scripts/
│   ├── check-api-contract.mjs
│   ├── check-boundaries.mjs
│   ├── check-browser-bundle.mjs
│   ├── check-dependency-policy.mjs
│   ├── check-production-smoke.mjs
│   └── lib/
└── src/
    ├── routes/
    │   ├── __root.tsx
    │   ├── _console.tsx
    │   ├── _console/
    │   │   └── index.tsx
    │   └── api/
    │       └── <feature>/
    ├── features/
    │   └── <feature>/
    │       ├── contracts.ts
    │       ├── api/
    │       │   ├── client.ts
    │       │   ├── queries.ts
    │       │   ├── mutations.ts
    │       │   └── server.ts
    │       ├── components/
    │       ├── <feature>.operation-manifest.json
    │       └── index.ts
    ├── infrastructure/
    │   ├── http/
    │   │   ├── browser-client.ts
    │   │   ├── browser-error.ts
    │   │   ├── server-client.ts
    │   │   ├── json-body.ts
    │   │   ├── errors.ts
    │   │   └── request-policy.ts
    │   ├── session/
    │   │   ├── browser-csrf.ts
    │   │   ├── port.ts
    │   │   ├── csrf.ts
    │   │   ├── guards.ts
    │   │   └── cookies.ts
    │   ├── observability/
    │   │   └── request-context.ts
    │   ├── env-startup-plugin.ts
    │   └── env.ts
    ├── integrations/
    │   └── <backend>/
    │       ├── client.ts
    │       ├── contracts.ts
    │       └── session-adapter.ts
    ├── shared/
    │   ├── components/
    │   │   ├── layout/
    │   │   ├── feedback/
    │   │   ├── table/
    │   │   └── ui/
    │   ├── validation/
    │   └── types/
    ├── styles/
    │   └── globals.css
    ├── routeTree.gen.ts
    ├── router.tsx
    ├── providers.tsx
    ├── start.ts
    └── server.ts
```

`<feature>` 只是结构占位符。具体 feature 名、路由文件、schema 和 query key 只在对应业务实施文档或代码中定义。公开路由布局 `_public.tsx` 在首个公开 feature 接入时随实际子路由一并创建，空布局不提前落盘，以避免与根路径 Console shell 产生路由歧义。

### 5.3 依赖方向

```text
composition root (routes, router.tsx, providers.tsx, start.ts, server.ts)
  -> features / integrations / infrastructure / shared
features -> integrations / infrastructure / shared
integrations -> infrastructure / shared
infrastructure -> shared
```

依赖规则：

1. `routes/` 只做路由声明、search/params 校验、loader 编排和页面组合，只能通过公开 API 导入 feature、integration、infrastructure 或 shared，不实现业务规则；
2. `features/<feature>` 持有该业务域的 Zod schema、API adapter、query key、表单和 UI；
3. feature 之间不能导入内部文件。跨 feature 流程由 route/application composition 编排，稳定公共能力下沉到 `shared/` 或 `infrastructure/`；
4. `integrations/<backend>` 实现某个具体后端的包络、credential 和 session adapter，可依赖 infrastructure/shared，不得引用 feature；
5. `infrastructure/` 只提供 HTTP、session port、环境变量、可观测性等技术机制，不理解具体后端包络、operation 或业务 reason；所有后端地址和密钥通过组合根注入，不由通用 client 自行决定；
6. `shared/` 必须业务无关，不能引用 routes、features、integrations 或 infrastructure；
7. 每个 feature 通过 `index.ts` 暴露最小公共 API，禁止跨目录深层导入；
8. 不建立全局业务 `contracts.ts`、`query-keys.ts` 或含义不明的 `utils.ts`；通用 transport/error 类型、后端包络和业务 schema 必须分开；
9. `routeTree.gen.ts` 由插件生成，禁止手工编辑。

`scripts/check-boundaries.mjs` 使用已安装的 TypeScript compiler API 解析 import graph，对上述依赖方向进行 CI 检查。不用字符串正则猜测 import，也不为首期边界检查引入重量级架构工具。

## 6. TanStack Start 运行时边界

### 6.1 入口职责

- `router.tsx` 通过 `getRouter()` 为每个运行时实例创建独立 Router、QueryClient 和 session coordinator；浏览器运行时只有一个实例，服务端禁止使用模块级可变 Router/cache；
- `providers.tsx` 从当前 Router context 接收 QueryClient，并组合 TanStack Query、nuqs 和 reduced-motion provider；HeroUI v3 的全局样式由 `globals.css` 引入，组件直接使用其 React Aria 上下文，不虚构 v2 的根 `HeroUIProvider`；
- `start.ts` 使用 `createStart` 设置 `defaultSsr: false`，注册全局 request middleware；
- `server.ts` 使用 `createServerEntry` 包装默认 fetch handler，并创建类型化的每请求 context；
- `routes/api/*` 定义显式 Server Route，由 feature server adapter 执行固定上游映射。

`RequestContext` 的平台字段只包含 request ID、开始时间和经过验证的基础请求元数据。Cookie 通过当前 `Request` 或 session adapter 读取，不把原始 token 复制进可序列化 route context。

自定义 `start.ts` 后显式注册 `createCsrfMiddleware({ filter: (ctx) => ctx.handlerType === 'serverFn' })`，保留 Start 对 Server Function 的默认同源保护。业务 Server Route 仍必须执行自己的 Origin 和 CSRF 策略。

### 6.2 客户端优先

- 首期所有业务页面继承 `defaultSsr: false`；
- loader 在浏览器导航阶段通过同源 BFF 和 `queryClient.ensureQueryData` 预取；
- 组件使用与 loader 相同的 query options，不维护服务端直连和浏览器 BFF 两套读取语义；
- 多个独立请求并行预取，使用 route pending UI，避免无意义瀑布；
- token、Cookie、服务端环境变量和未脱敏对象不得进入 loader 返回值或 hydration payload。

按路由开启 SSR 必须同时满足：存在可量化的首屏或可访问性收益；数据可安全序列化；服务端与客户端导航语义一致；缓存与多实例策略明确；已记录 ADR。SEO 本身不是内部 Console 开启 SSR 的充分理由。

### 6.3 BFF 边界

浏览器只能调用同源 `/api/*` Server Route。BFF 负责：

- 固定方法和路径映射；
- 请求体大小、超时和允许 header；
- credential policy、Cookie、Origin 与 CSRF；
- request ID/trace 关联；
- 响应 runtime schema 校验和错误归一化。

BFF 不负责资源授权和业务状态机，不提供 catch-all proxy，也不接受浏览器传入上游 origin、绝对 URL、`Authorization`、`Cookie` 或 `Host`。具体后端协议见后端接入文档。

`x-request-id` 只接受长度和字符集受限的关联值，否则由 Console 重新生成。入站 W3C `traceparent` 必须使用合法的 version `00`、非零 trace/span ID 和小写十六进制；`tracestate` 还需满足成员数、总长度、key/value 与重复 key 约束。非法 trace header 被丢弃，且调用方通过 `ServerRequestOptions.headers` 注入的 trace header 会先被清除。trace context 只用于链路关联，不参与身份或授权判断。

## 7. 数据与状态所有权

### 7.1 TanStack Query

TanStack Query 只管理远端服务端状态。每个 feature 在自己的 `api/queries.ts` 和 `api/mutations.ts` 中定义：

- query key factory；
- query options 和 mutation options；
- stale/gc/retry 策略；
- mutation 后需要失效或更新的精确 key。

禁止在 `shared/` 或根目录建立业务 query key 注册表。QueryClient 在单个 Router 运行时实例内唯一，但不能作为服务端进程级模块单例；cache key 和失效语义归 feature。会话边界发生变化时，由浏览器 session coordinator 触发当前实例的全局 cache 清理，服务端 session adapter 只负责 Cookie，避免混淆两个运行时。

默认不把私有 Query cache 持久化到 Web Storage。是否使用乐观更新由 feature 根据冲突语义决定，不能设为全局默认。

### 7.2 Router 与 nuqs

以下状态由 TanStack Router params/search schema 管理：

- 参与 loader 或 Query identity 的参数；
- 影响导航目标、重定向、权限或安全控制流的参数；
- 需要刷新、分享、前进/后退后仍保持的数据视图条件。

`nuqs` 只管理不影响请求和控制流的纯展示状态，例如视图模式、密度、面板开关。根 provider 使用 `nuqs/adapters/tanstack-router` 的 `NuqsAdapter`。同一个 key 不能同时由 Router schema 和 nuqs 管理。

### 7.3 React Hook Form 与 Zod

- feature 的 `contracts.ts` 或表单模块持有输入 schema；
- `@hookform/resolvers/zod` 连接 Zod 和 RHF；
- 客户端 schema 提供即时反馈，服务端仍是最终权威；
- 后端字段错误映射到 field，无法定位的错误进入 form root；
- mutation pending 时禁止重复提交；
- 密码、token 和敏感字段不得进入 URL、日志、telemetry 或 Query cache。

平台只提供通用表单布局、错误摘要、提交状态和可访问性组件，不提供业务字段 schema。

### 7.4 TanStack Table

Table 实例、row type、columns、分页、排序、筛选和行操作都归所属 feature。平台只规定：

- 远端分页使用 `manualPagination`，不得伪装成客户端全量分页；
- 后端不支持的排序或筛选不能在 UI 中表现为服务端能力；
- row identity 使用稳定业务标识，不使用数组索引；
- loading、empty、error、forbidden 和 stale 状态可区分；
- 表格支持键盘操作、可访问名称和窄屏滚动；
- 危险 mutation 使用确认对话框并在 pending 时锁定。

## 8. UI 与设计系统边界

- HeroUI 是可访问交互组件基础，`shared/components/ui` 只封装稳定 token、统一默认属性和跨 feature 复用行为；
- Tailwind CSS v4 使用 `@import "tailwindcss"`、`@theme` 和 CSS variables，不复制 Tailwind v3 大型配置；
- 颜色、间距、最大 8px 圆角、阴影和 focus ring 使用集中 token；首期固定明亮企业主题，暗色或高对比度主题必须在具有完整 token 和可访问性验证后再启用；
- Console 采用克制、可扫描、工作台式布局，避免营销式 hero、装饰性卡片堆叠和单色调页面；
- 图标使用 `lucide-react`，不熟悉的图标按钮提供 label 或 tooltip；
- Motion 只用于状态过渡和操作反馈，必须尊重 `prefers-reduced-motion`；
- 表单、表格、弹窗和导航达到 WCAG 2.2 AA 的键盘、焦点、对比度与错误关联要求。

业务组件留在 feature。只有至少两个真实 feature 复用且 API 稳定后，才允许提升到 shared。

## 9. 错误、加载与可观测性

- 根路由和布局提供 `pendingComponent`、`errorComponent` 与 `notFoundComponent`；
- route 控制流异常不能被宽泛 `try/catch` 吞掉；
- transport error、session error、validation error 和 domain error 使用可判别类型，不依赖错误文案分支；
- 未知错误使用安全兜底文案和 request ID，不透传堆栈、主机名或连接信息；
- request ID 只用于关联，不能作为身份或授权依据；
- 日志只记录路由模板、状态码、稳定错误码、耗时、request ID、trace ID 和脱敏主体标识；
- 密码、Cookie、Authorization、JWT、完整请求体和个人敏感字段不得进入日志或前端错误上报。

浏览器 telemetry 首期不是必需依赖。引入错误 SDK 或浏览器 OTel 前必须评审 PII、采样、CSP、成本和数据驻留。

## 10. 测试架构

### 10.1 测试分层

| 层级 | 工具 | 平台关注点 | Feature 关注点 |
| --- | --- | --- | --- |
| schema/纯函数 | Vitest | URL、安全 helper、错误基类 | 业务 schema、映射和 query key |
| 组件 | Vitest + Testing Library | provider、共享 UI、可访问性 | 表单、表格和业务状态 |
| Server Route | Vitest + MSW | Cookie、CSRF、header、超时、响应边界 | 固定路径映射和 domain error |
| 路由集成 | Vitest | loader/context/error boundary | feature loader、失效和导航 |
| 契约漂移 | Node 脚本 + Vitest | 契约工具可失败 | 当前 feature 所需 operation/schema |
| 端到端 | Playwright | shell、会话边界、响应式 | 每个业务主路径和异常路径 |
| 生产构建 | Vite build + Node smoke | route tree、Server Route、静态资源、环境变量 | feature 产物不泄漏服务端信息 |

平台测试不依赖具体业务账号或字段。业务 E2E 的数据准备、接口健康和验收场景由对应实施计划定义。

### 10.2 覆盖率与测试原则

- 遵循 Red-Green-Refactor；
- 对 HTTP 边界优先使用真实 `Request`/`Response` 和 MSW，避免过度 mock 实现细节；
- Cookie、CSRF、重定向、敏感信息过滤等安全关键 helper 要求分支全覆盖；
- 其余新增前端代码行/语句覆盖率下限为 80%，分支下限为 75%；
- 覆盖率不能替代关键行为断言；flaky 测试不能靠无限重试掩盖。

## 11. 工程命令与质量门禁

`web/package.json` 至少提供：

```json
{
  "scripts": {
    "dev": "node scripts/dev.mjs",
    "build": "vite build && tsc --noEmit",
    "start": "node .output/server/index.mjs",
    "typecheck": "tsc --noEmit",
    "lint": "oxlint .",
    "format": "oxfmt .",
    "format:check": "oxfmt --check .",
    "test": "vitest run",
    "test:watch": "vitest",
    "test:coverage": "vitest run --coverage",
    "test:e2e": "playwright test",
    "api:check": "node scripts/check-api-contract.mjs",
    "dependencies:check": "node scripts/check-dependency-policy.mjs",
    "boundaries:check": "node scripts/check-boundaries.mjs",
    "check": "pnpm format:check && pnpm lint && pnpm dependencies:check && pnpm boundaries:check && pnpm typecheck && pnpm api:check && pnpm test",
    "browser-bundle:check": "node scripts/check-browser-bundle.mjs",
    "production:smoke": "node scripts/check-production-smoke.mjs",
    "check:release": "pnpm check && pnpm test:coverage && pnpm build && pnpm browser-bundle:check && pnpm production:smoke && pnpm test:e2e"
  }
}
```

具体参数以锁定版本的 `--help` 为准。格式化只使用 Oxfmt，不并列引入 Prettier。

Husky pre-commit 只运行 staged 快速检查：

```text
*.{ts,tsx,js,mjs,cjs,json,css,md} -> oxfmt --write
*.{ts,tsx,js,mjs,cjs}             -> oxlint
```

完整门禁放在 CI，CI job 的工作目录固定为 `web`（或先从仓库根目录执行 `cd web`）：

```text
PNPM_CONFIG_DANGEROUSLY_ALLOW_ALL_BUILDS=true pnpm install --frozen-lockfile
  -> pnpm format:check
  -> pnpm lint
  -> pnpm dependencies:check
  -> pnpm boundaries:check
  -> pnpm typecheck
  -> pnpm api:check
  -> pnpm test:coverage
  -> pnpm build
  -> pnpm browser-bundle:check
  -> pnpm production:smoke
  -> pnpm test:e2e
```

PR 至少执行到 build；涉及 session、BFF、路由、依赖或部署的变更必须运行相应 E2E。主分支和发布流水线执行完整 `check:release`。

CI 在应用级 release gate 之后构建 `web/Dockerfile`，以非 root 用户启动容器，并强制 `--read-only` 与受限 `/tmp` tmpfs；健康检查通过后还需用 `docker inspect` 断言运行用户和 `ReadonlyRootfs`。容器验证不塞进本地 `pnpm check`，避免把 Docker 变成编辑阶段的硬依赖，但主分支和发布流水线不得跳过。

## 12. 构建、部署与环境变量

### 12.1 TanStack Start Node 构建

Vite 配置包含 Tailwind、TanStack Start、`nitroV2Plugin()` 和 React plugin。由于 Nitro 2 Vite plugin 仍标记为 Experimental，插件顺序以工程基建阶段通过的生产构建组合为准并由 smoke test 锁定，不把未经验证的顺序视为框架保证。

在 `web` 目录执行的生产启动命令固定为：

```bash
node .output/server/index.mjs
```

准入测试必须验证 `.output/server/index.mjs`、`.output/public`、自定义 `server.ts`、环境变量缺失时的启动失败、Server Route、Cookie、静态资源和 SIGTERM。Nitro 启动插件在进程绑定端口前调用统一环境 schema；`server.ts` 在请求边界再次读取同一 schema，不维护第二套配置规则。失败时阻断后续 feature 接入并记录替代部署 ADR。

### 12.2 环境变量边界

- 服务端变量不使用 `VITE_` 前缀，并在启动时用 Zod fail-fast 校验；
- `pnpm dev` 通过开发启动器读取 `.env*`，按“进程环境变量 > `.env*` > 本地默认值”的优先级解析配置；仅在前两者缺失时，为本机后端 `http://127.0.0.1:5558`、实际 Vite 端口对应的 Console Origin、请求超时和开发 session secret 提供非生产默认值；
- 若本机已有其他服务占用 `5558`，不得终止或复用该服务；在未提交的 `web/.env.local` 中将 `BACKEND_API_URL` 指向当前仓库 Go API 的隔离端口（例如 `http://127.0.0.1:5559`），并重启 `pnpm dev`；
- `pnpm start` 和生产构建产物不读取上述开发默认值，缺少必需服务端变量时仍必须在监听端口前 fail fast；
- `VITE_*` 只包含允许公开到浏览器的展示配置；
- 后端地址、session secret、service token 和基础设施凭据不得进入浏览器 bundle；
- 环境变量 schema 归 infrastructure，具体后端变量名由接入文档定义；
- Cookie 安全属性从可信运行环境和公开 Origin 推导，不能提供生产环境关闭安全属性的便捷开关。

### 12.3 容器与发布

运行镜像只复制生产依赖、Start/Vite 输出和实际静态资源，使用非 root 用户、只读文件系统、健康检查和显式终止超时。发布需要证明：

- Console 自身存活与上游依赖异常可区分；
- 新旧版本的 session/Cookie 契约可回滚；
- 日志和构建产物不包含 secret、token 或内部连接信息；
- 反向代理保持公开 Origin、转发协议和可信 header 语义一致。

推荐同域部署 Console 页面和 BFF。跨域、多子域、CDN/Edge BFF 或不同 runtime 必须单独记录 ADR 并重新评审 Cookie、CORS、CSRF 和私网访问。

## 13. 基建实施阶段

业务 feature 开发之前先完成以下阶段：

```text
F0 单应用依赖与工程脚本
  -> F1 TanStack Start 运行时与空 Console shell
       -> F2 基础设施端口与 feature 模板
            -> F3 测试、质量和契约工具
                 -> F4 生产构建与部署准入
```

### 13.1 F0：单应用依赖与工程脚本

- 在 `web` 建立单一 `package.json`、`pnpm-lock.yaml`、`.node-version` 和工程脚本；
- 不创建 `pnpm-workspace.yaml`、仓库根级 package.json 或多包委托脚本；
- 锁定第 4 节直接依赖，不主动引入 beta/nightly 版本，并对 Nitro/runtime 关键传递依赖做预发布扫描；
- 配置 TypeScript strict、oxlint、oxfmt、Husky 和 lint-staged。

### 13.2 F1：运行时与 Shell

- 建立 `router.tsx`、`providers.tsx`、`start.ts`、`server.ts` 和生成 route tree；
- 固定 `defaultSsr: false`，建立 public/console pathless layout；
- 建立空 Console shell、全局 loading、error、not-found、unauthorized 和 forbidden 状态；公开路由布局在首个公开 feature 接入时创建；
- 建立设计 token、focus ring、reduced-motion 和响应式布局，不填充业务假数据。

### 13.3 F2：基础设施端口与 Feature 模板

- 建立 credential-neutral server/browser client、request policy、session port 和 request context，并用 fixture backend 验证 integration 端口；
- 建立 query client、Router/nuqs 所有权规则、通用表单反馈和表格状态组件；
- 建立 `<feature>` 模板和 `check-boundaries.mjs`，示例不绑定真实业务 API；
- 建立 server-only 防护和浏览器 bundle 敏感变量扫描。

### 13.4 F3：质量与契约工具

- 配置 Vitest、Testing Library、MSW、Playwright 和覆盖率；
- 建立可由 feature 提供 operation manifest 的 `check-api-contract.mjs`，平台工具本身不硬编码业务 operation；空 Console 阶段可以不启用 `api:check`，首个后端 integration 接入后，所有相关 feature 都必须提交非空 manifest；
- 建立 CI 快速门禁和 release 门禁；
- 通过故意删除 manifest 必需项的 fixture 证明契约检查能 fail closed。

### 13.5 F4：生产部署准入

- 构建并启动 `.output/server/index.mjs`；
- 验证 Server Route、Cookie、静态资源、环境变量缺失和终止信号；
- 交付非 root、多阶段、只读文件系统的容器镜像；
- 若 Nitro 2 adapter 与当前 Start/Vite 不兼容，阻断业务接入并先完成部署 ADR。

## 14. 基建完成定义

前端平台基建只有同时满足以下条件才允许业务 feature 接入：

- 全新 checkout 可以使用固定 Node/pnpm 执行 frozen install、check、build 和 start；
- `web` 应用根只存在一份 `pnpm-lock.yaml`，Console 可独立构建和部署；
- 依赖方向有文档和自动化约束，示例 feature 不包含真实业务契约；
- transport、session、Router、Query、Form 和 Table 的 owner 边界明确；
- `server.ts`、`start.ts`、Server Route 和浏览器 bundle 的安全边界有测试；
- 缺少服务端环境变量时 fail fast，服务端配置不会进入浏览器 bundle；
- Console shell 在 375px、768px、1280px 和 1440px 无溢出、遮挡或布局跳动；
- 键盘、焦点、错误状态和 reduced-motion 基线通过；
- `pnpm format:check`、`pnpm lint`、`pnpm dependencies:check`、`pnpm boundaries:check`、`pnpm typecheck`、`pnpm test:coverage` 和 `pnpm build` 全部通过；
- 生产 smoke test 证明环境变量缺失时 fail fast，且 Node 输出、静态资源、Server Route、Cookie 和 SIGTERM 可用；
- 未完成项、adapter 风险和门禁豁免均有责任人、到期时间和 ADR，不以业务代码绕过基建缺口。

## 15. ADR 触发条件

以下变化必须记录 ADR：

- 为任一路由启用 SSR、预渲染或服务端数据直连；
- 浏览器绕过同源 BFF，或引入 catch-all proxy；
- 改为跨域、多子域、Edge runtime 或 CDN BFF；
- 引入第二个全局状态容器、第二套路由或表单框架；
- 抽取 `packages/ui`、`packages/api` 或微前端；
- 更换 Node 构建 adapter，或引入 beta/nightly runtime 依赖；
- 持久化私有 Query cache；
- 放宽 Cookie、CSRF、Origin、CSP 或敏感信息规则；
- major 依赖升级导致运行时、构建或 peer contract 变化。

ADR 至少记录背景、备选方案、决策、后果、迁移和回滚策略。架构基建保持最小、显式和可验证，不为未出现的业务场景预留复杂抽象。
