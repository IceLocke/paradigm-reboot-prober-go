<template>
  <div class="page-container">
    <div class="page-header">
      <h2>{{ t('term.charts') }}</h2>
      <div class="page-actions">
        <div class="search-box">
          <Search :size="16" />
          <input
            v-model="search"
            class="search-input"
            :placeholder="t('message.search_placeholder')"
          />
        </div>
        <UploadCart />
        <IconButton :icon="RefreshCw" :size="18" :title="t('common.refresh')" @click="loadCharts" />
        <div class="view-toggle">
          <button
            :class="['view-toggle-btn', { active: appStore.songsViewMode === 'grid' }]"
            :title="t('term.grid_view')"
            @click="appStore.songsViewMode = 'grid'"
          >
            <LayoutGrid :size="16" />
          </button>
          <button
            :class="['view-toggle-btn', { active: appStore.songsViewMode === 'table' }]"
            :title="t('term.table_view')"
            @click="appStore.songsViewMode = 'table'"
          >
            <List :size="16" />
          </button>
        </div>
      </div>
    </div>

    <!-- Search box (mobile: full width row) -->
    <div class="search-row-mobile">
      <div class="search-box search-box--mobile">
        <Search :size="16" />
        <input
          v-model="search"
          class="search-input"
          :placeholder="t('message.search_placeholder')"
        />
      </div>
    </div>

    <!-- Filters -->
    <div class="filters-row">
      <BaseTabs v-model="diffFilter" :tabs="diffTabs" />
      <BaseTabs v-model="versionFilter" :tabs="versionTabs" />
    </div>

    <!-- Advanced Filters -->
    <SongFilterPanel
      v-model:show="showAdvFilters"
      v-model:level-min="levelMin"
      v-model:level-max="levelMax"
      v-model:version-select="versionSelect"
      v-model:album-select="albumSelect"
      v-model:group-by="groupBy"
      :version-options="versionOptions"
      :album-options="albumOptions"
      :group-by-options="groupByOptions"
      :b50-filter="b50Filter"
      :b50-loading="b50Loading"
      @toggle-b50="toggleB50Filter"
    />

    <!-- Divider -->
    <hr class="section-divider" />

    <!-- Grid View -->
    <SongGridView
      v-if="appStore.songsViewMode === 'grid'"
      :groups="groupedData"
      :collapsed-levels="collapsedLevels"
      @toggle-level="toggleLevel"
      @click-chart="onClickTitle"
      @add-to-cart="onAddToCart"
      @quick-upload="onQuickUpload"
    />

    <!-- Table View -->
    <SongTableView
      v-else
      v-model:page-index="pageIndex"
      :data="paginatedData"
      :filtered-count="filteredData.length"
      :page-size="pageSize"
      @click-title="onClickTitle"
      @add-to-cart="onAddToCart"
      @quick-upload="onQuickUpload"
      @update:sorter="handleSorterUpdate"
    />

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
import { ref, defineAsyncComponent, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage } from 'naive-ui'
import { Search, RefreshCw, LayoutGrid, List } from '@lucide/vue'

import { useAppStore } from '@/stores/app'
import { getAllCharts, getSingleSongInfo } from '@/api/song'
import { USE_MOCK, getMockCharts } from '@/api/mock'
import type { ChartInfo, Song, Difficulty } from '@/api/types'

import { useChartFilters } from '@/composables/useChartFilters'
import { useChartGroups } from '@/composables/useChartGroups'

import BaseTabs from '@/components/ui/BaseTabs.vue'
import IconButton from '@/components/ui/IconButton.vue'
import SongDetailModal from '@/components/business/SongDetailModal.vue'
import QuickUploadModal from '@/components/business/QuickUploadModal.vue'
import UploadCart from '@/components/business/UploadCart.vue'
import SongFilterPanel from '@/components/business/SongFilterPanel.vue'

const SongGridView = defineAsyncComponent(() => import('@/components/business/SongGridView.vue'))
const SongTableView = defineAsyncComponent(() => import('@/components/business/SongTableView.vue'))

const { t } = useI18n()
const message = useMessage()
const appStore = useAppStore()

// --- Composables ---
const {
  search,
  diffFilter,
  versionFilter,
  pageIndex,
  pageSize,
  showAdvFilters,
  levelMin,
  levelMax,
  versionSelect,
  albumSelect,
  b50Filter,
  b50Loading,
  groupBy,
  diffTabs,
  versionTabs,
  versionOptions,
  albumOptions,
  groupByOptions,
  filteredData,
  paginatedData,
  handleSorterUpdate,
  toggleB50Filter,
} = useChartFilters()

const { groupedData, collapsedLevels, toggleLevel } = useChartGroups(filteredData, groupBy)

// --- Modals ---
const showSongDetail = ref(false)
const selectedSong = ref<Song | null>(null)
const showQuickUpload = ref(false)
const uploadTarget = ref({ title: '', difficulty: 'detected' as Difficulty, level: 0, chartId: 0 })

// --- Actions ---
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

// --- Load data ---
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
/* Search box in header (desktop) */
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
  max-width: 40vw;
}
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

/* Mobile search row (hidden on desktop) */
.search-row-mobile {
  display: none;
  margin-bottom: var(--space-3);
}
.search-box--mobile {
  max-width: none;
}

@media (max-width: 639px) {
  .page-actions .search-box {
    display: none;
  }
  .search-row-mobile {
    display: block;
  }
}

/* Filters row */
.filters-row {
  display: flex;
  gap: 0 var(--space-4);
  flex-wrap: wrap;
  margin-bottom: var(--space-2);
}
.filters-row :deep(.tabs__content) {
  display: none;
}
@media (max-width: 639px) {
  .filters-row {
    gap: var(--space-2) var(--space-3);
  }
}

/* View toggle */
.view-toggle {
  display: flex;
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 8px;
  overflow: hidden;
}
.view-toggle-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  background: none;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  transition: background var(--transition-fast), color var(--transition-fast);
}
.view-toggle-btn.active {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}
@media (hover: hover) {
  .view-toggle-btn:not(.active):hover {
    color: var(--text-secondary);
  }
}

/* Divider between filters and content */
.section-divider {
  border: none;
  border-top: 1px solid var(--border);
  margin: var(--space-4) 0;
}
</style>
