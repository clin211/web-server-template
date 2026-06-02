# RBAC 权限管理系统

## 一、功能概述

为通用后台管理 skeleton 构建完整的 RBAC（基于角色的访问控制）权限管理体系，支持菜单级和按钮级权限控制，提供可视化的权限管理后台。基于现有 Gin + GORM + PostgreSQL 技术栈，前端可按需接入任意管理后台框架，实现前后端无缝集成。

## 二、背景

当前系统已有 Casbin 基础设施，但缺乏完整的权限管理功能：

- 用户模型缺少角色关联（user 表 user_id 为 UUID）
- 缺少角色、权限、菜单管理表
- 缺少权限管理 API 和管理后台
- 无法实现细粒度的权限控制
- Casbin 现有策略为空，无需迁移，直接从新 permission 表同步策略

## 三、目标

### 3.1 核心目标

- 实现基于角色的访问控制（RBAC）
- 支持菜单/页面级和操作/按钮级的两级权限控制
- 提供可视化的权限管理后台（集成 vue-pure-admin 的树形组件和拖拽功能）
- 前后端双重权限验证（前端动态渲染，后端 Casbin 中间件）
- 支持与 SoybeanAdmin 前端的 Elegant Router 机制集成

### 3.2 预设角色

| 角色编码      | 角色名称   | 说明             |
|---------------|------------|------------------|
| `super_admin` | 超级管理员 | 系统最高权限     |
| `admin`       | 管理员     | 常规管理权限     |
| `operations`  | 运营人员   | 运营相关操作权限 |

> **说明**：系统当前预设 **3 个角色**。`super_admin` 拥有全权限，`admin` 管理核心模块，`operations` 处理运营业务。

**权限分配草图**（基于系统模块初步规划，后续可迭代）：

- `super_admin`：所有菜单、权限、用户角色全访问（包括权限管理模块本身）。
- `admin`：系统核心模块（用户管理、角色管理、菜单管理），但无权限管理模块访问。
- `operations`：运营模块（内容发布、数据统计），菜单/按钮限于读写操作。

权限分配将在初始化数据阶段通过脚本实现，支持后续可视化调整。

#### 3.2.1 角色权限策略配置

**权限策略实现说明**：

本系统使用 Casbin 实现权限控制，策略存储在 `casbin_rule` 表中。权限策略分为两类：

- **g 规则（角色继承）**：定义用户与角色的关联关系
- **p 规则（权限策略）**：定义角色对资源的访问权限

**权限策略格式**：

```sql
p, role::角色编码, 资源路径, HTTP方法, 效果
g, 用户ID, role::角色编码
```

**各角色权限详细配置**：

| 角色 | 策略数量 | 权限范围 | 说明 |
|------|---------|---------|------|
| `super_admin` | 5 条 | `/*` 全路径 | 系统最高权限，可访问所有接口 |
| `admin` | 30 条 | 用户/角色/权限/菜单管理 | 常规管理权限，含菜单角色管理 |
| `operations` | 37 条 | 文章/任务/证据块/项目管理 | 运营相关操作权限 |

**super_admin 权限策略**：

```sql
-- 全部接口访问权限
p, role::super_admin, /*, GET, allow
p, role::super_admin, /*, POST, allow
p, role::super_admin, /*, PUT, allow
p, role::super_admin, /*, PATCH, allow
p, role::super_admin, /*, DELETE, allow
```

**admin 权限策略**：

```sql
-- 用户管理
p, role::admin, /v1/users, GET, allow
p, role::admin, /v1/users, POST, allow
p, role::admin, /v1/users/*, GET, allow
p, role::admin, /v1/users/*, PUT, allow
p, role::admin, /v1/users/*, DELETE, allow
p, role::admin, /v1/users/*/roles, GET, allow
p, role::admin, /v1/users/*/roles, PUT, allow
p, role::admin, /v1/users/*/roles, POST, allow
p, role::admin, /v1/users/*/roles/*, DELETE, allow

-- 用户菜单
p, role::admin, /v1/users/menu-tree, GET, allow

-- 角色管理
p, role::admin, /v1/roles, GET, allow
p, role::admin, /v1/roles, POST, allow
p, role::admin, /v1/roles/*, GET, allow
p, role::admin, /v1/roles/*, PATCH, allow
p, role::admin, /v1/roles/*, DELETE, allow
p, role::admin, /v1/roles/*/permissions, GET, allow
p, role::admin, /v1/roles/*/permissions, POST, allow

-- 权限管理（仅查看，不含增删改）
p, role::admin, /v1/permissions, GET, allow
p, role::admin, /v1/permissions/*, GET, allow
p, role::admin, /v1/permissions/tree, GET, allow

-- 菜单管理
p, role::admin, /v1/menus, GET, allow
p, role::admin, /v1/menus, POST, allow
p, role::admin, /v1/menus/*, GET, allow
p, role::admin, /v1/menus/*, PATCH, allow
p, role::admin, /v1/menus/*, DELETE, allow
p, role::admin, /v1/menus/tree, GET, allow
p, role::admin, /v1/menus/*/roles, GET, allow
p, role::admin, /v1/menus/*/roles, PUT, allow
p, role::admin, /v1/menus/*/roles, POST, allow
p, role::admin, /v1/menus/*/roles/*, DELETE, allow
```

**operations 权限策略**：

```sql
-- 定时任务管理
p, role::operations, /v1/scheduled-tasks, GET, allow
p, role::operations, /v1/scheduled-tasks, POST, allow
p, role::operations, /v1/scheduled-tasks/:scheduledTaskID, GET, allow
p, role::operations, /v1/scheduled-tasks/:scheduledTaskID, PUT, allow
p, role::operations, /v1/scheduled-tasks/:scheduledTaskID, DELETE, allow
p, role::operations, /v1/scheduled-tasks/:scheduledTaskID/toggle, PUT, allow
p, role::operations, /v1/scheduled-tasks/:scheduledTaskID/trigger, POST, allow
p, role::operations, /v1/scheduled-tasks/:scheduledTaskID/executions, GET, allow

-- 用户菜单（运营人员可查看菜单）
p, role::operations, /v1/users/menu-tree, GET, allow
```

#### 3.2.2 角色数据库配置

**角色表 (role) 初始数据**：

> **注意**：`role_id` 使用 PostgreSQL 的 `gen_random_uuid()` 函数自动生成，无需手动指定。

