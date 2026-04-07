<template>
  <n-modal
    :show="show"
    preset="card"
    :title="t('common.import_csv')"
    style="width: 480px; max-width: 95vw;"
    :bordered="false"
    :auto-focus="false"
    @update:show="$emit('update:show', $event)"
  >
    <div class="import-form">
      <!-- File picker -->
      <div class="file-picker">
        <input
          ref="fileInputRef"
          type="file"
          accept=".csv,text/csv,application/csv,text/comma-separated-values"
          class="file-input-hidden"
          @change="onFileSelected"
        />
        <button type="button" class="btn btn--secondary file-btn" @click="fileInputRef?.click()">
          <File :size="16" />
          {{ fileName || t('common.select_file') }}
        </button>
      </div>

      <!-- Preview -->
      <div v-if="previewText" class="preview-box">
        <span class="preview-text">{{ previewText }}</span>
      </div>

      <!-- Error -->
      <div v-if="errorMsg" class="error-msg">{{ errorMsg }}</div>

      <!-- Replace mode -->
      <div v-if="filteredRecords.length > 0" class="form-field">
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

      <!-- Actions -->
      <div class="form-actions">
        <button type="button" class="btn btn--secondary" @click="$emit('update:show', false)">{{ t('common.cancel') }}</button>
        <button
          type="button"
          class="btn btn--primary"
          :disabled="filteredRecords.length === 0 || loading"
          @click="onUpload"
        >
          {{ loading ? t('common.loading') : t('common.upload_to_server') }}
        </button>
      </div>
    </div>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { NModal, useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { File } from '@lucide/vue';
import { useUserStore } from '@/stores/user'
import { uploadRecords, getAllChartsWithScores } from '@/api/record'
import { USE_MOCK, getMockAllCharts } from '@/api/mock'
import { decodeFileBuffer, parseCsvToRecords, filterUnchangedRecords } from '@/utils/csv'
import type { CsvRecord } from '@/utils/csv'

const { t } = useI18n()
const message = useMessage()
const userStore = useUserStore()

defineProps<{ show: boolean }>()
const emit = defineEmits<{
  'update:show': [value: boolean]
  'success': []
}>()

const fileInputRef = ref<HTMLInputElement | null>(null)
const fileName = ref('')
const parsedRecords = ref<CsvRecord[]>([])
const filteredRecords = ref<CsvRecord[]>([])
const previewText = ref('')
const errorMsg = ref('')
const isReplace = ref(false)
const loading = ref(false)

// Reset state when modal closes
watch(() => userStore.logged_in, () => { /* noop — keep reactivity */ })

function resetState() {
  fileName.value = ''
  parsedRecords.value = []
  filteredRecords.value = []
  previewText.value = ''
  errorMsg.value = ''
  isReplace.value = false
  loading.value = false
  if (fileInputRef.value) fileInputRef.value.value = ''
}

const onFileSelected = async (e: Event) => {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return

  fileName.value = file.name
  errorMsg.value = ''
  previewText.value = ''
  parsedRecords.value = []
  filteredRecords.value = []

  try {
    // Read and parse CSV
    const buffer = await file.arrayBuffer()
    const text = decodeFileBuffer(buffer)
    const records = parseCsvToRecords(text)

    if (records.length === 0) {
      errorMsg.value = t('message.csv_no_valid_records')
      return
    }
    parsedRecords.value = records

    // Fetch current best scores for filtering
    let bestMap: Map<number, number>
    if (USE_MOCK) {
      const mock = getMockAllCharts()
      bestMap = new Map(mock.charts.map((c) => [c.id, c.score]))
    } else {
      const res = await getAllChartsWithScores(userStore.username)
      bestMap = new Map(res.data.charts.map((c) => [c.id, c.score]))
    }

    const filtered = filterUnchangedRecords(records, bestMap)
    filteredRecords.value = filtered

    const skipped = records.length - filtered.length
    if (filtered.length === 0) {
      previewText.value = t('message.csv_no_valid_records')
    } else {
      previewText.value = t('message.csv_records_preview', {
        total: records.length,
        count: filtered.length,
      }) + (skipped > 0 ? ` (${skipped} skipped)` : '')
    }
  } catch {
    errorMsg.value = t('message.csv_parse_error')
  }
}

const BATCH_SIZE = 500

const onUpload = async () => {
  if (filteredRecords.value.length === 0) return
  loading.value = true
  errorMsg.value = ''

  try {
    const records = filteredRecords.value
    let uploaded = 0

    for (let i = 0; i < records.length; i += BATCH_SIZE) {
      const batch = records.slice(i, i + BATCH_SIZE)
      if (!USE_MOCK) {
        await uploadRecords(userStore.username, {
          play_records: batch.map((r) => ({ chart_id: r.chartId, score: r.score })),
          is_replace: isReplace.value,
        })
      }
      uploaded += batch.length
    }

    message.success(t('message.csv_import_success', { count: uploaded }))
    emit('success')
    emit('update:show', false)
    resetState()
  } catch (err: unknown) {
    const e = err as { response?: { data?: { error?: string } } }
    errorMsg.value = t('message.csv_import_failed') + (e.response?.data?.error ?? '')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.import-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}
.file-input-hidden {
  display: none;
}
.file-picker {
  display: flex;
}
.file-btn {
  flex: 1;
  gap: var(--space-2);
  justify-content: center;
}
.preview-box {
  padding: var(--space-3) var(--space-4);
  background: var(--bg-secondary);
  border-radius: 8px;
}
.preview-text {
  font-size: var(--text-sm);
  color: var(--text-secondary);
}
.error-msg {
  font-size: var(--text-sm);
  color: var(--color-danger);
  padding: var(--space-2) var(--space-3);
  background: rgba(239, 68, 68, 0.1);
  border-radius: 6px;
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
