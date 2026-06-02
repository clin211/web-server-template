<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue';
import type { FormInst } from 'naive-ui';
import { useFormRules } from '@/hooks/common/form';
import { $t } from '@/locales';

defineOptions({
  name: 'PermissionOperateDrawer'
});

interface Props {
  loading?: boolean;
}

defineProps<Props>();

const emit = defineEmits<{
  submit: [data: Api.Permission.CreatePermissionRequest | Api.Permission.UpdatePermissionRequest];
}>();

const visible = defineModel<boolean>('visible', { default: false });
const operateType = defineModel<'add' | 'edit'>('operateType', { default: 'add' });
const editingPermission = defineModel<Api.Permission.Permission | null>('editingPermission', { default: null });

const title = computed(() =>
  operateType.value === 'add'
    ? $t('page.system-manage.permission.drawer.addTitle')
    : $t('page.system-manage.permission.drawer.editTitle')
);

const formRef = ref<FormInst | null>(null);
const { createRequiredRule } = useFormRules();

const permissionTree = ref<Api.Permission.PermissionTreeNode[]>([]);
const permissionTreeLoading = ref(false);

const model = ref({
  permissionName: '',
  permissionCode: '',
  resourceType: 'button' as 'menu' | 'button',
  resourcePath: '',
  action: 'GET',
  description: '',
  parentID: null as string | null,
  status: true
});

const rules = computed<Record<string, App.Global.FormRule[]>>(() => ({
  permissionName: [createRequiredRule($t('page.system-manage.permission.form.permissionNameRequired'))],
  permissionCode: [createRequiredRule($t('page.system-manage.permission.form.permissionCodeRequired'))],
  resourceType: [createRequiredRule($t('page.system-manage.permission.form.resourceTypeRequired'))],
  action: [createRequiredRule($t('page.system-manage.permission.form.actionRequired'))]
}));

const resourceTypeOptions = [
  { label: $t('page.system-manage.permission.resourceType.menu'), value: 'menu' },
  { label: $t('page.system-manage.permission.resourceType.button'), value: 'button' }
];

const actionOptions = [
  { label: $t('page.system-manage.permission.action.GET'), value: 'GET' },
  { label: $t('page.system-manage.permission.action.POST'), value: 'POST' },
  { label: $t('page.system-manage.permission.action.PUT'), value: 'PUT' },
  { label: $t('page.system-manage.permission.action.DELETE'), value: 'DELETE' },
  { label: $t('page.system-manage.permission.action.PATCH'), value: 'PATCH' },
  { label: $t('page.system-manage.permission.action.export'), value: 'export' },
  { label: $t('page.system-manage.permission.action.import'), value: 'import' },
  { label: $t('page.system-manage.permission.action.query'), value: 'query' }
];

function getDefaultModel() {
  return {
    permissionName: '',
    permissionCode: '',
    resourceType: 'button' as 'menu' | 'button',
    resourcePath: '',
    action: 'GET',
    description: '',
    parentID: null as string | null,
    status: true
  };
}

function resetForm() {
  model.value = getDefaultModel();
  formRef.value?.restoreValidation();
}

async function fetchPermissionTree() {
  permissionTreeLoading.value = true;
  try {
    const res = await import('@/service/api/permission').then(m => m.fetchGetAllPermissionTree());
    if (!res.error && res.data) {
      function filterCurrentPermission(
        permissions: Api.Permission.PermissionTreeNode[]
      ): Api.Permission.PermissionTreeNode[] {
        return permissions
          .filter(permission => permission.permissionId !== editingPermission.value?.permissionId)
          .map(permission => ({
            ...permission,
            children: permission.children ? filterCurrentPermission(permission.children) : []
          }));
      }

      permissionTree.value = filterCurrentPermission(res.data.permissions);

      if (operateType.value === 'edit' && editingPermission.value) {
        const permission = editingPermission.value;
        const parentId =
          permission.parentId && permission.parentId !== '0' && permission.parentId !== '' ? permission.parentId : null;
        model.value.parentID = parentId;
      }
    }
  } finally {
    permissionTreeLoading.value = false;
  }
}

