<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue';
import { NButton, NPopconfirm, NSpace } from 'naive-ui';
import { $t } from '@/locales';
import { fetchCreateUser, fetchDeleteUser, fetchGetUserList, fetchUpdateUser } from '@/service/api/user';
import { useTableOperate } from '@/hooks/common/table';
import UserSearch from './modules/user-search.vue';
import UserOperateDrawer from './modules/user-operate-drawer.vue';
import UserDetailDrawer from './modules/user-detail-drawer.vue';
import UserRoleDrawer from './modules/user-role-drawer.vue';
import { createColumns, type UserTableRow } from './modules/user-table-columns';

defineOptions({
  name: 'SystemManageUser'
});

const columns = ref<NaiveUI.TableColumnCheck[]>([]);
const tableData = ref<UserTableRow[]>([]);
const searchModel = ref({ keyword: '' });
const tableLoading = ref(false);
const operateLoading = ref(false);

const pageSize = ref(10);
const nextPageToken = ref('');
const prevPageTokens = ref<string[]>([]);
const totalCount = ref(0);
const currentPage = ref(1);

const detailDrawerVisible = ref(false);
const detailUserId = ref<string | null>(null);
const roleDrawerVisible = ref(false);
const roleUserId = ref<string | null>(null);
const editingUserId = ref<string | null>(null);

const { drawerVisible, closeDrawer, operateType, editingData, handleAdd, handleEdit, onDeleted } = useTableOperate(
  tableData,
  'userID',
  getData
);

const filteredTableData = computed(() => {
  const keyword = searchModel.value.keyword.trim().toLowerCase();
  if (!keyword) return tableData.value;

  return tableData.value.filter(item => {
    const fields = [item.username, item.nickname, item.email, item.phone];
    return fields.some(field => field?.toLowerCase().includes(keyword));
  });
});

async function getData(pageToken = prevPageTokens.value.at(-1) ?? '') {
  tableLoading.value = true;
  try {
    const res = await fetchGetUserList({
      pageToken: pageToken || undefined,
      pageSize: pageSize.value
    });

    if (!res.error && res.data) {
      tableData.value = res.data.users;
      totalCount.value = res.data.totalCount;
      nextPageToken.value = res.data.nextPageToken;
    }
  } finally {
    tableLoading.value = false;
  }
}

async function handleRefresh() {
  await getData();
}

async function handleSearch() {
  await getData();
}

async function handleReset() {
  searchModel.value.keyword = '';
  await getData();
}

async function handleOperateSubmit(data: Api.User.CreateUserRequest) {
  operateLoading.value = true;

  try {
    if (operateType.value === 'add') {
      const res = await fetchCreateUser(data);
      if (!res.error) {
        window.$message?.success($t('common.addSuccess'));
        closeDrawer();
        await getData();
      }
      return;
    }

    if (!editingUserId.value) return;

    const { password: _password, ...updatePayload } = data;
    const res = await fetchUpdateUser(editingUserId.value, updatePayload);

    if (!res.error) {
      window.$message?.success($t('common.updateSuccess'));
      closeDrawer();
      await getData();
    }
  } finally {
    operateLoading.value = false;
  }
}

function handleViewDetail(user: UserTableRow) {
  detailUserId.value = user.userID;
  detailDrawerVisible.value = true;
}

function handleEditUser(user: UserTableRow) {
  editingUserId.value = user.userID;
  handleEdit(user.userID);
}

function handleAssignRole(user: UserTableRow) {
  roleUserId.value = user.userID;
  roleDrawerVisible.value = true;
}

async function handleDeleteUser(user: UserTableRow) {
  const res = await fetchDeleteUser(user.userID);
  if (!res.error) {
    await onDeleted();
  }
}

async function handlePrevPage() {
  if (currentPage.value <= 1) return;

  prevPageTokens.value.pop();
  currentPage.value -= 1;
  const pageToken = prevPageTokens.value.at(-1) ?? '';
  await getData(pageToken);
}

async function handleNextPage() {
  if (!nextPageToken.value) return;

  prevPageTokens.value.push(nextPageToken.value);
  currentPage.value += 1;
  await getData(nextPageToken.value);
}

const tableColumns = computed(() => {
  const cols = createColumns();
  const actionsCol = cols.find(column => 'key' in column && column.key === 'actions');

  if (!actionsCol) {
    return cols;
  }

  (actionsCol as NaiveUI.DataTableBaseColumn<UserTableRow>).render = (row: UserTableRow) =>
    h(
      NSpace,
      { justify: 'center' },
      {
        default: () => [
          h(
            NButton,
            {
              size: 'small',
              text: true,
              type: 'primary',
              onClick: () => handleViewDetail(row)
            },
            { default: () => $t('page.system-manage.user.actions.detail') }
          ),
          h(
            NButton,
            {
              size: 'small',
              text: true,
              type: 'primary',
              onClick: () => handleEditUser(row)
            },
            { default: () => $t('common.edit') }
          ),
          h(
            NButton,
            {
              size: 'small',
              text: true,
              type: 'primary',
              onClick: () => handleAssignRole(row)
            },
            { default: () => $t('page.system-manage.user.roleModal.button') }
          ),
          h(
            NPopconfirm,
            {
              onPositiveClick: () => handleDeleteUser(row)
            },
            {
              trigger: () =>
                h(
                  NButton,
                  {
                    size: 'small',
                    text: true,
                    type: 'error'
                  },
                  { default: () => $t('common.delete') }
                ),
              default: () => $t('common.confirmDelete')
            }
          )
        ]
      }
    );

  return cols;
});

onMounted(() => {
  getData();
});
</script>

<template>
  <div>
    <NSpace vertical :size="16">
      <UserSearch v-model="searchModel" :loading="tableLoading" @search="handleSearch" @reset="handleReset" />

      <NCard :bordered="false" size="small" class="card-wrapper">
        <NSpace vertical :size="12">
          <TableHeaderOperation
            v-model:columns="columns"
            :loading="tableLoading"
            :disabled-delete="true"
            @add="handleAdd"
            @refresh="handleRefresh"
          />

          <NDataTable
            :columns="tableColumns"
            :data="filteredTableData"
            :loading="tableLoading"
            :scroll-x="1280"
            :pagination="false"
            :bordered="false"
            :single-line="false"
            remote
          />

          <div class="flex flex-wrap items-center justify-between gap-12px">
            <span class="text-14px text-gray-500">{{ $t('datatable.itemCount', { total: totalCount }) }}</span>

            <NSpace>
              <NButton size="small" :disabled="currentPage <= 1 || tableLoading" @click="handlePrevPage">
                {{ $t('page.system-manage.user.pagination.prev') }}
              </NButton>
              <span class="flex items-center text-14px text-gray-500">
                {{ $t('page.system-manage.user.pagination.current', { page: currentPage }) }}
              </span>
              <NButton
                size="small"
                type="primary"
                :disabled="!nextPageToken || tableLoading"
                @click="handleNextPage"
              >
                {{ $t('page.system-manage.user.pagination.next') }}
              </NButton>
            </NSpace>
          </div>
        </NSpace>
      </NCard>
    </NSpace>

    <UserOperateDrawer
      v-model:visible="drawerVisible"
      v-model:operate-type="operateType"
      v-model:editing-user="editingData"
      :loading="operateLoading"
      @submit="handleOperateSubmit"
    />

    <UserDetailDrawer v-model:visible="detailDrawerVisible" :user-id="detailUserId" />
    <UserRoleDrawer v-model:visible="roleDrawerVisible" :user-id="roleUserId" />
  </div>
</template>
