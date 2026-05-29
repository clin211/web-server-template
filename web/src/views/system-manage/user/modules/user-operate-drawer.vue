<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import type { FormInst } from 'naive-ui';
import { REG_EMAIL, REG_PHONE } from '@/constants/reg';
import { useFormRules } from '@/hooks/common/form';
import { $t } from '@/locales';

defineOptions({
  name: 'UserOperateDrawer'
});

interface Props {
  loading?: boolean;
}

defineProps<Props>();

const emit = defineEmits<{
  submit: [data: Api.User.CreateUserRequest];
}>();

const visible = defineModel<boolean>('visible', { default: false });
const operateType = defineModel<'add' | 'edit'>('operateType', { default: 'add' });
const editingUser = defineModel<Api.User.User | null>('editingUser', { default: null });

const title = computed(() =>
  operateType.value === 'add' ? $t('page.system-manage.user.drawer.addTitle') : $t('page.system-manage.user.drawer.editTitle')
);

const formRef = ref<FormInst | null>(null);
const { patternRules, createRequiredRule } = useFormRules();

const model = ref({
  username: '',
  password: '',
  passwordConfirm: '',
  nickname: '',
  email: '',
  phone: ''
});

const rules = computed<Record<string, App.Global.FormRule[]>>(() => {
  const emailRule = model.value.email
    ? [{ pattern: REG_EMAIL, message: $t('form.email.invalid'), trigger: ['blur', 'input'] }]
    : [];

  const phoneRule = model.value.phone
    ? [{ pattern: REG_PHONE, message: $t('form.phone.invalid'), trigger: ['blur', 'input'] }]
    : [];

  const result: Record<string, App.Global.FormRule[]> = {
    username: [
      createRequiredRule($t('form.userName.required')),
      { ...patternRules.userName, trigger: ['blur', 'input'] }
    ],
    nickname: [],
    email: emailRule,
    phone: phoneRule
  };

  if (operateType.value === 'add') {
    result.password = [
      createRequiredRule($t('form.pwd.required')),
      { ...patternRules.pwd, trigger: ['blur', 'input'] }
    ];
    result.passwordConfirm = [
      createRequiredRule($t('form.confirmPwd.required')),
      {
        validator: (_rule, value: string) => value === model.value.password,
        message: $t('form.confirmPwd.invalid'),
        trigger: ['blur', 'input']
      }
    ];
  }

  return result;
});

function resetForm() {
  model.value = { username: '', password: '', passwordConfirm: '', nickname: '', email: '', phone: '' };
  formRef.value?.restoreValidation();
}

function syncFormByMode() {
  if (operateType.value === 'add') {
    resetForm();
    return;
  }

  model.value = {
    username: editingUser.value?.username || '',
    password: '',
    passwordConfirm: '',
    nickname: editingUser.value?.nickname || '',
    email: editingUser.value?.email || '',
    phone: editingUser.value?.phone || ''
  };
  formRef.value?.restoreValidation();
}

watch(
  () => visible.value,
  show => {
    if (show) syncFormByMode();
  }
);

function closeDrawer() {
  visible.value = false;
}

async function handleSubmit() {
  await formRef.value?.validate();
  const payload: Api.User.CreateUserRequest = {
    username: model.value.username,
    password: model.value.password,
    nickname: model.value.nickname || undefined,
    email: model.value.email || undefined,
    phone: model.value.phone || undefined
  };
  emit('submit', payload);
}
</script>

<template>
  <NDrawer v-model:show="visible" :width="500">
    <NDrawerContent :title="title" closable :native-scrollbar="false">
      <NForm ref="formRef" :model="model" :rules="rules" label-placement="top">
        <NFormItem :label="$t('page.system-manage.user.form.username')" path="username">
          <NInput v-model:value="model.username" :disabled="operateType === 'edit'" />
        </NFormItem>

        <NFormItem v-if="operateType === 'add'" :label="$t('page.system-manage.user.form.password')" path="password">
          <NInput v-model:value="model.password" type="password" show-password-on="click" />
        </NFormItem>

        <NFormItem
          v-if="operateType === 'add'"
          :label="$t('page.system-manage.user.form.passwordConfirm')"
          path="passwordConfirm"
        >
          <NInput v-model:value="model.passwordConfirm" type="password" show-password-on="click" />
        </NFormItem>

        <NFormItem :label="$t('page.system-manage.user.form.nickname')" path="nickname">
          <NInput v-model:value="model.nickname" />
        </NFormItem>

        <NFormItem :label="$t('page.system-manage.user.form.email')" path="email">
          <NInput v-model:value="model.email" />
        </NFormItem>

        <NFormItem :label="$t('page.system-manage.user.form.phone')" path="phone">
          <NInput v-model:value="model.phone" />
        </NFormItem>
      </NForm>

      <template #footer>
        <NSpace justify="end">
          <NButton @click="closeDrawer">{{ $t('common.cancel') }}</NButton>
          <NButton type="primary" :loading="loading" @click="handleSubmit">{{ $t('common.confirm') }}</NButton>
        </NSpace>
      </template>
    </NDrawerContent>
  </NDrawer>
</template>
