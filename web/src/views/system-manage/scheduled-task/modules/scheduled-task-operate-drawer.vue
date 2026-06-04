<script setup lang="ts">
import { computed, reactive, ref, watch, onMounted } from 'vue';
import { NDrawer, NDrawerContent, NForm, NFormItem, NInput, NSelect, NSwitch, NSpace, NButton, useMessage } from 'naive-ui';
import { $t } from '@/locales';
import { fetchListTaskDefinitions } from '@/service/api/scheduled-task';

interface Props {
  visible: boolean;
  operateType: 'add' | 'edit' | undefined;
  editingData?: Api.ScheduledTask.ScheduledTask | null;
  loading?: boolean;
}

interface Emits {
  (e: 'update:visible', visible: boolean): void;
  (e: 'submit', data: Api.ScheduledTask.CreateScheduledTaskRequest | Api.ScheduledTask.UpdateScheduledTaskRequest): void;
}

const props = defineProps<Props>();
const emit = defineEmits<Emits>();

const message = useMessage();

const drawerVisible = computed({
  get: () => props.visible,
  set: val => emit('update:visible', val)
});

const formData = reactive({
  name: '',
  taskType: '',
  cronExpr: '',
  queue: 'default',
  enabled: true,
  timezone: 'Asia/Shanghai',
  payload: ''
});

// 任务类型列表
const taskDefinitions = ref<Api.ScheduledTask.TaskDefinition[]>([]);

// 根据选择的任务类型获取允许的队列
const currentTaskDefinition = computed(() =>
  taskDefinitions.value.find(d => d.type === formData.taskType)
);

// 任务类型选项
const taskTypeOptions = computed(() =>
  taskDefinitions.value.map(d => ({
    label: `${d.type}${d.description ? ` (${d.description})` : ''}`,
    value: d.type
  }))
);

// 队列选项：根据选择的任务类型动态生成
const queueOptions = computed(() => {
  if (currentTaskDefinition.value?.allowedQueues?.length) {
    return currentTaskDefinition.value.allowedQueues.map(q => ({
      label: q,
      value: q
    }));
  }
  return [
    { label: 'default', value: 'default' },
    { label: 'low', value: 'low' },
    { label: 'critical', value: 'critical' }
  ];
});

const timezoneOptions = [
  { label: 'Asia/Shanghai (中国)', value: 'Asia/Shanghai' },
  { label: 'UTC', value: 'UTC' },
  { label: 'America/New_York (美国东部)', value: 'America/New_York' },
  { label: 'Europe/London (英国)', value: 'Europe/London' }
];

const isEdit = computed(() => props.operateType === 'edit');
const drawerTitle = computed(() =>
  isEdit.value ? $t('page.system-manage.scheduledTask.drawer.editTitle') : $t('page.system-manage.scheduledTask.drawer.addTitle')
);

// 加载任务类型列表
async function loadTaskDefinitions() {
  try {
    const res = await fetchListTaskDefinitions();
    if (!res.error && res.data?.definitions) {
      taskDefinitions.value = res.data.definitions;
    }
  } catch {
    // 使用默认选项作为降级方案
    taskDefinitions.value = [];
  }
}

// 监听编辑数据变化
watch(
  () => props.editingData,
  data => {
    if (data && isEdit.value) {
      formData.name = data.name;
      formData.taskType = data.taskType;
      formData.cronExpr = data.cronExpr;
      formData.queue = data.queue;
      formData.enabled = data.enabled;
      formData.timezone = data.timezone;
      formData.payload = data.payload ? JSON.stringify(data.payload, null, 2) : '';
    }
  },
  { immediate: true }
);

// 监听抽屉关闭
watch(
  () => props.visible,
  val => {
    if (!val) {
      resetForm();
    }
  }
);

function resetForm() {
  formData.name = '';
  formData.taskType = '';
  formData.cronExpr = '';
  formData.queue = 'default';
  formData.enabled = true;
  formData.timezone = 'Asia/Shanghai';
  formData.payload = '';
}

