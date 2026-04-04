<template>
  <n-modal
    :show="show"
    preset="card"
    :title="t('common.upload_record')"
    style="width: 420px; max-width: 95vw;"
    :bordered="false"
    @update:show="$emit('update:show', $event)"
  >
    <div class="upload-form">
      <div class="form-info">
        <div class="info-row">
          <span class="info-label">{{ t('term.title') }}</span>
          <span class="info-value">{{ title }}</span>
        </div>
        <div class="info-row">
          <span class="info-label">{{ t('term.difficulty') }}</span>
          <DifficultyBadge :difficulty="difficulty" :level="level" />
        </div>
      </div>

      <BaseInput
        v-model="scoreStr"
        :label="t('term.score')"
        type="number"
        placeholder="0 - 1010000"
      />

      <div class="form-field">
        <label class="field-label">{{ t('term.replace') }}</label>
        <div class="radio-group">
          <label class="radio-item">
            <input type="radio" :value="false" v-model="isReplace" />
            <span>{{ t('common.no') }}</span>
          </label>
          <label class="radio-item">
            <input type="radio" :value="true" v-model="isReplace" />
            <span>{{ t('common.yes') }}</span>
          </label>
        </div>
      </div>

      <div class="form-actions">
        <button class="btn btn--secondary" @click="$emit('update:show', false)">{{ t('common.cancel') }}</button>
        <button class="btn btn--primary" @click="onSubmit" :disabled="loading">{{ t('common.submit') }}</button>
      </div>
    </div>
  </n-modal>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { NModal } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { useUserStore } from '@/stores/user'
import { uploadRecords } from '@/api/record'
import type { Difficulty } from '@/api/types'
import { USE_MOCK } from '@/api/mock'
import DifficultyBadge from './DifficultyBadge.vue'
import BaseInput from '@/components/ui/BaseInput.vue'

const { t } = useI18n()
const userStore = useUserStore()

const props = defineProps<{
  show: boolean
  title: string
  difficulty: Difficulty
  level: number
  chartId: number
}>()

const emit = defineEmits<{
  'update:show': [value: boolean]
  'success': []
}>()

const scoreStr = ref('')
const isReplace = ref(false)
const loading = ref(false)

const onSubmit = async () => {
  const score = parseInt(scoreStr.value)
  if (isNaN(score) || score < 0 || score > 1010000) {
    return
  }

  loading.value = true
  try {
    if (!USE_MOCK) {
      await uploadRecords(userStore.username, {
        play_records: [{ chart_id: props.chartId, score }],
        is_replace: isReplace.value,
      })
    }
    emit('success')
    emit('update:show', false)
    scoreStr.value = ''
  } catch {
    // Error handled by interceptor
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.upload-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}
.form-info {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding: var(--space-4);
  background: var(--bg-secondary);
  border-radius: 8px;
}
.info-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
}
.info-label {
  font-size: var(--text-sm);
  color: var(--text-muted);
}
.info-value {
  font-size: var(--text-base);
  color: var(--text-primary);
  font-weight: 500;
}
.form-field {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}
.field-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary);
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
</style>
