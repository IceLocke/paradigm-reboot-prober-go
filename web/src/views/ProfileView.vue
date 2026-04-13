<template>
  <div v-if="!userStore.logged_in" class="login">
    <h2>{{ t('message.not_logged_in') }}</h2>
  </div>
  <div v-else class="page-container">
    <div class="page-header">
      <h2>{{ t('auth.profile') }}</h2>
    </div>

    <div v-if="userStore.profile" class="profile-form">
      <div class="form-column form-column--left">
        <BaseCard>
          <template #header>
            <h3>{{ t('auth.account_info') }}</h3>
          </template>
          <form @submit.prevent="onSave" class="form-section">
            <div class="form-row">
              <span class="form-label">{{ t('auth.username') }}</span>
              <span class="form-value">{{ userStore.profile.username }}</span>
            </div>

            <div class="form-row">
              <span class="form-label">{{ t('auth.email') }}</span>
              <span class="form-value">{{ userStore.profile.email }}</span>
            </div>

            <div class="form-row">
              <span class="form-label">{{ t('auth.password') }}</span>
              <BaseButton type="button" variant="secondary" size="sm" @click="showChangePassword = true" :text="t('auth.change_password')" />
            </div>

            <BaseInput
              v-model="form.nickname"
              :label="t('auth.nickname')"
            />

            <BaseInput
              v-model="form.qq_account"
              :label="t('auth.qq_account')"
              :placeholder="t('auth.qq_account')"
            />

            <div class="form-actions">
              <BaseButton type="submit" :disabled="loading" :text="t('common.save')" />
            </div>
          </form>
        </BaseCard>
      </div>

      <div class="form-column form-column--right">
        <BaseCard>
          <template #header>
            <h3>{{ t('auth.prober_settings') }}</h3>
          </template>
          <div class="form-section">
            <div class="form-row">
              <label class="form-label">{{ t('auth.anonymous_probe') }}</label>
              <n-radio-group
                v-model:value="anonymousProbe"
                @update:value="updateAnonymousProbe"
              >
                <span class="radio-group">
                  <n-radio :value="true">{{ t('common.yes') }}</n-radio>
                  <n-radio :value="false">{{ t('common.no') }}</n-radio>
                </span>
              </n-radio-group>
            </div>

            <div class="token-field">
              <BaseInput
                v-model="tokenDisplay"
                :label="t('auth.upload_token')"
                :readonly="true"
              />
              <IconButton type="button" :icon="Copy" :size="16" @click="onCopyToken" :title="t('common.copy')" />
              <IconButton type="button" :icon="RefreshCw" :size="16" @click="onRefreshToken" :title="t('common.refresh')" />
            </div>
          </div>
        </BaseCard>
      </div>
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
import { NRadioGroup, NRadio } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { Copy, RefreshCw } from '@lucide/vue';
import { toastSuccess, toastError } from '@/utils/toast'
import { useUserStore } from '@/stores/user'
import { updateMyInfo, refreshUploadToken } from '@/api/user'
import { USE_MOCK } from '@/api/mock'
import BaseCard from '@/components/ui/BaseCard.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseInput from '@/components/ui/BaseInput.vue'
import ConfirmModal from '@/components/ui/ConfirmModal.vue'
import IconButton from '@/components/ui/IconButton.vue'
import ChangePasswordModal from '@/components/business/ChangePasswordModal.vue'

const { t } = useI18n()
const userStore = useUserStore()

const form = reactive({
  nickname: '',
  qq_account: '',
})
const anonymousProbe = ref(false)
const loading = ref(false)
const showConfirm = ref(false)
const showChangePassword = ref(false)
const tokenDisplay = computed(() => userStore.profile?.upload_token ?? '')

const resetForm = () => {
  if (userStore.profile) {
    form.nickname = userStore.profile.nickname ?? ''
    form.qq_account = userStore.profile.qq_account ?? ''
    anonymousProbe.value = userStore.profile.anonymous_probe ?? false
  } else {
    form.nickname = ''
    form.qq_account = ''
    anonymousProbe.value = false
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
      })
    }
    if (userStore.profile) {
      userStore.profile.nickname = form.nickname
      userStore.profile.qq_account = form.qq_account
    }
    toastSuccess('message.update_profile_success')
  } catch {
    toastError('message.update_profile_failed')
  } finally {
    loading.value = false
  }
}

const onCopyToken = async () => {
  try {
    await navigator.clipboard.writeText(tokenDisplay.value)
    toastSuccess('message.copy_success')
  } catch {
    toastError('message.copy_failed')
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
    toastSuccess('message.refresh_upload_token_success')
  } catch {
    toastError('message.refresh_upload_token_failed')
  }
}

const updateAnonymousProbe = async () => {
  const allowAnonymousProbe = anonymousProbe.value
  try {
    if (!USE_MOCK) {
      await updateMyInfo({
        anonymous_probe: allowAnonymousProbe,
      })
    }
    if (userStore.profile) {
      userStore.profile.anonymous_probe = allowAnonymousProbe
    }
    toastSuccess('message.update_profile_success')
  } catch {
    anonymousProbe.value = !allowAnonymousProbe
    toastError('message.update_profile_failed')
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
  display: grid;
  grid-template-columns: 1fr 1fr;
  grid-template-areas: "left right";
  align-items: start;
  gap: var(--space-5);
}
.form-column {
  display: grid;
  row-gap: var(--space-5);
}
.form-column--left {
  grid-area: left;
}
.form-column--right {
  grid-area: right;
}
@media (max-width: 1023px) {
  .profile-form {
    grid-template-columns: 1fr;
    grid-template-areas: "left" "right";
  }
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
.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}
</style>
