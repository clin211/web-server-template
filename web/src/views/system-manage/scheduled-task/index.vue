<script setup lang="ts">
import { h, onMounted, ref } from 'vue';
import { NButton, NPopconfirm, NSpace, NSwitch, useMessage } from 'naive-ui';
import type { DataTableBaseColumn } from 'naive-ui';
import { $t } from '@/locales';
import {
  fetchDeleteScheduledTask,
  fetchListScheduledTasks,
  fetchToggleScheduledTask,
  fetchTriggerScheduledTask
} from '@/service/api/scheduled-task';
import { useTableOperate } from '@/hooks/common/table';
import { useColumnSetting } from '@/hooks/common/use-column-setting';
import ScheduledTaskSearch from './modules/scheduled-task-search.vue';
import ScheduledTaskOperateDrawer from './modules/scheduled-task-operate-drawer.vue';
import ScheduledTaskDetailDrawer from './modules/scheduled-task-detail-drawer.vue';
import ScheduledTaskExecutionsDrawer from './modules/scheduled-task-executions-drawer.vue';
import { createColumns, type ScheduledTaskTableRow } from './modules/scheduled-task-table-columns';

defineOptions({
  name: 'SystemManageScheduledTask'
});

const message = useMessage();

const searchModel = ref<{ enabled: string | null; taskType: string | null }>({
  enabled: null,
  taskType: null
});
const tableLoading = ref(false);
const operateLoading = ref(false);

const tableData = ref<ScheduledTaskTableRow[]>([]);

const detailDrawerVisible = ref(false);
const detailScheduledTaskId = ref<string | null>(null);
const executionsDrawerVisible = ref(false);
const executionsScheduledTaskId = ref<string | null>(null);
const editingScheduledTaskId = ref<string | null>(null);

const { columnChecks, finalColumns } = useColumnSetting<ScheduledTaskTableRow>({
  key: 'system-manage-scheduled-task',
  columnsFactory: createColumns
});

const { drawerVisible, closeDrawer, operateType, editingData, handleAdd, onDeleted } = useTableOperate(
  tableData,
  'scheduledTaskID',
  getData
);

async function getData() {
  tableLoading.value = true;
  try {
    const res = await fetchListScheduledTasks({
      enabled: searchModel.value.enabled === 'true' ? true : searchModel.value.enabled === 'false' ? false : undefined,
      taskType: searchModel.value.taskType || undefined
    });
    if (!res.error && res.data) {
      tableData.value = res.data.scheduledTasks;
    }
  } finally {
    tableLoading.value = false;
  }
}

async function handleRefresh() {
  await getData();
}

async function handleSearch() {
  await getData();
}

async function handleReset() {
  searchModel.value = { enabled: null, taskType: null };
  await getData();
}

async function handleOperateSubmit(
  data: Api.ScheduledTask.CreateScheduledTaskRequest | Api.ScheduledTask.UpdateScheduledTaskRequest
) {
  operateLoading.value = true;
  try {
    if (operateType.value === 'add') {
      const createData = data as Api.ScheduledTask.CreateScheduledTaskRequest;
      const res = await import('@/service/api/scheduled-task').then(m => m.fetchCreateScheduledTask(createData));
      if (!res.error) {
        message.success($t('common.addSuccess'));
        closeDrawer();
        await getData();
      }
      return;
    }

    if (!editingScheduledTaskId.value) return;

    const updateData = data as Api.ScheduledTask.UpdateScheduledTaskRequest;
    const res = await import('@/service/api/scheduled-task').then(m =>
      m.fetchUpdateScheduledTask(editingScheduledTaskId.value!, updateData)
    );

    if (!res.error) {
      message.success($t('common.updateSuccess'));
      closeDrawer();
      await getData();
    }
  } finally {
    operateLoading.value = false;
  }
}

function handleViewDetail(task: ScheduledTaskTableRow) {
  detailScheduledTaskId.value = task.scheduledTaskID;
  detailDrawerVisible.value = true;
}

function handleEditTask(task: ScheduledTaskTableRow) {
  editingScheduledTaskId.value = task.scheduledTaskID;
  operateType.value = 'edit';
  drawerVisible.value = true;
  editingData.value = task;
}

