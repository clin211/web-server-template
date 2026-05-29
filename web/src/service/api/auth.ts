import { request } from '../request';

/**
 * Login
 *
 * @param userName User name
 * @param password Password
 */
export function fetchLogin(userName: string, password: string) {
  return request<Api.User.LoginResponse>({
    url: '/v1/auth/login',
    method: 'post',
    data: {
      username: userName,
      password
    }
  });
}

/** Get user info */
export function fetchGetUserInfo() {
  return request<Api.User.GetMenuTreeResponse>({ url: '/v1/users/menu-tree' });
}

/**
 * Refresh token
 */
export function fetchRefreshToken() {
  return request<Api.User.LoginResponse>({
    url: '/v1/auth/refresh-token',
    method: 'put'
  });
}

/**
 * return custom backend error
 *
 * @param code error code
 * @param msg error message
 */
export function fetchCustomBackendError(code: string, msg: string) {
  return request({ url: '/auth/error', params: { code, msg } });
}
