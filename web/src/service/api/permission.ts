import { request } from '../request';

/**
 * Get permission list
 *
 * @param params pageToken, pageSize, status, resourceType, parentID
 */
export async function fetchGetPermissionList(params?: Api.Permission.ListPermissionRequest) {
  const result = await request<Api.Permission.ListPermissionResponse>({
    url: '/v1/permissions',
    params: {
      page_token: params?.pageToken,
      page_size: params?.pageSize,
      status: params?.status,
      resource_type: params?.resourceType,
      parent_id: params?.parentID
    }
  });

  if (result.error || !result.data) {
    return result;
  }

  return {
    ...result,
    data: {
      totalCount: result.data.totalCount,
      permissions: result.data.permissions,
      pageToken: result.data.pageToken ?? ''
    } satisfies Api.Permission.ListPermissionResponse
  };
}

/**
 * Create permission
 *
 * @param data Permission data
 */
export function fetchCreatePermission(data: Api.Permission.CreatePermissionRequest) {
  return request<Api.Permission.CreatePermissionResponse>({
    url: '/v1/permissions',
    method: 'post',
    data
  });
}

/**
 * Get all permission tree
 *
 * @param params status filter
 */
export function fetchGetAllPermissionTree(params?: { status?: number }) {
  return request<Api.Permission.ListPermissionTreeResponse>({
    url: '/v1/permissions/tree',
    params
  });
}

/**
 * Get permission by ID
 *
 * @param permissionId Permission ID
 */
export function fetchGetPermission(permissionId: string) {
  return request<Api.Permission.GetPermissionResponse>({ url: `/v1/permissions/${permissionId}` });
}

/**
 * Update permission
 *
 * @param permissionId Permission ID
 * @param data Update data
 */
export function fetchUpdatePermission(permissionId: string, data: Api.Permission.UpdatePermissionRequest) {
  return request<Api.Permission.UpdatePermissionResponse>({
    url: `/v1/permissions/${permissionId}`,
    method: 'put',
    data
  });
}

/**
 * Delete permission
 *
 * @param permissionId Permission ID
 */
export function fetchDeletePermission(permissionId: string) {
  return request<Api.Permission.DeletePermissionResponse>({
    url: `/v1/permissions/${permissionId}`,
    method: 'delete'
  });
}
