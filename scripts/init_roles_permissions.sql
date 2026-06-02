-- =============================================
-- Role & Permission 完整初始化脚本
-- 初始化 3 个预设角色及其权限关联
-- 容器首次启动时自动执行
-- =============================================

BEGIN;

-- =============================================
-- Step 1: 创建预设角色（如果不存在）
-- =============================================
INSERT INTO role (role_id, role_name, role_code, description, status, sort_order)
VALUES (gen_random_uuid(), '超级管理员', 'super_admin', '系统最高权限，拥有所有操作权限', 0, 1)
ON CONFLICT (role_code) DO NOTHING;

INSERT INTO role (role_id, role_name, role_code, description, status, sort_order)
VALUES (gen_random_uuid(), '管理员', 'admin', '常规管理权限，管理用户和内容', 0, 2)
ON CONFLICT (role_code) DO NOTHING;

INSERT INTO role (role_id, role_name, role_code, description, status, sort_order)
VALUES (gen_random_uuid(), '运营人员', 'operations', '运营相关操作权限', 0, 3)
ON CONFLICT (role_code) DO NOTHING;

-- =============================================
-- Step 2: 创建基础权限（如果不存在）
-- =============================================
INSERT INTO permission (permission_id, permission_name, permission_code, resource_type, resource_path, action, status)
SELECT * FROM (
    VALUES
        -- 首页
        (gen_random_uuid(), '首页', 'dashboard', 'menu'::resource_type, '/v1/home', 'GET', 0),
        -- 用户管理
        (gen_random_uuid(), '用户管理', 'system:user', 'menu'::resource_type, '/v1/users', 'GET', 0),
        (gen_random_uuid(), '用户列表', 'system:user:list', 'button'::resource_type, '/v1/users', 'GET', 0),
        (gen_random_uuid(), '用户创建', 'system:user:create', 'button'::resource_type, '/v1/users', 'POST', 0),
        (gen_random_uuid(), '用户编辑', 'system:user:update', 'button'::resource_type, '/v1/users/*', 'PUT', 0),
        (gen_random_uuid(), '用户删除', 'system:user:delete', 'button'::resource_type, '/v1/users/*', 'DELETE', 0),
        (gen_random_uuid(), '用户角色', 'system:user:role', 'button'::resource_type, '/v1/users/*/roles', 'GET', 0),
        (gen_random_uuid(), '分配角色', 'system:user:role:assign', 'button'::resource_type, '/v1/users/*/roles', 'POST', 0),
        (gen_random_uuid(), '用户菜单', 'system:user:menu', 'button'::resource_type, '/v1/users/menu-tree', 'GET', 0),
        -- 角色管理
        (gen_random_uuid(), '角色管理', 'system:role', 'menu'::resource_type, '/v1/roles', 'GET', 0),
        (gen_random_uuid(), '角色列表', 'system:role:list', 'button'::resource_type, '/v1/roles', 'GET', 0),
        (gen_random_uuid(), '角色创建', 'system:role:create', 'button'::resource_type, '/v1/roles', 'POST', 0),
        (gen_random_uuid(), '角色编辑', 'system:role:update', 'button'::resource_type, '/v1/roles/*', 'PUT', 0),
        (gen_random_uuid(), '角色删除', 'system:role:delete', 'button'::resource_type, '/v1/roles/*', 'DELETE', 0),
        (gen_random_uuid(), '角色权限', 'system:role:permission', 'button'::resource_type, '/v1/roles/*/permissions', 'GET', 0),
        (gen_random_uuid(), '分配权限', 'system:role:permission:assign', 'button'::resource_type, '/v1/roles/*/permissions', 'POST', 0),
        -- 权限管理
        (gen_random_uuid(), '权限管理', 'system:permission', 'menu'::resource_type, '/v1/permissions', 'GET', 0),
        (gen_random_uuid(), '权限列表', 'system:permission:list', 'button'::resource_type, '/v1/permissions', 'GET', 0),
        (gen_random_uuid(), '权限树', 'system:permission:tree', 'button'::resource_type, '/v1/permissions/tree', 'GET', 0),
        -- 菜单管理
        (gen_random_uuid(), '菜单管理', 'system:menu', 'menu'::resource_type, '/v1/menus', 'GET', 0),
        (gen_random_uuid(), '菜单列表', 'system:menu:list', 'button'::resource_type, '/v1/menus', 'GET', 0),
        (gen_random_uuid(), '菜单树', 'system:menu:tree', 'button'::resource_type, '/v1/menus/tree', 'GET', 0),
        (gen_random_uuid(), '菜单创建', 'system:menu:create', 'button'::resource_type, '/v1/menus', 'POST', 0),
        (gen_random_uuid(), '菜单编辑', 'system:menu:update', 'button'::resource_type, '/v1/menus/*', 'PUT', 0),
        (gen_random_uuid(), '菜单删除', 'system:menu:delete', 'button'::resource_type, '/v1/menus/*', 'DELETE', 0),
        (gen_random_uuid(), '菜单角色', 'system:menu:role', 'button'::resource_type, '/v1/menus/*/roles', 'GET', 0),
        (gen_random_uuid(), '分配菜单角色', 'system:menu:role:assign', 'button'::resource_type, '/v1/menus/*/roles', 'POST', 0),
        -- 定时任务管理
        (gen_random_uuid(), '定时任务', 'operations:scheduled-task', 'menu'::resource_type, '/v1/scheduled-tasks', 'GET', 0),
        (gen_random_uuid(), '任务列表', 'operations:scheduled-task:list', 'button'::resource_type, '/v1/scheduled-tasks', 'GET', 0),
        (gen_random_uuid(), '任务详情', 'operations:scheduled-task:detail', 'button'::resource_type, '/v1/scheduled-tasks/*', 'GET', 0),
        (gen_random_uuid(), '任务创建', 'operations:scheduled-task:create', 'button'::resource_type, '/v1/scheduled-tasks', 'POST', 0),
        (gen_random_uuid(), '任务编辑', 'operations:scheduled-task:update', 'button'::resource_type, '/v1/scheduled-tasks/*', 'PUT', 0),
        (gen_random_uuid(), '任务删除', 'operations:scheduled-task:delete', 'button'::resource_type, '/v1/scheduled-tasks/*', 'DELETE', 0),
        (gen_random_uuid(), '任务启停', 'operations:scheduled-task:toggle', 'button'::resource_type, '/v1/scheduled-tasks/*/toggle', 'PUT', 0),
        (gen_random_uuid(), '任务触发', 'operations:scheduled-task:trigger', 'button'::resource_type, '/v1/scheduled-tasks/*/trigger', 'POST', 0),
        (gen_random_uuid(), '执行记录', 'operations:scheduled-task:execution', 'button'::resource_type, '/v1/scheduled-tasks/*/executions', 'GET', 0)
) AS t(permission_id, permission_name, permission_code, resource_type, resource_path, action, status)
ON CONFLICT (permission_code) DO NOTHING;

