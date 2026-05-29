import { useAuthStore } from '@/store/modules/auth';
import { localStg } from '@/utils/storage';
import { fetchRefreshToken } from '../api';
import type { RequestInstanceState } from './type';

export function getAuthorization(useRefreshToken = false) {
  const token = useRefreshToken ? localStg.get('refreshToken') : localStg.get('token');
  const Authorization = token ? `Bearer ${token}` : null;

  return Authorization;
}

/** refresh token */
async function handleRefreshToken() {
  const authStore = useAuthStore();
  const refreshToken = localStg.get('refreshToken');

  if (!refreshToken) {
    await authStore.resetStore();
    return false;
  }

  const { error, data } = await fetchRefreshToken();
  if (!error && data) {
    localStg.set('token', data.accessToken);
    localStg.set('refreshToken', data.refreshToken);
    localStg.set('tokenExpireAt', data.expireAt);
    authStore.token = data.accessToken;
    return true;
  }

  await authStore.resetStore();

  return false;
}

export async function handleExpiredRequest(state: RequestInstanceState) {
  if (!state.refreshTokenPromise) {
    state.refreshTokenPromise = handleRefreshToken();
  }

  const success = await state.refreshTokenPromise;

  setTimeout(() => {
    state.refreshTokenPromise = null;
  }, 1000);

  return success;
}

export function showErrorMsg(state: RequestInstanceState, message: string) {
  if (!state.errMsgStack?.length) {
    state.errMsgStack = [];
  }

  const isExist = state.errMsgStack.includes(message);

  if (!isExist) {
    state.errMsgStack.push(message);

    window.$message?.error(message, {
      onLeave: () => {
        state.errMsgStack = state.errMsgStack.filter(msg => msg !== message);

        setTimeout(() => {
          state.errMsgStack = [];
        }, 5000);
      }
    });
  }
}
