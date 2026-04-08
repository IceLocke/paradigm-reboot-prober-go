<template>
  <ConfirmModal
    v-model:show="show"
    :message="t('message.logout_confirm')"
    @confirm="onLogout"
  />
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useMessage } from 'naive-ui'
import { useUserStore } from '@/stores/user'
import ConfirmModal from '@/components/ui/ConfirmModal.vue'

const { t } = useI18n()
const message = useMessage()
const userStore = useUserStore()

const show = defineModel<boolean>('show', { required: true })
const emit = defineEmits<{
  success: []
}>()

const onLogout = () => {
  userStore.$reset()
  message.success(t('message.logout_success'))
  emit('success')
}
</script>
