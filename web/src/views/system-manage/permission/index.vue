<script setup lang="ts">
import { h, onMounted, ref } from 'vue';
import { NButton, NPopconfirm, NSelect, NSpace } from 'naive-ui';
import type { DataTableBaseColumn, SelectOption } from 'naive-ui';
import { $t } from '@/locales';
import { fetchDeletePermission, fetchGetPermissionList } from '@/service/api/permission';
import { useTableOperate } from '@/hooks/common/table';
import { useColumnSetting } from '@/hooks/common/use-column-setting';
import PermissionSearch from './modules/permission-search.vue';
import PermissionOperateDrawer from './modules/permission-operate-drawer.vue';
import { createColumns, type PermissionTableRow } from './modules/permission-table-columns';

defineOptions({
  name: 'SystemManagePermission'
});

// 每页条数选项
const PAGE_SIZE_OPTIONS: SelectOption[] = [
  { label: '10条', value: 10 },
  { label: '20条', value: 20 },
  { label: '50条', value: 50 },
  { label: '100条', value: 100 }
];

const tableData = ref<PermissionTableRow[]>([]);
const searchModel = ref<{ status: string | null; resourceType: string | null }>({
  status: null,
  resourceType: null
});
const tableLoading = ref(false);
const operateLoading = ref(false);

// 分页相关
const pageSize = ref(PAGE_SIZE_OPTIONS[1].value);
const nextPageToken = ref('');
const prevPageTokens = ref<string[]>([]);
const totalCount = ref(0);
const currentPage = ref(1);

const editingPermissionId = ref<string | null>(null);

// 使用列设置 hook
const { columnChecks, finalColumns } = useColumnSetting<PermissionTableRow>({
  key: 'system-manage-permission',
  columnsFactory: createColumns
});

const {
  drawerVisible,
  closeDrawer,
  operateType,
  editingData,
  handleAdd,
  handleEdit,
  onDeleted
} = useTableOperate(tableData, 'permissionId', getData);

async function getData(pageToken = prevPageTokens.value.at(-1) ?? '') {
  tableLoading.value = true;
  try {
    const res = await fetchGetPermissionList({
      pageToken: pageToken || undefined,
      pageSize: Number(pageSize.value),
      status: searchModel.value.status ? Number(searchModel.value.status) : undefined,
      resourceType: searchModel.value.resourceType || undefined
    });

    if (!res.error && res.data) {
      tableData.value = res.data.permissions;
      totalCount.value = res.data.totalCount;
      nextPageToken.value = res.data.pageToken;
    }
  } finally {
    tableLoading.value = false;
  }
}

async function handleRefresh() {
  await getData();
}

async function handleSearch() {
  prevPageTokens.value = [];
  currentPage.value = 1;
  await getData();
}

async function handleReset() {
  searchModel.value = { status: null, resourceType: null };
  prevPageTokens.value = [];
  currentPage.value = 1;
  await getData();
}

async function handlePageSizeChange(val: number) {
  pageSize.value = val;
  prevPageTokens.value = [];
  currentPage.value = 1;
  await getData();
}

async function handleOperateSubmit(
  data: Api.Permission.CreatePermissionRequest | Api.Permission.UpdatePermissionRequest
) {
  operateLoading.value = true;

  try {
    if (operateType.value === 'add') {
      const { fetchCreatePermission } = await import('@/service/api/permission');
      const res = await fetchCreatePermission(data as Api.Permission.CreatePermissionRequest);
      if (!res.error) {
        window.$message?.success($t('common.addSuccess'));
        closeDrawer();
        await getData();
      }
      return;
    }

    if (!editingPermissionId.value) return;

    const { fetchUpdatePermission } = await import('@/service/api/permission');
    const res = await fetchUpdatePermission(
      editingPermissionId.value,
      data as Api.Permission.UpdatePermissionRequest
    );

    if (!res.error) {
      window.$message?.success($t('common.updateSuccess'));
      closeDrawer();
      await getData();
    }
  } finally {
    operateLoading.value = false;
  }
}

function handleEditPermission(permission: PermissionTableRow) {
  editingPermissionId.value = permission.permissionId;
  handleEdit(permission.permissionId);
}

async function handleDeletePermission(permission: PermissionTableRow) {
  const res = await fetchDeletePermission(permission.permissionId);
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

// 构建带操作列的最终列配置
function getTableColumns() {
  const cols = [...finalColumns.value];
  const actionsCol = cols.find(column => 'key' in column && column.key === 'actions');

  if (!actionsCol) {
    return cols;
  }

  (actionsCol as DataTableBaseColumn<PermissionTableRow>).render = (row: PermissionTableRow) =>
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
              onClick: () => handleEditPermission(row)
            },
            { default: () => $t('common.edit') }
          ),
          h(
            NPopconfirm,
            {
              onPositiveClick: () => handleDeletePermission(row)
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
}

onMounted(() => {
  getData();
});
</script>

<template>
  <div>
    <NSpace vertical :size="16">
      <PermissionSearch
        v-model="searchModel"
        :loading="tableLoading"
        @search="handleSearch"
        @reset="handleReset"
      />

      <NCard :bordered="false" size="small" class="card-wrapper">
        <NSpace vertical :size="12">
          <TableHeaderOperation
            v-model:columns="columnChecks"
            :loading="tableLoading"
            :disabled-delete="true"
            @add="handleAdd"
            @refresh="handleRefresh"
          />

          <NDataTable
            :columns="getTableColumns()"
            :data="tableData"
            :loading="tableLoading"
            :scroll-x="1280"
            :pagination="false"
            :bordered="false"
            :single-line="false"
            remote
          />

          <div class="flex flex-wrap items-center justify-between gap-12px">
            <span class="text-14px text-gray-500">{{ $t('datatable.itemCount', { total: totalCount }) }}</span>

            <NSpace align="center" :size="12">
              <NSelect
                v-model:value="pageSize"
                :options="PAGE_SIZE_OPTIONS"
                size="small"
                style="width: 100px"
                @update:value="handlePageSizeChange"
              />

              <NButton size="small" :disabled="currentPage <= 1 || tableLoading" @click="handlePrevPage">
                {{ $t('page.system-manage.user.pagination.prev') }}
              </NButton>
              <span class="flex items-center text-14px text-gray-500">
                {{ $t('page.system-manage.user.pagination.current', { page: currentPage }) }}
              </span>
              <NButton size="small" type="primary" :disabled="!nextPageToken || tableLoading" @click="handleNextPage">
                {{ $t('page.system-manage.user.pagination.next') }}
              </NButton>
            </NSpace>
          </div>
        </NSpace>
      </NCard>
    </NSpace>

    <PermissionOperateDrawer
      v-model:visible="drawerVisible"
      v-model:operate-type="operateType"
      v-model:editing-permission="editingData"
      :loading="operateLoading"
      @submit="handleOperateSubmit"
    />
  </div>
</template>