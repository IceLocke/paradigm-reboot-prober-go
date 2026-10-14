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
    <p v-if="fitting" class="fitting-description">{{ t('message.fitting_b50_description') }}</p>

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
import { formatAvgRating } from '@/utils/rating'
import { selectBest50 } from '@/utils/b50'

const { t } = useI18n()

const show = defineModel<boolean>('show', { required: true })

const props = withDefaults(defineProps<{
  username: string
  nickname: string
  rating: string
  b15Records: PlayRecordInfo[]
  b35Records: PlayRecordInfo[]
  b15Avg: string
  b35Avg: string
  fitting?: boolean
  fittingRecords?: PlayRecordInfo[]
}>(), {
  fitting: false,
  fittingRecords: () => [],
})

const mode = ref<'standard' | 'overflow' | 'global'>('standard')
const exporting = ref(false)

function avgRating(records: PlayRecordInfo[], count = records.length): string {
  const sum = records.reduce((s, r) => s + r.rating, 0)
  return formatAvgRating(sum, count)
}

const handleExport = async () => {
  if (exporting.value) return
  exporting.value = true
  const exportData = { ...props }
  const exportMode = mode.value

  try {
    let sections: B50Section[]
    let rating = exportData.rating
    let title = exportData.fitting ? t('term.fitting_b50_image_title') : undefined

    if (exportMode === 'standard') {
      sections = [
        { label: 'Best 15', avg: exportData.b15Avg, records: exportData.b15Records },
        { label: 'Best 35', avg: exportData.b35Avg, records: exportData.b35Records },
      ]
    } else if (exportMode === 'overflow') {
      if (USE_MOCK && !exportData.fitting) {
        toastError('message.export_image_failed')
        return
      }
      const { b15Records: b15, b35Records: b35 } = exportData.fitting
        ? selectBest50(exportData.fittingRecords, 5)
        : selectBest50((await getRecords(exportData.username, 'b50', 50, 1, 'rating', 'desc', undefined, 5)).data.records, 5)
      const best15 = b15.slice(0, 15)
      const best35 = b35.slice(0, 35)
      sections = [
        { label: 'Best 15', avg: avgRating(best15, exportData.fitting ? 15 : best15.length), records: b15, cutoff: 15 },
        { label: 'Best 35', avg: avgRating(best35, exportData.fitting ? 35 : best35.length), records: b35, cutoff: 35 },
      ]
    } else {
      if (USE_MOCK && !exportData.fitting) {
        toastError('message.export_image_failed')
        return
      }
      const records = exportData.fitting
        ? exportData.fittingRecords.slice(0, 50)
        : (await getRecords(exportData.username, 'best', 50, 1, 'rating', 'desc')).data.records
      sections = [
        { label: 'Best 50', avg: avgRating(records), records: records.slice(0, 50) },
      ]
      rating = avgRating(records.slice(0, 50))
      title = exportData.fitting
        ? t('term.fitting_b50_global_image_title')
        : 'Paradigm: Reboot Global Best Records'
    }

    const blob = await renderB50Image({
      sections,
      username: exportData.username,
      nickname: exportData.nickname,
      rating,
      title,
      fitting: exportData.fitting,
      footerNote: exportData.fitting && sections.some((section) => section.records.some((record) => record.chart.fitting_level == null))
        ? t('message.fitting_b50_fallback_image')
        : undefined,
    })
    saveAs(blob, `${exportData.fitting ? 'fitting_b50' : 'b50'}_${exportMode}_${Date.now()}.jpg`)
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
.fitting-description {
  color: var(--text-secondary);
  font-size: var(--text-sm);
  line-height: 1.6;
}
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