| role_code | role_name | description | status | sort_order |
|-----------|-----------|-------------|--------|------------|
| `super_admin` | 超级管理员 | 系统最高权限，拥有所有操作权限 | 0 | 1 |
| `admin` | 管理员 | 常规管理权限，管理用户和内容 | 0 | 2 |
| `operations` | 运营人员 | 运营相关操作权限 | 0 | 3 |

**初始化 SQL 脚本**：

```sql
-- 创建预设角色（role_id 自动生成）
INSERT INTO role (role_id, role_code, role_name, description, status, sort_order) VALUES
(gen_random_uuid(), 'super_admin', '超级管理员', '系统最高权限，拥有所有操作权限', 0, 1),
(gen_random_uuid(), 'admin', '管理员', '常规管理权限，管理用户和内容', 0, 2),
(gen_random_uuid(), 'operations', '运营人员', '运营相关操作权限', 0, 3);

-- 添加权限策略到 casbin_rule 表
-- super_admin 全权限
INSERT INTO casbin_rule (ptype, v0, v1, v2, v3) VALUES
('p', 'role::super_admin', '/*', 'GET', 'allow'),
('p', 'role::super_admin', '/*', 'POST', 'allow'),
('p', 'role::super_admin', '/*', 'PUT', 'allow'),
('p', 'role::super_admin', '/*', 'PATCH', 'allow'),
('p', 'role::super_admin', '/*', 'DELETE', 'allow');

-- 为现有用户分配角色（示例）
-- g, user_id, role::role_code
INSERT INTO casbin_rule (ptype, v0, v1) VALUES
('g', '2c2004b4-046d-4261-973c-7ac09b07c642', 'role::super_admin');
```

#### 3.2.3 新增接口权限添加流程

当开发新功能需要添加新的 API 接口时，需要同步更新权限配置。以下是标准操作流程：

**操作步骤**：

1. **确定接口所属模块**
   - 用户管理模块 → `admin` 角色
   - 内容/运营模块 → `operations` 角色
   - 系统管理模块 → `super_admin` 角色（无需手动添加）

2. **添加权限策略到 `casbin_rule` 表**

   ```sql
   -- 格式：ptype, v0=角色, v1=路径, v2=方法, v3=效果
   INSERT INTO casbin_rule (ptype, v0, v1, v2, v3) VALUES
   ('p', 'role::角色编码', '/v1/新接口路径', 'HTTP方法', 'allow');
   ```

3. **示例：新增用户导出接口**

   ```sql
   -- 确定：用户管理模块 → admin 角色
   -- 接口：GET /v1/users/export

   INSERT INTO casbin_rule (ptype, v0, v1, v2, v3) VALUES
   ('p', 'role::admin', '/v1/users/export', 'GET', 'allow');

   -- 如果 operations 角色也需要访问
   INSERT INTO casbin_rule (ptype, v0, v1, v2, v3) VALUES
   ('p', 'role::operations', '/v1/users/export', 'GET', 'allow');
   ```

4. **验证权限是否生效**

   ```sql
   -- 查看特定角色的权限
   SELECT * FROM casbin_rule
   WHERE ptype = 'p' AND v0 = 'role::admin' AND v1 = '/v1/users/export';
   ```

5. **更新文档**

   - 在本文档对应角色权限列表中添加新接口
   - 更新 `configs/permissions_init.sql` 脚本

**快速添加脚本模板**：

```bash
# 方式一：直接执行 SQL
docker exec infra_postgres psql -U postgres -d enterprise_template -c "
INSERT INTO casbin_rule (ptype, v0, v1, v2, v3) VALUES
('p', 'role::admin', '/v1/new-endpoint', 'GET', 'allow'),
('p', 'role::admin', '/v1/new-endpoint', 'POST', 'allow');
"

# 方式二：使用初始化脚本
docker exec infra_postgres psql -U postgres -d template -f configs/permissions_init.sql
```

**注意事项**：

- `super_admin` 角色拥有 `/*` 全权限，无需为该角色添加新接口
- 路径通配符规则：`/v1/users/*` 可匹配 `/v1/users/123`、`/v1/users/abc`
- HTTP 方法不区分大小写，但建议统一使用大写（GET/POST/PUT/PATCH/DELETE）
- 权限变更会自动生效（Casbin SyncedEnforcer 每 10 秒自动加载）

**权限管理方式对比**：

| 方式 | 命令 | 适用场景 |
|------|------|---------|
| 直接 SQL | `docker exec infra_postgres psql ...` | 快速添加、开发测试 |
| 初始化脚本 | `psql -U postgres -d enterprise_template -f scripts/init_roles_permissions.sql` | 批量添加、版本控制 |
| 管理 API | `POST /v1/permissions` | 生产环境、可视化操作 |

## 四、需求详情

### 4.1 功能性需求

#### 4.1.1 角色管理

- 支持角色的增删改查（支持分页、关键字搜索）
- 支持角色启用/禁用
- 支持给角色分配权限（追加/覆盖模式）
- 支持查看角色的权限列表（树形展示）

#### 4.1.2 权限管理

- 支持权限的增删改查
- 支持权限树形展示（父子关系，支持懒加载）
- 支持按资源类型筛选（菜单/按钮）
- 支持权限启用/禁用

#### 4.1.3 菜单管理

- 支持菜单的增删改查
- 支持菜单树形展示（多级目录）
- 支持拖拽排序（前端 vue-pure-admin 集成）
- 支持菜单关联权限
- 支持菜单图标、路由、组件配置（兼容 SoybeanAdmin 路由规范）
- 支持国际化（i18nKey）
- 支持菜单级角色控制（menu_role 关联表）

#### 4.1.4 用户角色管理

- 支持给用户分配角色（多角色）
- 支持查看用户的角色和权限（扁平化权限列表）
- 支持移除用户角色

#### 4.1.5 权限验证

- **前端验证**：根据用户权限动态显示/隐藏菜单和按钮（vue-pure-admin 的权限指令 `v-hasPerm`）
- **后端验证**：通过 Casbin 中间件验证 API 访问权限（Gin 插件集成）

### 4.2 非功能性需求

- 权限检查性能：< 10ms（角色/权限数 ~10，低并发内部使用，Redis 缓存用户角色）
- 支持权限变更实时生效（Casbin SyncedEnforcer 自动加载策略）
- 数据库表支持高并发查询（优化索引，GORM 连接池）
- 安全：API 统一错误码，拒绝访问记录日志（slog）

### 4.3 约束条件

- **单租户系统**：暂不支持多租户数据隔离
- **无数据级权限**：暂不支持行级数据权限控制
- **无组织架构**：暂不支持部门/团队概念
- **前端集成**：基于 vue-pure-admin，权限代码匹配按钮 `data-code` 属性
- **数据校验**：枚举值（如 status、resource_type）和业务联动（如菜单禁用时权限禁用）在应用层（GORM 模型验证 + Biz 逻辑）实现，避免数据库层约束

