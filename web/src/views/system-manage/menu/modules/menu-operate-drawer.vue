<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue';
import type { FormInst } from 'naive-ui';
import { useFormRules } from '@/hooks/common/form';
import { $t } from '@/locales';

defineOptions({
  name: 'MenuOperateDrawer'
});

interface Props {
  loading?: boolean;
}

defineProps<Props>();

const emit = defineEmits<{
  submit: [data: Api.Menu.CreateMenuRequest | Api.Menu.UpdateMenuRequest];
}>();

const visible = defineModel<boolean>('visible', { default: false });
const operateType = defineModel<'add' | 'edit'>('operateType', { default: 'add' });
const editingMenu = defineModel<Api.Menu.Menu | null>('editingMenu', { default: null });

const title = computed(() =>
  operateType.value === 'add'
    ? $t('page.system-manage.menu.drawer.addTitle')
    : $t('page.system-manage.menu.drawer.editTitle')
);

const formRef = ref<FormInst | null>(null);
const { createRequiredRule } = useFormRules();

const menuTree = ref<Api.Menu.MenuTreeNode[]>([]);
const menuTreeLoading = ref(false);

const model = ref({
  menuName: '',
  menuCode: '',
  menuType: 'menu' as 'menu' | 'page',
  i18nKey: '',
  parentID: null as string | null,
  path: '',
  component: '',
  icon: '',
  localIcon: '',
  sortOrder: 0,
  visible: true,
  status: true,
  constant: false,
  hideInMenu: false,
  keepAlive: false,
  href: ''
});

const rules = computed<Record<string, App.Global.FormRule[]>>(() => ({
  menuName: [createRequiredRule($t('page.system-manage.menu.form.menuNameRequired'))],
  menuCode: [createRequiredRule($t('page.system-manage.menu.form.menuCodeRequired'))],
  menuType: [createRequiredRule($t('page.system-manage.menu.form.menuTypeRequired'))]
}));

const menuTypeOptions = [
  { label: $t('page.system-manage.menu.type.directory'), value: 'menu' },
  { label: $t('page.system-manage.menu.type.page'), value: 'page' }
];

function getDefaultModel() {
  return {
    menuName: '',
    menuCode: '',
    menuType: 'menu' as 'menu' | 'page',
    i18nKey: '',
    parentID: null as string | null,
    path: '',
    component: '',
    icon: '',
    localIcon: '',
    sortOrder: 0,
    visible: true,
    status: true,
    constant: false,
    hideInMenu: false,
    keepAlive: false,
    href: ''
  };
}

function resetForm() {
  model.value = getDefaultModel();
  formRef.value?.restoreValidation();
}

async function fetchMenuTree() {
  menuTreeLoading.value = true;
  try {
    const res = await import('@/service/api/menu').then(m => m.fetchGetAllMenuTree());
    console.log('[fetchMenuTree] raw response:', res);
    console.log('[fetchMenuTree] res.data:', res.data);
    if (!res.error && res.data) {
      // 保留完整的树形结构，用于 NTreeSelect 的层级显示
      // 但需要过滤掉当前编辑的菜单及其子菜单，避免循环引用
      function filterCurrentMenu(menus: Api.Menu.MenuTreeNode[]): Api.Menu.MenuTreeNode[] {
        return menus
          .filter(menu => menu.menuID !== editingMenu.value?.menuID)
          .map(menu => ({
            ...menu,
            children: menu.children ? filterCurrentMenu(menu.children) : []
          }));
      }

      menuTree.value = filterCurrentMenu(res.data.menus);

      // 在菜单树加载完成后，立即同步 parentID 以确保正确选中
      if (operateType.value === 'edit' && editingMenu.value) {
        const menu = editingMenu.value;
        const parentId = menu.parentID && menu.parentID !== '0' && menu.parentID !== '' ? menu.parentID : null;
        model.value.parentID = parentId;
      }

      console.log(
        '[fetchMenuTree] menuTree:',
        menuTree.value.map(m => ({ name: m.menuName, id: m.menuID }))
      );
    }
  } finally {
    menuTreeLoading.value = false;
  }
}

function syncFormByMode() {
  if (operateType.value === 'add') {
    resetForm();
    return;
  }

  const menu = editingMenu.value;
  console.log('[syncFormByMode] editingMenu:', menu?.menuName, menu?.menuID, 'parentID:', menu?.parentID);

  if (menu) {
    // 处理 parentID: 顶级菜单 parentID 为 "0" 或空时设为 null
    const parentId = menu.parentID && menu.parentID !== '0' && menu.parentID !== '' ? menu.parentID : null;

    console.log('[syncFormByMode] parentId after processing:', parentId);

    model.value = {
      menuName: menu.menuName,
      menuCode: menu.menuCode,
      menuType: menu.menuType as 'menu' | 'page',
      i18nKey: menu.i18nKey || '',
      parentID: parentId,
      path: menu.path || '',
      component: menu.component || '',
      icon: menu.icon || '',
      localIcon: menu.localIcon || '',
      sortOrder: menu.sortOrder || 0,
      visible: menu.visible === 1,
      status: menu.status === 1,
      constant: menu.constant === 1,
      hideInMenu: menu.hideInMenu === 1,
      keepAlive: menu.keepAlive === 1,
      href: menu.href || ''
    };

    console.log('[syncFormByMode] model.parentID:', model.value.parentID);
  }
  formRef.value?.restoreValidation();
}

watch(
  () => visible.value,
  async show => {
    if (show) {
      // 先获取菜单树
      await fetchMenuTree();
      // 菜单树获取完成后，再同步表单数据
      // 确保 editingMenu.value 已经正确设置
      await nextTick();
      syncFormByMode();
    }
  }
);

