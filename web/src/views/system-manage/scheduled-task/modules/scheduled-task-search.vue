<script setup lang="ts">
import { computed, ref, onMounted } from 'vue';
import { $t } from '@/locales';
import { fetchListTaskDefinitions } from '@/service/api/scheduled-task';

defineOptions({
  name: 'ScheduledTaskSearch'
});

interface SearchModel {
  enabled: string | null;
  taskType: string | null;
}

interface Props {
  loading?: boolean;
  modelValue?: SearchModel;
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: () => ({ enabled: null, taskType: null })
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

// 任务类型列表
const taskDefinitions = ref<Api.ScheduledTask.TaskDefinition[]>([]);

const enabledOptions = [
  { label: $t('common.yes'), value: 'true' },
  { label: $t('common.no'), value: 'false' }
];

// 任务类型选项：从 API 获取
const taskTypeOptions = computed(() =>
  taskDefinitions.value.map(d => ({
    label: d.type,
    value: d.type
  }))
);

// 加载任务类型列表
async function loadTaskDefinitions() {
  try {
    const res = await fetchListTaskDefinitions();
    if (!res.error && res.data?.definitions) {
      taskDefinitions.value = res.data.definitions;
    }
  } catch {
    taskDefinitions.value = [];
  }
}

function handleSearch() {
  emit('search');
}

function handleReset() {
  emit('update:modelValue', { enabled: null, taskType: null });
  emit('reset');
}

onMounted(() => {
  loadTaskDefinitions();
});
</script>

<template>
  <NCard :bordered="false" size="small" class="mb-16px">
    <NGrid :cols="24" responsive="screen" item-responsive :x-gap="12">
      <NGi span="24 s:6 m:3">
        <NSelect
          v-model:value="model.enabled"
          :options="enabledOptions"
          :placeholder="$t('common.pleaseSelect') + $t('page.system-manage.scheduledTask.search.enabled')"
          clearable
          class="w-full"
        />
      </NGi>
      <NGi span="24 s:6 m:3">
        <NSelect
          v-model:value="model.taskType"
          :options="taskTypeOptions"
          :placeholder="$t('common.pleaseSelect') + $t('page.system-manage.scheduledTask.search.taskType')"
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
