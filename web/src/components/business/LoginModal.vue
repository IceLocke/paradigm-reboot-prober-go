<template>
  <n-modal
    :show="show"
    preset="card"
    :title="t('auth.login')"
    style="width: 400px; max-width: 95vw;"
    :bordered="false"
    @update:show="$emit('update:show', $event)"
  >
    <div class="auth-form">
      <BaseInput
        v-model="form.username"
        :label="t('auth.username')"
        :placeholder="t('auth.username')"
      />
      <BaseInput
        v-model="form.password"
        :label="t('auth.password')"
        type="password"
        :placeholder="t('auth.password')"
      />
      <div v-if="errorMsg" class="error-msg">{{ errorMsg }}</div>
      <button class="btn btn--primary btn--full" @click="onSubmit" :disabled="loading">
        {{ t('auth.login') }}
      </button>
      <p class="auth-switch">
        {{ t('auth.register') }}?
        <a @click.prevent="$emit('goRegister')" class="switch-link">{{ t('auth.register') }}</a>
      </p>
    </div>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { NModal } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { useUserStore } from '@/stores/user'
import { login } from '@/api/user'
import { USE_MOCK } from '@/api/mock'
import BaseInput from '@/components/ui/BaseInput.vue'

const { t } = useI18n()
const userStore = useUserStore()

defineProps<{ show: boolean }>()
const emit = defineEmits<{
  'update:show': [value: boolean]
  success: []
  goRegister: []
}>()

const form = reactive({ username: '', password: '' })
const loading = ref(false)
const errorMsg = ref('')

const onSubmit = async () => {
  if (!form.username || !form.password) {
    errorMsg.value = t('message.required')
    return
  }

  loading.value = true
  errorMsg.value = ''

  try {
    if (USE_MOCK) {
      userStore.$patch({
        logged_in: true,
        username: form.username.toLowerCase(),
        access_token: 'Bearer mock-token-12345',
      })
    } else {
      const res = await login(form.username, form.password)
      userStore.$patch({
        logged_in: true,
        username: form.username.toLowerCase(),
        access_token: `Bearer ${res.data.access_token}`,
      })
    }
    form.username = ''
    form.password = ''
    emit('success')
  } catch (err: unknown) {
    const error = err as { response?: { data?: { error?: string } } }
    errorMsg.value = t('message.login_failed') + (error.response?.data?.error ?? '')
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
