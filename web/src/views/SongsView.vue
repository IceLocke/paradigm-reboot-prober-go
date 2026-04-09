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
        <IconButton :icon="RefreshCw" :size="18" :title="t('common.refresh')" @click="loadCharts"/>
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

    <!-- Advanced Filters Toggle -->
    <button class="adv-filter-toggle" @click="showAdvFilters = !showAdvFilters">
      <ChevronRight :size="14" :class="['adv-filter-chevron', { open: showAdvFilters }]" />
      <span>{{ t('term.filters') }}</span>
    </button>

    <!-- Advanced Filters Panel -->
    <transition name="fade">
      <div v-if="showAdvFilters" class="adv-filters">
        <div class="adv-filters-grid">
          <!-- Level range -->
          <div class="filter-group">
            <label class="filter-label">{{ t('term.level_range') }}</label>
            <div class="level-range-inputs">
              <input
                v-model.number="levelMin"
                type="number"
                step="0.1"
                class="level-input"
                :placeholder="t('term.min_level')"
              />
              <span class="range-sep">~</span>
              <input
                v-model.number="levelMax"
                type="number"
                step="0.1"
                class="level-input"
                :placeholder="t('term.max_level')"
              />
            </div>
          </div>

          <!-- Version select -->
          <div class="filter-group">
            <label class="filter-label">{{ t('term.version') }}</label>
            <n-select
              v-model:value="versionSelect"
              :options="versionOptions"
              :placeholder="t('term.version_select')"
              clearable
              multiple
              size="small"
            />
          </div>

          <!-- Album select -->
          <div class="filter-group">
            <label class="filter-label">{{ t('term.album') }}</label>
            <n-select
              v-model:value="albumSelect"
              :options="albumOptions"
              :placeholder="t('term.album_select')"
              clearable
              multiple
              size="small"
            />
          </div>

          <!-- Group by -->
          <div class="filter-group">
            <label class="filter-label">{{ t('term.group_by') }}</label>
            <n-select
              v-model:value="groupBy"
              :options="groupByOptions"
              size="small"
            />
          </div>
        </div>

        <!-- B50 filter (full-width at bottom) -->
        <div class="adv-filters-bottom">
          <button
            :class="['b50-btn', { active: b50Filter }]"
            @click="toggleB50Filter"
          >
            <Star :size="14" />
            {{ t('term.in_b50') }}
          </button>
        </div>
      </div>
    </transition>

    <!-- Divider -->
    <hr class="section-divider" />

    <!-- Grid View (grouped, collapsible) -->
    <div v-if="appStore.songsViewMode === 'grid'" class="grid-view">
      <div v-if="groupedData.length > 0" class="level-groups">
        <div v-for="group in groupedData" :key="group.key" :class="['level-row', { 'level-row--collapsed': collapsedLevels.has(group.key) }]">
          <button class="level-label" @click="toggleLevel(group.key)">
            <ChevronRight :size="12" :class="['level-chevron', { open: !collapsedLevels.has(group.key) }]" />
            <span class="level-value">{{ group.key }}</span>
            <span class="level-count">({{ t('term.chart_count', { count: group.charts.length }) }})</span>
          </button>
          <div v-if="!collapsedLevels.has(group.key)" class="level-cards">
            <SongGridCard
              v-for="chart in group.charts"
              :key="chart.id"
              :chart="chart"
              @click="onClickTitle(chart.song_id)"
              @add-to-cart="onAddToCart(chart)"
              @quick-upload="onQuickUpload(chart)"
            />
          </div>
        </div>
      </div>
      <div v-else class="empty-state">
        <n-empty :description="t('common.no_data')" />
      </div>
    </div>

    <!-- Table View -->
    <div v-else class="table-wrapper">
      <n-data-table
        :columns="columns"
        :data="paginatedData"
        :bordered="false"
        :scroll-x="650"
        size="small"
        striped
        @update:sorter="handleSorterUpdate"
      />
    </div>

    <!-- Pagination (table view only) -->
    <div v-if="appStore.songsViewMode === 'table'" class="pagination-row">
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
import { NDataTable, NPagination, NSelect, NEmpty, useMessage } from 'naive-ui'
import type { DataTableColumns, DataTableSortState } from 'naive-ui'
import { Search, RefreshCw, Plus, Upload, LayoutGrid, List, ChevronRight, Star } from '@lucide/vue';

