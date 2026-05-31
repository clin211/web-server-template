# 菜单管理系统重设计方案

## 一、背景与目标

根据 SoybeanAdmin 前端的路由和菜单管理机制，重构服务端菜单管理模块，实现前后端菜单数据的无缝对接。

### 1.1 当前问题

- 后端菜单模型字段与前端期望不匹配
- 缺少与 Elegant Router 机制的集成
- API 返回格式与前端类型定义不一致

### 1.2 重构目标

- 重设计数据库表结构，兼容 SoybeanAdmin 的菜单元数据格式
- 实现与 Elegant Router 机制匹配的菜单管理 API
- 支持前端动态路由模式（dynamic route mode）

## 二、前端菜单机制分析

### 2.1 Elegant Router 机制

SoybeanAdmin 使用 `@elegant-router/vue` 实现基于文件系统的路由自动生成：

```
web/src/views/           → 页面视图
web/src/layouts/        → 布局组件
web/src/router/elegant/ → 生成的路由配置
```

**路由生成产物**（`src/router/elegant/routes.ts`）示例：

```typescript
{
  name: 'system-manage',
  path: '/system-manage',
  component: 'layout.base',
  meta: {
    title: 'system-manage',
    i18nKey: 'route.system-manage',
    icon: 'ph:gear-six',
    order: 2
  },
  children: [
    {
      name: 'system-manage_user',
      path: '/system-manage/user',
      component: 'view.system-manage_user',
      meta: {
        title: 'system-manage_user',
        i18nKey: 'route.system-manage_user',
        icon: 'ph:user-circle'
      }
    }
  ]
}
```

### 2.2 前端菜单类型定义

**路由元数据（ElegantConstRoute.meta）**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `title` | string | 菜单标题 |
| `i18nKey` | string | 国际化 key |
| `icon` | string | 图标名称 |
| `localIcon` | string | 本地图标 |
| `iconFontSize` | number | 图标大小 |
| `order` | number | 排序序号 |
| `hideInMenu` | boolean | 在菜单中隐藏 |
| `keepAlive` | boolean | 页面缓存 |
| `constant` | boolean | 常量路由（不参与权限过滤） |
| `roles` | string[] | 允许访问的角色列表 |
| `activeMenu` | string | 当前激活的菜单 |
| `href` | string | 外链地址 |

**前端菜单结构（App.Global.Menu）**：

```typescript
interface App.Global.Menu {
  key: string;              // 路由名称
  label: string;            // 菜单名称
  i18nKey: string;          // 国际化 key
  routeKey: RouteKey;       // 路由 key
  routePath: RoutePath;     // 路由路径
  icon: VNode;              // 图标组件
  children?: App.Global.Menu[];
}
```

### 2.3 动态路由模式

前端支持两种路由模式（`VITE_AUTH_ROUTE_MODE`）：

| 模式 | 说明 | 菜单来源 |
|------|------|---------|
| `static` | 开发环境，路由由前端自动生成 | 根据 `meta.roles` 过滤生成路由 |
| `dynamic` | 生产环境，路由由后端提供 | `/route/getUserRoutes` API |

**动态路由 API**：

```typescript
// GET /route/getConstantRoutes
// 返回常量路由（root, 404, 403 等）

// GET /route/getUserRoutes
// 返回用户可访问的路由和首页
{
  routes: MenuRoute[];  // 用户路由树
  home: string;         // 首页 routeKey
}
```

**菜单树 API**：

```typescript
// GET /v1/users/menu-tree
{
  menus: MenuTreeNode[];
}

interface MenuTreeNode {
  menu: Menu;
  children: MenuTreeNode[];
}

interface Menu {
  menuID: string;
  parentID: string;
  menuName: string;
  menuCode: string;
  menuType: string;    // menu=目录, page=页面
  i18nKey: string;      // 国际化 key
  icon: string;        // 图标名称
  localIcon: string;   // 本地图标
  iconFontSize: number; // 图标大小
  path: string;
  component: string;
  permissionID: string;
  sortOrder: number;
  visible: number;      // 0=隐藏, 1=显示
  status: number;       // 0=启用, 1=禁用
  constant: boolean;    // 常量路由
  hideInMenu: boolean;  // 在菜单中隐藏
  keepAlive: boolean;   // 页面缓存
  href: string;         // 外链地址
  activeMenu: string;   // 当前激活的菜单（用于面包屑）
  createdAt: number;
  updatedAt: number;
}
```

