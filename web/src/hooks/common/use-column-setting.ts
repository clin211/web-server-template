import { computed, effectScope, onScopeDispose, ref, watch } from 'vue';
import type { DataTableBaseColumn, DataTableColumns } from 'naive-ui';
import type { TableColumnCheck } from '@sa/hooks';
import { $t } from '@/locales';
import { jsonClone } from '@sa/utils';

const SELECTION_KEY = '__selection__';
const EXPAND_KEY = '__expand__';

interface UseColumnSettingOptions<T = unknown> {
  /** Unique key for storing column settings */
  key: string;
  /** Default columns factory */
  columnsFactory: () => DataTableColumns<T>;
  /** Transform column title (for i18n support) */
  transformColumnTitle?: (col: DataTableColumns<T>[number]) => string;
}

export function useColumnSetting<T = unknown>(options: UseColumnSettingOptions<T>) {
  const { key, columnsFactory, transformColumnTitle } = options;
  const scope = effectScope();

  const storageKey = `column-setting:${key}`;

  function loadFromStorage(): TableColumnCheck[] | null {
    try {
      const stored = localStorage.getItem(storageKey);
      if (stored) {
        return JSON.parse(stored) as TableColumnCheck[];
      }
    } catch {
      // ignore parse errors
    }
    return null;
  }

  function saveToStorage(checks: TableColumnCheck[]) {
    try {
      localStorage.setItem(storageKey, JSON.stringify(checks));
    } catch {
      // ignore storage errors
    }
  }

  function getDefaultColumnChecks(): TableColumnCheck[] {
    const cols = columnsFactory();
    const checks: TableColumnCheck[] = [];

    cols.forEach(col => {
      const column = col as DataTableBaseColumn<T> & { key?: string | number };
      if (column.key !== undefined && 'title' in column) {
        const title = transformColumnTitle
          ? transformColumnTitle(col)
          : (typeof column.title === 'string' ? column.title : String(column.key));
        checks.push({
          key: String(column.key),
          title,
          checked: true,
          fixed: (column.fixed as 'left' | 'right' | 'unFixed') ?? 'unFixed',
          visible: true
        });
      } else if (col.type === 'selection') {
        checks.push({
          key: SELECTION_KEY,
          title: $t('common.check'),
          checked: true,
          fixed: 'unFixed',
          visible: false
        });
      } else if (col.type === 'expand') {
        checks.push({
          key: EXPAND_KEY,
          title: $t('common.expandColumn'),
          checked: true,
          fixed: 'unFixed',
          visible: false
        });
      }
    });

    return checks;
  }

  // Load from storage or use defaults
  const columnChecks = ref<TableColumnCheck[]>(loadFromStorage() || jsonClone(getDefaultColumnChecks()));

  // Sync column checks to storage
  watch(
    columnChecks,
    checks => {
      saveToStorage(checks);
    },
    { deep: true }
  );

  // Calculate final columns based on columnChecks
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const finalColumns = computed<any[]>(() => {
    const defaultCols = columnsFactory();
    const columnMap = new Map<string, DataTableColumns<T>[number]>();

    defaultCols.forEach(col => {
      const column = col as DataTableBaseColumn<T> & { key?: string | number; type?: string };
      if (column.key !== undefined && 'title' in col) {
        columnMap.set(String(column.key), col);
      } else if (col.type === 'selection') {
        columnMap.set(SELECTION_KEY, col);
      } else if (col.type === 'expand') {
        columnMap.set(EXPAND_KEY, col);
      }
    });

    const result = columnChecks.value
      .filter(item => item.checked)
      .map(check => {
        const col = columnMap.get(check.key);
        if (!col) return null;

        return {
          ...col,
          fixed: check.fixed === 'unFixed' ? undefined : check.fixed
        };
      })
      .filter(Boolean);

    return result;
  });

  function reloadColumns() {
    const checkMap = new Map(columnChecks.value.map(col => [col.key, col.checked]));
    const fixedMap = new Map(columnChecks.value.map(col => [col.key, col.fixed]));

    const defaultChecks = getDefaultColumnChecks();

    columnChecks.value = defaultChecks.map(col => ({
      ...col,
      checked: checkMap.get(col.key) ?? col.checked,
      fixed: (fixedMap.get(col.key) !== 'unFixed' ? fixedMap.get(col.key) : undefined) ?? col.fixed
    }));
  }

  function resetToDefault() {
    columnChecks.value = jsonClone(getDefaultColumnChecks());
  }

  onScopeDispose(() => {
    scope.stop();
  });

  return {
    columnChecks,
    finalColumns,
    reloadColumns,
    resetToDefault
  };
}

export type { TableColumnCheck };