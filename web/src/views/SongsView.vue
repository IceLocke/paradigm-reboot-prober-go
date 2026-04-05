<template>
  <div class="page-container">
    <div class="page-header">
      <h2>{{ t('term.charts') }}</h2>
      <div class="page-actions">
        <div class="search-box">
          <svg class="search-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
          <input
            v-model="search"
            class="search-input"
            :placeholder="t('message.search_placeholder')"
          />
        </div>
        <n-popover trigger="click" placement="bottom-end" :style="{ maxWidth: '500px' }">
          <template #trigger>
            <button class="icon-btn" :title="t('term.upload_list')">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="8" cy="21" r="1"/><circle cx="19" cy="21" r="1"/><path d="M2.05 2.05h2l2.66 12.42a2 2 0 0 0 2 1.58h9.78a2 2 0 0 0 1.95-1.57l1.65-7.43H5.12"/></svg>
              <span v-if="appStore.uploadList.length > 0" class="badge">{{ appStore.uploadList.length }}</span>
            </button>
          </template>
          <UploadCartPanel />
        </n-popover>
        <button class="icon-btn" :title="t('common.refresh')" @click="loadCharts">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/></svg>
        </button>
      </div>
    </div>

    <!-- Filters -->
    <div class="filters-row">
      <BaseTabs v-model="diffFilter" :tabs="diffTabs" />
      <BaseTabs v-model="versionFilter" :tabs="versionTabs" />
    </div>

    <!-- Table -->
    <div class="table-wrapper">
      <n-data-table
        remote
        :columns="columns"
        :data="paginatedData"
        :bordered="false"
        :scroll-x="650"
        size="small"
        striped
        :row-key="(row: ChartInfo) => row.id"
        @update:sorter="handleSorterUpdate"
      />
    </div>

    <!-- Pagination -->
    <div class="pagination-row">
      <n-pagination
        v-model:page="pageIndex"
        :page-size="pageSize"
        :item-count="filteredData.length"
        :page-slot="5"
      />
    </div>

    <!-- Modals -->
    <SongDetailModal v-model:show="showSongDetail" :song="selectedSong" />
    <QuickUploadModal
      v-model:show="showQuickUpload"
      :title="uploadTarget.title"
      :difficulty="uploadTarget.difficulty"
      :level="uploadTarget.level"
      :chart-id="uploadTarget.chartId"
      @success="onUploadSuccess"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { NDataTable, NPagination, NPopover, useMessage } from 'naive-ui'
import type { DataTableColumns, DataTableSortState } from 'naive-ui'

import { useAppStore } from '@/stores/app'
import { getAllCharts, getSingleSongInfo } from '@/api/song'
import { USE_MOCK, getMockCharts } from '@/api/mock'
import type { ChartInfo, Song, Difficulty } from '@/api/types'
import BaseTabs from '@/components/ui/BaseTabs.vue'
import DifficultyBadge from '@/components/business/DifficultyBadge.vue'
import SongDetailModal from '@/components/business/SongDetailModal.vue'
import QuickUploadModal from '@/components/business/QuickUploadModal.vue'
import UploadCartPanel from '@/components/business/UploadCartPanel.vue'

const { t } = useI18n()
const message = useMessage()
const appStore = useAppStore()

const search = ref('')
const diffFilter = ref('all')
const versionFilter = ref('all')
const pageIndex = ref(1)
const pageSize = 20

const sortState = ref<DataTableSortState | null>(null)

const showSongDetail = ref(false)
const selectedSong = ref<Song | null>(null)
const showQuickUpload = ref(false)
const uploadTarget = ref({ title: '', difficulty: 'detected' as Difficulty, level: 0, chartId: 0 })

const diffTabs = [
  { key: 'all', label: t('common.all') },
  { key: 'detected', label: 'DET' },
  { key: 'invaded', label: 'IVD' },
  { key: 'massive', label: 'MSV' },
  { key: 'reboot', label: 'RBT' },
]

const versionTabs = [
  { key: 'all', label: t('common.all') },
  { key: 'new', label: t('term.current') },
  { key: 'old', label: t('term.past') },
]

const compareVersions = (a: string, b: string): number => {
  const aParts = a.split('.');
  const bParts = b.split('.');
  const len = Math.max(aParts.length, bParts.length);

  for (let i = 0; i < len; i++) {
    const diff = (Number(aParts[i]) || 0) - (Number(bParts[i]) || 0);
    if (diff) return diff;
  }
  return 0;
};

const filteredData = computed(() => {
  let data = Array.from(appStore.charts ?? [])

  if (search.value) {
    const q = search.value.toLowerCase()
    data = data.filter((c) => c.title.toLowerCase().includes(q) || c.artist.toLowerCase().includes(q))
  }

  if (diffFilter.value !== 'all') {
    data = data.filter((c) => c.difficulty === diffFilter.value)
  }

  if (versionFilter.value === 'old') data = data.filter((c) => !c.b15)
  else if (versionFilter.value === 'new') data = data.filter((c) => c.b15)

  if (sortState.value && sortState.value.order) {
    const { columnKey, order } = sortState.value

    data.sort((a, b) => {
      let result = 0

      switch (columnKey) {
        case 'title':
          result = a.title.localeCompare(b.title)
          break
        case 'version':
          result = compareVersions(a.version, b.version)
          break
        case 'level':
          result = a.level - b.level
          break
        case 'fitting_level':
          result = (a.fitting_level ?? 0) - (b.fitting_level ?? 0)
          break
      }

      return order === 'ascend' ? result : -result
    })
  }

  return data
})

