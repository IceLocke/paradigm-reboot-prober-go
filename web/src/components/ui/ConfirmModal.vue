<template>
  <n-modal
    :show="show"
    preset="card"
    :title="t('common.confirm')"
    style="width: 400px; max-width: 90vw; max-height: 90vh;"
    content-style="display: flex; flex-direction: column; overflow: hidden;"
    :bordered="false"
    :auto-focus="false"
    content-scrollable
    @update:show="$emit('update:show', $event)"
  >
    <div class="confirm-message">
      <slot><p>{{ message }}</p></slot>
    </div>
    <template #footer>
      <div class="confirm-actions">
        <BaseButton type="button" variant="secondary" @click="cancel" :text="t('common.cancel')" />
        <BaseButton type="button" @click="confirm" :text="t('common.confirm')" />
      </div>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { NModal } from 'naive-ui'
import BaseButton from '@/components/ui/BaseButton.vue'

const { t } = useI18n()

const show = defineModel<boolean>('show', { required: true })

defineProps<{
  message: string
}>()

const emit = defineEmits<{
  confirm: []
  cancel: []
}>()

const confirm = () => {
  show.value = false
  emit('confirm')
}

const cancel = () => {
  show.value = false
  emit('cancel')
}
</script>

<style scoped>
.confirm-message {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  margin: var(--space-3) 0;
}

.confirm-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}
</style>
