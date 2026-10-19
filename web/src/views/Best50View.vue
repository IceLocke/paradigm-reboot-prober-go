<template>
  <div class="page-container">
    <div class="page-header">
      <h2>{{ t('term.b50') }}</h2>
      <div class="page-actions">
        <IconButton
          :title="t('common.export_image')"
          :disabled="allRecords.length === 0"
          @click="showExportModal = true"
        >
          <ImageDown :size="18" />
        </IconButton>
        <IconButton
          :icon="RefreshCw"
          :size="18"
          :title="t('common.refresh')"
          @click="refreshData"
        />
      </div>
    </div>

    <!-- Version announcement -->
    <VersionAnnounceBanner />

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
  </div>

  <SongDetailModal v-model:show="showSongDetail" :song="selectedSong" :username="userStore.username" />
  <QuickUploadModal
    v-model:show="showQuickUpload"
    :title="uploadTarget.title"
    :difficulty="uploadTarget.difficulty"
    :level="uploadTarget.level"
    :chart-id="uploadTarget.chartId"
    :cover="uploadTarget.cover"
    @success="loadData"
  />
  <B50ExportModal
    v-model:show="showExportModal"
    :username="USE_MOCK ? 'demo_user' : userStore.username"
    :nickname="userStore.profile?.nickname || nickname"
    :rating="b50Rating"
    :b15-records="b15Records"
    :b35-records="b35Records"
    :b15-avg="b15Rating"
    :b35-avg="b35Rating"
  />
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { NDataTable } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { ImageDown, RefreshCw, Plus, Upload } from '@lucide/vue';
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { ScatterChart } from 'echarts/charts'
import { GridComponent, TooltipComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'

import { toastSuccess, toastError } from '@/utils/toast'
import { formatAvgRating, formatRating } from '@/utils/rating'

import { useUserStore } from '@/stores/user'
import { useAppStore } from '@/stores/app'
import { getRecords } from '@/api/record'
import { getSingleSongInfo } from '@/api/song'
import { USE_MOCK, getMockB50 } from '@/api/mock'
import type { PlayRecordInfo, Song, Difficulty } from '@/api/types'
import BaseCard from '@/components/ui/BaseCard.vue'
import IconButton from '@/components/ui/IconButton.vue'
import StatCard from '@/components/business/StatCard.vue'
import DifficultyBadge from '@/components/business/DifficultyBadge.vue'
import SongDetailModal from '@/components/business/SongDetailModal.vue'
import QuickUploadModal from '@/components/business/QuickUploadModal.vue'
import B50ExportModal from '@/components/business/B50ExportModal.vue'
import VersionAnnounceBanner from '@/components/business/VersionAnnounceBanner.vue'

use([ScatterChart, GridComponent, TooltipComponent, CanvasRenderer])

const { t } = useI18n()
const userStore = useUserStore()
const appStore = useAppStore()

const allRecords = ref<PlayRecordInfo[]>([])
const nickname = ref('')
const showSongDetail = ref(false)
const selectedSong = ref<Song | null>(null)

const showQuickUpload = ref(false)
const showExportModal = ref(false)
const uploadTarget = ref({ title: '', difficulty: 'detected' as Difficulty, level: 0, chartId: 0, cover: '' })

const b35Records = computed(() =>
  allRecords.value.filter((r) => !r.chart.b15).map((r, i) => ({ ...r, _index: i + 1 }))
)

const b15Records = computed(() =>
  allRecords.value.filter((r) => r.chart.b15).map((r, i) => ({ ...r, _index: i + 1 }))
)

const b50Rating = computed(() => {
  const sum = allRecords.value.reduce((s, r) => s + r.rating, 0)
  return formatAvgRating(sum, 50)
})

const b35Rating = computed(() => {
  const sum = b35Records.value.reduce((s, r) => s + r.rating, 0)
  return formatAvgRating(sum, 35)
})

const b15Rating = computed(() => {
  const sum = b15Records.value.reduce((s, r) => s + r.rating, 0)
  return formatAvgRating(sum, 15)
})

const onAddToCart = (record: PlayRecordInfo) => {
  const exists = appStore.uploadList.some((item) => item.chart_id === record.chart.id)
  if (exists) {
    toastError('message.add_to_upload_list_failed')
    return
  }
  appStore.uploadList.push({
    title: record.chart.title,
    difficulty: record.chart.difficulty,
    level: record.chart.level,
    chart_id: record.chart.id,
    score: record.score,
  })
  toastSuccess('message.add_to_upload_list_success')
}

const onQuickUpload = (record: PlayRecordInfo) => {
  uploadTarget.value = {
    title: record.chart.title,
    difficulty: record.chart.difficulty,
    level: record.chart.level,
    chartId: record.chart.id,
    cover: record.chart.cover,
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
        created_at: '', updated_at: '',
      }
    } else {
      const res = await getSingleSongInfo(songId)
      selectedSong.value = res.data
    }
  } catch (err: unknown) {
    toastError('message.get_song_failed', err)
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
        zIndex: 99,
        'width': 'trigger',
      },
    },
    fixed: "left",
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
    title: 'Rating',
    key: 'rating',
    width: 80,
    sorter: (a, b) => a.rating - b.rating,
    render(row) {
      return h('span', { class: 'mono' }, formatRating(row.rating))
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
        }, [
          h(Plus, { size: 14 })
        ]),
        h('button', {
          class: 'action-btn',
          title: t('message.quick_upload'),
          onClick: () => onQuickUpload(row),
        }, [
          h(Upload, { size: 14 })
        ]),
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
    return true
  }

  if (!userStore.logged_in) return false
  try {
    const res = await getRecords(userStore.username, 'b50')
    allRecords.value = res.data.records
    nickname.value = res.data.nickname
    return true
  } catch (err: unknown) {
    toastError('message.get_record_failed', err)
    return false
  }
}

const refreshData = async () => {
  const ok = await loadData()
  if (ok) toastSuccess('message.refresh_record_success')
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
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: 6px;
  transition: background var(--transition-fast), color var(--transition-fast);
}
@media (hover: hover) {
  :deep(.action-btn:hover) { background: rgba(255,255,255,0.06); color: var(--text-primary); }
}

</style>
