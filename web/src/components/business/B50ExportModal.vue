<template>
  <n-modal
    :show="show"
    preset="card"
    :title="t('common.export_image')"
    style="width: 420px; max-width: 90vw; max-height: 90vh;"
    content-style="display: flex; flex-direction: column; overflow: hidden;"
    :bordered="false"
    :auto-focus="false"
    content-scrollable
    @update:show="$emit('update:show', $event)"
  >
    <div class="export-options">
      <span class="option-label">{{ t('term.export_mode') }}</span>
      <n-radio-group v-model:value="mode" class="mode-group">
        <n-radio value="standard">{{ t('term.b50_standard') }}</n-radio>
        <n-radio value="overflow">{{ t('term.b50_overflow') }}</n-radio>
        <n-radio value="global">{{ t('term.b50_global') }}</n-radio>
      </n-radio-group>
    </div>

    <template #footer>
      <div class="confirm-actions">
        <BaseButton variant="secondary" @click="show = false" :text="t('common.cancel')" />
        <BaseButton
          :disabled="exporting"
          @click="handleExport"
          :text="exporting ? t('common.loading') : t('common.export_image')"
        />
      </div>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NModal, NRadioGroup, NRadio } from 'naive-ui'
import { saveAs } from 'file-saver'
import BaseButton from '@/components/ui/BaseButton.vue'
import { getRecords } from '@/api/record'
import { USE_MOCK } from '@/api/mock'
import { renderB50Image, type B50Section } from '@/utils/b50Canvas'
import type { PlayRecordInfo } from '@/api/types'
import { toastSuccess, toastError } from '@/utils/toast'

const { t } = useI18n()

const show = defineModel<boolean>('show', { required: true })

const props = defineProps<{
  username: string
  nickname: string
  rating: number
  b15Records: PlayRecordInfo[]
  b35Records: PlayRecordInfo[]
  b15Avg: number
  b35Avg: number
}>()

const mode = ref<'standard' | 'overflow' | 'global'>('standard')
const exporting = ref(false)

function avgRating(records: PlayRecordInfo[]): number {
  if (records.length === 0) return 0
  return records.reduce((s, r) => s + r.rating, 0) / (records.length * 100)
}

const handleExport = async () => {
  if (exporting.value) return
  exporting.value = true

  try {
    let sections: B50Section[]
    let rating = props.rating
    let title: string | undefined

    if (mode.value === 'standard') {
      sections = [
        { label: 'Best 15', avg: props.b15Avg, records: props.b15Records },
        { label: 'Best 35', avg: props.b35Avg, records: props.b35Records },
      ]
    } else if (mode.value === 'overflow') {
      if (USE_MOCK) {
        toastError('message.export_image_failed')
        return
      }
      const res = await getRecords(props.username, 'b50', 50, 1, 'rating', 'desc', undefined, 5)
      const records = res.data.records
      const b15 = records.filter((r) => r.chart.b15)
      const b35 = records.filter((r) => !r.chart.b15)
      sections = [
        { label: 'Best 15', avg: avgRating(b15.slice(0, 15)), records: b15, cutoff: 15 },
        { label: 'Best 35', avg: avgRating(b35.slice(0, 35)), records: b35, cutoff: 35 },
      ]
    } else {
      if (USE_MOCK) {
        toastError('message.export_image_failed')
        return
      }
      const res = await getRecords(props.username, 'all', 50, 1, 'rating', 'desc')
      const records = res.data.records
      sections = [
        { label: 'Best 50', avg: avgRating(records), records: records.slice(0, 50) },
      ]
      rating = avgRating(records.slice(0, 50))
      title = 'Paradigm: Reboot Global Best Records'
    }

    const blob = await renderB50Image({
      sections,
      username: props.username,
      nickname: props.nickname,
      rating,
      title,
    })
    saveAs(blob, `b50_${mode.value}_${Date.now()}.jpg`)
    toastSuccess('message.export_image_success')
    show.value = false
  } catch (err: unknown) {
    toastError('message.export_image_failed', err)
  } finally {
    exporting.value = false
  }
}
</script>

<style scoped>
.export-options {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  margin: var(--space-2) 0;
}
.option-label {
  font-size: var(--text-sm);
  color: var(--text-muted);
  font-weight: 500;
}
.mode-group {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}
.confirm-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}
</style>
