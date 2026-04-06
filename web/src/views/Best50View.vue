<template>
  <div class="page-container">
    <div class="page-header">
      <h2>{{ t('term.b50') }}</h2>
      <div class="page-actions">
        <n-popover trigger="click" placement="bottom-end" :style="{ maxWidth: '500px' }">
          <template #trigger>
            <button class="icon-btn" :title="t('term.upload_list')">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="8" cy="21" r="1"/><circle cx="19" cy="21" r="1"/><path d="M2.05 2.05h2l2.66 12.42a2 2 0 0 0 2 1.58h9.78a2 2 0 0 0 1.95-1.57l1.65-7.43H5.12"/></svg>
              <span v-if="appStore.uploadList.length > 0" class="badge">{{ appStore.uploadList.length }}</span>
            </button>
          </template>
          <UploadCartPanel />
        </n-popover>
        <button
          class="icon-btn"
          :title="t('common.export_image')"
          :disabled="exporting || allRecords.length === 0"
          @click="exportImage"
        >
          <svg v-if="!exporting" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
          <svg v-else width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="spin"><path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83"/></svg>
        </button>
        <button class="icon-btn" :title="t('common.refresh')" @click="refreshData">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/></svg>
        </button>
      </div>
    </div>

    <!-- Stats overview -->
    <div class="stats-row">
      <StatCard label="Rating" :value="b50Rating" :precision="4" />
      <StatCard label="B35 Avg" :value="b35Rating" :precision="4" />
      <StatCard label="B15 Avg" :value="b15Rating" :precision="4" />
    </div>

    <!-- Charts section -->
    <div class="b50-grid">
      <!-- B15 -->
      <BaseCard>
        <template #header>
          <div class="section-header">
            <h3>{{ t('term.current') }} <span class="record-count">({{ b15Records.length }})</span></h3>
          </div>
        </template>
        <div class="chart-wrapper" v-if="b15Records.length > 0">
          <VChart :option="scatterOption(b15Records)" :autoresize="true" class="scatter-chart" />
        </div>
        <div class="table-wrapper">
          <n-data-table
            :columns="recordColumns"
            :data="b15Records"
            :pagination="false"
            :bordered="false"
            :scroll-x="500"
            size="small"
            striped
          />
        </div>
      </BaseCard>

      <!-- B35 -->
      <BaseCard>
        <template #header>
          <div class="section-header">
            <h3>{{ t('term.past') }} <span class="record-count">({{ b35Records.length }})</span></h3>
          </div>
        </template>
        <div class="chart-wrapper" v-if="b35Records.length > 0">
          <VChart :option="scatterOption(b35Records)" :autoresize="true" class="scatter-chart" />
        </div>
        <div class="table-wrapper">
          <n-data-table
            :columns="recordColumns"
            :data="b35Records"
            :pagination="false"
            :bordered="false"
            :scroll-x="500"
            size="small"
            striped
          />
        </div>
      </BaseCard>
    </div>

    <!-- Song detail modal -->
    <SongDetailModal v-model:show="showSongDetail" :song="selectedSong" />
    <QuickUploadModal
      v-model:show="showQuickUpload"
      :title="uploadTarget.title"
      :difficulty="uploadTarget.difficulty"
      :level="uploadTarget.level"
      :chart-id="uploadTarget.chartId"
      @success="loadData"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch, h } from 'vue'
