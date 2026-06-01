import { request } from '../request';

/**
 * Get menu list
 *
 * @param params pageToken, pageSize, status, menuType, parentID
 */
export async function fetchGetMenuList(params?: Api.Menu.ListMenuRequest) {
  const result = await request<Api.Menu.ListMenuResponse>({
    url: '/v1/menus',
    params: {
      pageToken: params?.pageToken,
      pageSize: params?.pageSize,
      status: params?.status,
      menuType: params?.menuType,
      parentID: params?.parentID
    }
  });

  if (result.error || !result.data) {
    return result;
  }

  return {
    ...result,
    data: {
      totalCount: result.data.totalCount,
      menus: result.data.menus,
      pageToken: result.data.pageToken ?? ''
    } satisfies Api.Menu.ListMenuResponse
  };
}

/**
 * Create menu
 *
 * @param data Menu data
 */
export function fetchCreateMenu(data: Api.Menu.CreateMenuRequest) {
  return request<Api.Menu.CreateMenuResponse>({
    url: '/v1/menus',
    method: 'post',
    data
  });
}

/**
 * Get all menu tree (admin)
 *
 * @param params status filter
 */
export function fetchGetAllMenuTree(params?: { status?: number }) {
  return request<Api.Menu.ListMenuTreeResponse>({
    url: '/v1/menus/tree',
    params
  });
}

/**
 * Get menu by ID
 *
 * @param menuId Menu ID
 */
export function fetchGetMenu(menuId: string) {
  return request<Api.Menu.GetMenuResponse>({ url: `/v1/menus/${menuId}` });
}

/**
 * Update menu
 *
 * @param menuId Menu ID
 * @param data Update data
 */
export function fetchUpdateMenu(menuId: string, data: Api.Menu.UpdateMenuRequest) {
  return request<Api.Menu.UpdateMenuResponse>({
    url: `/v1/menus/${menuId}`,
    method: 'put',
    data
  });
}

/**
 * Delete menu
 *
 * @param menuId Menu ID
 */
export function fetchDeleteMenu(menuId: string) {
  return request<Api.Menu.DeleteMenuResponse>({
    url: `/v1/menus/${menuId}`,
    method: 'delete'
  });
}

/**
 * Get menu allowed roles
 *
 * @param menuId Menu ID
 */
export async function fetchGetMenuRoles(menuId: string) {
  const result = await request<Api.Menu.GetMenuRolesResponse>({
    url: `/v1/menus/${menuId}/roles`,
    method: 'get'
  });

  return result;
}

/**
 * Add role to menu
 *
 * @param menuId Menu ID
 * @param data Role data
 */
export async function fetchAddMenuRole(menuId: string, data: Api.Menu.AddMenuRoleRequest) {
  const result = await request({
    url: `/v1/menus/${menuId}/roles`,
    method: 'post',
    data
  });

  return result;
}

/**
 * Set menu roles (batch, overwrite mode)
 *
 * @param menuId Menu ID
 * @param data Role IDs
 */
export async function fetchSetMenuRoles(menuId: string, data: Api.Menu.SetMenuRolesRequest) {
  const result = await request<Api.Menu.SetMenuRolesResponse>({
    url: `/v1/menus/${menuId}/roles`,
    method: 'put',
    data
  });

  return result;
}

/**
 * Remove menu allowed role
 *
 * @param menuId Menu ID
 * @param roleId Role ID to remove
 */
export async function fetchRemoveMenuRole(menuId: string, roleId: string) {
  const result = await request({
    url: `/v1/menus/${menuId}/roles/${roleId}`,
    method: 'delete'
  });

  return result;
}
