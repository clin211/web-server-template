# ADR 0002：直接使用官方 Goose CLI

日期：2026-10-16。状态：采纳，替代此前的项目迁移适配方案。

## 背景与决策

这是项目模板。使用者只需理解 Goose 和 SQL 迁移，不应再学习项目专有的迁移命令、双账本、checksum 扩展和旧库接管协议。

直接使用固定版本 Goose v3.28.0 CLI 执行根目录 `migrations/` 中的 SQL，唯一版本历史为官方 `goose_db_version`。移除自定义 migrate 入口、`schema_migration` 校验值扩展、基线适配及其测试。项目不提供自动接管历史数据库的逻辑。

`make deps` 安装官方 CLI 到 `_output/tools/goose`，Docker 镜像携带同一个固定版本的 CLI 和 SQL 文件。Goose 是外部工具，业务 Go module 和三个应用二进制不依赖 Goose 库。应用启动只读官方版本历史，确认所需迁移已应用，不创建表或执行 DDL。`make generate` 从迁移文件名更新启动检查所需的版本列表，不计算 checksum。首次管理员初始化使用 `scripts/bootstrap-admin.sh`，不提供独立 Admin 应用。

## 使用和边界

开发和发布统一执行 `goose -dir migrations up`，使用标准 `GOOSE_DRIVER`、`GOOSE_DBSTRING` 连接配置。查看状态、创建迁移也使用官方命令。完整示例见[迁移和交付](../guide/migrations-and-release.md)。

迁移由部署流水线中的单个步骤串行执行，不在 API、Worker 或 Scheduler 启动时执行。当前固定版本 CLI 未配置迁移锁，因此同一数据库的部署任务必须互斥；模板不再维护自定义锁协议。已发布 SQL 保持不可变，新变更追加迁移。旧库的数据迁移由采用模板的项目按实际结构制定。

保留严格启动版本匹配；这能发现漏迁移和未知版本，但不证明实际表结构没有被手动修改，也不提供 checksum 验证。滚动发布需单独审核 expand/contract 兼容窗口。

依据：[官方 CLI](https://github.com/pressly/goose/tree/v3.28.0/cmd/goose)、[SQL 注解](https://pressly.github.io/goose/documentation/annotations/)。
