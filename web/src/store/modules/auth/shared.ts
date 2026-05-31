import { localStg } from '@/utils/storage';

/** Token expiration time buffer (ms), refresh before expiration to avoid race conditions */
export const TOKEN_REFRESH_BUFFER_MS = 60 * 1000; // 1 minute before expiration

/** Get token */
export function getToken() {
  return localStg.get('token') || '';
}

/** Get refresh token */
export function getRefreshToken() {
  return localStg.get('refreshToken') || '';
}

/** Get token expiration time (as Unix timestamp in ms) */
export function getTokenExpireAt(): number | undefined {
  const expireAt = localStg.get('tokenExpireAt');
  if (!expireAt) {
    return undefined;
  }
  // Handle ISO 8601 format: "2026-06-01T09:43:11+08:00"
  const date = new Date(expireAt);
  if (isNaN(date.getTime())) {
    return undefined;
  }
  return date.getTime();
}

/** Check if token is about to expire or already expired */
export function isTokenExpiredOrExpiring(expireAt: number | undefined): boolean {
  if (!expireAt) {
    return true;
  }
  const now = Date.now();
  return now >= expireAt - TOKEN_REFRESH_BUFFER_MS;
}

/** Check if token is completely expired (past expiration time) */
export function isTokenExpired(expireAt: number | undefined): boolean {
  if (!expireAt) {
    return true;
  }
  return Date.now() >= expireAt;
}

/** Clear auth storage */
export function clearAuthStorage() {
  localStg.remove('token');
  localStg.remove('refreshToken');
  localStg.remove('tokenExpireAt');
}
