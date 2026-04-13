<template>
  <n-modal
    :show="show"
    preset="card"
    :title="t('common.upload_record')"
    style="width: 400px; max-width: 90vw; max-height: 90vh;"
    :bordered="false"
    content-scrollable
    @update:show="$emit('update:show', $event)"
  >
    <form class="upload-form" @submit.prevent="onSubmit">
      <div class="form-info">
        <div class="chart-cover">
          <img
            :src="coverUrl"
            :alt="title"
            class="cover-img"
            loading="lazy"
          />
        </div>
        <div class="chart-info">
          <div class="info-row">
            <span class="info-label">{{ t('term.title') }}</span>
            <span class="info-value">{{ title }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">{{ t('term.difficulty') }}</span>
            <span><DifficultyBadge :difficulty="difficulty" :level="level" /></span>
          </div>
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
        <n-radio-group
          v-model:value="isReplace"
        >
          <span class="radio-group">
            <n-radio :value="true">{{ t('common.yes') }}</n-radio>
            <n-radio :value="false">{{ t('common.no') }}</n-radio>
          </span>
        </n-radio-group>
      </div>

      <div class="form-actions">
        <BaseButton type="button" variant="secondary" @click="$emit('update:show', false)" :text="t('common.cancel')" />
        <BaseButton type="submit" :disabled="loading" :text="t('common.submit')" />
      </div>
    </form>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { NModal, NRadioGroup, NRadio } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { useUserStore } from '@/stores/user'
import { uploadRecords } from '@/api/record'
import type { Difficulty } from '@/api/types'
import { USE_MOCK } from '@/api/mock'
import { toastSuccess, toastError } from '@/utils/toast'
import DifficultyBadge from './DifficultyBadge.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import BaseInput from '@/components/ui/BaseInput.vue'

const { t } = useI18n()
const userStore = useUserStore()

const show = defineModel<boolean>('show', { required: true })

const props = defineProps<{
  title: string
  difficulty: Difficulty
  level: number
  chartId: number
  cover: string
}>()

const emit = defineEmits<{
  'success': []
}>()

const coverUrl = computed(() => {
  if (!props.cover) return ''
  if (props.cover.startsWith('http')) return props.cover
  return `/cover/${props.cover}`
})

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
    toastSuccess('message.post_record_success')
    emit('success')
    show.value = false
    scoreStr.value = ''
  } catch (err: unknown) {
    toastError('message.post_record_failed', err)
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
  gap: var(--space-3);
  padding: var(--space-4);
  background: var(--bg-secondary);
  border-radius: 8px;
}
.chart-cover {
  display: flex;
  justify-content: center;
  align-items: center;
  width: 96px;
  max-width: 25%;
}
.cover-img {
  width: 100%;
  height: auto;
  border-radius: 8px;
}
.chart-info {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
.info-row {
  display: flex;
  flex-direction: column;
  gap: 4px;
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
  justify-content: space-between;
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
.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-3);
  padding-top: var(--space-3);
}
</style>