import { saveAs } from 'file-saver'
import { useI18n } from 'vue-i18n'
import { NDataTable, NPopover, useMessage } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { ScatterChart } from 'echarts/charts'
import { GridComponent, TooltipComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'

import { useUserStore } from '@/stores/user'
import { useAppStore } from '@/stores/app'
import { getRecords } from '@/api/record'
import { getSingleSongInfo } from '@/api/song'
import { USE_MOCK, getMockB50 } from '@/api/mock'
import type { PlayRecordInfo, Song, Difficulty } from '@/api/types'
import { renderB50Image } from '@/utils/b50Canvas'
import BaseCard from '@/components/ui/BaseCard.vue'
import StatCard from '@/components/business/StatCard.vue'
import DifficultyBadge from '@/components/business/DifficultyBadge.vue'
import SongDetailModal from '@/components/business/SongDetailModal.vue'
import QuickUploadModal from '@/components/business/QuickUploadModal.vue'
import UploadCartPanel from '@/components/business/UploadCartPanel.vue'

use([ScatterChart, GridComponent, TooltipComponent, CanvasRenderer])

const { t } = useI18n()
const message = useMessage()
const userStore = useUserStore()
const appStore = useAppStore()

const allRecords = ref<PlayRecordInfo[]>([])
const nickname = ref('')
const showSongDetail = ref(false)
const selectedSong = ref<Song | null>(null)
const exporting = ref(false)
const showQuickUpload = ref(false)
const uploadTarget = ref({ title: '', difficulty: 'detected' as Difficulty, level: 0, chartId: 0 })

const b35Records = computed(() =>
  allRecords.value.filter((r) => !r.chart.b15).map((r, i) => ({ ...r, _index: i + 1 }))
)

const b15Records = computed(() =>
  allRecords.value.filter((r) => r.chart.b15).map((r, i) => ({ ...r, _index: i + 1 }))
)

const b50Rating = computed(() => {
  const sum = allRecords.value.reduce((s, r) => s + r.rating, 0)
  return sum / 5000 // integer sum / (100 * 50), precision = 0.0002
})

const b35Rating = computed(() => {
  const sum = b35Records.value.reduce((s, r) => s + r.rating, 0)
  return sum / 3500
})

const b15Rating = computed(() => {
  const sum = b15Records.value.reduce((s, r) => s + r.rating, 0)
  return sum / 1500
})

const onAddToCart = (record: PlayRecordInfo) => {
  const exists = appStore.uploadList.some((item) => item.chart_id === record.chart.id)
  if (exists) {
    message.error(t('message.add_to_upload_list_failed'))
    return
  }
  appStore.uploadList.push({
    title: record.chart.title,
    difficulty: record.chart.difficulty,
    level: record.chart.level,
    chart_id: record.chart.id,
    score: record.score,
  })
  message.success(t('message.add_to_upload_list_success'))
}

const onQuickUpload = (record: PlayRecordInfo) => {
  uploadTarget.value = {
    title: record.chart.title,
    difficulty: record.chart.difficulty,
    level: record.chart.level,
    chartId: record.chart.id,
  }
  showQuickUpload.value = true
}

const onClickTitle = async (songId: number) => {
  showSongDetail.value = true
  selectedSong.value = null
  try {
    if (USE_MOCK) {
      selectedSong.value = {
        id: songId, title: 'Mock Song', artist: 'Mock Artist',
        bpm: '180', cover: '', illustrator: 'Artist', version: '1.0',
        album: 'Album', genre: 'Genre', length: '2:30', b15: false,
        wiki_id: '', charts: [],
      }
    } else {
      const res = await getSingleSongInfo(songId)
      selectedSong.value = res.data
    }
  } catch (err: unknown) {
    const e = err as { response?: { data?: { error?: string } } }
    message.error(t('message.get_song_failed') + (e.response?.data?.error ? ': ' + e.response.data.error : ''))
  }
}

const recordColumns = computed<DataTableColumns<PlayRecordInfo & { _index: number }>>(() => [
  { title: '#', key: '_index', width: 45 },
  {
    title: t('term.title'),
    key: 'title',
    minWidth: 140,
    ellipsis: {
      tooltip: {
        zIndex: 1,
      },
    },
    render(row) {
      return h('a', {
        class: 'link-text',
        onClick: () => onClickTitle(row.chart.song_id),
      }, row.chart.title)
    },
  },
  {
    title: t('term.difficulty'),
    key: 'difficulty',
    width: 110,
    render(row) {
      return h(DifficultyBadge, {
        difficulty: row.chart.difficulty,
        level: row.chart.level,
        short: true,
      })
    },
  },
  {
    title: t('term.score'),
    key: 'score',
    width: 100,
    sorter: (a, b) => a.score - b.score,
    render(row) {
      return h('span', { class: 'mono' }, row.score.toLocaleString())
    },
  },
  {
    title: 'Rt.',
    key: 'rating',
    width: 80,
    sorter: (a, b) => a.rating - b.rating,
    render(row) {
      return h('span', { class: 'mono' }, (row.rating / 100).toFixed(2))
    },
  },
  {
    title: '',
    key: 'actions',
    width: 80,
    render(row) {
      return h('div', { class: 'action-btns' }, [
        h('button', {
          class: 'action-btn',
          title: t('message.add_to_upload_list'),
          onClick: () => onAddToCart(row),
          innerHTML: '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 5v14M5 12h14"/></svg>',
        }),
        h('button', {
          class: 'action-btn',
          title: t('message.quick_upload'),
          onClick: () => onQuickUpload(row),
          innerHTML: '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>',
        }),
      ])
    },
  },
])

const scatterOption = (records: PlayRecordInfo[]) => {
  const data = records.map((r) => [r.chart.level, r.rating / 100])
  const levels = data.map((d) => d[0])
  const ratings = data.map((d) => d[1])
  const lvMin = Math.floor(Math.min(...levels) - 0.5)
  const lvMax = Math.ceil(Math.max(...levels) + 0.5)
  const rtMin = Math.floor(Math.min(...ratings) - 1)
  const rtMax = Math.ceil(Math.max(...ratings) + 1)
  return {
    backgroundColor: 'transparent',
    grid: { left: 50, right: 30, top: 36, bottom: 45 },
    xAxis: {
      type: 'value',
      name: 'Level',
      min: lvMin,
      max: lvMax,
      nameLocation: 'center',
      nameGap: 28,
      nameTextStyle: { color: '#a1a1aa' },
      axisLabel: { color: '#a1a1aa' },
      splitLine: { lineStyle: { color: '#27272a' } },
    },
    yAxis: {
      type: 'value',
      name: 'Rating',
      min: rtMin,
      max: rtMax,
      nameLocation: 'center',
      nameGap: 38,
      nameTextStyle: { color: '#a1a1aa' },
      axisLabel: { color: '#a1a1aa' },
      splitLine: { lineStyle: { color: '#27272a' } },
    },
    tooltip: {
      trigger: 'item',
      backgroundColor: '#1a1a22',
      borderColor: '#27272a',
      textStyle: { color: '#e4e4e7' },
      formatter: (params: { value: number[] }) =>
        `Level: ${params.value[0]}<br>Rating: ${params.value[1].toFixed(2)}`,
    },
    series: [{
      type: 'scatter',
      symbolSize: 8,
      data,
      itemStyle: { color: '#3b82f6' },
    }],
  }
}

const loadData = async () => {
  if (USE_MOCK) {
    const mock = getMockB50()
    allRecords.value = mock.records
    nickname.value = mock.nickname
    return
  }

  if (!userStore.logged_in) return
  try {
    const res = await getRecords(userStore.username, 'b50')
    allRecords.value = res.data.records
    nickname.value = res.data.nickname
  } catch (err: unknown) {
    const e = err as { response?: { data?: { error?: string } } }
    message.error(t('message.get_record_failed') + (e.response?.data?.error ?? ''))
  }
}

const refreshData = async () => {
  await loadData()
  message.success(t('message.refresh_record_success'))
}

const exportImage = async () => {
  if (exporting.value || allRecords.value.length === 0) return
  exporting.value = true
  try {
    const blob = await renderB50Image({
      b15Records: b15Records.value,
      b35Records: b35Records.value,
      username: USE_MOCK ? 'demo_user' : userStore.username,
      nickname: nickname.value,
      rating: b50Rating.value,
      b15Avg: b15Rating.value,
      b35Avg: b35Rating.value,
    })
    saveAs(blob, `b50_${Date.now()}.jpg`)
    message.success(t('message.export_image_success'))
  } catch {
    message.error(t('message.export_image_failed'))
  } finally {
    exporting.value = false
  }
}

watch(() => userStore.logged_in, (loggedIn) => {
  if (loggedIn) loadData()
})

onMounted(loadData)
</script>

<style scoped>
.stats-row {
  display: flex;
  gap: var(--space-4);
  margin-bottom: var(--space-6);
  flex-wrap: wrap;
}
.stats-row > * { flex: 1; min-width: 100px; }

.b50-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-5);
  align-items: start;
}
@media (max-width: 1023px) {
  .b50-grid { grid-template-columns: 1fr; }
}

