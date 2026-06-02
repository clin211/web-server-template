<script setup lang="ts">
import { computed } from 'vue';

defineOptions({
  name: 'PermissionSearch'
});

interface SearchModel {
  status: string | null;
  resourceType: string | null;
}

interface Props {
  loading?: boolean;
  modelValue?: SearchModel;
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: () => ({ status: null, resourceType: null })
});

const emit = defineEmits<{
  'update:modelValue': [value: SearchModel];
  search: [];
  reset: [];
}>();

const model = computed({
  get: () => props.modelValue,
  set: val => emit('update:modelValue', val)
});

const statusOptions = [
  { label: '启用', value: '0' },
  { label: '禁用', value: '1' }
];

const resourceTypeOptions = [
  { label: '菜单权限', value: 'menu' },
  { label: '按钮权限', value: 'button' }
];

async function handleSearch() {
  emit('search');
}

async function handleReset() {
  emit('update:modelValue', { status: null, resourceType: null });
  emit('reset');
}
</script>

<template>
  <NCard :bordered="false" size="small" class="mb-16px">
    <NGrid :cols="24" responsive="screen" item-responsive :x-gap="12">
      <NGi span="24 s:6 m:3">
        <NSelect
          v-model:value="model.status"
          :options="statusOptions"
          :placeholder="$t('common.pleaseSelect') + $t('page.system-manage.permission.form.status')"
          clearable
          class="w-full"
        />
      </NGi>
      <NGi span="24 s:6 m:3">
        <NSelect
          v-model:value="model.resourceType"
          :options="resourceTypeOptions"
          :placeholder="$t('common.pleaseSelect') + $t('page.system-manage.permission.form.resourceType')"
          clearable
          class="w-full"
        />
      </NGi>
      <NGi span="24 s:12 m:18" class="flex justify-end">
        <NSpace>
          <NButton type="primary" :loading="loading" @click="handleSearch">
            <template #icon>
              <icon-mdi-magnify class="text-icon" />
            </template>
            {{ $t('common.search') }}
          </NButton>
          <NButton @click="handleReset">
            <template #icon>
              <icon-mdi-refresh class="text-icon" />
            </template>
            {{ $t('common.reset') }}
          </NButton>
        </NSpace>
      </NGi>
    </NGrid>
  </NCard>
</template>
