<template>
  <n-modal
    :show="show"
    preset="card"
    :title="t('auth.logout')"
    style="width: 400px; max-width: 95vw;"
    :bordered="false"
    :auto-focus="false"
    @update:show="$emit('update:show', $event)"
  >
    <div class="logout-form">
      <p>{{ t('message.logout_confirm') }}</p>
      <div class="logout-actions">
        <button class="btn btn--secondary" @click="$emit('update:show', false)">{{ t('common.cancel') }}</button>
        <button class="btn btn--primary" @click="onLogout">{{ t('auth.logout') }}</button>
      </div>
    </div>
  </n-modal>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { NModal, useMessage } from 'naive-ui'
import { useUserStore } from '@/stores/user'

const { t } = useI18n()
const message = useMessage()
const userStore = useUserStore()

defineProps<{ show: boolean }>()
const emit = defineEmits<{
  'update:show': [value: boolean]
  success: []
}>()

const onLogout = () => {
  userStore.$reset()
  message.success(t('message.logout_success'))
  emit('success')
}
</script>

<style scoped>
.logout-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-3);
  padding-top: var(--space-3);
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
.btn--primary { background: var(--accent); color: #fff; }
.btn--secondary { background: transparent; border: 1px solid var(--border); color: var(--text-primary); }
@media (hover: hover) {
  .btn--primary:hover:not(:disabled) { background: var(--accent-hover); }
  .btn--secondary:hover { border-color: var(--border-hover); background: var(--bg-tertiary); }
}
</style>