function syncFormByMode() {
  if (operateType.value === 'add') {
    resetForm();
    return;
  }

  const permission = editingPermission.value;

  if (permission) {
    const parentId =
      permission.parentId && permission.parentId !== '0' && permission.parentId !== '' ? permission.parentId : null;

    model.value = {
      permissionName: permission.permissionName,
      permissionCode: permission.permissionCode,
      resourceType: permission.resourceType as 'menu' | 'button',
      resourcePath: permission.resourcePath || '',
      action: permission.action,
      description: permission.description || '',
      parentID: parentId,
      status: permission.status === 0
    };
  }
  formRef.value?.restoreValidation();
}

watch(
  () => visible.value,
  async show => {
    if (show) {
      await fetchPermissionTree();
      await nextTick();
      syncFormByMode();
    }
  }
);

watch(permissionTree, () => {
  if (visible.value && operateType.value === 'edit' && editingPermission.value) {
    const parentId =
      editingPermission.value.parentId &&
      editingPermission.value.parentId !== '0' &&
      editingPermission.value.parentId !== ''
        ? editingPermission.value.parentId
        : null;
    model.value.parentID = parentId;
  }
});

function closeDrawer() {
  visible.value = false;
}

async function handleSubmit() {
  await formRef.value?.validate();

  if (operateType.value === 'add') {
    const payload: Api.Permission.CreatePermissionRequest = {
      permissionName: model.value.permissionName,
      permissionCode: model.value.permissionCode,
      resourceType: model.value.resourceType,
      resourcePath: model.value.resourcePath || undefined,
      action: model.value.action,
      description: model.value.description || undefined,
      parentID: model.value.parentID || undefined,
      status: model.value.status ? 0 : 1
    };
    emit('submit', payload);
  } else {
    const payload: Api.Permission.UpdatePermissionRequest = {
      permissionName: model.value.permissionName,
      resourceType: model.value.resourceType,
      resourcePath: model.value.resourcePath || undefined,
      action: model.value.action,
      description: model.value.description || undefined,
      parentID: model.value.parentID || undefined,
      status: model.value.status ? 0 : 1
    };
    emit('submit', payload);
  }
}
</script>

<template>
  <NDrawer v-model:show="visible" :width="500">
    <NDrawerContent :title="title" closable :native-scrollbar="false">
      <NForm ref="formRef" :model="model" :rules="rules" label-placement="top">
        <NFormItem :label="$t('page.system-manage.permission.form.permissionName')" path="permissionName">
          <NInput v-model:value="model.permissionName" />
        </NFormItem>

        <NFormItem :label="$t('page.system-manage.permission.form.permissionCode')" path="permissionCode">
          <NInput v-model:value="model.permissionCode" :disabled="operateType === 'edit'" />
        </NFormItem>

        <NFormItem :label="$t('page.system-manage.permission.form.resourceType')" path="resourceType">
          <NSelect v-model:value="model.resourceType" :options="resourceTypeOptions" />
        </NFormItem>

        <NFormItem :label="$t('page.system-manage.permission.form.resourcePath')" path="resourcePath">
          <NInput v-model:value="model.resourcePath" placeholder="/system/user/list" />
        </NFormItem>

        <NFormItem :label="$t('page.system-manage.permission.form.action')" path="action">
          <NSelect v-model:value="model.action" :options="actionOptions" />
        </NFormItem>

        <NFormItem :label="$t('page.system-manage.permission.form.parentID')" path="parentID">
          <NTreeSelect
            v-model:value="model.parentID"
            :options="permissionTree"
            :loading="permissionTreeLoading"
            key-field="permissionId"
            label-field="permissionName"
            children-field="children"
            clearable
            show-checked-strategy="parent-first"
            filterable
          />
        </NFormItem>

        <NFormItem :label="$t('page.system-manage.permission.form.description')" path="description">
          <NInput v-model:value="model.description" type="textarea" :rows="3" />
        </NFormItem>

        <NFormItem :label="$t('page.system-manage.permission.form.status')" path="status">
          <NSwitch v-model:value="model.status" />
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