import { useAppStore } from '@/stores/app'
import { useUserStore } from '@/stores/user'
import { getAllCharts, getSingleSongInfo } from '@/api/song'
import { getRecords } from '@/api/record'
import { USE_MOCK, getMockCharts, getMockB50 } from '@/api/mock'
import type { ChartInfo, Song, Difficulty } from '@/api/types'
import BaseTabs from '@/components/ui/BaseTabs.vue'
import IconButton from '@/components/ui/IconButton.vue'
import DifficultyBadge from '@/components/business/DifficultyBadge.vue'
import SongDetailModal from '@/components/business/SongDetailModal.vue'
import QuickUploadModal from '@/components/business/QuickUploadModal.vue'
import UploadCart from '@/components/business/UploadCart.vue'
import SongGridCard from '@/components/business/SongGridCard.vue'

const { t } = useI18n()
const message = useMessage()
const appStore = useAppStore()
const userStore = useUserStore()

// --- Basic filters ---
const search = ref('')
const diffFilter = ref('all')
const versionFilter = ref('all')
const pageIndex = ref(1)
const pageSize = 24

const sortState = ref<DataTableSortState | null>(null)

// --- Advanced filters ---
const showAdvFilters = ref(false)
const collapsedLevels = ref(new Set<string>())

const toggleLevel = (key: string) => {
  const s = new Set(collapsedLevels.value)
  if (s.has(key)) {
    s.delete(key)
  } else {
    s.add(key)
  }
  collapsedLevels.value = s
}
const levelMin = ref<number | null>(null)
const levelMax = ref<number | null>(null)
const versionSelect = ref<string[] | null>(null)
const albumSelect = ref<string[] | null>(null)
const b50Filter = ref(false)
const b50Loading = ref(false)
const groupBy = ref<'level' | 'version' | 'album'>('level')

// --- Modals ---
const showSongDetail = ref(false)
const selectedSong = ref<Song | null>(null)
const showQuickUpload = ref(false)
const uploadTarget = ref({ title: '', difficulty: 'detected' as Difficulty, level: 0, chartId: 0 })

// --- Tabs ---
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

// --- Dynamic filter options ---
const versionOptions = computed(() => {
  if (!appStore.charts) return []
  const versions = [...new Set(appStore.charts.map((c) => c.version))].sort((a, b) => {
    const pa = a.split('.').map(Number)
    const pb = b.split('.').map(Number)
    for (let i = 0; i < Math.max(pa.length, pb.length); i++) {
      const diff = (pa[i] ?? 0) - (pb[i] ?? 0)
      if (diff) return diff
    }
    return 0
  })
  return versions.map((v) => ({ label: v, value: v }))
})

const albumOptions = computed(() => {
  if (!appStore.charts) return []
  const albums = [...new Set(appStore.charts.map((c) => c.album).filter(Boolean))].sort()
  return albums.map((a) => ({ label: a, value: a }))
})

const groupByOptions = computed(() => [
  { label: t('term.level'), value: 'level' },
  { label: t('term.version'), value: 'version' },
  { label: t('term.album'), value: 'album' },
])

// --- Comparisons ---
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

// --- Enhanced search: multi-field fuzzy ---
const matchesSearch = (chart: ChartInfo, query: string): boolean => {
  if (!query) return true
  const tokens = query.toLowerCase().split(/\s+/).filter(Boolean)
  if (tokens.length === 0) return true

  const fields = [
    chart.title,
    chart.artist,
    chart.album ?? '',
    chart.genre ?? '',
    chart.level_design ?? '',
  ].map((f) => f.toLowerCase())

  return tokens.every((token) => fields.some((field) => field.includes(token)))
}

