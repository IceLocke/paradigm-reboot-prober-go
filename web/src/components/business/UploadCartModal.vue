<template>
  <n-modal
    :show="show"
    preset="card"
    :title="t('term.upload_list')"
    style="width: 400px; max-width: 90vw; max-height: 90vh;"
    content-style="display: flex; flex-direction: column; overflow: hidden;"
    :bordered="false"
    :auto-focus="false"
    @update:show="$emit('update:show', $event)"
  >
    <div v-if="isEmpty" class="empty-state">
      <p>{{ t('common.no_data') }}</p>
    </div>
    <form v-else id="upload-form" class="cart-form" @submit.prevent="onSubmit">
      <div class="cart-list">
        <div v-for="(item, index) in appStore.uploadList" :key="item.chart_id" class="cart-item">
          <div class="cart-info">
            <span class="cart-title">{{ item.title }}</span>
            <span><DifficultyBadge :difficulty="item.difficulty as Difficulty" :level="item.level" :short="true" /></span>
          </div>
          <div class="cart-score">
            <input
              type="number"
              class="score-input"
              v-model.number="appStore.uploadList[index].new_score"
              v-bind:placeholder="String(appStore.uploadList[index].score ?? t('term.score'))"
              min="0"
              max="1010000"
            />
          </div>
          <IconButton type="button" class="remove-btn" :icon="X" :size="16" @click="removeItem(item.chart_id)" :title="t('common.cancel')" />
        </div>
      </div>
    </form>
    <template #footer>
      <div v-if="!isEmpty" class="cart-actions">
        <BaseButton form="upload-form" type="submit" size="sm" :disabled="loading" :text="t('common.submit') + '(' + appStore.uploadList.length + ')'" />
      </div>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NModal } from 'naive-ui'
import { X } from '@lucide/vue';
import { toastSuccess, toastError } from '@/utils/toast'
import { useAppStore } from '@/stores/app'
import { useUserStore } from '@/stores/user'
import { uploadRecords } from '@/api/record'
import type { Difficulty } from '@/api/types'
import { USE_MOCK } from '@/api/mock'
import BaseButton from '@/components/ui/BaseButton.vue'
import IconButton from '@/components/ui/IconButton.vue'
import DifficultyBadge from './DifficultyBadge.vue'

const { t } = useI18n()
const appStore = useAppStore()
const userStore = useUserStore()
const isEmpty = computed(() => appStore.uploadList.length === 0)
const loading = ref(false)

const show = defineModel<boolean>('show', { required: true })

const removeItem = (chartId: number) => {
  appStore.uploadList = appStore.uploadList.filter((item) => item.chart_id !== chartId)
}

const onSubmit = async () => {
  const records = appStore.uploadList
    .map((item) => ({
      chart_id: item.chart_id,
      score: item.new_score ?? 0,
    }))
    .filter((record) => record.score > 0)

  loading.value = true
  try {
    if (!USE_MOCK && records.length > 0) {
      await uploadRecords(userStore.username, { play_records: records })
    }
    appStore.uploadList = []
    toastSuccess('message.post_record_success')
  } catch (err: unknown) {
    toastError('message.post_record_failed', err)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.empty-state {
  padding: var(--space-6);
  text-align: center;
  color: var(--text-muted);
}
.cart-form {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}
.cart-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}
.cart-item {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-3);
  background: var(--bg-secondary);
  border-radius: 6px;
}
.cart-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.cart-title {
  font-size: var(--text-sm);
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.score-input {
  width: 100px;
  padding: 4px 8px;
  background: var(--bg-tertiary);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: var(--text-sm);
  font-family: var(--font-mono);
  outline: none;
  min-height: 36px;
}
.score-input:focus { border-color: var(--accent); }
.remove-btn {
  width: 32px;
  height: 32px;
  color: var(--text-muted);
  flex-shrink: 0;
}
@media (hover: hover) {
  .remove-btn:hover { background: rgba(239, 68, 68, 0.15); color: var(--color-danger); }
}
.cart-actions {
  display: flex;
  justify-content: flex-end;
}
</style>
