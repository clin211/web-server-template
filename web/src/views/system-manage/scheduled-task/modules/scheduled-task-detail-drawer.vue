<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { NDrawer, NDrawerContent, NDescriptions, NDescriptionsItem, NSpin } from 'naive-ui';
import dayjs from 'dayjs';
import { $t } from '@/locales';
import { fetchGetScheduledTask } from '@/service/api/scheduled-task';

interface Props {
  visible: boolean;
  scheduledTaskId: string | null;
}

interface Emits {
  (e: 'update:visible', visible: boolean): void;
}

const props = defineProps<Props>();
const emit = defineEmits<Emits>();

const drawerVisible = computed({
  get: () => props.visible,
  set: val => emit('update:visible', val)
});

const loading = ref(false);
const taskData = ref<Api.ScheduledTask.ScheduledTask | null>(null);

watch(
  () => props.visible,
  async val => {
    if (val && props.scheduledTaskId) {
      loading.value = true;
      try {
        const res = await fetchGetScheduledTask(props.scheduledTaskId);
        if (!res.error && res.data) {
          taskData.value = res.data.scheduledTask;
        }
      } finally {
        loading.value = false;
      }
    }
  }
);

function formatTime(timestamp: number | undefined | null): string {
  if (!timestamp) return '-';
  return dayjs(timestamp * 1000).format('YYYY-MM-DD HH:mm:ss');
}
</script>

<template>
  <NDrawer v-model:show="drawerVisible" :title="$t('page.system-manage.scheduledTask.detail.title')" :width="500">
    <NDrawerContent>
      <NSpin :show="loading">
        <NDescriptions v-if="taskData" :column="1" label-placement="left" bordered>
          <NDescriptionsItem :label="$t('page.system-manage.scheduledTask.columns.name')">
            {{ taskData.name }}
          </NDescriptionsItem>
          <NDescriptionsItem :label="$t('page.system-manage.scheduledTask.columns.scheduledTaskID')">
            {{ taskData.scheduledTaskID }}
          </NDescriptionsItem>
          <NDescriptionsItem :label="$t('page.system-manage.scheduledTask.columns.taskType')">
            {{ taskData.taskType }}
          </NDescriptionsItem>
          <NDescriptionsItem :label="$t('page.system-manage.scheduledTask.columns.cronExpr')">
            {{ taskData.cronExpr }}
          </NDescriptionsItem>
          <NDescriptionsItem :label="$t('page.system-manage.scheduledTask.columns.queue')">
            {{ taskData.queue }}
          </NDescriptionsItem>
          <NDescriptionsItem :label="$t('page.system-manage.scheduledTask.columns.enabled')">
            {{ taskData.enabled ? $t('common.yes') : $t('common.no') }}
          </NDescriptionsItem>
          <NDescriptionsItem :label="$t('page.system-manage.scheduledTask.columns.timezone')">
            {{ taskData.timezone }}
          </NDescriptionsItem>
          <NDescriptionsItem :label="$t('page.system-manage.scheduledTask.columns.nextRunTime')">
            {{ formatTime(taskData.nextRunTime) }}
          </NDescriptionsItem>
          <NDescriptionsItem :label="$t('page.system-manage.scheduledTask.columns.lastScheduledAt')">
            {{ formatTime(taskData.lastScheduledAt) }}
          </NDescriptionsItem>
          <NDescriptionsItem :label="$t('page.system-manage.scheduledTask.columns.lastExecutionID')">
            {{ taskData.lastExecutionID || '-' }}
          </NDescriptionsItem>
          <NDescriptionsItem :label="$t('page.system-manage.scheduledTask.columns.lastError')">
            {{ taskData.lastError || '-' }}
          </NDescriptionsItem>
          <NDescriptionsItem :label="$t('page.system-manage.scheduledTask.columns.userID')">
            {{ taskData.userID }}
          </NDescriptionsItem>
          <NDescriptionsItem :label="$t('page.system-manage.scheduledTask.columns.createdAt')">
            {{ formatTime(taskData.createdAt) }}
          </NDescriptionsItem>
          <NDescriptionsItem :label="$t('page.system-manage.scheduledTask.columns.updatedAt')">
            {{ formatTime(taskData.updatedAt) }}
          </NDescriptionsItem>
          <NDescriptionsItem :label="$t('page.system-manage.scheduledTask.columns.payload')">
            <pre style="white-space: pre-wrap; word-break: break-all; font-size: 12px">{{
              taskData.payload ? JSON.stringify(taskData.payload, null, 2) : '-'
            }}</pre>
          </NDescriptionsItem>
        </NDescriptions>
      </NSpin>
    </NDrawerContent>
  </NDrawer>
</template>