## 五、技术方案

### 5.1 技术架构

- **权限框架**：Casbin v2（现有基础设施，直接扩展）
- **模型扩展**：支持角色继承 (g) 和权限继承 (g2)，使用 SyncedPolicyAdapter 实时同步
- **数据库**：PostgreSQL (GORM v2)
- **API 协议**：RESTful (Gin) + gRPC (Protobuf，可选扩展）
- **缓存**：Redis (用户角色缓存，TTL 5min)
- **日志**：slog (权限拒绝审计)
- **前端集成**：支持 SoybeanAdmin 的 Elegant Router 机制

### 5.2 错误码规范

错误码遵循统一规范，格式为 `LLMMMNNN`（8位数字），参考 `@./01 user.md`：

| 级别 (LL) | 含义 | HTTP状态码 |
|-----------|------|-----------|
| 1 | 系统级错误 | 500 |
| 2 | 用户操作错误 | 200 |
| 3 | 业务逻辑错误 | 200 |
| 4 | 上游服务错误 | 502 |
| 5 | 下游服务错误 | 502 |

**模块代码 (MMM)**：

| 模块 | 代码 |
|------|------|
| 用户模块 | 001 |
| 角色模块 | 002 |
| 权限模块 | 003 |
| 菜单模块 | 004 |
| 认证模块 | 010 |

**当前已实现错误码**（`pkg/errorsx/errorsx.go`）：

| 错误码 | 说明 | 使用场景 |
|--------|------|----------|
| `CodeOK = 0` | 成功 | 成功响应 |
| `10101` | 内部服务器错误 | 系统异常 |
| `20101` | 用户不存在 | 用户登录/查询时 |
| `20102` | 用户已存在 | 用户创建时冲突 |
| `20107` | 权限不足 | 权限验证失败 |
| `50101` | 未认证 | Token 无效或过期 |
| `50102` | Token 无效 | Token 格式错误 |
| `50103` | Token 过期 | Token 已过期 |

**规划中错误码**（尚未实现，需后续补充）：

| 错误码 | 说明 | 使用场景 |
|--------|------|----------|
| `20201` | 角色不存在 | 角色操作时 |
| `20202` | 角色已存在 | 角色创建时冲突 |
| `20301` | 权限不存在 | 权限操作时 |
| `20401` | 菜单不存在 | 菜单操作时 |

**错误响应格式**：

```json
// 错误响应
{
  "code": 20107,
  "message": "permission denied",
  "reason": "PermissionDenied"
}

// 成功响应
{
  "code": 0,
  "message": "success",
  "data": { ... }
}
```

**错误包装规范**：

```go
// 必须使用 %w 包装底层错误
return nil, fmt.Errorf("failed to copy request: %w", err)
```

**日志记录规范**：

```go
// 错误日志
slog.ErrorContext(ctx, "Failed to copy request to model", "error", err)

// 警告日志
slog.WarnContext(ctx, "Username already exists", "username", userM.Username)

// 信息日志
slog.InfoContext(ctx, "Get users from backend storage", "count", len(users))

// 权限拒绝审计
slog.Info("permission_denied", slog.String("user_id", userID), slog.String("resource", resource), slog.String("action", action))
```

### 5.3 Casbin 模型设计

修正字段不匹配，优化 matcher 支持路径匹配。模型文件（`model.conf`）：

```
[request_definition]
r = sub, dom, obj, act

[policy_definition]
p = sub, dom, obj, act, eft

[role_definition]
g = _, _, _
g2 = _, _, _

[policy_effect]
e = some(where (p.eft == allow)) && !some(where (p.eft == deny))

[matchers]
m = g(r.sub, p.sub, r.dom) && keyMatch(r.obj, p.obj) && regexMatch(r.act, p.act)
```

**说明**：

- `g`：用户-角色关系（sub = user_id, obj = role_code）
- `g2`：角色-权限关系（sub = role_code, obj = permission_code）
- `dom`：预留 domain 用于多租户（当前为空字符串）
- `keyMatch`：支持资源路径匹配（e.g., `/user/*` 匹配 `/user/list`）
- `eft`：allow/deny，支持 deny 优先
- 策略同步：角色/权限变更时，调用 `Enforcer.LoadFilteredPolicy` 实时更新

### 5.4 权限控制流程

```
用户请求 → JWT 认证 (Gin middleware) → 从 Redis/数据库获取用户角色 → Casbin 授权检查 (Enforce(user_id, "", resource_path, action)) → 允许/拒绝 (日志拒绝)
                ↓
        前端：登录后调用 /auth/permissions 获取扁平权限列表 → vue-pure-admin 动态路由/按钮渲染
        后端：Casbin 中间件验证每个 API (e.g., c.GET("/user", casbinHandler, userListHandler))
```

**错误处理**：拒绝时返回 403 {code: "PERMISSION_DENIED", message: "无权限访问"}，记录 slog.Info("permission_denied", slog.String("user_id", user_id), slog.String("resource", resource), slog.String("action", action))。

### 5.5 双层权限控制机制

本系统实现前后端双重权限控制：

```
┌─────────────────────────────────────────────────────────────┐
│ 1. 后端菜单级权限过滤（menu_role 表）                          │
│    - 根据用户角色获取对应的菜单                               │
│    - constant=1 的菜单跳过权限过滤（常量路由）                  │
│    - API: /v1/users/menu-tree                                │
├─────────────────────────────────────────────────────────────┤
│ 2. 前端路由级权限过滤（meta.roles）                           │
│    - 根据用户角色过滤前端路由                                 │
│    - 匹配逻辑：routeRoles.some(role => userRoles.includes())  │
│    - 配合 meta.constant 实现灵活控制                          │
│    - API: /route/getUserRoutes                               │
└─────────────────────────────────────────────────────────────┘
```

**权限控制规则**：

| 条件 | 结果 |
|------|------|
| `menu_role` 无记录 | 该菜单对所有登录用户可见 |
| `menu_role` 有记录 | 只有记录中包含的角色可以访问 |
| `menu.constant = 1` | 该菜单为常量路由，不参与权限过滤 |

**菜单过滤 SQL 示例**：

```sql
-- 获取用户可见菜单的查询逻辑
SELECT m.* FROM menu m
WHERE m.status = 0                    -- 启用状态
  AND m.visible = 1                   -- 可见
  AND m.deleted_at IS NULL            -- 未删除
  AND (
    -- 条件1：无 menu_role 记录，对所有角色可见
    m.menu_id NOT IN (SELECT menu_id FROM menu_role)
    -- 条件2：有 menu_role 记录且用户拥有对应角色
    OR m.menu_id IN (
        SELECT menu_id FROM menu_role
        WHERE role_id IN (
            SELECT role_id FROM user_role WHERE user_id = ?
        )
    )
    -- 条件3：常量路由，不参与权限过滤
    OR m.constant = 1
  )
ORDER BY m.parent_id NULLS LAST, m.sort_order ASC;
```

## 六、数据库设计

### 6.1 核心表结构

#### 6.1.1 角色表 (role)

```sql
CREATE TABLE role (
    id              BIGSERIAL PRIMARY KEY,
    role_id         VARCHAR(64)  UNIQUE NOT NULL DEFAULT gen_random_uuid(),
    role_name       VARCHAR(50)  NOT NULL,
    role_code       VARCHAR(50)  UNIQUE NOT NULL,
    description     VARCHAR(200),
    status          SMALLINT     NOT NULL DEFAULT 0,
    sort_order      INT          DEFAULT 0,
    created_at      TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON COLUMN role.status IS '0=启用,1=禁用';
```

#### 6.1.2 用户角色关联表 (user_role)

```sql
CREATE TABLE user_role (
    id          BIGSERIAL PRIMARY KEY,
    user_id     VARCHAR(64) NOT NULL,
    role_id     VARCHAR(64) NOT NULL,
    assigned_at TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, role_id),
    FOREIGN KEY (user_id) REFERENCES "user"(user_id) ON DELETE CASCADE,
    FOREIGN KEY (role_id) REFERENCES role(role_id) ON DELETE CASCADE
);
```

#### 6.1.3 权限表 (permission)

```sql
CREATE TABLE permission (
    id              BIGSERIAL PRIMARY KEY,
    permission_id   VARCHAR(64)  UNIQUE NOT NULL DEFAULT gen_random_uuid(),
    permission_name VARCHAR(100) NOT NULL,
    permission_code VARCHAR(100) UNIQUE NOT NULL,
    resource_type   VARCHAR(20)  NOT NULL,
    resource_path   VARCHAR(200),  -- e.g., /system/user/list
    action          VARCHAR(20)  NOT NULL,  -- GET/POST 或 custom 如 export
    description     VARCHAR(200),
    parent_id       VARCHAR(64),
    path            VARCHAR(500),  -- 全路径 for 树形查询优化
    status          SMALLINT     NOT NULL DEFAULT 0,
    created_at      TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON COLUMN permission.action IS 'HTTP动词或自定义操作';
```

#### 6.1.4 角色权限关联表 (role_permission)

```sql
CREATE TABLE role_permission (
    id             BIGSERIAL PRIMARY KEY,
    role_id        VARCHAR(64) NOT NULL,
    permission_id  VARCHAR(64) NOT NULL,
    version        INT         DEFAULT 1,  -- 乐观锁
    created_at     TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(role_id, permission_id),
    FOREIGN KEY (role_id) REFERENCES role(role_id) ON DELETE CASCADE,
    FOREIGN KEY (permission_id) REFERENCES permission(permission_id) ON DELETE CASCADE
);
```

#### 6.1.5 菜单表 (menu)

> **注意**：与 SoybeanAdmin 前端集成的完整菜单表结构，包含国际化、本地图标、面包屑等字段。

```sql
CREATE TABLE menu (
    id              BIGSERIAL PRIMARY KEY,
    menu_id         VARCHAR(64)  UNIQUE NOT NULL DEFAULT gen_random_uuid(),
    parent_id       VARCHAR(64),                          -- 父菜单 UUID
    menu_name       VARCHAR(50)  NOT NULL,                -- 菜单名称
    menu_code       VARCHAR(50)  UNIQUE NOT NULL,         -- 菜单编码（唯一标识，作为路由 name）
    menu_type       VARCHAR(10)  NOT NULL,                -- 菜单类型: menu=目录, page=页面
    i18n_key        VARCHAR(100),                         -- 国际化 key（对应前端 meta.i18nKey）
    icon            VARCHAR(100),                         -- 图标名称（iconify 格式，如 ph:user-circle）
    local_icon      VARCHAR(100),                         -- 本地图标（可选，对应 meta.localIcon）
    icon_font_size  INT,                                  -- 图标大小（可选，对应 meta.iconFontSize）
    path            VARCHAR(200),                         -- 路由路径
    component       VARCHAR(200),                         -- 前端组件标识（如 view.system-manage_user）
    permission_id   VARCHAR(64),                          -- 关联权限 UUID
    sort_order      INT          DEFAULT 0,               -- 排序序号（对应 meta.order）
    visible         SMALLINT     NOT NULL DEFAULT 1,      -- 是否可见: 0=隐藏, 1=显示
    status          SMALLINT     NOT NULL DEFAULT 0,      -- 状态: 0=启用, 1=禁用
    constant        SMALLINT     NOT NULL DEFAULT 0,      -- 常量路由: 0=否, 1=是（不参与权限过滤）
    active_menu     VARCHAR(100),                         -- 当前激活的菜单（用于面包屑，对应 meta.activeMenu）
    hide_in_menu    SMALLINT     NOT NULL DEFAULT 0,      -- 菜单中隐藏: 0=否, 1=是（对应 meta.hideInMenu）
    keep_alive      SMALLINT     NOT NULL DEFAULT 0,      -- 页面缓存: 0=否, 1=是（对应 meta.keepAlive）
    href            VARCHAR(500),                         -- 外链地址（对应 meta.href）
    created_at      TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at      TIMESTAMP,                             -- 软删除

    CONSTRAINT fk_menu_parent FOREIGN KEY (parent_id) REFERENCES menu(menu_id) ON DELETE CASCADE,
    CONSTRAINT fk_menu_permission FOREIGN KEY (permission_id) REFERENCES permission(permission_id) ON DELETE SET NULL
);

-- 索引
CREATE INDEX idx_menu_parent_id ON menu(parent_id);
CREATE INDEX idx_menu_permission_id ON menu(permission_id);
CREATE INDEX idx_menu_path ON menu(path);
CREATE INDEX idx_menu_constant ON menu(constant);
CREATE INDEX idx_menu_hide_in_menu ON menu(hide_in_menu);
CREATE INDEX idx_menu_i18n_key ON menu(i18n_key);

COMMENT ON COLUMN menu.menu_type IS 'menu=目录, page=页面';
COMMENT ON COLUMN menu.visible IS '0=隐藏,1=显示';
COMMENT ON COLUMN menu.status IS '0=启用,1=禁用';
COMMENT ON COLUMN menu.constant IS '0=否,1=是（常量路由不参与权限过滤）';
COMMENT ON COLUMN menu.hide_in_menu IS '0=否,1=是';
COMMENT ON COLUMN menu.keep_alive IS '0=否,1=是';
```

#### 6.1.6 菜单角色关联表 (menu_role)

> **重要**：此表用于定义菜单允许访问的角色列表。配合 `constant` 字段，实现灵活的权限控制。

```sql
CREATE TABLE menu_role (
    id          BIGSERIAL PRIMARY KEY,
    menu_id     VARCHAR(64) NOT NULL,
    role_id     VARCHAR(64) NOT NULL,
    created_at  TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(menu_id, role_id),
    FOREIGN KEY (menu_id) REFERENCES menu(menu_id) ON DELETE CASCADE,
    FOREIGN KEY (role_id) REFERENCES role(role_id) ON DELETE CASCADE
);

CREATE INDEX idx_menu_role_menu_id ON menu_role(menu_id);
CREATE INDEX idx_menu_role_role_id ON menu_role(role_id);
```

**menu_role 表使用说明**：

| 场景 | menu_role 状态 | 结果 |
|------|----------------|------|
| 首页、对所有用户开放 | 无记录 | 所有登录用户可见 |
| 系统管理、仅管理员可访问 | 有记录 (admin, super_admin) | 仅指定角色可见 |
| 常量路由、登录页等 | 任意状态 + constant=1 | 所有用户可见，跳过权限过滤 |

**菜单角色设置示例**：

```sql
-- 设置用户管理菜单只有 admin 和 super_admin 可访问
INSERT INTO menu_role (menu_id, role_id) VALUES
('user-menu-id', 'admin-role-id'),
('user-menu-id', 'super-admin-role-id');

-- 移除某个角色的访问权限
DELETE FROM menu_role WHERE menu_id = 'user-menu-id' AND role_id = 'operations-role-id';

-- 查看菜单允许的角色
SELECT r.role_code, r.role_name FROM menu_role mr
JOIN role r ON mr.role_id = r.role_id
WHERE mr.menu_id = 'user-menu-id';
```

#### 6.1.7 审计日志表 (audit_log)

```sql
CREATE TABLE audit_log (
    id          BIGSERIAL PRIMARY KEY,
    user_id     VARCHAR(64) NOT NULL,
    action      VARCHAR(50) NOT NULL,  -- e.g., role_assign, permission_deny
    resource    VARCHAR(200),
    details     JSONB,  -- 变更前/后
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_audit_log_user_id ON audit_log(user_id);
CREATE INDEX idx_audit_log_created_at ON audit_log(created_at);
```

**应用层校验说明**：

- 枚举校验（如 status IN (0,1)、resource_type IN ('menu','button')、menu_type IN ('menu','page')、visible IN (0,1)）：在 GORM 模型中使用 validator 标签（e.g., `validate:"oneof=0 1"`），或 Biz 层自定义函数返回 ErrInvalidStatus。
- 业务联动（如菜单禁用时自动禁用关联权限）：在 menu Biz 的 Update 方法中，使用事务：更新 menu.status=1 → 查询 permission_id → 更新 permission.status=1 → 同步 Casbin。

### 6.2 索引设计

```sql
-- 用户角色关联
CREATE INDEX idx_user_role_user_id ON user_role(user_id);
CREATE INDEX idx_user_role_role_id ON user_role(role_id);

-- 角色权限关联
CREATE INDEX idx_role_permission_role_id ON role_permission(role_id);
CREATE INDEX idx_role_permission_permission_id ON role_permission(permission_id);

-- 权限树查询
CREATE INDEX idx_permission_parent_id ON permission(parent_id);
CREATE INDEX idx_permission_path ON permission(path);

-- 菜单树查询
CREATE INDEX idx_menu_parent_id ON menu(parent_id);
CREATE INDEX idx_menu_permission_id ON menu(permission_id);
CREATE INDEX idx_menu_path ON menu(path);
CREATE INDEX idx_menu_constant ON menu(constant);
CREATE INDEX idx_menu_hide_in_menu ON menu(hide_in_menu);
CREATE INDEX idx_menu_i18n_key ON menu(i18n_key);
```

### 6.3 字段映射说明

**菜单表字段与前端映射**：

| 后端字段 | 前端对应 | 说明 |
|---------|---------|------|
| `menu_id` | `menuID` | 业务唯一标识 |
| `parent_id` | `parentID` | 树形结构父节点 |
| `menu_name` | `menuName` | 菜单显示名称 |
| `menu_code` | `name` | 路由名称（唯一标识） |
| `menu_type` | - | menu=目录, page=页面 |
| `i18n_key` | `i18nKey` | 国际化 key（用于菜单翻译） |
| `icon` | `meta.icon` | 图标名称（iconify 格式） |
| `local_icon` | `meta.localIcon` | 本地图标（可选） |
| `icon_font_size` | `meta.iconFontSize` | 图标大小（可选） |
| `path` | `path` | 路由路径 |
| `component` | `component` | 组件标识 |
| `permission_id` | - | 权限关联 |
| `sort_order` | `meta.order` | 排序序号 |
| `visible` | - | 控制显示/隐藏 |
| `status` | - | 启用/禁用 |
| `constant` | `meta.constant` | 常量路由不参与权限过滤 |
| `active_menu` | `meta.activeMenu` | 当前激活的菜单（用于面包屑） |
| `hide_in_menu` | `meta.hideInMenu` | 在菜单中隐藏 |
| `keep_alive` | `meta.keepAlive` | 页面缓存 |
| `href` | `meta.href` | 外链地址 |

## 七、API 设计

### 7.1 设计规范

**路由组织**：

```go
// internal/apiserver/handler/handler.go
type Registrar func(v1 *gin.RouterGroup, h *Handler)
var registrars []Registrar

func Register(r Registrar) { registrars = append(registrars, r) }

func (h *Handler) InstallAll(v1 *gin.RouterGroup) {
    for _, r := range registrars { r(v1, h) }
}
```

**请求处理函数**（`pkg/core/core.go`）：

```go
// JSON 请求处理
core.HandleJSONRequest[T, R](c, handler, validators...)

// Query 参数请求处理
core.HandleQueryRequest[T, R](c, handler, validators...)

// URI 参数请求处理
core.HandleUriRequest[T, R](c, handler, validators...)

// URI + JSON 请求处理
core.HandleUriJSONRequest[T, R](c, handler, validators...)
```

**统一响应格式**（`pkg/errorsx/code.go`）：

```go
type APIResponse struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
    Reason  string      `json:"reason,omitempty"`
}

// 成功响应
errorsx.Success(data interface{}, message ...string)
// 返回: {"code": 0, "message": "success", "data": {...}}

// 错误响应
errorsx.FromBizError(err *BizError)
// 返回: {"code": <bizCode>, "message": "...", "reason": "..."}
```

**统一响应写入**（`pkg/core/core.go`）：

```go
func WriteResponse(c *gin.Context, data any, err error) {
    if err != nil {
        bizErr := errorsx.FromError(err)
        response := errorsx.FromBizError(bizErr)
        httpCode := errorsx.GetHTTPCode(bizErr.Code)
        c.JSON(httpCode, response)
        return
    }
    c.JSON(http.StatusOK, errorsx.Success(data, "success"))
}
```

### 7.2 角色管理 API

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/v1/roles` | 创建角色 |
| `GET` | `/v1/roles` | 获取角色列表（分页） |
| `GET` | `/v1/roles/:id` | 获取单个角色 |
| `PATCH` | `/v1/roles/:id` | 更新角色 |
| `DELETE` | `/v1/roles/:id` | 删除角色 |
| `GET` | `/v1/roles/:id/permissions` | 获取角色权限列表 |
| `POST` | `/v1/roles/:id/permissions` | 分配角色权限 |

**Handler 注册示例**（`internal/apiserver/handler/role.go`）：

```go
func init() {
    Register(func(v1 *gin.RouterGroup, h *Handler) {
        rg := v1.Group("/roles")
        rg.Use(h.mws...)
        rg.POST("", h.CreateRole)
        rg.GET("", h.ListRole)
        rg.GET("/:id", h.GetRole)
        rg.PATCH("/:id", h.UpdateRole)
        rg.DELETE("/:id", h.DeleteRole)
        rg.GET("/:id/permissions", h.GetRolePermissions)
        rg.PUT("/:id/permissions", h.SetRolePermissions)
        rg.POST("/:id/permissions", h.AddRolePermissions)
    })
}
```

### 7.3 权限管理 API

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/v1/permissions` | 创建权限 |
| `GET` | `/v1/permissions` | 获取权限列表（分页） |
| `GET` | `/v1/permissions/tree` | 获取权限树 |
| `GET` | `/v1/permissions/:id` | 获取单个权限 |
| `PATCH` | `/v1/permissions/:id` | 更新权限 |
| `DELETE` | `/v1/permissions/:id` | 删除权限 |

**Handler 注册示例**（`internal/apiserver/handler/permission.go`）：

```go
func init() {
    Register(func(v1 *gin.RouterGroup, h *Handler) {
        rg := v1.Group("/permissions")
        rg.Use(h.mws...)
        rg.POST("", h.CreatePermission)
        rg.GET("", h.ListPermission)
        rg.GET("/tree", h.ListPermissionTree)
        rg.GET("/:id", h.GetPermission)
        rg.PATCH("/:id", h.UpdatePermission)
        rg.DELETE("/:id", h.DeletePermission)
    })
}
```

### 7.4 菜单管理 API

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/v1/menus` | 创建菜单 |
| `GET` | `/v1/menus` | 获取菜单列表（分页） |
| `GET` | `/v1/menus/tree` | 获取菜单树 |
| `GET` | `/v1/menus/:id` | 获取单个菜单 |
| `PATCH` | `/v1/menus/:id` | 更新菜单 |
| `DELETE` | `/v1/menus/:id` | 删除菜单 |
| `PUT` | `/v1/menus/:id/sort` | 更新菜单排序 |

**Handler 注册示例**（`internal/apiserver/handler/menu.go`）：

```go
func init() {
    Register(func(v1 *gin.RouterGroup, h *Handler) {
        rg := v1.Group("/menus")
        rg.Use(h.mws...)
        rg.POST("", h.CreateMenu)
        rg.PATCH("/:id", h.UpdateMenu)
        rg.DELETE("/:id", h.DeleteMenu)
        rg.GET("/:id", h.GetMenu)
        rg.GET("", h.ListMenu)
        rg.GET("/tree", h.ListMenuTree)
        rg.PUT("/:id/sort", h.SortMenu)
    })
}
```

### 7.5 菜单角色管理 API

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/v1/menus/:id/roles` | 获取菜单允许访问的角色列表 |
| `PUT` | `/v1/menus/:id/roles` | 批量设置菜单允许的角色（覆盖模式） |
| `POST` | `/v1/menus/:id/roles` | 追加菜单允许的角色 |
| `DELETE` | `/v1/menus/:id/roles/:roleID` | 移除菜单允许的角色 |

**Handler 注册示例**（`internal/apiserver/handler/menu.go`）：

```go
func init() {
    Register(func(v1 *gin.RouterGroup, h *Handler) {
        rg := v1.Group("/menus")
        rg.Use(h.mws...)
        // ... 菜单 CRUD 路由
        rg.GET("/:id/roles", h.GetMenuRoles)
        rg.PUT("/:id/roles", h.SetMenuRoles)
        rg.POST("/:id/roles", h.AddMenuRoles)
        rg.DELETE("/:id/roles/:roleID", h.RemoveMenuRole)
    })
}
```

**响应示例**：

```json
// GET /v1/menus/:id/roles - 成功响应
{
  "code": 0,
  "message": "success",
  "data": {
    "menuId": "menu-uuid",
    "roles": [
      {
        "roleId": "admin-role-id",
        "roleCode": "admin",
        "roleName": "管理员"
      }
    ],
    "count": 1
  }
}

// 错误响应示例
{
  "code": 20401,
  "message": "menu not found",
  "reason": "MenuNotFound"
}
```

**Proto 定义**（`pkg/api/apiserver/v1/menu.proto`）：

```protobuf
message GetMenuRolesRequest {
    string menu_id = 1;
}

message GetMenuRolesResponse {
    string menu_id = 1;
    repeated Role roles = 2;
    int32 count = 3;
}

message SetMenuRolesRequest {
    string menu_id = 1;
    repeated string role_ids = 2;
}

message SetMenuRolesResponse {
    string menu_id = 1;
    repeated string role_ids = 2;
    int32 count = 3;
}

message AddMenuRolesRequest {
    string menu_id = 1;
    repeated string role_ids = 2;
}

message RemoveMenuRoleRequest {
    string menu_id = 1;
    string role_id = 2;
}
```

### 7.6 用户菜单 API

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/v1/users/menu-tree` | 获取当前用户的菜单树 |
| `GET` | `/v1/route` | 获取用户可访问路由 |
| `GET` | `/v1/route/constant` | 获取常量路由 |

**Handler 注册示例**（`internal/apiserver/handler/route.go`）：

```go
func init() {
    Register(func(v1 *gin.RouterGroup, h *Handler) {
        rg := v1.Group("/route")
        rg.Use(h.mws...)
        rg.GET("", h.GetUserRoutes)
        rg.GET("/constant", h.GetConstantRoutes)
    })
    // 用户菜单树在 user handler 中
    Register(func(v1 *gin.RouterGroup, h *Handler) {
        rg := v1.Group("/users")
        rg.Use(h.mws...)
        rg.GET("/menu-tree", h.GetUserMenuTree)
    })
}
```

**常量路由响应**（`/v1/route/constant`）：

> 常量路由不参与权限过滤，前端直接硬编码。返回的路由包括：root、not-found、403、404、500、login 等。

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "routes": [
      {
        "name": "root",
        "path": "/",
        "redirect": "/home",
        "meta": {
          "constant": true
        }
      },
      {
        "name": "login",
        "path": "/login",
        "component": "layout.blank$view.login",
        "meta": {
          "title": "login",
          "constant": true,
          "hideInMenu": true
        }
      },
      {
        "name": "not-found",
        "path": "/:pathMatch(.*)*",
        "component": "layout.blank$view.404",
        "meta": {
          "title": "not-found",
          "constant": true,
          "hideInMenu": true
        }
      }
    ]
  }
}
```

**用户菜单树响应**（`/v1/users/menu-tree`）：

> 返回当前用户可访问的完整菜单树结构。

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "menus": [
      {
        "menuId": "home-uuid",
        "parentId": null,
        "menuName": "首页",
        "menuCode": "home",
        "menuType": "page",
        "i18nKey": "route.home",
        "icon": "mdi:monitor-dashboard",
        "path": "/home",
        "component": "view.home",
        "sortOrder": 1,
        "visible": 1,
        "status": 0,
        "constant": false,
        "hideInMenu": false,
        "keepAlive": false,
        "createdAt": "2026-05-29T00:00:00Z",
        "updatedAt": "2026-05-29T00:00:00Z",
        "children": []
      },
      {
        "menuId": "system-manage-uuid",
        "parentId": null,
        "menuName": "系统管理",
        "menuCode": "system-manage",
        "menuType": "menu",
        "i18nKey": "route.system-manage",
        "icon": "ph:gear-six",
        "path": "/system-manage",
        "sortOrder": 2,
        "visible": 1,
        "status": 0,
        "constant": false,
        "hideInMenu": false,
        "keepAlive": false,
        "createdAt": "2026-05-29T00:00:00Z",
        "updatedAt": "2026-05-29T00:00:00Z",
        "children": [
          {
            "menuId": "user-uuid",
            "parentId": "system-manage-uuid",
            "menuName": "用户管理",
            "menuCode": "system-manage_user",
            "menuType": "page",
            "i18nKey": "route.system-manage_user",
            "icon": "ph:user-circle",
            "path": "/system-manage/user",
            "component": "view.system-manage_user",
            "sortOrder": 1,
            "visible": 1,
            "status": 0,
            "constant": false,
            "hideInMenu": false,
            "keepAlive": false,
            "createdAt": "2026-05-29T00:00:00Z",
            "updatedAt": "2026-05-29T00:00:00Z",
            "children": []
          }
        ]
      }
    ]
  }
}
```

### 7.7 用户角色管理 API

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/v1/users/:id/roles` | 获取用户角色列表 |
| `POST` | `/v1/users/:id/roles` | 分配用户角色 |

