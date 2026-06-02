<script setup lang="ts">
import { computed } from 'vue';

defineOptions({
  name: 'MenuSearch'
});

interface SearchModel {
  status: string | null;
  menuType: string | null;
}

interface Props {
  loading?: boolean;
  modelValue?: SearchModel;
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: () => ({ status: null, menuType: null })
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

const menuTypeOptions = [
  { label: '目录', value: 'menu' },
  { label: '页面', value: 'page' }
];

async function handleSearch() {
  emit('search');
}

async function handleReset() {
  emit('update:modelValue', { status: null, menuType: null });
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
          :placeholder="$t('common.pleaseSelect') + $t('page.system-manage.menu.form.status')"
          clearable
          class="w-full"
        />
      </NGi>
      <NGi span="24 s:6 m:3">
        <NSelect
          v-model:value="model.menuType"
          :options="menuTypeOptions"
          :placeholder="$t('common.pleaseSelect') + $t('page.system-manage.menu.form.menuType')"
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
