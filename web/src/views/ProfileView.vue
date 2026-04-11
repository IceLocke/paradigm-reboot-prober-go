<template>
  <div v-if="!userStore.logged_in" class="login">
    <h2>{{ t('message.not_logged_in') }}</h2>
  </div>
  <div v-else class="page-container">
    <div class="page-header">
      <h2>{{ t('auth.profile') }}</h2>
    </div>

    <div v-if="userStore.profile" class="profile-form">
      <!-- Read-only info -->
      <div class="form-section">
        <div class="form-row">
          <span class="form-label">{{ t('auth.username') }}</span>
          <span class="form-value">{{ userStore.profile.username }}</span>
        </div>

        <div class="form-row">
          <span class="form-label">{{ t('auth.email') }}</span>
          <span class="form-value">{{ userStore.profile.email }}</span>
        </div>

        <!-- Upload Token -->
        <div class="token-field">
          <BaseInput
            v-model="tokenDisplay"
            :label="t('auth.upload_token')"
            :readonly="true"
          />
          <IconButton type="button" :icon="Copy" :size="16" @click="onCopyToken" :title="t('common.copy')" />
          <IconButton type="button" :icon="RefreshCw" :size="16" @click="onRefreshToken" :title="t('common.refresh')" />
        </div>

        <div class="form-row">
          <span class="form-label">{{ t('auth.password') }}</span>
          <BaseButton type="button" variant="secondary" size="sm" @click="showChangePassword = true" :text="t('auth.change_password')" />
        </div>
      </div>

      <n-divider />

      <form @submit.prevent="onSave" class="form-section">
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

        <!-- Anonymous probe -->
        <div class="form-row">
          <label class="form-label">{{ t('auth.anonymous_probe') }}</label>
          <span class="radio-group">
            <label class="radio-item">
              <input type="radio" :value="true" v-model="form.anonymous_probe" />
              <span>{{ t('common.yes') }}</span>
            </label>
            <label class="radio-item">
              <input type="radio" :value="false" v-model="form.anonymous_probe" />
              <span>{{ t('common.no') }}</span>
            </label>
          </span>
        </div>

        <div class="form-actions">
          <BaseButton type="submit" :disabled="loading" :text="t('common.save')" />
        </div>
      </form>
    </div>
  </div>

  <ConfirmModal
    v-model:show="showConfirm"
    :message="t('message.token_refresh_confirm')"
    @confirm="refreshToken"
  />
  <ChangePasswordModal
    v-model:show="showChangePassword"
  />
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted } from 'vue'
import { useMessage, NDivider } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { Copy, RefreshCw } from '@lucide/vue';
import { useUserStore } from '@/stores/user'
import { updateMyInfo, refreshUploadToken } from '@/api/user'
import { USE_MOCK } from '@/api/mock'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseInput from '@/components/ui/BaseInput.vue'
import ConfirmModal from '@/components/ui/ConfirmModal.vue'
import IconButton from '@/components/ui/IconButton.vue'
import ChangePasswordModal from '@/components/business/ChangePasswordModal.vue'

const { t } = useI18n()
const message = useMessage()
const userStore = useUserStore()

const form = reactive({
  nickname: '',
  qq_account: '',
  anonymous_probe: false,
})
const loading = ref(false)
const showConfirm = ref(false)
const showChangePassword = ref(false)
const tokenDisplay = computed(() => userStore.profile?.upload_token ?? '')

const resetForm = () => {
  if (userStore.profile) {
    form.nickname = userStore.profile.nickname ?? ''
    form.qq_account = userStore.profile.qq_account ?? ''
    form.anonymous_probe = userStore.profile.anonymous_probe ?? false
  } else {
    form.nickname = ''
    form.qq_account = ''
    form.anonymous_probe = false
  }
}
onMounted(resetForm)
watch(() => userStore.profile, resetForm)

const onSave = async () => {
  loading.value = true

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
    message.success(t('message.update_profile_success'))
  } catch {
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

const onRefreshToken = () => {
  showConfirm.value = true
}

const refreshToken = async () => {
  showConfirm.value = false
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
.login {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100%;
}
.profile-form {
  display: flex;
  flex-direction: column;
  margin-top: var(--space-4);
}
.form-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}
.form-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
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
}
.radio-item input[type="radio"] {
  accent-color: var(--accent);
  width: 18px;
  height: 18px;
}
.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}
</style>