**Handler 注册示例**（`internal/apiserver/handler/user.go`）：

```go
func init() {
    Register(func(v1 *gin.RouterGroup, h *Handler) {
        rg := v1.Group("/users")
        rg.Use(h.mws...)
        // ... 用户 CRUD 路由
        rg.GET("/:id/roles", h.GetUserRoles)
        rg.POST("/:id/roles", h.AssignRolesToUser)
        rg.DELETE("/:id/roles/:roleID", h.RemoveRoleFromUser)
        rg.GET("/menu-tree", h.GetUserMenuTree)
    })
}
```

**Proto 定义**（`pkg/api/apiserver/v1/user_role.proto`）：

```protobuf
message GetUserRolesRequest {
    string user_id = 1;
}

message GetUserRolesResponse {
    string user_id = 1;
    repeated Role roles = 2;
    int32 count = 3;
}

message AssignRolesToUserRequest {
    string user_id = 1;
    repeated string role_ids = 2;
    string mode = 3;  // "override" 或 "append"
}

message RemoveUserRoleRequest {
    string user_id = 1;
    string role_id = 2;
}
```

### 7.8 认证 API

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/v1/auth/login` | 用户登录 |
| `PUT` | `/v1/auth/refresh-token` | 刷新令牌 |
| `GET` | `/v1/auth/permissions` | 获取用户权限列表（扁平） |

**路由注册示例**（`internal/apiserver/httpserver.go`）：

```go
v1.POST("/auth/login", hdl.Login)
v1.PUT("/auth/refresh-token", hdl.RefreshToken)

