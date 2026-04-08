<template>
  <n-modal
    :show="show"
    preset="card"
    :title="t('auth.register')"
    style="width: 400px; max-width: 95vw;"
    :bordered="false"
    :auto-focus="false"
    @update:show="$emit('update:show', $event)"
  >
    <form class="auth-form" @submit.prevent="onSubmit">
      <BaseInput
        v-model="form.username"
        :label="t('auth.username')"
        :placeholder="t('message.username_character')"
      />
      <BaseInput
        v-model="form.email"
        :label="t('auth.email')"
        type="email"
        placeholder="user@example.com"
      />
      <BaseInput
        v-model="form.password"
        :label="t('auth.password')"
        type="password"
        :placeholder="t('message.password_character')"
      />
      <BaseInput
        v-model="form.confirmPassword"
        :label="t('auth.confirm_password')"
        type="password"
      />
      <div v-if="errorMsg" class="error-msg">{{ errorMsg }}</div>
      <BaseButton type="submit" full :disabled="loading" :text="t('auth.register')" />
      <p class="auth-switch">
        {{ t('auth.has_account') }}?
        <a @click.prevent="$emit('goLogin')" class="switch-link">{{ t('auth.login') }}</a>
      </p>
    </form>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { NModal, useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { register } from '@/api/user'
import { USE_MOCK } from '@/api/mock'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseInput from '@/components/ui/BaseInput.vue'

const { t } = useI18n()
const message = useMessage()

const show = defineModel<boolean>('show', { required: true })
const emit = defineEmits<{
  success: []
  goLogin: []
}>()

const form = reactive({
  username: '',
  email: '',
  password: '',
  confirmPassword: '',
})
const loading = ref(false)
const errorMsg = ref('')

const onSubmit = async () => {
  if (!form.username || !form.email || !form.password || !form.confirmPassword) {
    errorMsg.value = t('message.required')
    return
  }
  if (form.password !== form.confirmPassword) {
    errorMsg.value = t('message.password_mismatch')
    return
  }

  loading.value = true
  errorMsg.value = ''

  try {
    if (!USE_MOCK) {
      await register({
        username: form.username,
        email: form.email,
        password: form.password,
      })
    }
    form.username = ''
    form.email = ''
    form.password = ''
    form.confirmPassword = ''
    message.success(t('message.register_success'))
    emit('success')
  } catch (err: unknown) {
    const error = err as { response?: { data?: { error?: string } } }
    const msg = t('message.register_failed') + (error.response?.data?.error ?? '')
    errorMsg.value = msg
    message.error(msg)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.auth-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}
.error-msg {
  font-size: var(--text-sm);
  color: var(--color-danger);
  padding: var(--space-2) var(--space-3);
  background: rgba(239, 68, 68, 0.1);
  border-radius: 6px;
}
.auth-switch {
  display: flex;
  justify-content: center;
  gap: var(--space-2);
  font-size: var(--text-sm);
  color: var(--text-muted);
}
.switch-link {
  color: var(--accent);
  cursor: pointer;
  text-decoration: none;
}
.switch-link:hover { text-decoration: underline; }
</style>
