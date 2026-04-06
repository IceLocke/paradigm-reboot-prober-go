<template>
  <div class="upload-cart">
    <div v-if="appStore.uploadList.length === 0" class="empty-state">
      <p>{{ t('common.no_data') }}</p>
    </div>
    <template v-else>
      <div class="cart-list">
        <div v-for="(item, index) in appStore.uploadList" :key="item.chart_id" class="cart-item">
          <div class="cart-info">
            <span class="cart-title">{{ item.title }}</span>
            <DifficultyBadge :difficulty="item.difficulty as Difficulty" :level="item.level" :short="true" />
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
          <button class="remove-btn" @click="removeItem(item.chart_id)" :title="t('common.cancel')">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6 6 18M6 6l12 12"/></svg>
          </button>
        </div>
      </div>
      <div class="cart-actions">
        <button class="btn btn--primary btn--sm" @click="onSubmit" :disabled="loading">
          {{ t('common.submit') }} ({{ appStore.uploadList.length }})
        </button>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useUserStore } from '@/stores/user'
import { uploadRecords } from '@/api/record'
import type { Difficulty } from '@/api/types'
import { USE_MOCK } from '@/api/mock'
import DifficultyBadge from './DifficultyBadge.vue'

const { t } = useI18n()
const message = useMessage()
const appStore = useAppStore()
const userStore = useUserStore()
const loading = ref(false)

const removeItem = (chartId: number) => {
  appStore.uploadList = appStore.uploadList.filter((item) => item.chart_id !== chartId)
}

const onSubmit = async () => {
  const records = appStore.uploadList.map((item) => ({
    chart_id: item.chart_id,
    score: item.new_score ?? 0,
  }))

  loading.value = true
  try {
    if (!USE_MOCK) {
      await uploadRecords(userStore.username, { play_records: records })
    }
    appStore.uploadList = []
    message.success(t('message.post_record_success'))
  } catch (err: unknown) {
    const e = err as { response?: { data?: { error?: string } } }
    message.error(t('message.post_record_failed') + (e.response?.data?.error ?? ''))
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.upload-cart {
  min-width: 300px;
}
.empty-state {
  padding: var(--space-6);
  text-align: center;
  color: var(--text-muted);
}
.cart-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  max-height: 400px;
  overflow-y: auto;
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
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  background: none;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  border-radius: 6px;
  flex-shrink: 0;
}
@media (hover: hover) {
  .remove-btn:hover { background: rgba(239, 68, 68, 0.15); color: var(--color-danger); }
}
.cart-actions {
  display: flex;
  justify-content: flex-end;
  padding-top: var(--space-3);
  margin-top: var(--space-2);
  border-top: 1px solid var(--border);
}
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
  font-weight: 500;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  transition: background var(--transition-base);
  font-family: inherit;
  white-space: nowrap;
}
.btn:disabled { opacity: 0.4; cursor: not-allowed; }
.btn--sm { padding: 6px 12px; font-size: 13px; min-height: 36px; }
.btn--primary { background: var(--accent); color: #fff; }
@media (hover: hover) { .btn--primary:hover:not(:disabled) { background: var(--accent-hover); } }
</style>
