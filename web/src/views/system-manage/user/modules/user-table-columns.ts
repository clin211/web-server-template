import { h } from 'vue';
import type { DataTableColumns } from 'naive-ui';
import { NTag } from 'naive-ui';
import dayjs from 'dayjs';
import { $t } from '@/locales';

export type UserTableRow = Api.User.User;

export function createColumns(): DataTableColumns<UserTableRow> {
  return [
    {
      key: 'username',
      title: $t('page.system-manage.user.columns.username'),
      align: 'center',
      width: 120
    },
    {
      key: 'nickname',
      title: $t('page.system-manage.user.columns.nickname'),
      align: 'center',
      width: 120,
      render: row => row.nickname || '-'
    },
    {
      key: 'email',
      title: $t('page.system-manage.user.columns.email'),
      align: 'center',
      width: 180,
      render: row => row.email || '-'
    },
    {
      key: 'phone',
      title: $t('page.system-manage.user.columns.phone'),
      align: 'center',
      width: 140,
      render: row => row.phone || '-'
    },
    {
      key: 'gender',
      title: $t('page.system-manage.user.columns.gender'),
      align: 'center',
      width: 100,
      render: row => {
        const map: Record<number, string> = {
          0: $t('page.system-manage.user.gender.unknown'),
          1: $t('page.system-manage.user.gender.male'),
          2: $t('page.system-manage.user.gender.female')
        };
        return map[row.gender] ?? '-';
      }
    },
    {
      key: 'status',
      title: $t('page.system-manage.user.columns.status'),
      align: 'center',
      width: 100,
      render: row => {
        const tagType = row.status === 0 ? 'success' : 'error';
        const label = row.status === 0 ? $t('page.system-manage.user.status.normal') : $t('page.system-manage.user.status.disabled');
        return h(NTag, { type: tagType }, { default: () => label });
      }
    },
    {
      key: 'createdAt',
      title: $t('page.system-manage.user.columns.createdAt'),
      align: 'center',
      width: 180,
      render: row => (row.createdAt ? dayjs(row.createdAt * 1000).format('YYYY-MM-DD HH:mm') : '-')
    },
    {
      key: 'lastLoginAt',
      title: $t('page.system-manage.user.columns.lastLoginAt'),
      align: 'center',
      width: 180,
      render: row => (row.lastLoginAt ? dayjs(row.lastLoginAt * 1000).format('YYYY-MM-DD HH:mm') : '-')
    },
    {
      key: 'actions',
      title: $t('common.operate'),
      align: 'center',
      width: 260,
      fixed: 'right'
    }
  ];
}
