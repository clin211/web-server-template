import { request } from '../request';

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
export function fetchGetUserList(params?: Api.User.ListUserRequest) {
  return request<Api.User.ListUserResponse>({ url: '/v1/users', params });
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
export function fetchGetUserRoles(userId: string) {
  return request<Api.User.GetUserRolesResponse>({ url: `/v1/users/${userId}/roles` });
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