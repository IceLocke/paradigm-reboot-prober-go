<template>
  <n-modal
    :show="show"
    preset="card"
    :title="t('auth.profile')"
    style="width: 500px; max-width: 95vw;"
    :bordered="false"
    :auto-focus="false"
    @update:show="$emit('update:show', $event)"
  >
    <form v-if="userStore.profile" class="profile-form" @submit.prevent="onSave">
      <!-- Read-only info -->
      <div class="form-row">
        <span class="form-label">{{ t('auth.username') }}</span>
        <span class="form-value">{{ userStore.profile.username }}</span>
      </div>
      <div class="form-row">
        <span class="form-label">{{ t('auth.email') }}</span>
        <span class="form-value">{{ userStore.profile.email }}</span>
      </div>

      <!-- Editable fields -->
      <BaseInput
        v-model="form.nickname"
        :label="t('auth.nickname')"
      />

      <BaseInput
        v-model="form.qq_account"
        :label="t('auth.qq_account')"
        :placeholder="t('auth.qq_account')"
      />

      <!-- Upload Token -->
      <div class="token-field">
        <BaseInput
          v-model="tokenDisplay"
          :label="t('auth.upload_token')"
          :readonly="true"
        />
        <button type="button" class="icon-btn" @click="onCopyToken" :title="t('common.copy')">
          <Copy :size="16" />
        </button>
        <button type="button" class="icon-btn" @click="onRefreshToken" :title="t('common.refresh')">
          <RefreshCw :size="16" />
        </button>
      </div>

      <!-- Anonymous probe -->
      <div class="form-field">
        <label class="form-label">{{ t('auth.anonymous_probe') }}</label>
        <div class="radio-group">
          <label class="radio-item">
            <input type="radio" :value="true" v-model="form.anonymous_probe" />
            <span>{{ t('common.yes') }}</span>
          </label>
          <label class="radio-item">
            <input type="radio" :value="false" v-model="form.anonymous_probe" />
            <span>{{ t('common.no') }}</span>
          </label>
        </div>
      </div>

      <div v-if="errorMsg" class="error-msg">{{ errorMsg }}</div>
      <div v-if="successMsg" class="success-msg">{{ successMsg }}</div>

      <div class="form-actions">
        <button type="button" class="btn btn--secondary" @click="$emit('update:show', false)">{{ t('common.cancel') }}</button>
        <button type="submit" class="btn btn--primary" :disabled="loading">{{ t('common.save') }}</button>
      </div>
    </form>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, reactive, watch, computed } from 'vue'
import { NModal, useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { Copy, RefreshCw } from '@lucide/vue';
import { useUserStore } from '@/stores/user'
import { updateMyInfo, refreshUploadToken } from '@/api/user'
import { USE_MOCK } from '@/api/mock'
import BaseInput from '@/components/ui/BaseInput.vue'

const { t } = useI18n()
const message = useMessage()
const userStore = useUserStore()

const props = defineProps<{ show: boolean }>()
defineEmits<{ 'update:show': [value: boolean] }>()

const form = reactive({
  nickname: '',
  qq_account: '',
  anonymous_probe: false,
})
const loading = ref(false)
const errorMsg = ref('')
const successMsg = ref('')

const tokenDisplay = computed(() => userStore.profile?.upload_token ?? '')

watch(() => props.show, (val) => {
  if (val && userStore.profile) {
    form.nickname = userStore.profile.nickname ?? ''
    form.qq_account = userStore.profile.qq_account ?? ''
    form.anonymous_probe = userStore.profile.anonymous_probe ?? false
    errorMsg.value = ''
    successMsg.value = ''
  }
})

const onSave = async () => {
  loading.value = true
  errorMsg.value = ''
  successMsg.value = ''

  try {
    if (!USE_MOCK) {
      await updateMyInfo({
        nickname: form.nickname,
        qq_account: form.qq_account || undefined,
        anonymous_probe: form.anonymous_probe,
      })
    }
    if (userStore.profile) {
      userStore.profile.nickname = form.nickname
      userStore.profile.qq_account = form.qq_account
      userStore.profile.anonymous_probe = form.anonymous_probe
    }
    successMsg.value = t('message.update_profile_success')
    message.success(t('message.update_profile_success'))
  } catch {
    errorMsg.value = t('message.update_profile_failed')
    message.error(t('message.update_profile_failed'))
  } finally {
    loading.value = false
  }
}

const onCopyToken = async () => {
  try {
    await navigator.clipboard.writeText(tokenDisplay.value)
    message.success(t('message.copy_success'))
  } catch {
    message.error(t('message.copy_failed'))
  }
}

const onRefreshToken = async () => {
  try {
    if (USE_MOCK) {
      if (userStore.profile) {
        userStore.profile.upload_token = 'mock-token-' + Math.random().toString(36).slice(2, 10)
      }
    } else {
      const res = await refreshUploadToken()
      if (userStore.profile) {
        userStore.profile.upload_token = res.data.upload_token
      }
    }
    message.success(t('message.refresh_upload_token_success'))
  } catch {
    message.error(t('message.refresh_upload_token_failed'))
  }
}
</script>

<style scoped>
.profile-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}
.form-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-2) 0;
}
.form-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary);
}
.form-value {
  font-size: var(--text-base);
  color: var(--text-primary);
}
.token-field {
  display: flex;
  gap: var(--space-2);
  align-items: flex-end;
}
.token-field > :first-child { flex: 1; }
.form-field {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}
.radio-group {
  display: flex;
  gap: var(--space-4);
}
.radio-item {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  cursor: pointer;
  color: var(--text-primary);
  font-size: var(--text-base);
  min-height: 44px;
}
.radio-item input[type="radio"] {
  accent-color: var(--accent);
  width: 18px;
  height: 18px;
}
.error-msg {
  font-size: var(--text-sm);
  color: var(--color-danger);
  padding: var(--space-2) var(--space-3);
  background: rgba(239, 68, 68, 0.1);
  border-radius: 6px;
}
.success-msg {
  font-size: var(--text-sm);
  color: var(--color-success);
  padding: var(--space-2) var(--space-3);
  background: rgba(34, 197, 94, 0.1);
  border-radius: 6px;
}
.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-3);
  padding-top: var(--space-3);
}
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 8px 16px;
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
.btn--secondary { background: transparent; border: 1px solid var(--border); color: var(--text-primary); }
@media (hover: hover) {
  .btn--primary:hover:not(:disabled) { background: var(--accent-hover); }
  .btn--secondary:hover { border-color: var(--border-hover); background: var(--bg-tertiary); }
}
.icon-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: 8px;
  transition: background var(--transition-fast);
}
@media (hover: hover) {
  .icon-btn:hover { background: rgba(255,255,255,0.06); color: var(--text-primary); }
}
</style>