.section-header {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}
.record-count {
  font-size: var(--text-sm);
  color: var(--text-muted);
  font-weight: 400;
}

.chart-wrapper {
  margin-bottom: var(--space-4);
}
.scatter-chart {
  height: 280px;
  width: 100%;
}

.table-wrapper {
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
}

.icon-btn {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: 8px;
  transition: background var(--transition-fast);
}
.badge {
  position: absolute;
  top: 4px;
  right: 4px;
  background: var(--accent);
  color: #fff;
  font-size: 10px;
  font-weight: 600;
  min-width: 16px;
  height: 16px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 4px;
}
@media (hover: hover) {
  .icon-btn:hover { background: rgba(255,255,255,0.06); color: var(--text-primary); }
}

:deep(.link-text) {
  color: var(--accent);
  cursor: pointer;
  text-decoration: none;
  font-size: var(--text-sm);
}
:deep(.link-text:hover) { text-decoration: underline; }
:deep(.mono) { font-family: var(--font-mono); font-size: var(--text-sm); }
:deep(.action-btns) {
  display: flex;
  gap: 2px;
}
:deep(.action-btn) {
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
  transition: background var(--transition-fast), color var(--transition-fast);
}
@media (hover: hover) {
  :deep(.action-btn:hover) { background: rgba(255,255,255,0.06); color: var(--text-primary); }
}

.icon-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
.spin {
  animation: spin 1s linear infinite;
}
</style>
