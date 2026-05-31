import { computed, reactive, ref } from 'vue';
import { useRoute } from 'vue-router';
import { defineStore } from 'pinia';
import { useLoading } from '@sa/hooks';
import { fetchGetMenuTree, fetchGetUser, fetchGetUserRoles, fetchLogin, fetchRefreshToken } from '@/service/api';
import { useRouterPush } from '@/hooks/common/router';
import { localStg } from '@/utils/storage';
import { SetupStoreId } from '@/enum';
import { $t } from '@/locales';
import { useRouteStore } from '../route';
import { useTabStore } from '../tab';
import { clearAuthStorage, getToken, isTokenExpiredOrExpiring, getTokenExpireAt } from './shared';

export const useAuthStore = defineStore(SetupStoreId.Auth, () => {
  const route = useRoute();
  const routeStore = useRouteStore();
  const tabStore = useTabStore();
  const { toLogin, redirectFromLogin } = useRouterPush(false);
  const { loading: loginLoading, startLoading, endLoading } = useLoading();

  const token = ref('');

  const userInfo: Api.Auth.UserInfo = reactive({
    userId: '',
    userName: '',
    roles: [],
    buttons: []
  });

  /** Menu tree from backend */
  const menuTree = ref<Api.User.MenuTreeNode[]>([]);

  /** is super role in static route */
  const isStaticSuper = computed(() => {
    const { VITE_AUTH_ROUTE_MODE, VITE_STATIC_SUPER_ROLE } = import.meta.env;

    return VITE_AUTH_ROUTE_MODE === 'static' && userInfo.roles.includes(VITE_STATIC_SUPER_ROLE);
  });

  /** Is login */
  const isLogin = computed(() => Boolean(token.value));

  /** Extract user ID from JWT token */
  function extractUserIdFromToken(tokenStr: string): string {
    try {
      const parts = tokenStr.split('.');
      if (parts.length === 3) {
        const payload = JSON.parse(atob(parts[1].replace(/-/g, '+').replace(/_/g, '/')));
        return payload['x-user-id'] || payload.userId || '';
      }
    } catch {
      // ignore
    }
    return '';
  }

  /** Reset auth store */
  async function resetStore() {
    recordUserId();

    stopTokenRefreshTimer();

    clearAuthStorage();
    localStg.remove('userId');
    localStg.remove('tokenExpireAt');

    // Reset user info
    Object.assign(userInfo, {
      userId: '',
      userName: '',
      roles: [],
      buttons: []
    });

    // Reset menu tree
    menuTree.value = [];

    token.value = '';

    if (!route.meta.constant) {
      await toLogin();
    }

    tabStore.cacheTabs();
    routeStore.resetStore();
  }

  /** Record the user ID of the previous login session */
  function recordUserId() {
    if (!userInfo.userId) {
      return;
    }
    localStg.set('lastLoginUserId', userInfo.userId);
  }

  /**
   * Check if current login user is different from previous login user
   *
   * @returns {boolean} Whether to clear all tabs
   */
  function checkTabClear(): boolean {
    if (!userInfo.userId) {
      return false;
    }

    const lastLoginUserId = localStg.get('lastLoginUserId');

    if (!lastLoginUserId || lastLoginUserId !== userInfo.userId) {
      localStg.remove('globalTabs');
      tabStore.clearTabs();
      localStg.remove('lastLoginUserId');
      return true;
    }

    localStg.remove('lastLoginUserId');
    return false;
  }

  /**
   * Login
   *
   * @param userName User name
   * @param password Password
   * @param [redirect=true] Whether to redirect after login. Default is `true`
   */
  async function login(userName: string, password: string, redirect = true) {
    startLoading();

    try {
      const { data, error } = await fetchLogin(userName, password);

      if (!error && data) {
        // Store tokens
        localStg.set('token', data.accessToken);
        localStg.set('refreshToken', data.refreshToken);
        localStg.set('tokenExpireAt', data.expireAt);

        token.value = data.accessToken;

        // Extract userID from token and fetch user info and roles
        const userId = extractUserIdFromToken(data.accessToken);
        if (userId) {
          await syncUserProfile(userId);
        }

        // Menu tree failure should not break login state
        try {
          await fetchMenuTree();
        } catch {
          // ignore - menu tree is optional for basic functionality
        }

        const isClear = checkTabClear();
        let needRedirect = redirect;

        if (isClear) {
          needRedirect = false;
        }
        await redirectFromLogin(needRedirect);

        startTokenRefreshTimer();

        window.$notification?.success({
          title: $t('page.login.common.loginSuccess'),
          content: $t('page.login.common.welcomeBack', { userName: userInfo.userName }),
          duration: 4500
        });

        return;
      }

      await resetStore();
    } finally {
      endLoading();
    }
  }

  async function syncUserProfile(userId: string) {
    localStg.set('userId', userId);
    await fetchUserInfoById(userId);
    await fetchUserRolesById(userId);
  }

  /** Fetch user info by user ID */
  async function fetchUserInfoById(userId: string) {
    const { data, error } = await fetchGetUser(userId);

    if (!error && data) {
      Object.assign(userInfo, {
        userId: data.user.userID,
        userName: data.user.nickname || data.user.username,
        roles: userInfo.roles,
        buttons: userInfo.buttons
      });
    }
  }

  /** Fetch user roles and permission codes by user ID */
  async function fetchUserRolesById(userId: string) {
    const { data, error } = await fetchGetUserRoles(userId);

    if (!error && data) {
      Object.assign(userInfo, {
        roles: data.roles.map(role => role.code),
        buttons: data.permissionCodes || []
      });
    }
  }

  /** Fetch menu tree */
  async function fetchMenuTree() {
    const { data, error } = await fetchGetMenuTree();

    if (!error && data) {
      menuTree.value = data.menus || [];
    }
  }

  /**
   * Logout
   */
  function logout() {
    resetStore();
  }

  /** Token refresh interval ID for cleanup */
  let tokenRefreshTimer: ReturnType<typeof setInterval> | null = null;

  /** Stop periodic token refresh check */
  function stopTokenRefreshTimer() {
    if (tokenRefreshTimer) {
      clearInterval(tokenRefreshTimer);
      tokenRefreshTimer = null;
    }
  }

  /** Refresh token in background */
  async function refreshTokenInBackground(): Promise<boolean> {
    const refreshToken = localStg.get('refreshToken');

    if (!refreshToken) {
      return false;
    }

    const { error, data } = await fetchRefreshToken();
    if (!error && data) {
      localStg.set('token', data.accessToken);
      localStg.set('refreshToken', data.refreshToken);
      localStg.set('tokenExpireAt', data.expireAt);
      token.value = data.accessToken;
      return true;
    }

    stopTokenRefreshTimer();
    await resetStore();
    return false;
  }

  /** Start periodic token refresh check */
  function startTokenRefreshTimer() {
    stopTokenRefreshTimer();

    const CHECK_INTERVAL_MS = 30 * 1000;

    const checkAndRefresh = async () => {
      if (!token.value) {
        return;
      }

      const expireAt = getTokenExpireAt();
      if (isTokenExpiredOrExpiring(expireAt)) {
        await refreshTokenInBackground();
      }
    };

    setTimeout(checkAndRefresh, 5000);
    tokenRefreshTimer = setInterval(checkAndRefresh, CHECK_INTERVAL_MS);
  }

  async function initUserInfo() {
    const maybeToken = getToken();

    if (!maybeToken) {
      return;
    }

    token.value = maybeToken;

    const userId = localStg.get('userId') || extractUserIdFromToken(maybeToken);

    if (userId) {
      await syncUserProfile(userId);

      try {
        await fetchMenuTree();
      } catch {
        // ignore - menu tree is optional for basic functionality
      }
    }

    if (!userInfo.userId) {
      await resetStore();
    } else {
      startTokenRefreshTimer();
    }
  }

  return {
    token,
    userInfo,
    menuTree,
    isStaticSuper,
    isLogin,
    loginLoading,
    resetStore,
    login,
    logout,
    initUserInfo,
    startTokenRefreshTimer,
    stopTokenRefreshTimer
  };
});
