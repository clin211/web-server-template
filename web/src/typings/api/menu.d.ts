declare namespace Api {
  /**
   * namespace Menu
   *
   * backend api module: "menu" (apiserver/v1)
   */
  namespace Menu {
    /**
     * 菜单基础字段
     */
    interface Menu {
      menuID: string;
      parentID: string; // 父菜单ID，顶级菜单为 "0"
      menuName: string;
      menuCode: string;
      menuType: 'menu' | 'page'; // menu=目录, page=页面
      i18nKey?: string; // 国际化key
      icon?: string;
      localIcon?: string;
      iconFontSize?: number;
      path?: string;
      component?: string;
      permissionID?: string;
      sortOrder: number;
      visible: number; // 0=隐藏, 1=显示
      status: number;
      constant: number; // 0=否, 1=是
      activeMenu?: string;
      hideInMenu: number; // 0=否, 1=是
      keepAlive: number; // 0=否, 1=是
      href?: string;
      createdAt: number;
      updatedAt: number;
    }

    /**
     * 菜单树节点
     */
    interface MenuTreeNode extends Menu {
      children: MenuTreeNode[];
    }

    /**
     * 列表菜单请求
     */
    interface ListMenuRequest {
      pageToken?: string;
      pageSize?: number;
      status?: number;
      menuType?: string;
      parentID?: string;
    }

    /**
     * 列表菜单响应
     */
    interface ListMenuResponse {
      totalCount: number;
      menus: MenuTreeNode[];
      pageToken: string;
    }

    /**
     * 列表菜单树响应
     */
    interface ListMenuTreeResponse {
      menus: MenuTreeNode[];
    }

    /**
     * 获取菜单响应
     */
    interface GetMenuResponse {
      menu: Menu;
    }

    /**
     * 创建菜单请求
     */
    interface CreateMenuRequest {
      parentID?: string;
      menuName: string;
      menuCode: string;
      menuType: string;
      i18nKey?: string;
      icon?: string;
      localIcon?: string;
      iconFontSize?: number;
      path?: string;
      component?: string;
      permissionID?: string;
      sortOrder?: number;
      visible?: number;
      constant?: number;
      activeMenu?: string;
      hideInMenu?: number;
      keepAlive?: number;
      href?: string;
    }

    /**
     * 创建菜单响应
     */
    interface CreateMenuResponse {
      menuID: string;
    }

    /**
     * 更新菜单请求
     */
    interface UpdateMenuRequest {
      parentID?: string;
      menuName?: string;
      i18nKey?: string;
      icon?: string;
      localIcon?: string;
      iconFontSize?: number;
      path?: string;
      component?: string;
      sortOrder?: number;
      visible?: number;
      status?: number;
      constant?: number;
      activeMenu?: string;
      hideInMenu?: number;
      keepAlive?: number;
      href?: string;
    }

    /**
     * 更新菜单响应
     */
    interface UpdateMenuResponse {}

    /**
     * 删除菜单响应
     */
    interface DeleteMenuResponse {}

    // ============= Menu Role Management =============

    /**
     * 获取菜单允许的角色列表响应
     */
    interface GetMenuRolesResponse {
      roleIds: string[];
      roleCodes: string[];
    }

    /**
     * 追加菜单允许的角色请求
     */
    interface AddMenuRoleRequest {
      roleId: string;
    }

    /**
     * 批量设置菜单允许的角色请求
     */
    interface SetMenuRolesRequest {
      roleIds: string[];
    }

    /**
     * 批量设置菜单允许的角色响应
     */
    interface SetMenuRolesResponse {
      menuId: string;
      roleIds: string[];
      count: number;
    }
  }
}