### 2.4 路由权限过滤

前端根据用户角色过滤路由：

```typescript
// src/store/modules/route/shared.ts
function filterAuthRouteByRoles(route: ElegantConstRoute, roles: string[]): ElegantConstRoute[] {
  const routeRoles = route.meta?.roles || [];
  const isEmptyRoles = !routeRoles.length;
  const hasPermission = routeRoles.some(role => roles.includes(role));

  // 空 roles 表示无需权限
  return hasPermission || isEmptyRoles ? [route] : [];
}
```

## 三、数据库设计

### 3.1 菜单表（menu）

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
```

### 3.2 字段映射说明

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

### 3.3 菜单角色关联表（menu_role）

> 此表用于定义菜单允许访问的角色列表。配合 `constant` 字段，实现灵活的权限控制。

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

**权限控制规则**：

| 条件 | 结果 |
|------|------|
| `menu_role` 无记录 | 该菜单对所有登录用户可见 |
| `menu_role` 有记录 | 只有记录中包含的角色可以访问 |
| `menu.constant = 1` | 该菜单为常量路由，不参与权限过滤 |

**示例**：

```sql
-- 首页对所有用户可见（menu_role 无记录）
-- 用户管理页面只有 super_admin 和 admin 可访问
INSERT INTO menu_role (menu_id, role_id) VALUES
  ('home-menu-id', 'super-admin-role-id'),
  ('user-menu-id', 'super-admin-role-id'),
  ('user-menu-id', 'admin-role-id');
```

## 四、API 设计

### 4.1 菜单管理 API

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/v1/menus` | 创建菜单 |
| `GET` | `/v1/menus` | 获取菜单列表（分页） |
| `GET` | `/v1/menus/tree` | 获取菜单树 |
| `GET` | `/v1/menus/:id` | 获取单个菜单 |
| `PATCH` | `/v1/menus/:id` | 更新菜单 |
| `DELETE` | `/v1/menus/:id` | 删除菜单 |
| `PUT` | `/v1/menus/:id/sort` | 更新菜单排序 |

### 4.2 菜单角色管理 API

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/v1/menus/:id/roles` | 获取菜单允许访问的角色列表 |
| `PUT` | `/v1/menus/:id/roles` | 批量设置菜单允许的角色（覆盖模式） |
| `POST` | `/v1/menus/:id/roles` | 追加菜单允许的角色 |
| `DELETE` | `/v1/menus/:id/roles/:roleId` | 移除菜单允许的角色 |

### 4.3 用户菜单 API

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/v1/users/menu-tree` | 获取当前用户的菜单树 |
| `GET` | `/route/getConstantRoutes` | 获取常量路由 |
| `GET` | `/route/getUserRoutes` | 获取用户可访问路由 |

**常量路由响应**（`/route/getConstantRoutes`）：

> 常量路由不参与权限过滤，前端直接硬编码。返回的路由包括：root、not-found、403、404、500、login 等。

```json
{
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
```

### 4.4 请求/响应定义

**获取用户路由响应**：

**创建菜单请求**：

```json
{
  "parentId": "optional-parent-id",
  "menuName": "用户管理",
  "menuCode": "system-manage_user",
  "menuType": "page",
  "i18nKey": "route.system-manage_user",
  "icon": "ph:user-circle",
  "localIcon": null,
  "iconFontSize": 20,
  "path": "/system-manage/user",
  "component": "view.system-manage_user",
  "sortOrder": 1,
  "visible": 1,
  "keepAlive": false,
  "hideInMenu": false,
  "constant": false,
  "activeMenu": null,
  "href": null
}
```

**菜单响应**：

