---
name: Git 提交格式
description: "应用约定式提交格式规则。在生成提交消息或创建提交时使用。"
---

# Git 提交消息格式规则

为项目中的所有 git 提交应用约定式提交格式。

## 提交消息格式

```sh
<类型>(<范围>): <描述>

[可选正文]

[脚注]
```

## 提交类型

- **feat**: 新功能
- **fix**: 缺陷修复
- **docs**: 文档变更
- **style**: 代码风格变更（格式化等）
- **refactor**: 代码重构（无功能变更）
- **test**: 添加/更新测试
- **chore**: 维护任务
- **build**: 构建系统或依赖变更
- **ci**: CI/CD 变更
- **perf**: 性能优化
- **revert**: 回滚之前的提交

## 破坏性变更

### 使用 ! 以引起注意

```sh
feat!: 产品发货时发送邮件
```

### 使用 BREAKING CHANGE 脚注

```sh
feat: 允许配置扩展其他配置

BREAKING CHANGE: `extends` 键现在用于扩展配置文件
```

### 同时使用 ! 和 BREAKING CHANGE

```sh
chore!: 移除对 Node 6 的支持

BREAKING CHANGE: 使用了 Node 6 中不可用的 JavaScript 特性。
```

## 必需的脚注

### Signed-off-by 脚注

**始终包含** 带有姓名和邮箱的 `Signed-off-by` 脚注。

按以下优先级顺序获取凭据：

1. 环境变量：`$GIT_AUTHOR_NAME` 和 `$GIT_AUTHOR_EMAIL`
2. Git 配置：`git config user.name` 和 `git config user.email`
3. 如果两者都未配置，询问用户提供详细信息

## Gitlint 验证规则

- 运行 `make run-gitlint` 来验证提交消息
- **标题行**：最多 120 个字符
- **正文行**：每行最多 140 个字符
- 使用约定式提交格式
- 包含必需的脚注（Signed-off-by）
- 无尾随空格

## 示例

### 简单提交

```sh
docs: 更正 CHANGELOG 拼写
```

### 带范围

```sh
feat(azure): 添加工作负载身份支持
```

### 多段落带脚注

```sh
fix: 防止请求竞态

- 引入请求 ID 并引用最新请求。忽略
- 来自最新请求之外的传入响应。

- 移除用于缓解竞态但现在已过时的超时设置。
```

## 快速检查清单

创建提交时：

- [ ] 使用约定式提交格式：`<类型>(<范围>): <描述>`
- [ ] 标题少于 120 个字符
- [ ] 正文行少于 140 个字符
- [ ] 使用 `make run-gitlint` 验证
- [ ] 破坏性变更使用 "!" 或 `BREAKING CHANGE`

## 参考

约定式提交规范：<https://www.conventionalcommits.org/en/v1.0.0/#specification>
