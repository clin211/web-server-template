<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import dayjs from 'dayjs';
import { useI18n } from 'vue-i18n';

defineOptions({
  name: 'MenuDetailDrawer'
});

interface Props {
  menuId: string | null;
  visible: boolean;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  'update:visible': [value: boolean];
}>();

const { t } = useI18n();

const visible = computed({
  get: () => props.visible,
  set: (val: boolean) => emit('update:visible', val)
});

const loading = ref(false);
const menuDetail = ref<Api.Menu.Menu | null>(null);

async function fetchDetail() {
  const menuId = props.menuId;
  if (!menuId) return;

  loading.value = true;
  try {
    const res = await import('@/service/api/menu').then(m => m.fetchGetMenu(menuId));
    if (!res.error && res.data) {
      menuDetail.value = res.data.menu;
    }
  } finally {
    loading.value = false;
  }
}

watch(
  () => props.visible,
  (val: boolean) => {
    if (val) {
      menuDetail.value = null;
      fetchDetail();
    }
  }
);

function formatTime(ts: number | undefined) {
  if (!ts) return '-';
  return dayjs(ts * 1000).format('YYYY-MM-DD HH:mm:ss');
}

function formatYesOrNo(val: number | undefined) {
  if (val === undefined || val === null) return '-';
  return val === 1 ? t('common.yesOrNo.yes') : t('common.yesOrNo.no');
}
</script>

<template>
  <NDrawer v-model:show="visible" :width="560">
    <NDrawerContent :title="t('page.system-manage.menu.detail.title')" closable :native-scrollbar="false">
      <NSpin :show="loading">
        <NDescriptions v-if="menuDetail" :column="2" label-placement="left" size="small" bordered>
          <NDescriptionsItem :label="t('page.system-manage.menu.columns.menuName')">
            {{ menuDetail.menuName }}
          </NDescriptionsItem>
          <NDescriptionsItem :label="t('page.system-manage.menu.columns.menuCode')">
            {{ menuDetail.menuCode || '-' }}
          </NDescriptionsItem>
          <NDescriptionsItem :label="t('page.system-manage.menu.columns.menuType')">
            <NTag :type="menuDetail.menuType === 'page' ? 'info' : 'default'">
              {{
                menuDetail.menuType === 'page'
                  ? t('page.system-manage.menu.menuType.page')
                  : t('page.system-manage.menu.menuType.menu')
              }}
            </NTag>
          </NDescriptionsItem>
          <NDescriptionsItem :label="t('page.system-manage.menu.form.parentID')">
            {{ menuDetail.parentID || '-' }}
          </NDescriptionsItem>
          <NDescriptionsItem :label="t('page.system-manage.menu.columns.path')">
            {{ menuDetail.path || '-' }}
          </NDescriptionsItem>
          <NDescriptionsItem :label="t('page.system-manage.menu.form.component')">
            {{ menuDetail.component || '-' }}
          </NDescriptionsItem>
          <NDescriptionsItem :label="t('page.system-manage.menu.form.icon')">
            {{ menuDetail.icon || menuDetail.localIcon || '-' }}
          </NDescriptionsItem>
          <NDescriptionsItem :label="t('page.system-manage.menu.columns.sortOrder')">
            {{ menuDetail.sortOrder }}
          </NDescriptionsItem>
          <NDescriptionsItem :label="t('page.system-manage.menu.columns.visible')">
            <NTag :type="menuDetail.visible === 1 ? 'success' : 'default'">
              {{ formatYesOrNo(menuDetail.visible) }}
            </NTag>
          </NDescriptionsItem>
          <NDescriptionsItem :label="t('page.system-manage.menu.columns.status')">
            <NTag :type="menuDetail.status === 0 ? 'success' : 'error'">
              {{
                menuDetail.status === 0
                  ? t('page.system-manage.menu.status.normal')
                  : t('page.system-manage.menu.status.disabled')
              }}
            </NTag>
          </NDescriptionsItem>
          <NDescriptionsItem :label="t('page.system-manage.menu.form.constant')">
            <NTag :type="menuDetail.constant === 1 ? 'warning' : 'default'">
              {{ formatYesOrNo(menuDetail.constant) }}
            </NTag>
          </NDescriptionsItem>
          <NDescriptionsItem :label="t('page.system-manage.menu.form.hideInMenu')">
            <NTag :type="menuDetail.hideInMenu === 1 ? 'default' : 'success'">
              {{ formatYesOrNo(menuDetail.hideInMenu) }}
            </NTag>
          </NDescriptionsItem>
          <NDescriptionsItem :label="t('page.system-manage.menu.form.keepAlive')">
            <NTag :type="menuDetail.keepAlive === 1 ? 'info' : 'default'">
              {{ formatYesOrNo(menuDetail.keepAlive) }}
            </NTag>
          </NDescriptionsItem>
          <NDescriptionsItem :label="t('page.system-manage.menu.form.href')">
            {{ menuDetail.href || '-' }}
          </NDescriptionsItem>
          <NDescriptionsItem :label="t('page.system-manage.menu.columns.createdAt')">
            {{ formatTime(menuDetail.createdAt) }}
          </NDescriptionsItem>
          <NDescriptionsItem :label="t('page.system-manage.menu.detail.updatedAt')">
            {{ formatTime(menuDetail.updatedAt) }}
          </NDescriptionsItem>
        </NDescriptions>
        <div v-else class="py-20px text-center text-gray-400">
          {{ t('common.noData') }}
        </div>
      </NSpin>

      <template #footer>
        <NSpace justify="end">
          <NButton @click="visible = false">{{ t('common.close') }}</NButton>
        </NSpace>
      </template>
    </NDrawerContent>
  </NDrawer>
</template>