```json
{
  "menuID": "uuid",
  "parentID": "parent-uuid",
  "menuName": "用户管理",
  "menuCode": "system-manage_user",
  "menuType": "page",
  "i18nKey": "route.system-manage_user",
  "icon": "ph:user-circle",
  "localIcon": null,
  "iconFontSize": 20,
  "path": "/system-manage/user",
  "component": "view.system-manage_user",
  "permissionID": "permission-uuid",
  "sortOrder": 1,
  "visible": 1,
  "status": 0,
  "constant": false,
  "activeMenu": null,
  "hideInMenu": false,
  "keepAlive": false,
  "href": null,
  "createdAt": "2026-05-29T00:00:00Z",
  "updatedAt": "2026-05-29T00:00:00Z"
}
```

**获取用户路由响应**：

> **注意**：返回的 `routes` 必须是完整的嵌套树结构，前端直接使用 `router.addRoute()` 添加。
> `meta.roles` 用于前端权限过滤，当角色为空或包含用户角色时允许访问。

```json
{
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
  "menuId": "menu-uuid",
  "roleIds": ["role-uuid-1", "role-uuid-2"],
  "count": 2
}
```

## 五、技术实现

### 5.1 后端菜单模型

```go
// internal/apiserver/model/menu.gen.go

type MenuM struct {
    ID           int64      `gorm:"column:id;primaryKey;autoIncrement"`
    MenuID       string     `gorm:"column:menu_id;not null;default:gen_random_uuid()"`
    ParentID     *string    `gorm:"column:parent_id"`
    MenuName     string     `gorm:"column:menu_name;not null"`
    MenuCode     string     `gorm:"column:menu_code;not null"`
    MenuType     string     `gorm:"column:menu_type;not null"`
    I18nKey      *string    `gorm:"column:i18n_key"`              // 国际化 key
    Icon         *string    `gorm:"column:icon"`
    LocalIcon    *string    `gorm:"column:local_icon"`            // 本地图标
    IconFontSize *int       `gorm:"column:icon_font_size"`        // 图标大小
    Path         *string    `gorm:"column:path"`
    Component    *string    `gorm:"column:component"`
    PermissionID *string    `gorm:"column:permission_id"`
    SortOrder    int32      `gorm:"column:sort_order;default:0"`
    Visible      int16      `gorm:"column:visible;not null;default:1"`
    Status       int16      `gorm:"column:status;not null;default:0"`
    Constant     int16      `gorm:"column:constant;not null;default:0"`
    ActiveMenu  *string    `gorm:"column:active_menu"`              // 当前激活的菜单
    HideInMenu   int16      `gorm:"column:hide_in_menu;not null;default:0"`
    KeepAlive    int16      `gorm:"column:keep_alive;not null;default:0"`
    Href         *string    `gorm:"column:href"`
    CreatedAt    time.Time  `gorm:"column:created_at;not null"`
    UpdatedAt    time.Time  `gorm:"column:updated_at;not null"`
    DeletedAt    *time.Time `gorm:"column:deleted_at"`
}
```

### 5.2 前端类型适配

前端期望的 `MenuRoute` 接口：

```typescript
// src/typings/api/route.d.ts
interface MenuRoute extends ElegantConstRoute {
  id: string;
}

interface UserRoute {
  routes: MenuRoute[];
  home: string;
}
```

后端需要转换为符合前端期望的格式：

```go
// internal/apiserver/pkg/conversion/menu.go

func MenuToRouteProto(menu *model.MenuM, roles []string) *apiserver.MenuRoute {
    return &apiserver.MenuRoute{
        Id:       menu.MenuID,
        Name:     menu.MenuCode,
        Path:     menu.Path,
        Component: menu.Component,
        Meta: &apiserver.MenuRouteMeta{
            Title:      menu.MenuName,
            I18nKey:    menu.I18nKey,
            Icon:       menu.Icon,
            LocalIcon:  menu.LocalIcon,
            IconFontSize: menu.IconFontSize,
            Order:      menu.SortOrder,
            ActiveMenu: menu.ActiveMenu,
            HideInMenu: menu.HideInMenu == 1,
            KeepAlive:  menu.KeepAlive == 1,
            Constant:   menu.Constant == 1,
            Href:       menu.Href,
            Roles:      roles,
        },
    }
}
```