-- =============================================
-- Step 3: 角色权限关联（role_permission 表）
-- =============================================

-- super_admin: 全权限（关联所有启用的权限）
INSERT INTO role_permission (role_id, permission_id)
SELECT r.role_id, p.permission_id
FROM role r
CROSS JOIN permission p
WHERE r.role_code = 'super_admin'
  AND p.status = 0
  AND NOT EXISTS (
    SELECT 1 FROM role_permission rp
    WHERE rp.role_id = r.role_id AND rp.permission_id = p.permission_id
  );

-- admin: 系统管理权限
INSERT INTO role_permission (role_id, permission_id)
SELECT r.role_id, p.permission_id
FROM role r
CROSS JOIN permission p
WHERE r.role_code = 'admin'
  AND p.status = 0
  AND (
    p.permission_code LIKE 'system:%'
    OR p.permission_code = 'dashboard'
  )
  AND NOT EXISTS (
    SELECT 1 FROM role_permission rp
    WHERE rp.role_id = r.role_id AND rp.permission_id = p.permission_id
  );

-- operations: 运营权限
INSERT INTO role_permission (role_id, permission_id)
SELECT r.role_id, p.permission_id
FROM role r
CROSS JOIN permission p
WHERE r.role_code = 'operations'
  AND p.status = 0
  AND (
    p.permission_code LIKE 'operations:%'
    OR p.permission_code = 'dashboard'
  )
  AND NOT EXISTS (
    SELECT 1 FROM role_permission rp
    WHERE rp.role_id = r.role_id AND rp.permission_id = p.permission_id
  );

-- =============================================
-- Step 4: Casbin p 规则同步（角色 → 资源路径 → 方法）
-- =============================================

-- super_admin: 全路径
INSERT INTO casbin_rule (ptype, v0, v1, v2, v3)
SELECT 'p', 'role::super_admin', '/*', method, 'allow'
FROM unnest(ARRAY['GET', 'POST', 'PUT', 'PATCH', 'DELETE']) AS method
WHERE NOT EXISTS (
    SELECT 1 FROM casbin_rule cr
    WHERE cr.ptype = 'p'
      AND cr.v0 = 'role::super_admin'
      AND cr.v1 = '/*'
      AND cr.v2 = method
);

