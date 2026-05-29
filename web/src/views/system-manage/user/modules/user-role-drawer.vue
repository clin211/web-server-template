<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { $t } from '@/locales';

defineOptions({
  name: 'UserRoleDrawer'
});

interface Props {
  userId: string | null;
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

async function fetchUserRoles() {
  const userId = props.userId;
  if (!userId) return;

  const res = await import('@/service/api/user').then(m => m.fetchGetUserRoles(userId));
  if (!res.error && res.data) {
    const roleIds = res.data.roles.map(role => role.id || role.roleID);
    assignedRoleIds.value = roleIds;
    checkedRoleIds.value = roleIds;
  }
}

async function initializeRoleData() {
  if (!props.userId) return;
  loading.value = true;
  try {
    await Promise.all([fetchAllRoles(), fetchUserRoles()]);
  } finally {
    loading.value = false;
  }
}

async function handleRemoveRole(roleId: string) {
  if (!props.userId) return;

  removingRoleId.value = roleId;
  try {
    const res = await import('@/service/api/user').then(m => m.fetchRemoveUserRole(props.userId!, roleId));

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
  if (!props.userId) return;
  submitting.value = true;
  try {
    const res = await import('@/service/api/user').then(m =>
      m.fetchAssignUserRoles(props.userId!, checkedRoleIds.value)
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
  <NDrawer v-model:show="visible" :width="500">
    <NDrawerContent
      :title="$t('page.system-manage.user.roleModal.title')"
      closable
      :native-scrollbar="false"
    >
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
                    {{ $t('page.system-manage.user.roleModal.assigned') }}
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
                        {{ $t('page.system-manage.user.roleModal.remove') }}
                      </NButton>
                    </template>
                    {{ $t('page.system-manage.user.roleModal.removeConfirm') }}
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

      <template #footer>
        <NSpace justify="end">
          <NButton :disabled="submitting || !!removingRoleId" @click="visible = false">{{ $t('common.cancel') }}</NButton>
          <NButton type="primary" :loading="submitting" :disabled="!!removingRoleId" @click="handleSubmit">
            {{ $t('common.confirm') }}
          </NButton>
        </NSpace>
      </template>
    </NDrawerContent>
  </NDrawer>
</template>
