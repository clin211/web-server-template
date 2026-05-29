import { request } from '../request';

type BackendListUserRequest = {
  page_token?: string;
  page_size?: number;
};

type BackendListUserResponse = {
  totalCount: number;
  users: Api.User.User[];
  page_token?: string;
  pageToken?: string;
};

type BackendRole = Omit<Api.Role.Role, 'id' | 'name' | 'code' | 'label' | 'value'>;

type BackendGetUserRolesResponse = {
  roles: BackendRole[];
  permissionCodes: string[];
};

/**
 * Get current user's menu tree
 */
export function fetchGetMenuTree() {
  return request<Api.User.GetMenuTreeResponse>({ url: '/v1/users/menu-tree' });
}

/**
 * Get user info
 *
 * @param userId User ID
 */
export function fetchGetUser(userId: string) {
  return request<Api.User.GetUserResponse>({ url: `/v1/users/${userId}` });
}

/**
 * Get user list
 *
 * @param params pageToken and pageSize
 */
export async function fetchGetUserList(params?: Api.User.ListUserRequest) {
  const result = await request<BackendListUserResponse>({
    url: '/v1/users',
    params: {
      page_token: params?.pageToken,
      page_size: params?.pageSize
    } satisfies BackendListUserRequest
  });

  if (result.error || !result.data) {
    return result;
  }

  return {
    ...result,
    data: {
      totalCount: result.data.totalCount,
      users: result.data.users,
      nextPageToken: result.data.page_token ?? result.data.pageToken ?? ''
    } satisfies Api.User.ListUserResponse
  };
}

/**
 * Create user
 *
 * @param data User data
 */
export function fetchCreateUser(data: Api.User.CreateUserRequest) {
  return request<Api.User.CreateUserResponse>({
    url: '/v1/users',
    method: 'post',
    data
  });
}

/**
 * Update user
 *
 * @param userId User ID
 * @param data Update data
 */
export function fetchUpdateUser(userId: string, data: Api.User.UpdateUserRequest) {
  return request({
    url: `/v1/users/${userId}`,
    method: 'put',
    data
  });
}

/**
 * Delete user
 *
 * @param userId User ID
 */
export function fetchDeleteUser(userId: string) {
  return request({
    url: `/v1/users/${userId}`,
    method: 'delete'
  });
}

/**
 * Get user roles
 *
 * @param userId User ID
 */
export async function fetchGetUserRoles(userId: string) {
  const result = await request<BackendGetUserRolesResponse>({
    url: `/v1/users/${userId}/roles`
  });

  if (result.error || !result.data) {
    return result;
  }

  return {
    ...result,
    data: {
      roles: result.data.roles.map(role => ({
        ...role,
        id: role.roleID,
        name: role.roleName,
        code: role.roleCode,
        label: role.roleName,
        value: role.roleID
      })),
      permissionCodes: result.data.permissionCodes || []
    } satisfies Api.User.GetUserRolesResponse
  };
}

/**
 * Assign roles to user
 *
 * @param userId User ID
 * @param roleIDs Role IDs to assign
 */
export function fetchAssignUserRoles(userId: string, roleIDs: string[]) {
  return request({
    url: `/v1/users/${userId}/roles`,
    method: 'post',
    data: { roleIDs }
  });
}

/**
 * Remove role from user
 *
 * @param userId User ID
 * @param roleId Role ID to remove
 */
export function fetchRemoveUserRole(userId: string, roleId: string) {
  return request({
    url: `/v1/users/${userId}/roles/${roleId}`,
    method: 'delete'
  });
}
