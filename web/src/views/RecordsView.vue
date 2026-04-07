<template>
  <div class="page-container">
    <div class="page-header">
      <h2>{{ t('term.records') }}</h2>
      <div class="page-actions">
        <IconButton :icon="FileUp" :size="18" :title="t('common.import_csv')" @click="showCsvImport = true"/>
        <IconButton :icon="FileDown" :size="18" :title="t('common.export_csv')" @click="onExportCsv"/>
        <UploadCart />
        <IconButton :icon="RefreshCw" :size="18" :title="t('common.refresh')" @click="refreshRecords"/>
      </div>
    </div>

    <!-- Scope filter -->
    <div class="filters-row">
      <BaseTabs v-model="scope" :tabs="scopeTabs" />
    </div>

    <!-- Table -->
    <div class="table-wrapper">
      <n-data-table
        remote
        :columns="columns"
        :data="records"
        :bordered="false"
        :scroll-x="750"
        size="small"
        striped
        :loading="loading"
        @update:sorter="handleSorterUpdate"
      />
    </div>

    <!-- Pagination -->
    <div class="pagination-row">
      <n-pagination
        v-model:page="pageIndex"
        :page-size="pageSize"
        :item-count="total"
        :page-slot="5"
        @update:page="loadRecords"
      />
    </div>

    <!-- Modals -->
    <SongDetailModal v-model:show="showSongDetail" :song="selectedSong" />
    <CsvImportModal v-model:show="showCsvImport" @success="loadRecords" />
    <QuickUploadModal
      v-model:show="showQuickUpload"
      :title="uploadTarget.title"
      :difficulty="uploadTarget.difficulty"
      :level="uploadTarget.level"
      :chart-id="uploadTarget.chartId"
      @success="loadRecords"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { NDataTable, NPagination, useMessage } from 'naive-ui'
import type { DataTableColumns, DataTableSortState } from 'naive-ui'
import { FileUp, FileDown, RefreshCw, Plus, Upload } from '@lucide/vue';
import dayjs from 'dayjs'

import { useUserStore } from '@/stores/user'
import { useAppStore } from '@/stores/app'
import { getRecords, getAllChartsWithScores } from '@/api/record'
import { getSingleSongInfo } from '@/api/song'
import { USE_MOCK, getMockRecords, getMockAllCharts } from '@/api/mock'
import { exportCsv } from '@/utils/csv'
import { saveAs } from 'file-saver'
import type { PlayRecordInfo, Song, Difficulty } from '@/api/types'
import BaseTabs from '@/components/ui/BaseTabs.vue'
import IconButton from '@/components/ui/IconButton.vue'
import DifficultyBadge from '@/components/business/DifficultyBadge.vue'
import SongDetailModal from '@/components/business/SongDetailModal.vue'
import QuickUploadModal from '@/components/business/QuickUploadModal.vue'
import UploadCart from '@/components/business/UploadCart.vue'
import CsvImportModal from '@/components/business/CsvImportModal.vue'

const { t } = useI18n()
const message = useMessage()
const userStore = useUserStore()
const appStore = useAppStore()

const scope = ref('best')
const pageIndex = ref(1)
const pageSize = 20
const total = ref(0)
const records = ref<PlayRecordInfo[]>([])
const loading = ref(false)

const sortState = ref<DataTableSortState | null>(null)

const showSongDetail = ref(false)
const selectedSong = ref<Song | null>(null)
const showQuickUpload = ref(false)
const showCsvImport = ref(false)
const uploadTarget = ref({ title: '', difficulty: 'detected' as Difficulty, level: 0, chartId: 0 })

const scopeTabs = [
  { key: 'best', label: t('term.best_only') },
  { key: 'all', label: t('term.all_records') },
]

watch(scope, () => {
  pageIndex.value = 1
  loadRecords()
})

const handleSorterUpdate = (
  sorter: DataTableSortState | DataTableSortState[] | null
) => {
  if (Array.isArray(sorter)) {
    sortState.value = sorter[0] ?? null
  } else {
    sortState.value = sorter
  }
  loadRecords()
}

