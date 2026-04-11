<template>
  <n-modal
    :show="show"
    preset="card"
    :title="t('auth.change_password')"
    style="width: 400px; max-width: 90vw; max-height: 90vh;"
    :bordered="false"
    :auto-focus="false"
    @update:show="$emit('update:show', $event)"
  >
    <form class="auth-form" @submit.prevent="onSubmit">
      <BaseInput
        v-model="form.old_password"
        :label="t('auth.old_password')"
        type="password"
      />
      <BaseInput
        v-model="form.new_password"
        :label="t('auth.new_password')"
        type="password"
      />
      <BaseInput
        v-model="form.confirm_password"
        :label="t('auth.confirm_password')"
        type="password"
      />
      <div v-if="errorMsg" class="error-msg">{{ errorMsg }}</div>
      <div class="form-actions">
        <BaseButton type="button" variant="secondary" @click="$emit('update:show', false)" :text="t('common.cancel')" />
        <BaseButton type="submit" :disabled="loading" :text="t('common.confirm')" />
      </div>
    </form>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, reactive, watch } from 'vue'
import { NModal, useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { changePassword } from '@/api/user'
import { USE_MOCK } from '@/api/mock'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseInput from '@/components/ui/BaseInput.vue'

const { t } = useI18n()
const message = useMessage()

const show = defineModel<boolean>('show', { required: true })
const emit = defineEmits<{ success: [] }>()

const form = reactive({
  old_password: '',
  new_password: '',
  confirm_password: '',
})
const loading = ref(false)
const errorMsg = ref('')

watch(show, (val) => {
  if (val) {
    form.old_password = ''
    form.new_password = ''
    form.confirm_password = ''
    errorMsg.value = ''
  }
})

const onSubmit = async () => {
  if (!form.old_password || !form.new_password || !form.confirm_password) {
    errorMsg.value = t('message.required')
    return
  }
  if (form.old_password === form.new_password) {
    errorMsg.value = t('message.password_duplicate')
    return
  }
  if (form.confirm_password !== form.new_password) {
    errorMsg.value = t('message.password_mismatch')
    return
  }

  loading.value = true
  errorMsg.value = ''

  try {
    if (!USE_MOCK) {
      await changePassword({
        old_password: form.old_password,
        new_password: form.new_password,
      })
    }
    form.old_password = ''
    form.new_password = ''
    form.confirm_password = ''
    show.value = false
    message.success(t('message.change_password_success'))
    emit('success')
  } catch (err: unknown) {
    const error = err as { status?: number, response?: { data?: { error?: string } } }
    if (error.status === 401) {
      errorMsg.value = t('message.password_incorrect')
    } else {
      const msg = t('message.change_password_failed') + (error.response?.data?.error ?? '')
      errorMsg.value = msg
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
.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-3);
  padding-top: var(--space-3);
}
</style>
