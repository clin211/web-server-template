import { h } from 'vue';
import type { DataTableColumns } from 'naive-ui';
import { NTag } from 'naive-ui';
import dayjs from 'dayjs';
import { $t } from '@/locales';
import SvgIcon from '@/components/custom/svg-icon.vue';

export type MenuTableRow = Api.Menu.Menu;

export function createColumns(): DataTableColumns<MenuTableRow> {
  return [
    {
      key: 'menuName',
      title: $t('page.system-manage.menu.columns.menuName'),
      align: 'center',
      minWidth: 150,
      ellipsis: { tooltip: true },
      render: row => {
        if (row.i18nKey) {
          const translated = $t(row.i18nKey as App.I18n.I18nKey);
          return translated !== row.i18nKey ? translated : row.menuName;
        }
        return row.menuName;
      }
    },
    {
      key: 'menuCode',
      title: $t('page.system-manage.menu.columns.menuCode'),
      align: 'center',
      minWidth: 120,
      ellipsis: { tooltip: true }
    },
    {
      key: 'menuType',
      title: $t('page.system-manage.menu.columns.menuType'),
      align: 'center',
      width: 90,
      render: row => {
        const map: Record<string, string> = {
          menu: $t('page.system-manage.menu.type.directory'),
          page: $t('page.system-manage.menu.type.page')
        };
        const type = map[row.menuType] ?? '-';
        const tagType = row.menuType === 'menu' ? 'default' : 'info';
        return h(NTag, { type: tagType, size: 'small' }, { default: () => type });
      }
    },
    {
      key: 'icon',
      title: $t('page.system-manage.menu.columns.icon'),
      align: 'center',
      width: 80,
      render: row =>
        row.icon || row.localIcon
          ? h(SvgIcon, { icon: row.icon, localIcon: row.localIcon, class: 'text-16px' })
          : '-'
    },
    {
      key: 'path',
      title: $t('page.system-manage.menu.columns.path'),
      align: 'center',
      minWidth: 180,
      ellipsis: { tooltip: true }
    },
    {
      key: 'component',
      title: $t('page.system-manage.menu.columns.component'),
      align: 'center',
      minWidth: 180,
      ellipsis: { tooltip: true }
    },
    {
      key: 'visible',
      title: $t('page.system-manage.menu.columns.visible'),
      align: 'center',
      width: 80,
      render: row => {
        const type = row.visible === 1 ? 'success' : 'warning';
        const label = row.visible === 1 ? $t('common.yes') : $t('common.no');
        return h(NTag, { type, size: 'small' }, { default: () => label });
      }
    },
    {
      key: 'status',
      title: $t('page.system-manage.menu.columns.status'),
      align: 'center',
      width: 80,
      render: row => {
        const tagType = row.status === 0 ? 'success' : 'error';
        const label = row.status === 0 ? $t('page.system-manage.menu.status.normal') : $t('page.system-manage.menu.status.disabled');
        return h(NTag, { type: tagType, size: 'small' }, { default: () => label });
      }
    },
    {
      key: 'sortOrder',
      title: $t('page.system-manage.menu.columns.sortOrder'),
      align: 'center',
      width: 80
    },
    {
      key: 'constant',
      title: $t('page.system-manage.menu.columns.constant'),
      align: 'center',
      width: 80,
      render: row => {
        const type = row.constant === 1 ? 'info' : 'default';
        const label = row.constant === 1 ? $t('common.yes') : $t('common.no');
        return h(NTag, { type, size: 'small' }, { default: () => label });
      }
    },
    {
      key: 'hideInMenu',
      title: $t('page.system-manage.menu.columns.hideInMenu'),
      align: 'center',
      width: 100,
      render: row => {
        const type = row.hideInMenu === 1 ? 'warning' : 'success';
        const label = row.hideInMenu === 1 ? $t('common.yes') : $t('common.no');
        return h(NTag, { type, size: 'small' }, { default: () => label });
      }
    },
    {
      key: 'createdAt',
      title: $t('page.system-manage.menu.columns.createdAt'),
      align: 'center',
      width: 160,
      render: row => (row.createdAt ? dayjs(row.createdAt * 1000).format('YYYY-MM-DD HH:mm') : '-')
    },
    {
      key: 'actions',
      title: $t('common.operate'),
      align: 'center',
      width: 240,
      fixed: 'right'
    }
  ];
}