<script setup lang="ts">
import { computed, ref, watch, h } from 'vue';
import { NDrawer, NDrawerContent, NDataTable, NTag, NButton, NSpace, NSpin } from 'naive-ui';
import type { DataTableColumns } from 'naive-ui';
import dayjs from 'dayjs';
import { $t } from '@/locales';
import { fetchListScheduledTaskExecutions } from '@/service/api/scheduled-task';

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
const executions = ref<Api.ScheduledTask.ScheduledTaskExecution[]>([]);
const totalCount = ref(0);
const pageToken = ref('');
const pageSize = ref(20);

function formatTime(timestamp: number | undefined | null): string {
  if (!timestamp) return '-';
  return dayjs(timestamp * 1000).format('YYYY-MM-DD HH:mm:ss');
}

function getDispatchStatusType(status: string): 'success' | 'warning' | 'error' | 'info' {
  switch (status) {
    case 'enqueued':
      return 'success';
    case 'pending':
      return 'warning';
    case 'enqueue_failed':
      return 'error';
    case 'skipped':
      return 'info';
    default:
      return 'info';
  }
}

function getProcessStatusType(status: string): 'success' | 'warning' | 'error' | 'info' | 'default' {
  switch (status) {
    case 'succeeded':
      return 'success';
    case 'running':
      return 'info';
    case 'failed':
      return 'error';
    case 'retrying':
      return 'warning';
    case 'dead':
      return 'error';
    case 'pending':
      return 'default';
    default:
      return 'default';
  }
}

const columns: DataTableColumns<Api.ScheduledTask.ScheduledTaskExecution> = [
  {
    key: 'executionID',
    title: 'Execution ID',
    width: 200,
    ellipsis: { tooltip: true }
  },
  {
    key: 'triggerType',
    title: $t('page.system-manage.scheduledTask.execution.columns.triggerType'),
    width: 100,
    align: 'center'
  },
  {
    key: 'dispatchStatus',
    title: $t('page.system-manage.scheduledTask.execution.columns.dispatchStatus'),
    width: 120,
    align: 'center',
    render: row => {
      return h(
        NTag,
        { type: getDispatchStatusType(row.dispatchStatus), size: 'small' },
        { default: () => row.dispatchStatus }
      );
    }
  },
  {
    key: 'processStatus',
    title: $t('page.system-manage.scheduledTask.execution.columns.processStatus'),
    width: 120,
    align: 'center',
    render: row => {
      return h(
        NTag,
        { type: getProcessStatusType(row.processStatus), size: 'small' },
        { default: () => row.processStatus }
      );
    }
  },
  {
    key: 'attempt',
    title: $t('page.system-manage.scheduledTask.execution.columns.attempt'),
    width: 80,
    align: 'center'
  },
  {
    key: 'durationMs',
    title: $t('page.system-manage.scheduledTask.execution.columns.durationMs'),
    width: 100,
    align: 'center',
    render: row => (row.durationMs ? `${row.durationMs}ms` : '-')
  },
  {
    key: 'startedAt',
    title: $t('page.system-manage.scheduledTask.execution.columns.startedAt'),
    width: 170,
    align: 'center',
    render: row => formatTime(row.startedAt)
  },
  {
    key: 'finishedAt',
    title: $t('page.system-manage.scheduledTask.execution.columns.finishedAt'),
    width: 170,
    align: 'center',
    render: row => formatTime(row.finishedAt)
  },
  {
    key: 'createdAt',
    title: $t('page.system-manage.scheduledTask.execution.columns.createdAt'),
    width: 170,
    align: 'center',
    render: row => formatTime(row.createdAt)
  }
];

async function loadExecutions() {
  if (!props.scheduledTaskId) return;
  loading.value = true;
  try {
    const res = await fetchListScheduledTaskExecutions(props.scheduledTaskId, {
      pageToken: pageToken.value || undefined,
      pageSize: pageSize.value
    });
    if (!res.error && res.data) {
      executions.value = res.data.executions;
      totalCount.value = res.data.totalCount;
      pageToken.value = res.data.pageToken;
    }
  } finally {
    loading.value = false;
  }
}

watch(
  () => props.visible,
  val => {
    if (val && props.scheduledTaskId) {
      pageToken.value = '';
      loadExecutions();
    }
  }
);

function handleLoadMore() {
  if (!pageToken.value) return;
  loadExecutions();
}
</script>

<template>
  <NDrawer v-model:show="drawerVisible" :title="$t('page.system-manage.scheduledTask.execution.title')" :width="900">
    <NDrawerContent>
      <NSpin :show="loading">
        <NSpace vertical :size="12">
          <NSpace justify="end">
            <NButton size="small" :disabled="!pageToken" @click="handleLoadMore">
              {{ $t('common.loadMore') }}
            </NButton>
          </NSpace>
          <NDataTable
            :columns="columns"
            :data="executions"
            :bordered="false"
            :single-line="false"
            :pagination="false"
          />
        </NSpace>
      </NSpin>
    </NDrawerContent>
  </NDrawer>
</template>