function validateForm(): boolean {
  if (!formData.name.trim()) {
    message.warning($t('page.system-manage.scheduledTask.form.nameRequired'));
    return false;
  }
  if (!formData.taskType) {
    message.warning($t('page.system-manage.scheduledTask.form.taskTypeRequired'));
    return false;
  }
  if (!formData.cronExpr.trim()) {
    message.warning($t('page.system-manage.scheduledTask.form.cronExprRequired'));
    return false;
  }
  // 简单的 cron 表达式校验
  const cronRegex = /^(\*|[0-9,\-/]+)\s+(\*|[0-9,\-/]+)\s+(\*|[0-9,\-/]+)\s+(\*|[0-9,\-/]+)\s+(\*|[0-9,\-/]+)$/;
  if (!cronRegex.test(formData.cronExpr.trim())) {
    message.warning($t('page.system-manage.scheduledTask.form.cronExprInvalid'));
    return false;
  }
  // 校验 payload JSON 格式
  if (formData.payload.trim()) {
    try {
      JSON.parse(formData.payload);
    } catch {
      message.warning($t('page.system-manage.scheduledTask.form.payloadInvalid'));
      return false;
    }
  }
  return true;
}

function handleSubmit() {
  if (!validateForm()) return;

  const payload = formData.payload.trim() ? JSON.parse(formData.payload) : undefined;

  if (isEdit.value && props.editingData) {
    const data: Api.ScheduledTask.UpdateScheduledTaskRequest = {
      name: formData.name,
      cronExpr: formData.cronExpr,
      queue: formData.queue,
      enabled: formData.enabled,
      timezone: formData.timezone
    };
    if (payload) data.payload = payload;
    emit('submit', data);
  } else {
    const data: Api.ScheduledTask.CreateScheduledTaskRequest = {
      name: formData.name,
      taskType: formData.taskType,
      cronExpr: formData.cronExpr,
      queue: formData.queue,
      enabled: formData.enabled,
      timezone: formData.timezone
    };
    if (payload) data.payload = payload;
    emit('submit', data);
  }
}

onMounted(() => {
  loadTaskDefinitions();
});
</script>

<template>
  <NDrawer v-model:show="drawerVisible" :title="drawerTitle" :width="500">
    <NDrawerContent>
      <NForm label-placement="left" label-width="120">
        <NFormItem :label="$t('page.system-manage.scheduledTask.form.name')" required>
          <NInput v-model:value="formData.name" :placeholder="$t('page.system-manage.scheduledTask.form.namePlaceholder')" />
        </NFormItem>
        <NFormItem :label="$t('page.system-manage.scheduledTask.form.taskType')" required>
          <NSelect
            v-model:value="formData.taskType"
            :options="taskTypeOptions"
            :placeholder="$t('common.pleaseSelect')"
            :disabled="isEdit"
          />
        </NFormItem>
        <NFormItem :label="$t('page.system-manage.scheduledTask.form.cronExpr')" required>
          <NInput v-model:value="formData.cronExpr" placeholder="0 3 * * *" />
        </NFormItem>
        <NFormItem :label="$t('page.system-manage.scheduledTask.form.queue')">
          <NSelect v-model:value="formData.queue" :options="queueOptions" />
        </NFormItem>
        <NFormItem :label="$t('page.system-manage.scheduledTask.form.timezone')">
          <NSelect v-model:value="formData.timezone" :options="timezoneOptions" />
        </NFormItem>
        <NFormItem :label="$t('page.system-manage.scheduledTask.form.payload')">
          <NInput
            v-model:value="formData.payload"
            type="textarea"
            placeholder='{"key": "value"}'
            :rows="4"
          />
        </NFormItem>
        <NFormItem :label="$t('page.system-manage.scheduledTask.form.enabled')">
          <NSwitch v-model:checked="formData.enabled" />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="drawerVisible = false">{{ $t('common.cancel') }}</NButton>
          <NButton type="primary" :loading="props.loading" @click="handleSubmit">
            {{ $t('common.confirm') }}
          </NButton>
        </NSpace>
      </template>
    </NDrawerContent>
  </NDrawer>
</template>
