-- =============================================
-- Admin User & Role 初始化脚本
-- 用于给 root 用户建立完整的角色、权限关联
-- =============================================
-- =============================================
-- Step 1: 创建 admin 角色（如果不存在）
-- =============================================
INSERT INTO role (
    role_id,
    role_name,
    role_code,
    description,
    status,
    sort_order
  )
SELECT gen_random_uuid(),
  '管理员',
  'admin',
  '系统管理员角色，拥有所有权限',
  0,
  1
WHERE NOT EXISTS (
    SELECT 1
    FROM role
    WHERE role_code = 'admin'
  );
-- =============================================
-- Step 2: 创建基础权限（如果不存在）
-- =============================================
INSERT INTO permission (
    permission_id,
    permission_name,
    permission_code,
    resource_type,
    resource_path,
    action,
    status
  )
SELECT *
FROM (
    VALUES (
        gen_random_uuid(),
        '用户管理',
        'system:user',
        'menu',
        '/manage/user',
        'GET',
        0
      ),
      (
        gen_random_uuid(),
        '用户列表',
        'system:user:list',
        'button',
        '/manage/user',
        'GET',
        0
      ),
      (
        gen_random_uuid(),
        '用户创建',
        'system:user:create',
        'button',
        '/manage/user',
        'POST',
        0
      ),
      (
        gen_random_uuid(),
        '用户编辑',
        'system:user:update',
        'button',
        '/manage/user',
        'PUT',
        0
      ),
      (
        gen_random_uuid(),
        '用户删除',
        'system:user:delete',
        'button',
        '/manage/user',
        'DELETE',
        0
      ),
      (
        gen_random_uuid(),
        '角色管理',
        'system:role',
        'menu',
        '/manage/role',
        'GET',
        0
      ),
      (
        gen_random_uuid(),
        '角色列表',
        'system:role:list',
        'button',
        '/manage/role',
        'GET',
        0
      ),
      (
        gen_random_uuid(),
        '角色创建',
        'system:role:create',
        'button',
        '/manage/role',
        'POST',
        0
      ),
      (
        gen_random_uuid(),
        '角色编辑',
        'system:role:update',
        'button',
        '/manage/role',
        'PUT',
        0
      ),
      (
        gen_random_uuid(),
        '角色删除',
        'system:role:delete',
        'button',
        '/manage/role',
        'DELETE',
        0
      ),
      (
        gen_random_uuid(),
        '菜单管理',
        'system:menu',
        'menu',
        '/manage/menu',
        'GET',
        0
      ),
      (
        gen_random_uuid(),
        '菜单列表',
        'system:menu:list',
        'button',
        '/manage/menu',
        'GET',
        0
      ),
      (
        gen_random_uuid(),
        '首页',
        'dashboard',
        'menu',
        '/dashboard',
        'GET',
        0
      )
  ) AS t(
    permission_id,
    permission_name,
    permission_code,
    resource_type,
    resource_path,
    action,
    status
  ) ON CONFLICT (permission_code) DO NOTHING;
-- =============================================
-- Step 3: 建立 role → permission 关联
-- =============================================
INSERT INTO role_permission (role_id, permission_id)
SELECT r.role_id,
  p.permission_id
FROM role r
  CROSS JOIN permission p
WHERE r.role_code = 'admin'
  AND p.status = 0
  AND NOT EXISTS (
    SELECT 1
    FROM role_permission rp
    WHERE rp.role_id = r.role_id
      AND rp.permission_id = p.permission_id
  );
-- =============================================
-- Step 4: 建立 user → role 关联（root 用户）
-- =============================================
INSERT INTO user_role (user_id, role_id)
SELECT u.user_id,
  r.role_id
FROM "user" u
  CROSS JOIN role r
WHERE u.username = 'root'
  AND r.role_code = 'admin'
  AND NOT EXISTS (
    SELECT 1
    FROM user_role ur
    WHERE ur.user_id = u.user_id
      AND ur.role_id = r.role_id
  );
-- =============================================
-- Step 5: 同步到 Casbin（g 规则：用户-角色）
-- =============================================
INSERT INTO casbin_rule (ptype, v0, v1)
SELECT 'g',
  u.user_id::text,
  'role::admin'
FROM "user" u
WHERE u.username = 'root'
  AND NOT EXISTS (
    SELECT 1
    FROM casbin_rule cr
    WHERE cr.ptype = 'g'
      AND cr.v0 = u.user_id::text
      AND cr.v1 = 'role::admin'
  );
-- =============================================
-- Step 6: 同步到 Casbin（p 规则：角色-权限）
-- =============================================
INSERT INTO casbin_rule (ptype, v0, v1, v2, v3)
SELECT 'p',
  'role::admin',
  p.resource_path,
  p.action,
  'allow'
FROM permission p
WHERE p.status = 0
  AND p.resource_path IS NOT NULL
  AND NOT EXISTS (
    SELECT 1
    FROM casbin_rule cr
    WHERE cr.ptype = 'p'
      AND cr.v0 = 'role::admin'
      AND cr.v1 = p.resource_path
      AND cr.v2 = p.action
  );
-- =============================================
-- 验证查询
-- =============================================
-- \echo '=== root 用户信息 ==='
-- SELECT user_id, username FROM "user" WHERE username = 'root';
-- \echo '=== admin 角色信息 ==='
-- SELECT role_id, role_name, role_code FROM role WHERE role_code = 'admin';
-- \echo '=== root 的角色列表 ==='
-- SELECT r.role_name, r.role_code
-- FROM user_role ur
-- JOIN role r ON ur.role_id = r.role_id
-- JOIN "user" u ON ur.user_id = u.user_id
-- WHERE u.username = 'root';
-- \echo '=== admin 角色的权限数量 ==='
-- SELECT COUNT(*) AS permission_count
-- FROM role_permission rp
-- JOIN role r ON rp.role_id = r.role_id
-- WHERE r.role_code = 'admin';
-- \echo '=== Casbin g 规则 ==='
-- SELECT ptype, v0, v1 FROM casbin_rule WHERE ptype = 'g';
-- \echo '=== Casbin p 规则（role::admin）==='
-- SELECT ptype, v0, v1, v2, v3 FROM casbin_rule WHERE ptype = 'p' AND v0 = 'role::admin';