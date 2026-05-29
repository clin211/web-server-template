# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 工作约束

- 与仓库现有协作约定保持一致，默认使用简体中文。
- 这是一个 `pnpm` monorepo 的 Vue 3 管理后台前端，依赖 `pnpm-workspace.yaml` 管理 `packages/*` 工作区包；不要切换到 npm 或 yarn。

## 常用命令

```bash
pnpm i
pnpm dev               # 本地开发，Vite dev server 监听 0.0.0.0:9527 并自动打开浏览器
pnpm dev:prod          # 用 prod 环境变量本地启动
pnpm build             # 生产构建（mode=prod）
pnpm build:test        # 测试环境构建（mode=test）
pnpm preview           # 预览构建产物，端口 9725
pnpm typecheck         # vue-tsc --noEmit --skipLibCheck
pnpm lint              # oxlint --fix && eslint --fix .
pnpm fmt               # oxfmt
pnpm gen-route         # 重新生成 Elegant Router 产物
pnpm cleanup           # 清理 Soybean 生成物
pnpm commit            # 生成 Conventional Commit 提交信息
pnpm commit:zh         # 生成中文 Conventional Commit 提交信息
pnpm exec eslint src/path/to/file.ts --fix  # 单文件定向检查
```

- 提交前钩子会运行 `pnpm typecheck && pnpm lint && pnpm fmt && git diff --exit-code`；格式化或自动修复导致工作区变脏时，提交会失败。
- 当前仓库没有根级 `test` script，也未发现业务测试文件；没有“运行单个测试”的现成命令。前端改动主要依赖 `pnpm typecheck`、`pnpm lint` 和浏览器手工验证。

## 仓库分层

- `src/`: 应用源码。
- `build/`: Vite 插件与构建辅助逻辑。
- `packages/*`: 工作区基础能力，当前主要包含 `@sa/axios`、`@sa/hooks`、`@sa/materials`、`@sa/utils`、`@sa/color`、`@sa/uno-preset`、`@sa/scripts`。
- `public/`: 静态资源。

## 应用启动链路

- 入口在 `src/main.ts`：先加载 `src/plugins/assets.ts`，再依次初始化 loading、nprogress、离线图标、dayjs，随后创建 Vue app，并按顺序挂载 Pinia、Router、i18n、版本通知和根节点校验。
- 根组件在 `src/App.vue`：通过 Naive UI 的 `NConfigProvider` 注入主题与语言，再通过 `AppProvider` 暴露全局 message / dialog 等上下文；全局水印也在这里挂载。

## 路由与权限模型

- 路由生成依赖 `@elegant-router/vue`，配置入口在 `build/plugins/router.ts`。
- `src/router/elegant/routes.ts`、`src/router/elegant/imports.ts`、`src/router/elegant/transform.ts` 是生成产物；不要手改，优先修改 `src/views/**/*`、布局映射或 `src/router/routes/index.ts` 中的 `customRoutes`，再按需执行 `pnpm gen-route`。
- 运行时路由入口在 `src/router/index.ts`，内建常量路由在 `src/router/routes/builtin.ts`。
- `src/router/routes/index.ts` 会把生成路由与自定义路由合并后，拆成 constant routes 和 auth routes。
- `src/store/modules/route/index.ts` 是路由编排中心：
  - `static` 模式直接使用生成路由，并按 `meta.roles` 过滤权限；
  - `dynamic` 模式通过 `src/service/api/route.ts` 拉取常量路由和用户路由，再在运行时 `router.addRoute`；
  - 同时维护全局菜单、搜索菜单、面包屑、缓存路由与首页重定向。
- `src/router/guard/route.ts` 负责懒初始化 constant/auth routes、登录跳转、403/404 区分，以及 `meta.href` 外链处理。

## 状态管理

- Pinia 初始化在 `src/store/index.ts`，并通过 `src/store/plugins/index.ts` 注入 setup store 的重置能力。
- 主要 store 模块：
  - `app`: 语言、布局、响应式界面状态；
  - `theme`: 深色模式、主题 token、Naive UI 覆盖、水印；
  - `auth`: token、用户信息、菜单树；登录后会先从 JWT 中提取用户 ID，再请求 `/v1/users/:id` 和 `/v1/users/menu-tree`；
  - `route`: 权限路由初始化、菜单、面包屑、keep-alive 名单；
  - `tab`: 多标签页状态与缓存。

## 请求层与后端对接

- 请求统一经过 `src/service/request/index.ts`，底层封装来自工作区包 `@sa/axios`。
- `src/utils/service.ts` 负责把环境变量转换成 `baseURL` / proxy pattern；`build/config/proxy.ts` 复用这份配置生成 Vite 开发代理，因此修改服务地址时要同时考虑运行时和开发代理。
- API 按领域拆在 `src/service/api/auth.ts`、`src/service/api/route.ts`、`src/service/api/user.ts`。
- 请求层会自动注入 `Authorization`，并根据环境变量处理：后端成功码、直接登出码、弹窗登出码、token 过期刷新与失败提示。

## 构建与样式系统

- `vite.config.ts` 统一装配 `build/plugins/index.ts` 中的插件：Vue、JSX、devtools、Elegant Router、UnoCSS、自动组件/图标注册、构建进度、HTML build time 注入、根节点校验。
- `uno.config.ts` 基于 `@sa/uno-preset` 和 `src/theme/vars` 定义主题 token；全局 SCSS 通过 `vite.config.ts` 注入 `src/styles/scss/global.scss`。
- 行为上最关键的环境变量是：`VITE_AUTH_ROUTE_MODE`、`VITE_ROUTE_HOME`、`VITE_ROUTER_HISTORY_MODE`、`VITE_SERVICE_BASE_URL`、`VITE_OTHER_SERVICE_BASE_URL`、`VITE_HTTP_PROXY`、`VITE_PROXY_LOG`。

## 修改代码时的切入点

- 新增页面或调整页面路由：优先看 `src/views/**/*`、`src/layouts/**/*`、`src/router/routes/index.ts` 和 `build/plugins/router.ts`。
- 调整登录、菜单、权限、首页跳转：联动检查 `src/store/modules/auth`、`src/store/modules/route`、`src/router/guard/route.ts`、`src/service/api/route.ts`。
- 调整接口调用或鉴权行为：联动检查 `src/service/request/*`、`src/service/api/*`、`src/utils/service.ts`。
- 调整主题、国际化或全局 UI 能力：联动检查 `src/store/modules/theme`、`src/locales`、`src/theme`、`uno.config.ts`、`src/App.vue`。
