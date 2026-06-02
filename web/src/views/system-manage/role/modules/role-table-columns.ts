import type { DataTableColumns } from 'naive-ui';
import { $t } from '@/locales';

export type RoleTableRow = Api.Role.Role;

export function createColumns(): DataTableColumns<RoleTableRow> {
  return [
    {
      key: 'index',
      width: 80,
      render: (_, index) => (index as number) + 1
    },
    {
      key: 'roleName',
      title: $t('page.system-manage.role.columns.roleName'),
      align: 'center'
    },
    {
      key: 'roleCode',
      title: $t('page.system-manage.role.columns.roleCode'),
      align: 'center'
    },
    {
      key: 'description',
      title: $t('page.system-manage.role.columns.description'),
      align: 'center',
      ellipsis: { tooltip: true }
    },
    {
      key: 'status',
      title: $t('page.system-manage.role.columns.status'),
      align: 'center',
      width: 100
    },
    {
      key: 'createdAt',
      title: $t('page.system-manage.role.columns.createdAt'),
      align: 'center',
      width: 180,
      render: row => {
        const date = new Date(row.createdAt * 1000);
        return date.toLocaleString('zh-CN');
      }
    },
    {
      key: 'actions',
      title: $t('common.operate'),
      align: 'center',
      width: 120,
      fixed: 'right'
    }
  ];
}