// --- Filtered data ---
const filteredData = computed(() => {
  let data = Array.from(appStore.charts ?? [])

  // Text search (enhanced multi-field fuzzy)
  if (search.value) {
    data = data.filter((c) => matchesSearch(c, search.value))
  }

  // Difficulty tab filter
  if (diffFilter.value !== 'all') {
    data = data.filter((c) => c.difficulty === diffFilter.value)
  }

  // Version tab filter (new/old)
  if (versionFilter.value === 'old') data = data.filter((c) => !c.b15)
  else if (versionFilter.value === 'new') data = data.filter((c) => c.b15)

  // Advanced: level range
  if (levelMin.value != null && !isNaN(levelMin.value)) {
    data = data.filter((c) => c.level >= levelMin.value!)
  }
  if (levelMax.value != null && !isNaN(levelMax.value)) {
    data = data.filter((c) => c.level <= levelMax.value!)
  }

  // Advanced: version select
  if (versionSelect.value && versionSelect.value.length > 0) {
    const vs = new Set(versionSelect.value)
    data = data.filter((c) => vs.has(c.version))
  }

  // Advanced: album select
  if (albumSelect.value && albumSelect.value.length > 0) {
    const as = new Set(albumSelect.value)
    data = data.filter((c) => as.has(c.album))
  }

  // Advanced: B50 filter
  if (b50Filter.value && appStore.b50ChartIds) {
    const b50Set = new Set(appStore.b50ChartIds)
    data = data.filter((c) => b50Set.has(c.id))
  }

  // Sort (only for table view, but apply regardless for consistency)
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

// Group filtered data for grid view by selected criteria
interface ChartGroup {
  key: string
  charts: ChartInfo[]
}

const groupedData = computed<ChartGroup[]>(() => {
  const map = new Map<string, ChartInfo[]>()

  for (const chart of filteredData.value) {
    let key: string
    switch (groupBy.value) {
      case 'version':
        key = chart.version
        break
      case 'album':
        key = chart.album || '-'
        break
      default: // level
        key = (Math.round(chart.level * 10) / 10).toFixed(1)
        break
    }
    const arr = map.get(key)
    if (arr) {
      arr.push(chart)
    } else {
      map.set(key, [chart])
    }
  }

  const groups: ChartGroup[] = []
  for (const [key, charts] of map) {
    groups.push({ key, charts })
  }

  // Sort groups
  if (groupBy.value === 'level') {
    groups.sort((a, b) => parseFloat(b.key) - parseFloat(a.key))
  } else if (groupBy.value === 'version') {
    groups.sort((a, b) => compareVersions(b.key, a.key))
  } else {
    groups.sort((a, b) => a.key.localeCompare(b.key))
  }

  return groups
})

// Auto-collapse: only show first ~20 items, collapse groups beyond threshold
const AUTO_EXPAND_LIMIT = 20

const computeAutoCollapsed = (groups: ChartGroup[]): Set<string> => {
  const collapsed = new Set<string>()
  let total = 0
  for (const group of groups) {
    if (total >= AUTO_EXPAND_LIMIT) {
      collapsed.add(group.key)
    } else {
      total += group.charts.length
    }
  }
  return collapsed
}

watch(groupBy, () => {
  collapsedLevels.value = computeAutoCollapsed(groupedData.value)
})

// Set initial auto-collapse when data first loads
watch(groupedData, (groups) => {
  if (collapsedLevels.value.size === 0 && groups.length > 0) {
    collapsedLevels.value = computeAutoCollapsed(groups)
  }
}, { immediate: true })

const handleSorterUpdate = (sorter: DataTableSortState | DataTableSortState[] | null) => {
  if (Array.isArray(sorter)) {
    sortState.value = sorter[0] ?? null
  } else {
    sortState.value = sorter
  }
}

// --- B50 filter toggle ---
const toggleB50Filter = async () => {
  if (b50Filter.value) {
    b50Filter.value = false
    return
  }

  // Need to load B50 data
  if (!appStore.b50ChartIds) {
    if (!userStore.logged_in && !USE_MOCK) {
      message.warning(t('message.login_required_b50'))
      return
    }

    b50Loading.value = true
    try {
      if (USE_MOCK) {
        const mock = getMockB50()
        appStore.b50ChartIds = mock.records.map((r) => r.chart.id)
      } else {
        const res = await getRecords(userStore.username, 'b50')
        appStore.b50ChartIds = res.data.records.map((r) => r.chart.id)
      }
    } catch {
      message.error(t('message.get_record_failed'))
      b50Loading.value = false
      return
    }
    b50Loading.value = false
  }

  b50Filter.value = true
}

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

// --- Table columns ---
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
    fixed: "left",
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
    key: 'difficulty',
    width: 90,
    render(row) {
      return h(DifficultyBadge, { difficulty: row.difficulty, short: false })
    },
  },
  {
    title: t('term.level'),
    key: 'level',
    width: 80,
    sorter: true,
    render(row) {
      return h('span', { class: 'mono' }, row.level.toFixed(1))
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

/* Filters row */

/* Advanced filter toggle */
.adv-filter-toggle {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  background: none;
  border: none;
  color: var(--text-secondary);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
  padding: var(--space-1) 0;
  margin-bottom: var(--space-2);
  font-family: inherit;
  transition: color var(--transition-fast);
}
@media (hover: hover) {
  .adv-filter-toggle:hover { color: var(--text-primary); }
}
.adv-filter-chevron {
  transition: transform var(--transition-fast);
  flex-shrink: 0;
}
.adv-filter-chevron.open {
  transform: rotate(90deg);
}

/* Advanced filters panel */
.adv-filters {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: var(--space-4);
}

.adv-filters-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: var(--space-4);
  align-items: end;
}

.filter-group {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.filter-label {
  font-size: var(--text-xs);
  color: var(--text-muted);
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.level-range-inputs {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.level-input {
  width: 80px;
  min-height: 34px;
  padding: 0 var(--space-2);
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 16px;
  font-family: var(--font-mono);
  outline: none;
  transition: border-color var(--transition-fast);
}
.level-input:focus {
  border-color: var(--accent);
}
.level-input::placeholder {
  color: var(--text-muted);
  font-family: var(--font-sans);
}

.range-sep {
  color: var(--text-muted);
  font-size: var(--text-sm);
}

.adv-filters-bottom {
  margin-top: var(--space-3);
  padding-top: var(--space-3);
  border-top: 1px solid var(--border);
}

.b50-btn {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  padding: 7px 14px;
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text-secondary);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
  transition: background var(--transition-fast), color var(--transition-fast), border-color var(--transition-fast);
  font-family: inherit;
  white-space: nowrap;
  min-height: 34px;
}
.b50-btn.active {
  background: var(--accent-muted);
  border-color: var(--accent);
  color: var(--accent);
}
@media (hover: hover) {
  .b50-btn:not(.active):hover {
    border-color: var(--border-hover);
    color: var(--text-primary);
  }
}

/* Divider between filters and content */
.section-divider {
  border: none;
  border-top: 1px solid var(--border);
  margin: var(--space-4) 0;
}

/* Grid view - grouped by level */
.grid-view {
  margin-top: 0;
}

.level-groups {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.level-row {
  display: flex;
  gap: var(--space-3);
  align-items: stretch;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: var(--space-3);
  transition: width var(--transition-base);
}

.level-row--collapsed {
  padding: var(--space-1) var(--space-3);
}

.level-label {
  display: flex;
  align-items: baseline;
  gap: var(--space-1);
  flex-shrink: 0;
  width: 140px;
  min-width: 140px;
  padding: var(--space-1) 0;
  background: none;
  border: none;
  cursor: pointer;
  color: var(--text-primary);
  font-family: inherit;
  transition: color var(--transition-fast);
  overflow: hidden;
}
@media (hover: hover) {
  .level-label:hover { color: var(--accent); }
}

.level-chevron {
  color: var(--text-muted);
  flex-shrink: 0;
  transition: transform var(--transition-fast);
}
.level-chevron.open {
  transform: rotate(90deg);
}

.level-value {
  font-family: var(--font-mono);
  font-size: var(--text-lg);
  font-weight: 700;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 80px;
}

.level-count {
  font-size: 11px;
  color: var(--text-muted);
  font-weight: 400;
  white-space: nowrap;
  flex-shrink: 0;
}

.level-cards {
  display: flex;
  gap: var(--space-3);
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
  scrollbar-width: thin;
  scrollbar-color: var(--border) transparent;
  padding-bottom: var(--space-1);
  flex: 1;
  min-width: 0;
}
.level-cards::-webkit-scrollbar {
  height: 4px;
}
.level-cards::-webkit-scrollbar-track {
  background: transparent;
}
.level-cards::-webkit-scrollbar-thumb {
  background: var(--border);
  border-radius: 2px;
}

/* Cards in level rows should have fixed width */
.level-cards :deep(.song-grid-card) {
  flex-shrink: 0;
  width: 130px;
}

@media (max-width: 639px) {
  .level-row {
    flex-wrap: wrap;
    gap: var(--space-2);
    padding: var(--space-2);
  }
  .level-label {
    width: 100%;
    min-width: 0;
  }
  .level-value {
    font-size: var(--text-base);
  }
  .level-cards {
    width: 100%;
    gap: var(--space-2);
  }
  .level-cards :deep(.song-grid-card) {
    width: 105px;
  }
}

.empty-state {
  padding: var(--space-10) 0;
  display: flex;
  justify-content: center;
}

/* Table view */
.table-wrapper {
  -webkit-overflow-scrolling: touch;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 10px;
  overflow: hidden;
  padding: var(--space-3);
}

/* Pagination */
.pagination-row {
  display: flex;
  justify-content: center;
  margin-top: var(--space-5);
}

/* Table styles */
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
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: 6px;
  transition: background var(--transition-fast), color var(--transition-fast);
}
@media (hover: hover) {
  :deep(.action-btn:hover) { background: rgba(255,255,255,0.06); color: var(--text-primary); }
}
</style>
