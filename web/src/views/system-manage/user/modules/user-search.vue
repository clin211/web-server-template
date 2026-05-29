<script setup lang="ts">
import { computed } from 'vue';

defineOptions({
  name: 'UserSearch'
});

interface SearchModel {
  keyword: string;
}

interface Props {
  loading?: boolean;
  modelValue?: SearchModel;
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: () => ({ keyword: '' })
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

async function handleSearch() {
  emit('search');
}

async function handleReset() {
  emit('update:modelValue', { keyword: '' });
  emit('reset');
}
</script>

<template>
  <NCard :bordered="false" size="small" class="mb-16px">
    <NGrid :cols="24" responsive="screen" item-responsive>
      <NGi span="24 s:12 m:10">
        <NInput
          v-model:value="model.keyword"
          :placeholder="$t('common.keywordSearch')"
          clearable
          @keyup.enter="handleSearch"
        >
          <template #prefix>
            <icon-mdi-magnify class="mr-4px text-16px text-gray-400" />
          </template>
        </NInput>
      </NGi>
      <NGi span="24 s:24 m:14" class="flex justify-end">
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
