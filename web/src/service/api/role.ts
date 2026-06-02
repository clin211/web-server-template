import { request } from '../request';

type BackendListRoleRequest = {
  pageToken?: string;
  pageSize?: number;
  status?: number;
  keyword?: string;
};

type BackendListRoleResponse = {
  totalCount: number;
  roles: Array<{
    roleID: string;
    roleName: string;
    roleCode: string;
    description: string;
    status: number;
    sortOrder: number;
    createdAt: number;
    updatedAt: number;
  }>;
  pageToken?: string;
};

/**
 * Get role list
 *
 * @param params pageToken, pageSize, status and keyword
 */
export async function fetchGetRoleList(params?: Api.Role.ListRoleRequest) {
  const result = await request<BackendListRoleResponse>({
    url: '/v1/roles',
    params: {
      pageToken: params?.pageToken,
      pageSize: params?.pageSize,
      status: params?.status,
      keyword: params?.keyword
    } satisfies BackendListRoleRequest
  });

  if (result.error || !result.data) {
    return result;
  }

  return {
    ...result,
    data: {
      totalCount: result.data.totalCount,
      roles: result.data.roles.map(role => ({
        ...role,
        id: role.roleID,
        name: role.roleName,
        code: role.roleCode,
        label: role.roleName,
        value: role.roleID
      })),
      nextPageToken: result.data.pageToken ?? ''
    } satisfies Api.Role.ListRoleResponse
  };
}

/**
 * Get role permissions (tree structure)
 *
 * @param roleId Role ID
 */
export async function fetchGetRolePermissions(roleId: string) {
  const result = await request<Api.Role.GetRolePermissionsResponse>({
    url: `/v1/roles/${roleId}/permissions`,
    method: 'get'
  });

  return result;
}

/**
 * Assign permissions to role
 *
 * @param roleId Role ID
 * @param data Permission assignment data
 */
export function fetchAssignPermissionsToRole(roleId: string, data: Api.Role.AssignPermissionsToRoleRequest) {
  return request({
    url: `/v1/roles/${roleId}/permissions`,
    method: 'post',
    data
  });
}
