import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { refDebounced } from '@vueuse/core'
import type { DataTableSortState } from 'naive-ui'

import { toastWarning, toastError } from '@/utils/toast'

import { useAppStore } from '@/stores/app'
import { useUserStore } from '@/stores/user'
import { getRecords } from '@/api/record'
import { USE_MOCK, getMockB50 } from '@/api/mock'
import type { ChartInfo } from '@/api/types'
import { buildLevelBrackets } from '@/utils/levelBrackets'
import type { LevelBracket } from '@/utils/levelBrackets'
import KanaUtils from '@/utils/kana'

const compareVersions = (a: string, b: string): number => {
  const aParts = a.split('.')
  const bParts = b.split('.')
  const len = Math.max(aParts.length, bParts.length)
  for (let i = 0; i < len; i++) {
    const diff = (Number(aParts[i]) || 0) - (Number(bParts[i]) || 0)
    if (diff) return diff
  }
  return 0
}

const matchesSearch = (chart: ChartInfo, query: string): boolean => {
  if (!query) return true
  const tokens = KanaUtils.toHiraganaCase(KanaUtils.toZenkanaCase(query))
    .toLowerCase().split(/\s+/).filter(Boolean)
  if (tokens.length === 0) return true

  const fields = [
    chart.title,
    chart.artist,
    chart.genre ?? '',
    chart.level_design ?? '',
  ].map((f) => KanaUtils.toHiraganaCase(KanaUtils.toZenkanaCase(f)).toLowerCase())

  return tokens.every((token) => fields.some((field) => field.includes(token)))
}

export function useChartFilters() {
  const { t } = useI18n()
  const appStore = useAppStore()
  const userStore = useUserStore()

  // --- Basic filters ---
  const search = ref('')
  const searchDebounced = refDebounced(search, 300)
  const diffFilter = ref<string[]>([])
  const versionFilter = ref('all')
  const pageIndex = ref(1)
  const pageSize = 24

  const sortState = ref<DataTableSortState | null>(null)

  // --- Advanced filters ---
  const showAdvFilters = ref(false)
  const levelMin = ref<number | null>(null)
  const levelMax = ref<number | null>(null)
  const versionSelect = ref<string[] | null>(null)
  const albumSelect = ref<string[] | null>(null)
  const b50Filter = ref(false)
  const b50Loading = ref(false)
  const groupBy = ref<'level' | 'version' | 'album'>('level')

  // --- Difficulty options (multi-select) ---
  const diffOptions = [
    { key: 'detected', label: 'DET' },
    { key: 'invaded', label: 'IVD' },
    { key: 'massive', label: 'MSV' },
    { key: 'reboot', label: 'RBT' },
  ]

  const toggleDiff = (key: string) => {
    if (key === 'all') {
      diffFilter.value = []
      return
    }
    const idx = diffFilter.value.indexOf(key)
    if (idx >= 0) {
      diffFilter.value = diffFilter.value.filter((k) => k !== key)
    } else {
      diffFilter.value = [...diffFilter.value, key]
    }
    if (diffFilter.value.length === diffOptions.length) {
      diffFilter.value = []
    }
  }

  const versionTabs = [
    { key: 'all', label: t('common.all') },
    { key: 'new', label: t('term.current') },
    { key: 'old', label: t('term.past') },
  ]

  // --- Dynamic filter options ---
  const versionOptions = computed(() => {
    if (!appStore.charts) return []
    const versions = [...new Set(appStore.charts.map((c) => c.version))].sort((a, b) => compareVersions(b, a))
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

  // --- Level bracket quick-select options ---
  const levelBrackets = computed<LevelBracket[]>(() => {
    if (!appStore.charts) return []
    return buildLevelBrackets(appStore.charts)
  })

  // --- Filtered data ---
  const filteredData = computed(() => {
    let data = Array.from(appStore.charts ?? [])

    if (searchDebounced.value) {
      data = data.filter((c) => matchesSearch(c, searchDebounced.value))
    }

    if (diffFilter.value.length > 0) {
      const diffs = new Set(diffFilter.value)
      data = data.filter((c) => diffs.has(c.difficulty))
    }

    if (versionFilter.value === 'old') data = data.filter((c) => !c.b15)
    else if (versionFilter.value === 'new') data = data.filter((c) => c.b15)

    if (levelMin.value != null && !isNaN(levelMin.value)) {
      data = data.filter((c) => c.level >= levelMin.value!)
    }
    if (levelMax.value != null && !isNaN(levelMax.value)) {
      data = data.filter((c) => c.level <= levelMax.value!)
    }

    if (versionSelect.value && versionSelect.value.length > 0) {
      const vs = new Set(versionSelect.value)
      data = data.filter((c) => vs.has(c.version))
    }

    if (albumSelect.value && albumSelect.value.length > 0) {
      const as = new Set(albumSelect.value)
      data = data.filter((c) => as.has(c.album))
    }

    if (b50Filter.value && appStore.b50ChartIds) {
      const b50Set = new Set(appStore.b50ChartIds)
      data = data.filter((c) => b50Set.has(c.id))
    }

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

  // --- B50 filter toggle ---
  const toggleB50Filter = async () => {
    if (b50Filter.value) {
      b50Filter.value = false
      return
    }

    if (!appStore.b50ChartIds) {
      if (!userStore.logged_in && !USE_MOCK) {
        toastWarning('message.login_required_b50')
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
      } catch (err: unknown) {
        toastError('message.get_record_failed', err)
        b50Loading.value = false
        return
      }
      b50Loading.value = false
    }

    b50Filter.value = true
  }

  return {
    search,
    diffFilter,
    diffOptions,
    toggleDiff,
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
    versionTabs,
    versionOptions,
    albumOptions,
    groupByOptions,
    levelBrackets,
    filteredData,
    paginatedData,
    handleSorterUpdate,
    toggleB50Filter,
  }
}
