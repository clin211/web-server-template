<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { NButton, NCheckbox, NSpace, NSpin, NSwitch, NTree } from 'naive-ui';
import type { TreeOption } from 'naive-ui';
import { $t } from '@/locales';
import { fetchAssignPermissionsToRole, fetchGetRolePermissions } from '@/service/api/role';

defineOptions({
  name: 'RolePermissionDrawer'
});

interface Props {
  roleId: string | null;
  roleName?: string;
  visible: boolean;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  'update:visible': [value: boolean];
  assigned: [];
}>();

const visible = computed({
  get: () => props.visible,
  set: (val: boolean) => emit('update:visible', val)
});

const loading = ref(false);
const submitting = ref(false);

// 权限树数据
const permissionTreeData = ref<TreeOption[]>([]);

// 已分配的权限ID集合（用于记录原始状态）
const assignedPermissionIds = ref<Set<string>>(new Set());

// 当前选中的权限ID集合
const checkedPermissionIds = ref<string[]>([]);

// 模式切换（覆盖/追加）
const appendMode = ref(false);

function buildTreeOptions(nodes: Api.Role.PermissionTreeNode[]): TreeOption[] {
  return nodes.map(node => ({
    key: node.permissionID,
    label: node.permissionName,
    children: node.children?.length ? buildTreeOptions(node.children) : undefined,
    disabled: false
  }));
}

async function fetchPermissions() {
  if (!props.roleId) return;

  loading.value = true;
  try {
    const res = await fetchGetRolePermissions(props.roleId);
    if (!res.error && res.data) {
      // 构建树形选项
      permissionTreeData.value = buildTreeOptions(res.data.permissions);

      // 收集已分配的权限ID
      const assignedIds = new Set<string>();
      const checkedIds: string[] = [];

      function collectAssignedIds(nodes: Api.Role.PermissionTreeNode[]) {
        for (const node of nodes) {
          if (node.assigned) {
            assignedIds.add(node.permissionID);
            checkedIds.push(node.permissionID);
          }
          if (node.children?.length) {
            collectAssignedIds(node.children);
          }
        }
      }

      collectAssignedIds(res.data.permissions);
      assignedPermissionIds.value = assignedIds;
      checkedPermissionIds.value = checkedIds;
    }
  } finally {
    loading.value = false;
  }
}

// 获取所有叶子节点的权限ID（用于全选/取消全选）
function getAllLeafPermissionIds(permissionNodes: Api.Role.PermissionTreeNode[]): string[] {
  const ids: string[] = [];
  function collect(nodes: Api.Role.PermissionTreeNode[]) {
    for (const node of nodes) {
      if (!node.children?.length) {
        ids.push(node.permissionID);
      } else {
        collect(node.children);
      }
    }
  }
  collect(permissionNodes);
  return ids;
}

// 全选
function handleSelectAll() {
  const allLeafIds = getAllLeafPermissionIds(permissionTreeData.value as unknown as Api.Role.PermissionTreeNode[]);
  checkedPermissionIds.value = allLeafIds;
}

// 取消全选
function handleUnselectAll() {
  checkedPermissionIds.value = [];
}

// 统计已选数量
const selectedCount = computed(() => checkedPermissionIds.value.length);

// 提交分配
async function handleSubmit() {
  if (!props.roleId) return;

  submitting.value = true;
  try {
    const res = await fetchAssignPermissionsToRole(props.roleId, {
      permissionIDs: checkedPermissionIds.value,
      mode: appendMode.value ? 'append' : 'override'
    });

    if (!res.error) {
      window.$message?.success($t('common.modifySuccess'));
      emit('assigned');
      visible.value = false;
    }
  } finally {
    submitting.value = false;
  }
}

// 处理树节点勾选变化
function handleCheckedKeys(keys: string[]) {
  checkedPermissionIds.value = keys;
}

watch(
  () => props.visible,
  val => {
    if (val) {
      // 重置状态
      permissionTreeData.value = [];
      assignedPermissionIds.value = new Set();
      checkedPermissionIds.value = [];
      appendMode.value = false;
      fetchPermissions();
    }
  }
);
</script>

<template>
  <NModal
    v-model:show="visible"
    :mask-closable="false"
    preset="card"
    :title="`${$t('page.system-manage.role.permissionDrawer.title')}${roleName ? ` - ${roleName}` : ''}`"
    style="width: 600px; max-width: 90vw"
  >
    <NCard :bordered="false" size="small" class="min-h-300px">
      <NSpin :show="loading">
        <!-- 操作栏 -->
        <div class="flex items-center justify-between mb-16px">
          <div class="flex items-center gap-12px">
            <NCheckbox
              :checked="checkedPermissionIds.length === permissionTreeData.length && permissionTreeData.length > 0"
              :indeterminate="
                checkedPermissionIds.length > 0 && checkedPermissionIds.length < permissionTreeData.length
              "
              @update:checked="(val: boolean) => (val ? handleSelectAll() : handleUnselectAll())"
            >
              {{ $t('common.selectAll') }}
            </NCheckbox>
            <span class="text-14px text-gray-500">
              {{ $t('page.system-manage.role.permissionDrawer.selectedCount', { count: selectedCount }) }}
            </span>
          </div>

          <div class="flex items-center gap-8px">
            <span class="text-14px">{{ $t('page.system-manage.role.permissionDrawer.appendMode') }}:</span>
            <NSwitch v-model:value="appendMode" />
          </div>
        </div>

        <!-- 权限树 -->
        <div class="max-h-400px overflow-y-auto border border-gray-200 rounded-8px p-12px">
          <NTree
            v-if="permissionTreeData.length"
            :data="permissionTreeData"
            :checked-keys="checkedPermissionIds"
            :default-expand-all="true"
            checkable
            selectable
            expand-on-click
            virtual-scroll
            block-line
            @update:checked-keys="handleCheckedKeys"
          />
          <div v-else class="py-40px text-center text-gray-400">
            {{ $t('common.noData') }}
          </div>
        </div>
      </NSpin>
    </NCard>

    <template #footer>
      <NSpace justify="end">
        <NButton :disabled="submitting" @click="visible = false">{{ $t('common.cancel') }}</NButton>
        <NButton type="primary" :loading="submitting" @click="handleSubmit">
          {{ $t('common.confirm') }}
        </NButton>
      </NSpace>
    </template>
  </NModal>
</template>
