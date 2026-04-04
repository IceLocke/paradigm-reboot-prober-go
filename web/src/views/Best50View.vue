<template>
  <div class="page-container">
    <div class="page-header">
      <h2>{{ t('term.b50') }}</h2>
      <div class="page-actions">
        <button class="icon-btn" :title="t('common.refresh')" @click="loadData">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/></svg>
        </button>
      </div>
    </div>

    <!-- Stats overview -->
    <div class="stats-row">
      <StatCard label="Best 50 Avg" :value="b50Rating" :precision="4" />
      <StatCard label="B35 Avg" :value="b35Rating" :precision="4" />
      <StatCard label="B15 Avg" :value="b15Rating" :precision="4" />
    </div>

    <!-- Charts section -->
    <div class="b50-grid">
      <!-- B35 -->
      <BaseCard>
        <template #header>
          <div class="section-header">
            <h3>{{ t('term.b35') }} <span class="record-count">({{ b35Records.length }})</span></h3>
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

      <!-- B15 -->
      <BaseCard>
        <template #header>
          <div class="section-header">
            <h3>{{ t('term.b15') }} <span class="record-count">({{ b15Records.length }})</span></h3>
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
    </div>

    <!-- Song detail modal -->
    <SongDetailModal v-model:show="showSongDetail" :song="selectedSong" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { NDataTable } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { ScatterChart } from 'echarts/charts'
import { GridComponent, TooltipComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'

import { useUserStore } from '@/stores/user'
import { getRecords } from '@/api/record'
import { getSingleSongInfo } from '@/api/song'
import { USE_MOCK, getMockB50 } from '@/api/mock'
import type { PlayRecordInfo, Song } from '@/api/types'
import BaseCard from '@/components/ui/BaseCard.vue'
import StatCard from '@/components/business/StatCard.vue'
import DifficultyBadge from '@/components/business/DifficultyBadge.vue'
import SongDetailModal from '@/components/business/SongDetailModal.vue'

use([ScatterChart, GridComponent, TooltipComponent, CanvasRenderer])

const { t } = useI18n()
const userStore = useUserStore()

const allRecords = ref<PlayRecordInfo[]>([])
const showSongDetail = ref(false)
const selectedSong = ref<Song | null>(null)

const b35Records = computed(() =>
  allRecords.value.filter((r) => !r.chart.b15).map((r, i) => ({ ...r, _index: i + 1 }))
)

const b15Records = computed(() =>
  allRecords.value.filter((r) => r.chart.b15).map((r, i) => ({ ...r, _index: i + 1 }))
)

const b50Rating = computed(() => {
  if (allRecords.value.length === 0) return 0
  const sum = allRecords.value.reduce((s, r) => s + r.rating, 0)
  return sum / 100 / allRecords.value.length
})

const b35Rating = computed(() => {
  if (b35Records.value.length === 0) return 0
  const sum = b35Records.value.reduce((s, r) => s + r.rating, 0)
  return sum / 100 / b35Records.value.length
})

const b15Rating = computed(() => {
  if (b15Records.value.length === 0) return 0
  const sum = b15Records.value.reduce((s, r) => s + r.rating, 0)
  return sum / 100 / b15Records.value.length
})

const onClickSongTitle = async (songId: number) => {
  showSongDetail.value = true
  selectedSong.value = null
  try {
    if (USE_MOCK) {
      selectedSong.value = {
        song_id: songId, title: 'Mock Song', artist: 'Mock Artist',
        bpm: '180', cover: '', illustrator: 'Artist', version: '1.0',
        album: 'Album', genre: 'Genre', length: '2:30', b15: false,
        wiki_id: '', charts: [],
      }
    } else {
      const res = await getSingleSongInfo(songId)
      selectedSong.value = res.data
    }
  } catch { /* handled */ }
}

const recordColumns = computed<DataTableColumns<PlayRecordInfo & { _index: number }>>(() => [
  { title: '#', key: '_index', width: 45 },
  {
    title: t('term.title'),
    key: 'title',
    minWidth: 140,
    ellipsis: { tooltip: true },
    render(row) {
      return h('a', {
        class: 'link-text',
        onClick: () => onClickSongTitle(row.chart.song_id),
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
])

const scatterOption = (records: PlayRecordInfo[]) => {
  const data = records.map((r) => [r.chart.level, r.rating / 100])
  return {
    backgroundColor: 'transparent',
    grid: { left: 50, right: 20, top: 20, bottom: 40 },
    xAxis: {
      type: 'value',
      name: 'Level',
      nameTextStyle: { color: '#a1a1aa' },
      axisLabel: { color: '#a1a1aa' },
      splitLine: { lineStyle: { color: '#27272a' } },
    },
    yAxis: {
      type: 'value',
      name: 'Rating',
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
    return
  }

  if (!userStore.logged_in) return
  try {
    const res = await getRecords(userStore.username, 'b50')
    allRecords.value = res.data.records
  } catch { /* handled */ }
}

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
  height: 250px;
  width: 100%;
}

.table-wrapper {
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
}

.icon-btn {
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
</style>