watch(
  () => filteredData.value.length,
  (length: number) => {
    const maxPage = Math.max(1, Math.ceil(length / pageSize))
    pageIndex.value = Math.min(pageIndex.value, maxPage)
  },
)

const paginatedData = computed(() => {
  const start = (pageIndex.value - 1) * pageSize
  return filteredData.value.slice(start, start + pageSize)
})

const handleSorterUpdate = (sorter: DataTableSortState | DataTableSortState[] | null) => {
  if (Array.isArray(sorter)) {
    sortState.value = sorter[0] ?? null
  } else {
    sortState.value = sorter
  }
}

const onClickTitle = async (songId: number) => {
  showSongDetail.value = true
  selectedSong.value = null
  try {
    if (USE_MOCK) {
      const chart = appStore.charts?.find((c) => c.song_id === songId)
      selectedSong.value = {
        id: songId,
        title: chart?.title ?? '',
        artist: chart?.artist ?? '',
        bpm: chart?.bpm ?? '',
        cover: chart?.cover ?? '',
        illustrator: chart?.illustrator ?? '',
        version: chart?.version ?? '',
        album: chart?.album ?? '',
        genre: chart?.genre ?? '',
        length: chart?.length ?? '',
        b15: chart?.b15 ?? false,
        wiki_id: chart?.wiki_id ?? '',
        charts: (appStore.charts ?? [])
          .filter((c) => c.song_id === songId)
          .map((c) => ({
            id: c.id,
            song_id: c.song_id,
            difficulty: c.difficulty,
            level: c.level,
            fitting_level: c.fitting_level,
            level_design: c.level_design,
            notes: c.notes,
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

const onQuickUpload = (chart: ChartInfo) => {
  uploadTarget.value = {
    title: chart.title,
    difficulty: chart.difficulty,
    level: chart.level,
    chartId: chart.id,
  }
  showQuickUpload.value = true
}

const onAddToCart = (chart: ChartInfo) => {
  const exists = appStore.uploadList.some((item) => item.chart_id === chart.id)
  if (exists) {
    message.error(t('message.add_to_upload_list_failed'))
    return
  }
  appStore.uploadList.push({
    title: chart.title,
    difficulty: chart.difficulty,
    level: chart.level,
    chart_id: chart.id,
  })
  message.success(t('message.add_to_upload_list_success'))
}

const onUploadSuccess = () => {
  // Refresh could happen here
}

const columns = computed<DataTableColumns<ChartInfo>>(() => [
  {
    title: t('term.title'),
    key: 'title',
    minWidth: 150,
    ellipsis: {
      tooltip: {
        zIndex: 1,
      },
    },
    sorter: true,
    render(row) {
      return h('a', {
        class: 'link-text',
        onClick: () => onClickTitle(row.song_id),
      }, row.title)
    },
  },
  {
    title: t('term.version'),
    key: 'version',
    width: 90,
    sorter: true,
  },
  {
    title: t('term.season'),
    key: 'b15',
    width: 90,
    render(row) {
      return h('span', {
        class: row.b15 ? 'version-badge version-badge--new' : 'version-badge version-badge--old',
      }, row.b15 ? t('term.current') : t('term.past'))
    },
  },
  {
    title: t('term.difficulty'),
    key: 'level',
    width: 110,
    sorter: true,
    render(row) {
      return h(DifficultyBadge, { difficulty: row.difficulty, level: row.level.toFixed(1), short: true })
    },
  },
  {
    title: t('term.fitting_level'),
    key: 'fitting_level',
    width: 90,
    sorter: true,
    render(row) {
      return h('span', { class: 'mono' }, row.fitting_level != null ? row.fitting_level.toFixed(1) : '-')
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

const loadCharts = async () => {
  if (USE_MOCK) {
    appStore.charts = getMockCharts()
    return
  }
  try {
    const res = await getAllCharts()
    appStore.charts = res.data
    message.success(t('message.get_charts_success'))
  } catch (err: unknown) {
    const e = err as { response?: { data?: { error?: string } } }
    message.error(t('message.get_charts_failed') + (e.response?.data?.error ? ': ' + e.response.data.error : ''))
  }
}

onMounted(() => {
  if (!appStore.charts) loadCharts()
})
</script>

<style scoped>
.filters-row {
  display: flex;
  gap: 0 var(--space-4);
  flex-wrap: wrap;
}
.search-box {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 0 var(--space-3);
  min-height: 40px;
  flex: 1;
  max-width: 300px;
}
.search-icon { color: var(--text-muted); flex-shrink: 0; }
.search-input {
  border: none;
  background: none;
  color: var(--text-primary);
  font-size: 16px;
  outline: none;
  width: 100%;
  font-family: inherit;
}
.search-input::placeholder { color: var(--text-muted); }

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
  position: relative;
}
@media (hover: hover) {
  .icon-btn:hover { background: rgba(255,255,255,0.06); color: var(--text-primary); }
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

:deep(.link-text) {
  color: var(--accent);
  cursor: pointer;
  text-decoration: none;
  font-size: var(--text-sm);
}
:deep(.link-text:hover) { text-decoration: underline; }
:deep(.mono) { font-family: var(--font-mono); font-size: var(--text-sm); }

:deep(.version-badge) {
  display: inline-flex;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
}
:deep(.version-badge--new) { background: var(--accent-muted); color: var(--accent); }
:deep(.version-badge--old) { background: rgba(161,161,170,0.1); color: var(--text-secondary); }

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
</style>