### 5.3 菜单树构建

从数据库获取用户可见的菜单后，在内存中构建树形结构：

```go
// internal/apiserver/store/menu.go

// GetUserMenus 获取用户可见的菜单
// 过滤逻辑：
// 1. 菜单状态必须启用且可见
// 2. 满足以下条件之一：
//    - 该菜单在 menu_role 表中无记录（对所有角色可见）
//    - 该菜单的 menu_role 记录中至少有一个角色是用户拥有的
func (s *menuStore) GetUserMenus(ctx context.Context, userID string) ([]*model.MenuM, error) {
    // 获取用户拥有的角色ID列表
    var roleIDs []string
    err := s.core.DB(ctx).
        Table("user_role").
        Select("role_id").
        Where("user_id = ?", userID).
        Pluck("role_id", &roleIDs).Error
    if err != nil {
        return nil, err
    }

    // 如果用户没有任何角色，返回空
    if len(roleIDs) == 0 {
        return []*model.MenuM{}, nil
    }

    var menus []*model.MenuM

    // 查询满足以下条件的菜单：
    // 1. 状态启用且可见
    // 2. 软删除未删除
    // 3. 满足：
    //    - menu_role 表中无记录（对所有角色可见）
    //    - 或 menu_role 表中有该用户的角色
    err = s.core.DB(ctx).
        Where("menu.status = ? AND menu.visible = ?", 0, 1).
        Where("menu.deleted_at IS NULL").
        Where("(menu_id NOT IN (SELECT menu_id FROM menu_role)) OR " +
              "(menu_id IN (SELECT menu_id FROM menu_role WHERE role_id IN ?)))", roleIDs).
        Order("menu.parent_id NULLS LAST, menu.sort_order ASC").
        Find(&menus).Error
    if err != nil {
        return nil, err
    }

    return menus, nil
}

// GetMenuAllowedRoles 获取菜单允许访问的角色列表
// 从 menu_role 表中查询该菜单允许的角色，返回角色代码列表
func (s *menuStore) GetMenuAllowedRoles(ctx context.Context, menuID string) ([]string, error) {
    var roleCodes []string
    err := s.core.DB(ctx).
        Table("menu_role").
        Select("role.role_code").
        Joins("JOIN role ON menu_role.role_id = role.role_id").
        Where("menu_role.menu_id = ?", menuID).
        Pluck("role.role_code", &roleCodes).Error
    if err != nil {
        return nil, err
    }
    return roleCodes, nil
}

// BuildMenuTree 将扁平菜单列表转换为嵌套树结构，同时填充 roles 信息
func BuildMenuTree(menus []*model.MenuM, store MenuStore) ([]*apiserver.MenuRoute, error) {
    childrenMap := make(map[string][]*model.MenuM)

    // 按父ID分组
    for _, menu := range menus {
        parentID := ""
        if menu.ParentID != nil {
            parentID = *menu.ParentID
        }
        childrenMap[parentID] = append(childrenMap[parentID], menu)
    }

    // 收集根节点
    rootMenus := childrenMap[""]

    // 递归构建树
    var buildRoutes func(parentID string) ([]*apiserver.MenuRoute, error)
    buildRoutes = func(parentID string) ([]*apiserver.MenuRoute, error) {
        children := childrenMap[parentID]
        routes := make([]*apiserver.MenuRoute, 0, len(children))
        for _, menu := range children {
            // 获取菜单允许的角色
            roles, err := store.GetMenuAllowedRoles(context.Background(), menu.MenuID)
            if err != nil {
                return nil, err
            }
            route := MenuToRouteProto(menu, roles)
            // 递归构建子路由
            childRoutes, err := buildRoutes(menu.MenuID)
            if err != nil {
                return nil, err
            }
            route.Children = childRoutes
            routes = append(routes, route)
        }
        return routes, nil
    }

    return buildRoutes("")
}
```

## 六、前端集成

### 6.1 动态路由初始化