// 通过 InstallAll 注册
hdl.InstallAll(v1)  // 包含 /auth/permissions
```

### 7.9 请求/响应定义

**获取用户路由响应**（`GET /v1/route`）：

> **注意**：返回的 `routes` 必须是完整的嵌套树结构，前端直接使用 `router.addRoute()` 添加。
> `meta.roles` 用于前端权限过滤，当角色为空或包含用户角色时允许访问。

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "routes": [
      {
        "name": "home",
        "path": "/home",
        "component": "layout.base$view.home",
        "meta": {
          "title": "home",
          "i18nKey": "route.home",
          "icon": "mdi:monitor-dashboard",
          "order": 1,
          "hideInMenu": false,
          "keepAlive": false,
          "roles": []
        },
        "children": []
      },
      {
        "name": "system-manage",
        "path": "/system-manage",
        "component": "layout.base",
        "meta": {
          "title": "system-manage",
          "i18nKey": "route.system-manage",
          "icon": "ph:gear-six",
          "order": 2,
          "roles": ["super_admin", "admin"]
        },
        "children": [
          {
            "name": "system-manage_user",
            "path": "/system-manage/user",
            "component": "view.system-manage_user",
            "meta": {
              "title": "system-manage_user",
              "i18nKey": "route.system-manage_user",
              "icon": "ph:user-circle",
              "roles": ["super_admin", "admin"]
            }
          }
        ]
      }
    ],
    "home": "home"
  }
}
```