function handleViewExecutions(task: ScheduledTaskTableRow) {
  executionsScheduledTaskId.value = task.scheduledTaskID;
  executionsDrawerVisible.value = true;
}

async function handleToggleTask(task: ScheduledTaskTableRow) {
  const res = await fetchToggleScheduledTask(task.scheduledTaskID, !task.enabled);
  if (!res.error) {
    message.success($t('common.updateSuccess'));
    await getData();
  }
}

async function handleTriggerTask(task: ScheduledTaskTableRow) {
  const res = await fetchTriggerScheduledTask(task.scheduledTaskID);
  if (!res.error) {
    message.success($t('page.system-manage.scheduledTask.actions.triggerSuccess'));
    await getData();
  }
}

async function handleDeleteTask(task: ScheduledTaskTableRow) {
  const res = await fetchDeleteScheduledTask(task.scheduledTaskID);
  if (!res.error) {
    await onDeleted();
  }
}

function getTableColumns() {
  const cols = [...finalColumns.value];
  const actionsCol = cols.find(column => 'key' in column && column.key === 'actions');

  if (!actionsCol) {
    return cols;
  }

  (actionsCol as DataTableBaseColumn<ScheduledTaskTableRow>).render = (row: ScheduledTaskTableRow) =>
    h(
      NSpace,
      { justify: 'center' },
      {
        default: () => [
          h(
            NButton,
            {
              size: 'small',
              text: true,
              type: 'primary',
              onClick: () => handleViewDetail(row)
            },
            { default: () => $t('page.system-manage.user.actions.detail') }
          ),
          h(
            NButton,
            {
              size: 'small',
              text: true,
              type: 'primary',
              onClick: () => handleEditTask(row)
            },
            { default: () => $t('common.edit') }
          ),
          h(
            NButton,
            {
              size: 'small',
              text: true,
              type: 'info',
              onClick: () => handleViewExecutions(row)
            },
            {
              default: () => $t('page.system-manage.scheduledTask.actions.executions')
            }
          ),
          h(
            NButton,
            {
              size: 'small',
              text: true,
              type: 'warning',
              onClick: () => handleTriggerTask(row)
            },
            { default: () => $t('common.trigger') }
          ),
          h(
            NSwitch,
            {
              size: 'small',
              value: row.enabled,
              onUpdateValue: () => handleToggleTask(row)
            },
            { default: () => '' }
          ),
          h(
            NPopconfirm,
            {
              onPositiveClick: () => handleDeleteTask(row)
            },
            {
              trigger: () =>
                h(
                  NButton,
                  {
                    size: 'small',
                    text: true,
                    type: 'error'
                  },
                  { default: () => $t('common.delete') }
                ),
              default: () => $t('common.confirmDelete')
            }
          )
        ]
      }
    );

  return cols;
}

onMounted(() => {
  getData();
});
</script>

<template>
  <div>
    <NSpace vertical :size="16">
      <ScheduledTaskSearch v-model="searchModel" :loading="tableLoading" @search="handleSearch" @reset="handleReset" />

      <NCard :bordered="false" size="small" class="card-wrapper">
        <NSpace vertical :size="12">
          <TableHeaderOperation
            v-model:columns="columnChecks"
            :loading="tableLoading"
            :disabled-delete="true"
            @add="handleAdd"
            @refresh="handleRefresh"
          />

          <NDataTable
            :columns="getTableColumns()"
            :data="tableData"
            :loading="tableLoading"
            :scroll-x="1800"
            :pagination="false"
            :bordered="false"
            :single-line="false"
            :row-key="(row: ScheduledTaskTableRow) => row.scheduledTaskID"
          />
        </NSpace>
      </NCard>
    </NSpace>

    <ScheduledTaskOperateDrawer
      v-model:visible="drawerVisible"
      v-model:operate-type="operateType"
      v-model:editing-data="editingData"
      :loading="operateLoading"
      @submit="handleOperateSubmit"
    />

    <ScheduledTaskDetailDrawer v-model:visible="detailDrawerVisible" :scheduled-task-id="detailScheduledTaskId" />
    <ScheduledTaskExecutionsDrawer
      v-model:visible="executionsDrawerVisible"
      :scheduled-task-id="executionsScheduledTaskId"
    />
  </div>
</template>
