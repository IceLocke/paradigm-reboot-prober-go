<template>
  <n-modal
    :show="show"
    preset="card"
    :title="t('auth.register')"
    style="width: 420px; max-width: 95vw;"
    :bordered="false"
    @update:show="$emit('update:show', $event)"
  >
    <div class="auth-form">
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
      <button class="btn btn--primary btn--full" @click="onSubmit" :disabled="loading">
        {{ t('auth.register') }}
      </button>
      <p class="auth-switch">
        {{ t('auth.login') }}?
        <a @click.prevent="$emit('goLogin')" class="switch-link">{{ t('auth.login') }}</a>
      </p>
    </div>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { NModal } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { register } from '@/api/user'
import { USE_MOCK } from '@/api/mock'
import BaseInput from '@/components/ui/BaseInput.vue'

const { t } = useI18n()

defineProps<{ show: boolean }>()
const emit = defineEmits<{
  'update:show': [value: boolean]
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
    emit('success')
  } catch (err: unknown) {
    const error = err as { response?: { data?: { error?: string } } }
    errorMsg.value = t('message.register_failed') + (error.response?.data?.error ?? '')
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
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 10px 16px;
  font-weight: 500;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  transition: background var(--transition-base);
  font-family: inherit;
  font-size: var(--text-base);
  min-height: 44px;
}
.btn:disabled { opacity: 0.4; cursor: not-allowed; }
.btn--primary { background: var(--accent); color: #fff; }
.btn--full { width: 100%; }
@media (hover: hover) {
  .btn--primary:hover:not(:disabled) { background: var(--accent-hover); }
}
.auth-switch {
  text-align: center;
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
