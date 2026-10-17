### 总体评价

本次新增能力保持模块化单体和小接口，关键状态由数据库约束与事务保护，完成了实际故障路径测试；可进入人工代码审查，生产发布仍需远程门禁和环境演练。

审查范围为 `internal/job/`、`internal/job/runtime/`、任务/计划/授权 Store、报表 Biz/Handler、迁移、遥测和交付。准则为用户给出的规范及实施开始时读取的 `.claude/constitution.md`；该文件随后由主分支 `8c2adc7` 删除，本次未恢复它。

### 优点 (值得称赞的地方)

- 第一条/第四条：只引入 Cron 解析依赖；任务执行使用现有 PostgreSQL，事件使用现有 Kafka，没有通用插件、多后端或重复队列。消费者定义执行仓储接口，业务示例贯穿 HTTP→Biz→Store。
- 第二条/第三条：故障测试使用真实 PG/Kafka/Redis、synctest 和有限 fake，保留原有效断言；事务上下文、旧 fencing、参数脱敏、退出和错误分类都有实际测试证据。审核中发现并修复嵌套事务逃逸、SQL 凭据日志和空数组响应问题。

### 待改进项 (按优先级排序)

#### [高优先级] 违宪问题 (Violations of Constitution)

本次审查未发现尚未修复、足以阻止本地交付的新增违宪问题。此结论针对已检查的代码及实际测试，不能替代生产压测、环境授权核验或独立人工审查。

#### [中优先级] 建议性改进

- **位置:** `internal/apiserver/biz/v1/task/service.go`
- **问题描述:** 模板的任务、计划、报表、RBAC 和 DLQ 动作集中在一个业务文件；函数复杂度已通过原 lint 门禁，后续增加领域动作会降低阅读效率。
- **修改建议:** 扩展业务时按现有 user 模式拆成 submit/query/schedule/report/admin 动作文件，保留当前具体依赖，不先增加服务框架。

- **位置:** `internal/apiserver/pkg/metrics/tasks.go`
- **问题描述:** 缓存型聚合避免 scrape 执行 SQL，但 15 秒刷新涉及多次数据库计数；大量历史数据需要负载验证。执行计数反映本地 Handler 结果，最终持久化状态及撤权/取消应以 task_state_count、task_attempt_outcomes 为准。
- **修改建议:** 随真实数据规模评估清理、索引与聚合查询成本；告警使用数据库状态与积压，新增类型保持标签有限。不要加入 task_id/owner_id 等标签。

- **位置:** `internal/pkg/dbschema/`、`docs/guide/migrations-and-release.md`
- **问题描述:** 严格版本匹配会拒绝旧二进制连接较新数据库；迁移直接使用官方 Goose CLI，同库发布需要串行执行，模板不再提供旧库接管逻辑。
- **修改建议:** 每个后续版本审核 expand/contract 和滚动兼容窗口；存量数据库按实际结构制定迁移映射与备份恢复演练。

- **位置:** `.github/workflows/quality.yml`、`.github/workflows/deploy.yml`
- **问题描述:** 本地完整门禁及镜像闭环已通过，远程 secret scan、Actions 权限、环境审批和 registry digest 发布尚未执行。
- **修改建议:** 变更进入仓库后检查成功的完整 Quality Gate，再配置 release 环境审批并对同一 artifact 执行授权发布。远程状态必须单独记录。