**角色过滤说明**：

- `roles: []` 表示该菜单对所有登录用户可见
- `roles: ["super_admin", "admin"]` 表示只有 super_admin 和 admin 角色可见
- 前端根据用户角色列表过滤路由：`routeRoles.some(role => userRoles.includes(role))`

**菜单角色设置请求**（批量覆盖）：

```json
{
  "roleIds": ["role-uuid-1", "role-uuid-2"]
}
```

**菜单角色设置响应**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "menuId": "menu-uuid",
    "roleIds": ["role-uuid-1", "role-uuid-2"],
    "count": 2
  }
}
```

## 八、实施计划

### 8.1 实施顺序

1. **数据库模型层** (`internal/model/`)
   - 定义 GORM 模型 (role, user_role, menu_role 等)，添加 validator 标签和钩子 (AfterUpdate 同步 Casbin + 业务校验)
   - 编写模型转换 (ToProto)，AutoMigrate + 种子脚本

2. **API 定义** (`pkg/api/v1/`)
   - 定义 Proto (可选)，Gin 路由 + Handler 骨架

3. **Store 层** (`internal/store/`)
   - GORM DAO (role.Store, permission.Store, menu.Store)，支持分页 (gorm-paginator)

4. **Biz 层** (`internal/biz/v1/`)
   - 业务逻辑 (e.g., AssignRole: 事务 + Casbin sync + 枚举校验 + 联动更新)

5. **Handler 层** (`internal/handler/v1/`)
   - Gin Handler (权限中间件: casbin.Enforce)

6. **Casbin 集成** (`pkg/auth/casbin/`)
   - 初始化 Enforcer (SyncedPostgreSQLAdapter)
   - 中间件: gin-casbin

7. **前端集成** (vue-pure-admin / SoybeanAdmin)
   - 添加权限页面 (角色/菜单 CRUD，树形 el-tree + 拖拽)
   - 权限指令插件，动态路由守卫

8. **初始化数据**
   - 种子脚本: 创建预设角色 + 分配权限 (super_admin 全权限)
   - 基础菜单 (e.g., 系统管理、运营模块)，兼容 SoybeanAdmin 格式

9. **单元测试**
   - Go: testify/table-driven，覆盖 Casbin Enforce 和 Biz 校验
   - 前端: Vitest，模拟权限渲染

### 8.2 里程碑

| 阶段 | 内容 | 交付物 |
|------|------|--------|
| Phase 1 | 数据库 + API 定义 | SQL 迁移 + Gin 路由 |
| Phase 2 | Store + Biz 层 | DAO + 业务逻辑 (含校验) |
| Phase 3 | Handler + Casbin | 完整后端 API + Swagger |
| Phase 4 | 前端集成 + 初始化 | vue-pure-admin/SoybeanAdmin 页面 + 种子数据 |
| Phase 5 | 测试 + 部署 | 测试报告 + Docker 镜像 |

## 九、验收标准

### 9.1 功能验收

- [ ] 可以创建、编辑、删除角色（分页搜索，校验枚举）
- [ ] 可以给角色分配权限（树形预览）
- [ ] 可以创建、编辑、删除权限/菜单（拖拽排序，校验联动）
- [ ] 可以给用户分配角色（多选）
- [ ] 可以查看用户的菜单树（权限过滤，vue-pure-admin / SoybeanAdmin 渲染）
- [ ] 前端动态显示菜单/按钮（v-hasPerm 测试）
- [ ] 后端 API 验证（Postman 403 测试）
- [ ] 权限变更实时生效（无延迟）
- [ ] 菜单角色管理 API 正常工作（增删改查）

### 9.2 性能验收

- [ ] 权限检查 < 10ms (ab 测试 100 QPS)
- [ ] 菜单树查询 < 100ms
- [ ] 角色列表分页支持

### 9.3 初始数据验收

- [ ] 3 个预设角色创建 + 权限分配 (super_admin 全开)
- [ ] 基础菜单初始化 (5-10 项，兼容 SoybeanAdmin)
- [ ] menu_role 关联表正确管理菜单与角色的关系

### 9.4 安全/集成验收

- [ ] SQL 注入/越权测试 (e.g., 非 admin 删角色失败)
- [ ] 应用层校验测试 (e.g., 无效 status 返回 ErrInvalidStatus)
- [ ] 审计日志记录 (查阅 /audit/logs)
- [ ] 前端权限同步 (登录后菜单刷新无旧项)

## 十、风险与依赖

### 10.1 风险

| 风险                  | 影响       | 缓解措施                  |
|-----------------------|------------|---------------------------|
| Casbin 策略同步延迟   | 变更不生效 | SyncedEnforcer + Redis 缓存 |
| 权限检查性能问题      | API 慢     | 角色缓存 + 索引优化       |
| 树形结构查询性能      | 加载慢     | path 字段 + 递归 CTE      |
| 前端集成兼容性        | 渲染异常   | vue-pure-admin / SoybeanAdmin 版本 Pin |
| 应用层校验遗漏        | 数据不一致 | 单元测试覆盖 Biz 校验     |

### 10.2 外部依赖

- PostgreSQL + GORM v2
- Casbin v2 + gin-casbin
- Redis (go-redis)
- vue-pure-admin 或 SoybeanAdmin (GitHub 最新 stable)
- slog 日志

## 十一、后续扩展

### 11.1 短期扩展

- 权限变更审计查询 (API /audit/logs)
- 角色模板 (一键分配预设权限组)
- 用户权限临时提升 (JWT claim 时效)

### 11.2 长期扩展

- 多租户 (domain 隔离)
- 数据级权限 (Casbin filter)
- 组织架构 (部门角色继承)
- ABAC 动态权限 (表达式支持)

## 十二、参考文档

- [Casbin 官方文档](https://casbin.org/docs/overview)
- [RBAC 设计最佳实践](https://en.wikipedia.org/wiki/Role-based_access_control)
- [vue-pure-admin 文档](https://github.com/pure-admin/vue-pure-admin)
- [SoybeanAdmin 文档](https://github.com/soybeanjs/soybean-admin)
- [Elegant Router 文档](https://github.com/soybeanjs/elegant-router)
- 项目 README: `@./README.md`
- 项目宪法: `@.claude/constitution.md`
- 菜单管理系统设计: `@./03 menu.md`
- 用户模块业务逻辑: `@./01 user.md`