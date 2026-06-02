declare namespace Api {
  /**
   * namespace Role
   *
   * backend api module: "role" (apiserver/v1)
   */
  namespace Role {
    interface Role {
      roleID: string;
      roleName: string;
      roleCode: string;
      description: string;
      status: number;
      sortOrder: number;
      createdAt: number;
      updatedAt: number;
    }

    interface ListRoleRequest {
      pageToken?: string;
      pageSize?: number;
      status?: number;
      keyword?: string;
    }

    interface ListRoleResponse {
      totalCount: number;
      roles: Role[];
      nextPageToken: string;
    }

    interface RoleOption {
      label: string;
      value: string;
      roleCode: string;
      roleName: string;
      status: number;
    }

    // ============= Role Permission Management =============

    /**
     * 权限树节点
     */
    interface PermissionTreeNode {
      permissionID: string;
      permissionName: string;
      permissionCode: string;
      resourceType: string; // menu=菜单, button=按钮
      resourcePath: string;
      action: string; // GET/POST/PUT/DELETE
      assigned: boolean;
      children: PermissionTreeNode[];
    }

    /**
     * 获取角色权限响应
     */
    interface GetRolePermissionsResponse {
      permissions: PermissionTreeNode[];
    }

    /**
     * 分配角色权限请求
     */
    interface AssignPermissionsToRoleRequest {
      permissionIDs: string[];
      mode: 'override' | 'append'; // override=覆盖模式, append=追加模式
    }
  }
}
