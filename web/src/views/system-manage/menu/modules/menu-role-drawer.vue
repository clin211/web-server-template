<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { $t } from '@/locales';

defineOptions({
  name: 'MenuRoleDrawer'
});

interface Props {
  menuId: string | null;
  visible: boolean;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  'update:visible': [value: boolean];
}>();

const visible = computed({
  get: () => props.visible,
  set: (val: boolean) => emit('update:visible', val)
});

const loading = ref(false);
const submitting = ref(false);
const removingRoleId = ref<string | null>(null);
const allRoles = ref<Api.Role.Role[]>([]);
const checkedRoleIds = ref<string[]>([]);
const assignedRoleIds = ref<string[]>([]);

function getRoleId(role: Api.Role.Role) {
  return role.id || role.roleID;
}

function isAssignedRole(roleId: string) {
  return assignedRoleIds.value.includes(roleId);
}

async function fetchAllRoles() {
  const res = await import('@/service/api/role').then(m => m.fetchGetRoleList({ status: 0 }));
  if (!res.error && res.data) {
    allRoles.value = res.data.roles;
  }
}

async function fetchMenuRoles() {
  const menuId = props.menuId;
  if (!menuId) return;

  const res = await import('@/service/api/menu').then(m => m.fetchGetMenuRoles(menuId));
  if (!res.error && res.data) {
    assignedRoleIds.value = res.data.roleIds;
    checkedRoleIds.value = res.data.roleIds;
  }
}

async function initializeRoleData() {
  if (!props.menuId) return;
  loading.value = true;
  try {
    await Promise.all([fetchAllRoles(), fetchMenuRoles()]);
  } finally {
    loading.value = false;
  }
}

async function handleRemoveRole(roleId: string) {
  if (!props.menuId) return;

  removingRoleId.value = roleId;
  try {
    const res = await import('@/service/api/menu').then(m => m.fetchRemoveMenuRole(props.menuId!, roleId));

    if (!res.error) {
      assignedRoleIds.value = assignedRoleIds.value.filter(id => id !== roleId);
      checkedRoleIds.value = checkedRoleIds.value.filter(id => id !== roleId);
      window.$message?.success($t('common.modifySuccess'));
    }
  } finally {
    removingRoleId.value = null;
  }
}

async function handleSubmit() {
  if (!props.menuId) return;
  submitting.value = true;
  try {
    const res = await import('@/service/api/menu').then(m =>
      m.fetchSetMenuRoles(props.menuId!, { roleIds: checkedRoleIds.value })
    );
    if (!res.error) {
      window.$message?.success($t('common.modifySuccess'));
      visible.value = false;
    }
  } finally {
    submitting.value = false;
  }
}

watch(
  () => props.visible,
  val => {
    if (val) {
      allRoles.value = [];
      checkedRoleIds.value = [];
      assignedRoleIds.value = [];
      initializeRoleData();
    }
  }
);
</script>

<template>
  <NModal v-model:show="visible" :mask-closable="false" preset="card" :title="$t('page.system-manage.menu.roleDrawer.title')" style="width: 500px; max-width: 90vw;">
    <NCard :bordered="false" size="small" class="min-h-200px">
      <NSpin :show="loading">
        <div v-if="allRoles.length" class="max-h-400px overflow-y-auto">
          <NCheckboxGroup v-model:value="checkedRoleIds">
            <NSpace vertical>
              <div v-for="role in allRoles" :key="getRoleId(role)" class="flex items-center justify-between gap-12px">
                <NCheckbox :value="getRoleId(role)" size="large">
                  {{ role.name || role.roleName }}
                </NCheckbox>

                <NSpace v-if="isAssignedRole(getRoleId(role))" size="small" align="center">
                  <NTag size="small" type="success">
                    {{ $t('page.system-manage.menu.roleDrawer.assigned') }}
                  </NTag>

                  <NPopconfirm @positive-click="handleRemoveRole(getRoleId(role))">
                    <template #trigger>
                      <NButton
                        size="small"
                        text
                        type="error"
                        :loading="removingRoleId === getRoleId(role)"
                        :disabled="submitting"
                      >
                        {{ $t('page.system-manage.menu.roleDrawer.remove') }}
                      </NButton>
                    </template>
                    {{ $t('page.system-manage.menu.roleDrawer.removeConfirm') }}
                  </NPopconfirm>
                </NSpace>
              </div>
            </NSpace>
          </NCheckboxGroup>
        </div>
        <div v-else class="py-20px text-center text-gray-400">
          {{ $t('common.noData') }}
        </div>
      </NSpin>
    </NCard>

    <template #footer>
      <NSpace justify="end">
        <NButton :disabled="submitting || !!removingRoleId" @click="visible = false">{{ $t('common.cancel') }}</NButton>
        <NButton type="primary" :loading="submitting" :disabled="!!removingRoleId" @click="handleSubmit">
          {{ $t('common.confirm') }}
        </NButton>
      </NSpace>
    </template>
  </NModal>
</template>
