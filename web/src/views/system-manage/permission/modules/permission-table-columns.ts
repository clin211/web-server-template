import { h } from 'vue';
import type { DataTableColumns } from 'naive-ui';
import { NTag } from 'naive-ui';
import SvgIcon from '@/components/custom/svg-icon.vue';
import { $t } from '@/locales';

export type PermissionTableRow = Api.Permission.Permission;

export function createColumns(): DataTableColumns<PermissionTableRow> {
  return [
    {
      key: 'permissionName',
      title: $t('page.system-manage.permission.columns.permissionName'),
      align: 'center',
      minWidth: 150,
      ellipsis: { tooltip: true }
    },
    {
      key: 'permissionCode',
      title: $t('page.system-manage.permission.columns.permissionCode'),
      align: 'center',
      minWidth: 160,
      ellipsis: { tooltip: true }
    },
    {
      key: 'resourceType',
      title: $t('page.system-manage.permission.columns.resourceType'),
      align: 'center',
      width: 100,
      render: row => {
        const map: Record<string, string> = {
          menu: $t('page.system-manage.permission.resourceType.menu'),
          button: $t('page.system-manage.permission.resourceType.button')
        };
        const type = map[row.resourceType] ?? '-';
        const tagType = row.resourceType === 'menu' ? 'default' : 'info';
        return h(NTag, { type: tagType, size: 'small' }, { default: () => type });
      }
    },
    {
      key: 'resourceTypeIcon',
      title: $t('page.system-manage.permission.columns.icon'),
      align: 'center',
      width: 80,
      render: row => {
        const icon = row.resourceType === 'menu' ? 'mdi:folder-outline' : 'mdi:button-cursor';
        return h(SvgIcon, { icon, class: 'text-16px' });
      }
    },
    {
      key: 'resourcePath',
      title: $t('page.system-manage.permission.columns.resourcePath'),
      align: 'center',
      minWidth: 180,
      ellipsis: { tooltip: true },
      render: row => row.resourcePath || '-'
    },
    {
      key: 'action',
      title: $t('page.system-manage.permission.columns.action'),
      align: 'center',
      width: 100,
      render: row => {
        const tagType =
          row.action === 'GET'
            ? 'success'
            : row.action === 'POST'
              ? 'warning'
              : row.action === 'DELETE'
                ? 'error'
                : 'info';
        return h(NTag, { type: tagType, size: 'small' }, { default: () => row.action || '-' });
      }
    },
    {
      key: 'status',
      title: $t('page.system-manage.permission.columns.status'),
      align: 'center',
      width: 80,
      render: row => {
        const tagType = row.status === 0 ? 'success' : 'error';
        const label =
          row.status === 0
            ? $t('page.system-manage.permission.status.normal')
            : $t('page.system-manage.permission.status.disabled');
        return h(NTag, { type: tagType, size: 'small' }, { default: () => label });
      }
    },
    {
      key: 'description',
      title: $t('page.system-manage.permission.columns.description'),
      align: 'center',
      minWidth: 150,
      ellipsis: { tooltip: true },
      render: row => row.description || '-'
    },
    {
      key: 'createdAt',
      title: $t('page.system-manage.permission.columns.createdAt'),
      align: 'center',
      width: 160,
      render: row => {
        const date = new Date(row.createdAt * 1000);
        const formatDate = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`;
        return formatDate;
      }
    },
    {
      key: 'actions',
      title: $t('common.operate'),
      align: 'center',
      width: 150,
      fixed: 'right'
    }
  ];
}
