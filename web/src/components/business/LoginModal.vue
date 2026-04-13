<template>
  <n-modal
    :show="show"
    preset="card"
    :title="t('auth.login')"
    style="width: 400px; max-width: 90vw; max-height: 90vh;"
    :bordered="false"
    :auto-focus="false"
    content-scrollable
    @update:show="$emit('update:show', $event)"
  >
    <form class="auth-form" @submit.prevent="onSubmit">
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
      <BaseButton type="submit" full :disabled="loading" :text="t('auth.login')" />
      <p class="auth-switch">
        {{ t('auth.no_account') }}?
        <a @click.prevent="$emit('goRegister')" class="switch-link">{{ t('auth.register') }}</a>
      </p>
    </form>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { NModal } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { useUserStore } from '@/stores/user'
import { login } from '@/api/user'
import { USE_MOCK } from '@/api/mock'
import { toastSuccess, formatApiError } from '@/utils/toast'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseInput from '@/components/ui/BaseInput.vue'

const { t } = useI18n()
const userStore = useUserStore()

const show = defineModel<boolean>('show', { required: true })
const emit = defineEmits<{
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
    toastSuccess('message.login_success')
    emit('success')
  } catch (err: unknown) {
    const error = err as { status: number }
    if (error.status === 401) {
      errorMsg.value = t('message.password_incorrect')
    } else {
      errorMsg.value = formatApiError('message.login_failed', err)
    }
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
