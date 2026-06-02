<script setup lang="ts">
import { h, onMounted, ref } from 'vue';
import { NButton, NDataTable, NSpace } from 'naive-ui';
import type { DataTableBaseColumn } from 'naive-ui';
import { $t } from '@/locales';
import { fetchGetRoleList } from '@/service/api/role';
import { useColumnSetting } from '@/hooks/common/use-column-setting';
import RolePermissionDrawer from './modules/role-permission-drawer.vue';
import { createColumns, type RoleTableRow } from './modules/role-table-columns';

defineOptions({
  name: 'SystemManageRole'
});

const tableData = ref<RoleTableRow[]>([]);
const tableLoading = ref(false);
const totalCount = ref(0);

const permissionDrawerVisible = ref(false);
const permissionRoleId = ref<string | null>(null);
const permissionRoleName = ref<string | null>(null);

// 使用列设置 hook
const { columnChecks, finalColumns, reloadColumns } = useColumnSetting<RoleTableRow>({
  key: 'system-manage-role',
  columnsFactory: createColumns
});

async function getData() {
  tableLoading.value = true;
  try {
    const res = await fetchGetRoleList({ status: undefined });
    if (!res.error && res.data) {
      tableData.value = res.data.roles;
      totalCount.value = res.data.totalCount;
    }
  } finally {
    tableLoading.value = false;
  }
}

function handleAssignPermission(row: RoleTableRow) {
  permissionRoleId.value = row.roleID;
  permissionRoleName.value = row.roleName;
  permissionDrawerVisible.value = true;
}

// 构建带操作列的最终列配置
function getTableColumns() {
  const cols = [...finalColumns.value];
  const actionsCol = cols.find(column => 'key' in column && column.key === 'actions');

  if (!actionsCol) {
    return cols;
  }

  (actionsCol as DataTableBaseColumn<RoleTableRow>).render = (row: RoleTableRow) =>
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
              onClick: () => handleAssignPermission(row)
            },
            { default: () => $t('common.assign') }
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
      <NCard :bordered="false" size="small" class="card-wrapper">
        <NSpace vertical :size="12">
          <TableHeaderOperation
            v-model:columns="columnChecks"
            :loading="tableLoading"
            :disabled-delete="true"
            @refresh="getData"
          />

          <NDataTable
            :columns="getTableColumns()"
            :data="tableData"
            :loading="tableLoading"
            :scroll-x="960"
            :pagination="false"
            :bordered="false"
            :single-line="false"
          />
        </NSpace>
      </NCard>
    </NSpace>

    <RolePermissionDrawer
      v-model:visible="permissionDrawerVisible"
      :role-id="permissionRoleId"
      :role-name="permissionRoleName || undefined"
      @assigned="getData"
    />
  </div>
</template>