const onClickTitle = async (songId: number) => {
  showSongDetail.value = true
  selectedSong.value = null
  try {
    if (USE_MOCK) {
      const charts = appStore.charts ?? []
      const chart = charts.find((c) => c.song_id === songId)
      selectedSong.value = {
        id: songId, title: chart?.title ?? 'Song', artist: chart?.artist ?? '',
        bpm: chart?.bpm ?? '', cover: chart?.cover ?? '', illustrator: chart?.illustrator ?? '',
        version: chart?.version ?? '', album: chart?.album ?? '', genre: chart?.genre ?? '',
        length: chart?.length ?? '', b15: chart?.b15 ?? false, wiki_id: chart?.wiki_id ?? '',
        charts: charts.filter((c) => c.song_id === songId).map((c) => ({
          id: c.id, song_id: c.song_id, difficulty: c.difficulty,
          level: c.level, fitting_level: c.fitting_level, level_design: c.level_design, notes: c.notes,
        })),
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

const onQuickUpload = (record: PlayRecordInfo) => {
  uploadTarget.value = {
    title: record.chart.title,
    difficulty: record.chart.difficulty,
    level: record.chart.level,
    chartId: record.chart.id,
  }
  showQuickUpload.value = true
}

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

const columns = computed<DataTableColumns<PlayRecordInfo>>(() => [
  {
    title: '#',
    key: 'index',
    width: 50,
    render(_row, index) {
      return h('span', { class: 'mono' }, (pageIndex.value - 1) * pageSize + index + 1)
    },
  },
  {
    title: t('term.title'),
    key: 'title',
    minWidth: 150,
    ellipsis: {
      tooltip: {
        zIndex: 1,
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
    title: t('term.season'),
    key: 'b15',
    width: 90,
    render(row) {
      return h('span', {
        class: row.chart.b15 ? 'version-badge version-badge--new' : 'version-badge version-badge--old',
      }, row.chart.b15 ? t('term.current') : t('term.past'))
    },
  },
  {
    title: t('term.difficulty'),
    key: 'difficulty',
    width: 110,
    render(row) {
      return h(DifficultyBadge, { difficulty: row.chart.difficulty, level: row.chart.level, short: true })
    },
  },
  {
    title: t('term.score'),
    key: 'score',
    width: 100,
    sorter: true,
    render(row) {
      return h('span', { class: 'mono' }, row.score.toLocaleString())
    },
  },
  {
    title: 'Rating',
    key: 'rating',
    width: 80,
    sorter: true,
    render(row) {
      return h('span', { class: 'mono' }, (row.rating / 100).toFixed(2))
    },
  },
  {
    title: t('term.record_time'),
    key: 'record_time',
    width: 130,
    sorter: true,
    render(row) {
      return h('span', { class: 'time-text' }, dayjs(row.record_time).format('YY/MM/DD HH:mm'))
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

const loadRecords = async () => {
  loading.value = true
  try {
    if (USE_MOCK) {
      const mock = getMockRecords(scope.value, pageSize, pageIndex.value)
      records.value = mock.records
      total.value = mock.total
    } else {
      if (!userStore.logged_in) return
      const { columnKey, order } = sortState.value ?? {}
      const hasActiveSort = (order === 'ascend' || order === 'descend') && columnKey != null
      const sortBy = hasActiveSort ? String(columnKey) : 'rating'
      const sortOrder = hasActiveSort ? (order === 'ascend' ? 'asc' : 'desc') : 'desc'
      const res = await getRecords(userStore.username, scope.value, pageSize, pageIndex.value, sortBy, sortOrder)
      records.value = res.data.records
      total.value = res.data.total
    }
  } catch (err: unknown) {
    const e = err as { response?: { data?: { error?: string } } }
    message.error(t('message.get_record_failed') + (e.response?.data?.error ?? ''))
  } finally {
    loading.value = false
  }
}

const refreshRecords = async () => {
  await loadRecords()
  message.success(t('message.refresh_record_success'))
}

const onExportCsv = async () => {
  try {
    let csvData
    if (USE_MOCK) {
      csvData = getMockAllCharts()
    } else {
      const res = await getAllChartsWithScores(userStore.username)
      csvData = res.data
    }
    const csvStr = exportCsv(csvData.charts)
    const blob = new Blob([csvStr], { type: 'text/csv;charset=utf-8' })
    saveAs(blob, `${userStore.username}_records.csv`)
    message.success(t('message.csv_export_success'))
  } catch {
    message.error(t('message.csv_export_failed'))
  }
}

watch(() => userStore.logged_in, (loggedIn) => {
  if (loggedIn) loadRecords()
})

onMounted(loadRecords)
</script>

<style scoped>
.filters-row {
  display: flex;
  gap: 0 var(--space-4);
  flex-wrap: wrap;
}
.table-wrapper {
  -webkit-overflow-scrolling: touch;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 10px;
  overflow: hidden;
  padding: var(--space-3);
}
.pagination-row {
  display: flex;
  justify-content: center;
  margin-top: var(--space-5);
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

:deep(.link-text) { color: var(--accent); cursor: pointer; text-decoration: none; font-size: var(--text-sm); }
:deep(.link-text:hover) { text-decoration: underline; }
:deep(.mono) { font-family: var(--font-mono),monospace; font-size: var(--text-sm); }
:deep(.time-text) { font-size: var(--text-xs); color: var(--text-secondary); }
:deep(.version-badge) { display: inline-flex; padding: 2px 8px; border-radius: 4px; font-size: 12px; font-weight: 500; }
:deep(.version-badge--new) { background: var(--accent-muted); color: var(--accent); }
:deep(.version-badge--old) { background: rgba(161,161,170,0.1); color: var(--text-secondary); }
:deep(.action-btns) { display: flex; gap: 2px; }
:deep(.action-btn) {
  display: flex; align-items: center; justify-content: center;
  width: 32px; height: 32px; background: none; border: none;
  color: var(--text-secondary); cursor: pointer; border-radius: 6px;
  transition: background var(--transition-fast), color var(--transition-fast);
}
@media (hover: hover) {
  :deep(.action-btn:hover) { background: rgba(255,255,255,0.06); color: var(--text-primary); }
}
</style>