-- admin: 系统管理路径
INSERT INTO casbin_rule (ptype, v0, v1, v2, v3)
SELECT 'p', 'role::admin', p.resource_path, p.action, 'allow'
FROM permission p
WHERE p.status = 0
  AND p.resource_path LIKE '/v1/users%'
   OR p.resource_path LIKE '/v1/roles%'
   OR p.resource_path LIKE '/v1/permissions%'
   OR p.resource_path LIKE '/v1/menus%'
   OR p.resource_path = '/v1/home'
  AND NOT EXISTS (
    SELECT 1 FROM casbin_rule cr
    WHERE cr.ptype = 'p'
      AND cr.v0 = 'role::admin'
      AND cr.v1 = p.resource_path
      AND cr.v2 = p.action
  );

-- operations: 运营路径 + 用户菜单
INSERT INTO casbin_rule (ptype, v0, v1, v2, v3)
SELECT 'p', 'role::operations', p.resource_path, p.action, 'allow'
FROM permission p
WHERE p.status = 0
  AND (
    p.resource_path LIKE '/v1/scheduled-tasks%'
    OR p.resource_path = '/v1/users/menu-tree'
    OR p.resource_path = '/v1/home'
  )
  AND NOT EXISTS (
    SELECT 1 FROM casbin_rule cr
    WHERE cr.ptype = 'p'
      AND cr.v0 = 'role::operations'
      AND cr.v1 = p.resource_path
      AND cr.v2 = p.action
  );

-- =============================================
-- Step 5: 用户角色关联（user_role 表）
-- 说明：根据实际需求取消相应注释
-- =============================================

-- root 用户 → super_admin
INSERT INTO user_role (user_id, role_id)
SELECT u.user_id, r.role_id
FROM "user" u
CROSS JOIN role r
WHERE u.username = 'root'
  AND r.role_code = 'super_admin'
  AND NOT EXISTS (
    SELECT 1 FROM user_role ur
    WHERE ur.user_id = u.user_id AND ur.role_id = r.role_id
  );

-- 备用：admin 用户 → admin（如果存在）
-- INSERT INTO user_role (user_id, role_id)
-- SELECT u.user_id, r.role_id
-- FROM "user" u
-- CROSS JOIN role r
-- WHERE u.username = 'admin'
--   AND r.role_code = 'admin'
--   AND NOT EXISTS (
--     SELECT 1 FROM user_role ur
--     WHERE ur.user_id = u.user_id AND ur.role_id = r.role_id
--   );

-- =============================================
-- Step 6: Casbin g 规则同步（用户 → 角色）
-- =============================================

-- root 用户 → super_admin
INSERT INTO casbin_rule (ptype, v0, v1)
SELECT 'g', u.user_id::text, 'role::super_admin'
FROM "user" u
WHERE u.username = 'root'
  AND NOT EXISTS (
    SELECT 1 FROM casbin_rule cr
    WHERE cr.ptype = 'g'
      AND cr.v0 = u.user_id::text
      AND cr.v1 = 'role::super_admin'
  );

-- 备用：admin 用户 → admin
-- INSERT INTO casbin_rule (ptype, v0, v1)
-- SELECT 'g', u.user_id::text, 'role::admin'
-- FROM "user" u
-- WHERE u.username = 'admin'
--   AND NOT EXISTS (
--     SELECT 1 FROM casbin_rule cr
--     WHERE cr.ptype = 'g'
--       AND cr.v0 = u.user_id::text
--       AND cr.v1 = 'role::admin'
--   );

-- =============================================
-- Step 7: 菜单角色关联（menu_role 表，可选）
-- 用于控制菜单对特定角色的可见性
-- =============================================

-- 获取各角色 ID（供后续使用）
-- SELECT role_id, role_code FROM role WHERE role_code IN ('super_admin', 'admin', 'operations');

-- 示例：设置用户管理菜单只有 admin 和 super_admin 可访问
-- INSERT INTO menu_role (menu_id, role_id)
-- SELECT m.menu_id, r.role_id
-- FROM menu m
-- CROSS JOIN role r
-- WHERE m.menu_code = 'system-manage_user'
--   AND r.role_code IN ('admin', 'super_admin')
--   AND NOT EXISTS (
--     SELECT 1 FROM menu_role mr
--     WHERE mr.menu_id = m.menu_id AND mr.role_id = r.role_id
--   );

COMMIT;