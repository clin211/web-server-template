import { h } from 'vue';
import type { DataTableColumns } from 'naive-ui';
import { NTag } from 'naive-ui';
import dayjs from 'dayjs';
import { $t } from '@/locales';

export type ScheduledTaskTableRow = Api.ScheduledTask.ScheduledTask;

export function createColumns(): DataTableColumns<ScheduledTaskTableRow> {
  return [
    {
      key: 'name',
      title: $t('page.system-manage.scheduledTask.columns.name'),
      align: 'center',
      minWidth: 150,
      ellipsis: { tooltip: true }
    },
    {
      key: 'taskType',
      title: $t('page.system-manage.scheduledTask.columns.taskType'),
      align: 'center',
      width: 140,
      ellipsis: { tooltip: true }
    },
    {
      key: 'cronExpr',
      title: $t('page.system-manage.scheduledTask.columns.cronExpr'),
      align: 'center',
      width: 160,
      ellipsis: { tooltip: true }
    },
    {
      key: 'queue',
      title: $t('page.system-manage.scheduledTask.columns.queue'),
      align: 'center',
      width: 100
    },
    {
      key: 'enabled',
      title: $t('page.system-manage.scheduledTask.columns.enabled'),
      align: 'center',
      width: 100
    },
    {
      key: 'nextRunTime',
      title: $t('page.system-manage.scheduledTask.columns.nextRunTime'),
      align: 'center',
      width: 170,
      render: row => (row.nextRunTime ? dayjs(row.nextRunTime * 1000).format('YYYY-MM-DD HH:mm:ss') : '-')
    },
    {
      key: 'lastScheduledAt',
      title: $t('page.system-manage.scheduledTask.columns.lastScheduledAt'),
      align: 'center',
      width: 170,
      render: row => (row.lastScheduledAt ? dayjs(row.lastScheduledAt * 1000).format('YYYY-MM-DD HH:mm:ss') : '-')
    },
    {
      key: 'lastError',
      title: $t('page.system-manage.scheduledTask.columns.lastError'),
      align: 'center',
      minWidth: 150,
      ellipsis: { tooltip: true },
      render: row => {
        if (!row.lastError) return '-';
        return h(NTag, { type: 'error', size: 'small', round: true }, { default: () => row.lastError });
      }
    },
    {
      key: 'createdAt',
      title: $t('page.system-manage.scheduledTask.columns.createdAt'),
      align: 'center',
      width: 170,
      render: row => (row.createdAt ? dayjs(row.createdAt * 1000).format('YYYY-MM-DD HH:mm:ss') : '-')
    },
    {
      key: 'actions',
      title: $t('common.operate'),
      align: 'center',
      width: 300,
      fixed: 'right'
    }
  ];
}