```typescript
// src/store/modules/route/index.ts

async function initDynamicAuthRoute() {
  const { data, error } = await fetchGetUserRoutes();

  if (!error) {
    const { routes, home } = data;
    addAuthRoutes(routes);
    handleConstantAndAuthRoutes();
    setRouteHome(home);
  }
}
```

### 6.2 菜单状态管理

```typescript
// src/store/modules/auth/index.ts

async function fetchMenuTree() {
  const { data, error } = await fetchGetMenuTree();
  if (!error && data) {
    menuTree.value = data.menus || [];
  }
}
```

### 6.3 国际化支持

前端使用 `i18nKey` 进行国际化：

```typescript
// src/store/modules/route/shared.ts

function updateLocaleOfGlobalMenus(menus: App.Global.Menu[]) {
  return menus.map(menu => ({
    ...menu,
    label: menu.i18nKey ? $t(menu.i18nKey) : menu.label,
    children: menu.children ? updateLocaleOfGlobalMenus(menu.children) : undefined
  }));
}
```

后端需要支持 `i18nKey` 字段：

```go
type MenuM struct {
    // ...
    I18nKey *string `gorm:"column:i18n_key"`  // 新增字段
    // ...
}
```

## 七、实施计划

### 7.1 阶段一：数据库重构

1. 创建新的 `menu` 表结构
2. 创建 `menu_role` 关联表（菜单允许的角色）
3. 编写数据迁移脚本

### 7.2 阶段二：后端实现

1. 更新 GORM 模型（新增 `i18n_key`, `local_icon`, `icon_font_size`, `active_menu` 字段）
2. 实现 Store 层（`GetUserMenus` 支持菜单角色过滤）
3. 实现 Biz 层（菜单 CRUD + 菜单角色管理）
4. 实现 Handler 层（适配前端 API 格式）
5. 实现 `/route/getUserRoutes` 接口（返回嵌套树结构）
6. 更新 API 响应格式适配前端

### 7.3 阶段三：菜单角色管理

1. 实现 `/v1/menus/:id/roles` GET 接口
2. 实现 `/v1/menus/:id/roles` PUT 接口（批量覆盖）
3. 实现 `/v1/menus/:id/roles` POST 接口（追加）
4. 实现 `/v1/menus/:id/roles/:roleId` DELETE 接口

### 7.4 阶段四：集成测试

1. 实现 `/v1/menus/:id/roles` GET 接口
2. 实现 `/v1/menus/:id/roles` PUT 接口（批量覆盖）
3. 实现 `/v1/menus/:id/roles` POST 接口（追加）
4. 实现 `/v1/menus/:id/roles/:roleId` DELETE 接口

### 7.5 阶段五：集成测试

1. 前后端联调测试
2. 动态路由模式验证
3. 权限过滤功能验证

### 7.6 阶段六：初始化数据

1. 迁移现有菜单数据
2. 初始化默认菜单角色关联
3. 创建前端期望的示例菜单

## 八、验收标准

- [ ] 数据库表结构符合设计规范
- [ ] 菜单 CRUD API 正常工作
- [ ] `/v1/users/menu-tree` 返回正确的菜单树结构
- [ ] `/route/getConstantRoutes` 返回正确的常量路由
- [ ] `/route/getUserRoutes` 返回符合前端期望的路由格式（嵌套树结构 + meta.roles）
- [ ] 前端动态路由模式正常加载菜单
- [ ] 菜单排序和层级关系正确维护
- [ ] 权限过滤功能正常工作（根据 menu_role 表过滤）
- [ ] 菜单角色管理 API 正常工作（增删改查）
- [ ] `menu_role` 关联表正确管理菜单与角色的关系
- [ ] 用户根据角色正确获取对应的菜单树
- [ ] `constant = 1` 的菜单不参与权限过滤

## 九、参考文档

- 前端路由机制：[Elegant Router](https://github.com/soybeanjs/elegant-router)
- 前端类型定义：`web/src/typings/api/route.d.ts`
- 前端菜单管理：`web/src/store/modules/route/`
- 现有权限设计：`docs/features/02 permission.md`
