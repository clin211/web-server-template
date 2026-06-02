<script setup lang="ts">
import { h, onMounted, ref } from 'vue';
import { NButton, NPopconfirm, NSpace } from 'naive-ui';
import type { DataTableBaseColumn } from 'naive-ui';
import { $t } from '@/locales';
import { fetchDeleteMenu, fetchGetAllMenuTree, fetchGetMenuList } from '@/service/api/menu';
import { useTableOperate } from '@/hooks/common/table';
import { useColumnSetting } from '@/hooks/common/use-column-setting';
import MenuSearch from './modules/menu-search.vue';
import MenuOperateDrawer from './modules/menu-operate-drawer.vue';
import MenuDetailDrawer from './modules/menu-detail-drawer.vue';
import MenuRoleDrawer from './modules/menu-role-drawer.vue';
import { createColumns, type MenuTableRow } from './modules/menu-table-columns';

defineOptions({
  name: 'SystemManageMenu'
});

const searchModel = ref<{ status: string | null; menuType: string | null }>({ status: null, menuType: null });
const tableLoading = ref(false);
const operateLoading = ref(false);

// 菜单树数据（用于树形表格展示）
const menuTreeData = ref<MenuTableRow[]>([]);

const detailDrawerVisible = ref(false);
const detailMenuId = ref<string | null>(null);
const roleDrawerVisible = ref(false);
const roleMenuId = ref<string | null>(null);
const editingMenuId = ref<string | null>(null);

// 使用列设置 hook
const { columnChecks, finalColumns } = useColumnSetting<MenuTableRow>({
  key: 'system-manage-menu',
  columnsFactory: createColumns
});

const { drawerVisible, closeDrawer, operateType, editingData, handleAdd, onDeleted } = useTableOperate(
  menuTreeData,
  'menuID',
  getData
);

// 菜单树数据（用于筛选）
const menuTree = ref<Api.Menu.MenuTreeNode[]>([]);

// 后端过滤：直接使用筛选条件请求数据
async function getData() {
  tableLoading.value = true;
  try {
    const res = await fetchGetMenuList({
      status: searchModel.value.status ? Number(searchModel.value.status) : undefined,
      menuType: searchModel.value.menuType || undefined
    });

    if (!res.error && res.data) {
      // 保留树形结构用于表格展示
      menuTreeData.value = res.data.menus;
    }
  } finally {
    tableLoading.value = false;
  }
}

// 获取菜单树（用于筛选）
async function fetchMenuTreeData() {
  const res = await fetchGetAllMenuTree({ status: 0 });
  if (!res.error && res.data) {
    menuTree.value = res.data.menus;
  }
}

async function handleRefresh() {
  await getData();
}

async function handleSearch() {
  await getData();
}

async function handleReset() {
  searchModel.value = { status: null, menuType: null };
  await getData();
}

async function handleOperateSubmit(data: Api.Menu.CreateMenuRequest | Api.Menu.UpdateMenuRequest) {
  operateLoading.value = true;

  try {
    if (operateType.value === 'add') {
      const createData = data as Api.Menu.CreateMenuRequest;
      const res = await import('@/service/api/menu').then(m => m.fetchCreateMenu(createData));
      if (!res.error) {
        window.$message?.success($t('common.addSuccess'));
        closeDrawer();
        await getData();
      }
      return;
    }

    if (!editingMenuId.value) return;

    const updateData = data as Api.Menu.UpdateMenuRequest;
    const res = await import('@/service/api/menu').then(m => m.fetchUpdateMenu(editingMenuId.value!, updateData));

    if (!res.error) {
      window.$message?.success($t('common.updateSuccess'));
      closeDrawer();
      await getData();
    }
  } finally {
    operateLoading.value = false;
  }
}

function handleViewDetail(menu: MenuTableRow) {
  detailMenuId.value = menu.menuID;
  detailDrawerVisible.value = true;
}

function handleEditMenu(menu: MenuTableRow) {
  editingMenuId.value = menu.menuID;
  // 先设置 operateType 和打开抽屉
  operateType.value = 'edit';
  drawerVisible.value = true;

  // 直接设置 editingData
  editingData.value = menu;

  console.log('[handleEditMenu] set editingData:', menu.menuName, 'parentID:', menu.parentID);
}

function handleAssignRole(menu: MenuTableRow) {
  roleMenuId.value = menu.menuID;
  roleDrawerVisible.value = true;
}

async function handleDeleteMenu(menu: MenuTableRow) {
  const res = await fetchDeleteMenu(menu.menuID);
  if (!res.error) {
    await onDeleted();
  }
}

// 构建带操作列的最终列配置
function getTableColumns() {
  const cols = [...finalColumns.value];
  const actionsCol = cols.find(column => 'key' in column && column.key === 'actions');

  if (!actionsCol) {
    return cols;
  }

  (actionsCol as DataTableBaseColumn<MenuTableRow>).render = (row: MenuTableRow) =>
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
              onClick: () => handleEditMenu(row)
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
            { default: () => $t('page.system-manage.menu.roleDrawer.title') }
          ),
          h(
            NPopconfirm,
            {
              onPositiveClick: () => handleDeleteMenu(row)
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
  fetchMenuTreeData();
});
</script>

<template>
  <div>
    <NSpace vertical :size="16">
      <MenuSearch v-model="searchModel" :loading="tableLoading" @search="handleSearch" @reset="handleReset" />

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
            :data="menuTreeData"
            :loading="tableLoading"
            :scroll-x="1280"
            :pagination="false"
            :bordered="false"
            :single-line="false"
            :row-key="(row: MenuTableRow) => row.menuID"
            tree
            default-expand-all
          />
        </NSpace>
      </NCard>
    </NSpace>

    <MenuOperateDrawer
      v-model:visible="drawerVisible"
      v-model:operate-type="operateType"
      v-model:editing-menu="editingData"
      :loading="operateLoading"
      @submit="handleOperateSubmit"
    />

    <MenuDetailDrawer v-model:visible="detailDrawerVisible" :menu-id="detailMenuId" />
    <MenuRoleDrawer v-model:visible="roleDrawerVisible" :menu-id="roleMenuId" />
  </div>
</template>
