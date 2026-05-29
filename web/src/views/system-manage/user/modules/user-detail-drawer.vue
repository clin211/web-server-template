<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import dayjs from 'dayjs';

defineOptions({
  name: 'UserDetailDrawer'
});

interface Props {
  userId: string | null;
  visible: boolean;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  'update:visible': [value: boolean];
}>();

const visible = computed({
  get: () => props.visible,
  set: (val: boolean) => emit('update:visible', val)
});

const loading = ref(false);
const userDetail = ref<Api.User.User | null>(null);

async function fetchDetail() {
  const userId = props.userId;
  if (!userId) return;

  loading.value = true;
  try {
    const res = await import('@/service/api/user').then(m => m.fetchGetUser(userId));
    if (!res.error && res.data) {
      userDetail.value = res.data.user;
    }
  } finally {
    loading.value = false;
  }
}

watch(
  () => props.visible,
  (val: boolean) => {
    if (val) {
      userDetail.value = null;
      fetchDetail();
    }
  }
);

function formatTime(ts: number | undefined) {
  if (!ts) return '-';
  return dayjs(ts * 1000).format('YYYY-MM-DD HH:mm:ss');
}
</script>

<template>
  <NDrawer v-model:show="visible" :width="560">
    <NDrawerContent :title="$t('page.system-manage.user.detail.title')" closable :native-scrollbar="false">
      <NSpin :show="loading">
        <NDescriptions v-if="userDetail" :column="2" label-placement="left" size="small" bordered>
          <NDescriptionsItem :label="$t('page.system-manage.user.columns.username')">
            {{ userDetail.username }}
          </NDescriptionsItem>
          <NDescriptionsItem :label="$t('page.system-manage.user.columns.nickname')">
            {{ userDetail.nickname || '-' }}
          </NDescriptionsItem>
          <NDescriptionsItem :label="$t('page.system-manage.user.columns.email')">
            {{ userDetail.email || '-' }}
          </NDescriptionsItem>
          <NDescriptionsItem :label="$t('page.system-manage.user.columns.phone')">
            {{ userDetail.phone || '-' }}
          </NDescriptionsItem>
          <NDescriptionsItem :label="$t('page.system-manage.user.columns.gender')">
            <NTag
              v-if="userDetail.gender !== undefined && userDetail.gender !== null"
              :type="userDetail.gender === 1 ? 'info' : userDetail.gender === 2 ? 'warning' : 'default'"
            >
              {{
                userDetail.gender === 1
                  ? $t('page.system-manage.user.gender.male')
                  : userDetail.gender === 2
                    ? $t('page.system-manage.user.gender.female')
                    : $t('page.system-manage.user.gender.unknown')
              }}
            </NTag>
            <span v-else>-</span>
          </NDescriptionsItem>
          <NDescriptionsItem :label="$t('page.system-manage.user.columns.status')">
            <NTag :type="userDetail.status === 0 ? 'success' : 'error'">
              {{
                userDetail.status === 0
                  ? $t('page.system-manage.user.status.normal')
                  : $t('page.system-manage.user.status.disabled')
              }}
            </NTag>
          </NDescriptionsItem>
          <NDescriptionsItem :label="$t('page.system-manage.user.detail.avatar')">
            <NAvatar v-if="userDetail.avatar" :src="userDetail.avatar" :size="40" round />
            <span v-else>-</span>
          </NDescriptionsItem>
          <NDescriptionsItem :label="$t('page.system-manage.user.detail.postCount')">
            {{ userDetail.postCount ?? 0 }}
          </NDescriptionsItem>
          <NDescriptionsItem :label="$t('page.system-manage.user.columns.createdAt')">
            {{ formatTime(userDetail.createdAt) }}
          </NDescriptionsItem>
          <NDescriptionsItem :label="$t('page.system-manage.user.columns.lastLoginAt')">
            {{ formatTime(userDetail.lastLoginAt) }}
          </NDescriptionsItem>
          <NDescriptionsItem :label="$t('page.system-manage.user.detail.description')" :span="2">
            {{ userDetail.description || '-' }}
          </NDescriptionsItem>
        </NDescriptions>
        <div v-else class="py-20px text-center text-gray-400">
          {{ $t('common.noData') }}
        </div>
      </NSpin>

      <template #footer>
        <NSpace justify="end">
          <NButton @click="visible = false">{{ $t('common.close') }}</NButton>
        </NSpace>
      </template>
    </NDrawerContent>
  </NDrawer>
</template>
