# 数据库迁移、升级和镜像交付

## 使用官方 Goose

模板直接使用 Goose v3.28.0 CLI，SQL 位于 `migrations/`，唯一版本历史是 `goose_db_version`。Goose 使用标准连接配置，不读取应用的 `configs.yaml`。

```bash
make tools.goose                 # make deps 也会安装，产物为 _output/tools/goose
export PATH="$PWD/_output/tools:$PATH"
export GOOSE_DRIVER=postgres
# 仅用于仓库本地开发容器；生产连接由 secret manager 注入
export GOOSE_DBSTRING='postgres://postgres:dev-only-postgres@127.0.0.1:54321/enterprise_template?sslmode=disable'
goose -dir migrations up
goose -dir migrations status
goose -dir migrations version
```

创建迁移使用官方命令：

```bash
goose -dir migrations -s create add_example sql
# 编辑生成的 SQL，然后更新应用启动检查需要的版本列表
make generate
goose -dir migrations up
```

SQL 使用 `-- +goose Up`；PL/pgSQL 使用 `StatementBegin/End`。已发布文件保持不可变，后续变更追加新文件。`configs/schema.sql` 只用于参考，Compose 不会自动导入。模板不提供旧库 baseline 接管、第二套账本或 checksum 扩展；采用模板时以空库初始化，存量业务数据库按实际结构制定迁移计划。

API、Worker、Scheduler 启动只读取 Goose 历史并匹配当前构建所需版本，不执行迁移，也不需要 Goose 库或 SQL 文件。检查使用每个版本的最后一条记录，能识别回滚及重新应用；不会校验 SQL checksum 或实际表结构。严格版本匹配不支持任意新旧 schema 混跑，滚动发布需要审核兼容窗口。首次管理员授权由运维机器上的 `scripts/bootstrap-admin.sh` 完成，依赖 `psql`、仓库中的迁移文件和标准 `PG*` 连接配置。

## 容器运行

镜像包含 `/app/goose` 和 `/app/migrations/`，迁移直接调用官方工具：

```bash
# GOOSE_DBSTRING 通过环境注入；数据库主机名须在容器网络内可达
docker run --rm --network <database-network> \
  -e GOOSE_DRIVER=postgres -e GOOSE_DBSTRING \
  --entrypoint /app/goose "$APP_IMAGE" -env=none -dir /app/migrations up
```

Compose 使用相同镜像中的 Goose：

```bash
# 数据库地址使用 Compose 服务名 postgres:5432，生产凭据由外部注入
export GOOSE_DBSTRING='postgres://postgres:dev-only-postgres@postgres:5432/enterprise_template?sslmode=disable'
docker compose -f docker-compose.env.yml -f docker-compose.app.yml run --rm goose
```

同一数据库的迁移由一个发布任务串行执行。当前固定版本 CLI 未配置迁移锁，部署平台应将同库发布设为互斥；不要让多个副本同时运行 `up`。`-env=none` 避免容器内读取隐式 `.env`，只使用注入配置。

## 质量和发布

Quality Gate 包含格式、vet、race+覆盖率、增量 lint、Proto/OpenAPI/Wire/版本列表生成一致性、构建、漏洞与 secret scan，以及真实 PostgreSQL/Redis/Kafka 集成和镜像冒烟。

image job 构建一次并记录 image ID、完整 commit 和 tar 的 SHA256，作为 Actions artifact 保存。integration job 下载同一产物，执行官方 Goose、API、Worker、Scheduler 和报表冒烟。发布工作流核验成功的 release push quality run 与完整 commit，校验 artifact 和 OCI revision 后发布同一镜像，记录 digest；不补建产物。

部署顺序：

1. 保存数据库备份、当前 digest/config 和目标 release manifest。
2. 在串行发布步骤使用目标镜像运行 `goose up`，查看 `status`。
3. 用相同 digest 启动 API、Worker、Scheduler，检查各自 `/readyz`。
4. 检查任务、计划、通知及队列积压。

后续数据库变更采用 expand→兼容读写→迁移数据→contract。当前初始化 SQL 未提供 Down 段；回滚应用前确认数据库兼容，不兼容时按经过演练的备份恢复方案处理。恢复会丢失备份后的写入，需对账。

## 验证

`make integration.test` 使用官方 CLI 在每个测试的临时数据库执行初始化和重复迁移，并检查任务/计划/权限行为。`TestIntegrationGooseHistory` 验证无自定义账本、缺失版本、未来版本、回滚、重新应用及未初始化数据库。`TestIntegrationBootstrapAdminScript` 验证 Shell 初始化脚本的授权与审计、重复拒绝、账号/版本校验、SQL 字面量处理、失败回滚和并发互斥。测试需要 `TEST_POSTGRES_DSN` 和可从 `PATH` 调用的 `psql`，通过 Makefile 运行会自动准备固定版本 Goose。

`make verify.build` 检查三个应用二进制均不嵌入迁移 SQL，也不依赖 Goose 库。`scripts/image-smoke.sh` 用待发布的同一镜像执行官方 Goose 后启动三个长期进程。历史验证记录不能替代当前改动的实际验证结果。
