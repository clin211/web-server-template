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
      id: string;
      name: string;
      code: string;
      label: string;
      value: string;
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
  }
}
