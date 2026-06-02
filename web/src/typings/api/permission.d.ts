declare namespace Api {
  /**
   * namespace Permission
   *
   * backend api module: "permission"
   */
  namespace Permission {
    /**
     * 权限基础字段
     */
    interface Permission {
      id: number; // 内部主键ID
      permissionId: string; // 权限业务唯一UUID
      permissionName: string; // 权限名称
      permissionCode: string; // 权限编码（唯一标识）
      resourceType: 'menu' | 'button'; // 资源类型：menu=菜单, button=按钮
      resourcePath?: string; // 资源路径（如 /system/user/list）
      action: string; // HTTP动词或自定义操作（GET/POST/export等）
      description?: string; // 权限描述
      parentId?: string; // 父权限UUID（用于构建权限树）
      path?: string; // 全路径（用于树形查询优化）
      status: number; // 权限状态：0=启用, 1=禁用
      createdAt: number; // 创建时间
      updatedAt: number; // 更新时间
    }

    /**
     * 权限树节点
     */
    interface PermissionTreeNode extends Permission {
      children: PermissionTreeNode[];
    }

    /**
     * 列表权限请求
     */
    interface ListPermissionRequest {
      pageToken?: string;
      pageSize?: number;
      status?: number;
      resourceType?: string;
      parentID?: string;
    }

    /**
     * 列表权限响应
     */
    interface ListPermissionResponse {
      totalCount: number;
      permissions: Permission[];
      pageToken: string;
    }

    /**
     * 列表权限树响应
     */
    interface ListPermissionTreeResponse {
      permissions: PermissionTreeNode[];
    }

    /**
     * 获取权限响应
     */
    interface GetPermissionResponse {
      permission: Permission;
    }

    /**
     * 创建权限请求
     */
    interface CreatePermissionRequest {
      parentID?: string;
      permissionName: string;
      permissionCode: string;
      resourceType: string;
      resourcePath?: string;
      action: string;
      description?: string;
      status?: number;
    }

    /**
     * 创建权限响应
     */
    interface CreatePermissionResponse {
      permissionId: string;
    }

    /**
     * 更新权限请求
     */
    interface UpdatePermissionRequest {
      parentID?: string;
      permissionName?: string;
      resourceType?: string;
      resourcePath?: string;
      action?: string;
      description?: string;
      status?: number;
    }

    /**
     * 更新权限响应
     */
    interface UpdatePermissionResponse {}

    /**
     * 删除权限响应
     */
    interface DeletePermissionResponse {}
  }
}
