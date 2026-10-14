# Integration 模板

Integration 只隔离具体后端协议，不承载 feature 的业务用例。按真实需要建立：

```text
integrations/<backend>/
├── client.ts             # 注入 base URL、session port 和通用 ServerHttpClient
├── contracts.ts          # 后端 envelope、reason 与协议级解析
├── session-adapter.ts    # 后端 credential/session 契约适配
└── index.ts              # 仅在存在稳定外部消费者时提供
```

规则：

- 可以依赖 `infrastructure/` 和 `shared/`，不能导入 `features/` 或 `routes/`；
- 不包含某个 feature 的表单 schema、query key、mutation key 或页面错误文案；
- 后端地址、secret 和 credential 由组合根注入，不能在 integration 内硬编码；
- envelope 解析必须 fail closed，不能把未知结构强制断言为成功响应；
- 只有出现第二个真实后端时才新增 integration，不能为假设场景预建 adapter。

复制完成后运行 integration 契约测试、`pnpm boundaries:check` 和 `pnpm typecheck`。