// 监听 menuTree 变化，确保 parentID 能正确选中
watch(menuTree, () => {
  if (visible.value && operateType.value === 'edit' && editingMenu.value) {
    // menuTree 加载完成后，重新设置 parentID
    const menu = editingMenu.value;
    const parentId = menu.parentID && menu.parentID !== '0' && menu.parentID !== '' ? menu.parentID : null;
    model.value.parentID = parentId;
  }
});

function closeDrawer() {
  visible.value = false;
}

async function handleSubmit() {
  await formRef.value?.validate();

  if (operateType.value === 'add') {
    const payload: Api.Menu.CreateMenuRequest = {
      menuName: model.value.menuName,
      menuCode: model.value.menuCode,
      menuType: model.value.menuType,
      i18nKey: model.value.i18nKey || undefined,
      parentID: model.value.parentID || undefined,
      path: model.value.path || undefined,
      component: model.value.component || undefined,
      icon: model.value.icon || undefined,
      localIcon: model.value.localIcon || undefined,
      sortOrder: model.value.sortOrder,
      visible: model.value.visible ? 1 : 0,
      constant: model.value.constant ? 1 : 0,
      hideInMenu: model.value.hideInMenu ? 1 : 0,
      keepAlive: model.value.keepAlive ? 1 : 0,
      href: model.value.href || undefined
    };
    emit('submit', payload);
  } else {
    const payload: Api.Menu.UpdateMenuRequest = {
      menuName: model.value.menuName,
      i18nKey: model.value.i18nKey || undefined,
      parentID: model.value.parentID || undefined,
      path: model.value.path || undefined,
      component: model.value.component || undefined,
      icon: model.value.icon || undefined,
      localIcon: model.value.localIcon || undefined,
      sortOrder: model.value.sortOrder,
      visible: model.value.visible ? 1 : 0,
      status: model.value.status ? 1 : 0,
      constant: model.value.constant ? 1 : 0,
      hideInMenu: model.value.hideInMenu ? 1 : 0,
      keepAlive: model.value.keepAlive ? 1 : 0,
      href: model.value.href || undefined
    };
    emit('submit', payload);
  }
}
</script>

<template>
  <NDrawer v-model:show="visible" :width="500">
    <NDrawerContent :title="title" closable :native-scrollbar="false">
      <NForm ref="formRef" :model="model" :rules="rules" label-placement="top">
        <NFormItem :label="$t('page.system-manage.menu.form.menuName')" path="menuName">
          <NInput v-model:value="model.menuName" />
        </NFormItem>

        <NFormItem :label="$t('page.system-manage.menu.form.menuCode')" path="menuCode">
          <NInput v-model:value="model.menuCode" :disabled="operateType === 'edit'" />
        </NFormItem>

        <NFormItem :label="$t('page.system-manage.menu.form.menuType')" path="menuType">
          <NSelect v-model:value="model.menuType" :options="menuTypeOptions" />
        </NFormItem>

        <NFormItem :label="$t('page.system-manage.menu.form.i18nKey')" path="i18nKey">
          <NInput v-model:value="model.i18nKey" placeholder="route.xxx" />
        </NFormItem>

        <NFormItem :label="$t('page.system-manage.menu.form.parentID')" path="parentID">
          <NTreeSelect
            v-model:value="model.parentID"
            :options="menuTree"
            :loading="menuTreeLoading"
            key-field="menuID"
            label-field="menuName"
            children-field="children"
            clearable
            show-checked-strategy="parent-first"
            filterable
          />
        </NFormItem>

        <NFormItem :label="$t('page.system-manage.menu.form.path')" path="path">
          <NInput v-model:value="model.path" />
        </NFormItem>

        <NFormItem :label="$t('page.system-manage.menu.form.component')" path="component">
          <NInput v-model:value="model.component" />
        </NFormItem>

        <NFormItem :label="$t('page.system-manage.menu.form.icon')" path="icon">
          <NInput v-model:value="model.icon" placeholder="mdi:xxx" />
        </NFormItem>

        <NFormItem :label="$t('page.system-manage.menu.form.localIcon')" path="localIcon">
          <NInput v-model:value="model.localIcon" />
        </NFormItem>

        <NFormItem :label="$t('page.system-manage.menu.form.sortOrder')" path="sortOrder">
          <NInputNumber v-model:value="model.sortOrder" :min="0" :max="9999" class="w-full" />
        </NFormItem>

        <NFormItem :label="$t('page.system-manage.menu.form.visible')" path="visible">
          <NSwitch v-model:value="model.visible" />
        </NFormItem>

        <NFormItem :label="$t('page.system-manage.menu.form.status')" path="status">
          <NSwitch v-model:value="model.status" />
        </NFormItem>

        <NFormItem :label="$t('page.system-manage.menu.form.constant')" path="constant">
          <NSwitch v-model:value="model.constant" />
        </NFormItem>

        <NFormItem :label="$t('page.system-manage.menu.form.hideInMenu')" path="hideInMenu">
          <NSwitch v-model:value="model.hideInMenu" />
        </NFormItem>

        <NFormItem :label="$t('page.system-manage.menu.form.keepAlive')" path="keepAlive">
          <NSwitch v-model:value="model.keepAlive" />
        </NFormItem>

        <NFormItem :label="$t('page.system-manage.menu.form.href')" path="href">
          <NInput v-model:value="model.href" />
        </NFormItem>
      </NForm>

      <template #footer>
        <NSpace justify="end">
          <NButton @click="closeDrawer">{{ $t('common.cancel') }}</NButton>
          <NButton type="primary" :loading="loading" @click="handleSubmit">{{ $t('common.confirm') }}</NButton>
        </NSpace>
      </template>
    </NDrawerContent>
  </NDrawer>
</template>
