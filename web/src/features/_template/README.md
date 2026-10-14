# Feature 模板

新增业务 feature 时复制本目录约定，而不是保留空文件或示例业务数据。只创建当前 feature 实际需要的文件：

```text
features/<feature>/
├── contracts.ts                         # 业务输入、输出和 Zod schema
├── api/
│   ├── client.ts                        # 浏览器到同源 BFF 的 adapter
│   ├── queries.ts                       # query key factory 与 queryOptions
│   ├── mutations.ts                     # mutation key factory 与 mutationOptions
│   └── server.ts                        # server-only 固定上游映射
├── components/                          # 该 feature 独占的表单、表格和视图
├── <feature>.operation-manifest.json    # 当前 feature 使用的上游 operation
└── index.ts                             # 最小公开 API
```

规则：

- `routes/` 只通过 `index.ts` 组合 feature，不能深层导入内部文件；
- schema、错误映射、query/mutation identity 和缓存失效语义归当前 feature；
- feature 不能导入另一个 feature 的内部文件，跨 feature 流程由 route 组合；
- `api/server.ts` 必须包含 `@tanstack/react-start/server-only`，且只能映射显式上游路径；
- 不创建 catch-all proxy、全局业务 registry、假数据或尚未存在的业务能力；
- 不需要 query、mutation、组件或上游 API 时，删除对应可选文件。

复制完成后运行 `pnpm boundaries:check`、`pnpm api:check`、相关测试和 `pnpm typecheck`。
