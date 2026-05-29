import { computed, onMounted, ref } from 'vue';
import { fetchGetRoleList, fetchGetUserList } from '@/service/api';

type UseUserTableOptions = {
  immediate?: boolean;
  pageSize?: number;
  rolePageSize?: number;
};

export function useUserTable(options: UseUserTableOptions = {}) {
  const { immediate = true, pageSize: initialPageSize = 10, rolePageSize = 100 } = options;

  const items = ref<Api.User.User[]>([]);
  const totalCount = ref(0);
  const loading = ref(false);
  const roleLoading = ref(false);
  const currentPage = ref(1);
  const pageSize = ref(initialPageSize);
  const nextPageToken = ref('');
  const pageTokens = ref<string[]>(['']);
  const roleOptions = ref<Api.Role.RoleOption[]>([]);

  const hasNextPage = computed(() => Boolean(nextPageToken.value));

  async function loadRoles() {
    roleLoading.value = true;

    try {
      const allRoles: Api.Role.Role[] = [];
      let pageToken = '';

      do {
        const { data, error } = await fetchGetRoleList({
          pageToken,
          pageSize: rolePageSize,
          status: 0
        });

        if (error || !data) {
          return;
        }

        allRoles.push(...data.roles);
        pageToken = data.nextPageToken;
      } while (pageToken);

      roleOptions.value = allRoles.map(role => ({
        label: role.roleName,
        value: role.roleID,
        roleCode: role.roleCode,
        roleName: role.roleName,
        status: role.status
      }));
    } finally {
      roleLoading.value = false;
    }
  }

  async function loadPage(page: number = currentPage.value) {
    const pageToken = pageTokens.value[page - 1] ?? '';

    loading.value = true;

    try {
      const { data, error } = await fetchGetUserList({
        pageToken,
        pageSize: pageSize.value
      });

      if (error || !data) {
        return;
      }

      items.value = data.users;
      totalCount.value = data.totalCount;
      nextPageToken.value = data.nextPageToken;
      currentPage.value = page;

      if (data.nextPageToken) {
        pageTokens.value[page] = data.nextPageToken;
      } else {
        pageTokens.value = pageTokens.value.slice(0, page);
      }
    } finally {
      loading.value = false;
    }
  }

  async function refresh() {
    currentPage.value = 1;
    nextPageToken.value = '';
    pageTokens.value = [''];

    await Promise.all([loadRoles(), loadPage(1)]);
  }

  async function changePage(page: number) {
    if (page < 1) {
      return;
    }

    if (page > currentPage.value + 1) {
      return;
    }

    if (page > currentPage.value && !hasNextPage.value) {
      return;
    }

    await loadPage(page);
  }

  async function loadMore() {
    if (!hasNextPage.value) {
      return;
    }

    await changePage(currentPage.value + 1);
  }

  async function changePageSize(size: number) {
    pageSize.value = size;
    await refresh();
  }

  onMounted(async () => {
    if (immediate) {
      await refresh();
    }
  });

  return {
    items,
    totalCount,
    loading,
    roleLoading,
    roleOptions,
    currentPage,
    pageSize,
    nextPageToken,
    hasNextPage,
    refresh,
    loadMore,
    changePage,
    changePageSize
  };
}
