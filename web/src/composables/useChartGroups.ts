import { ref, computed, watch } from 'vue'
import type { Ref, ComputedRef } from 'vue'
import type { ChartInfo } from '@/api/types'

export interface ChartGroup {
  key: string
  charts: ChartInfo[]
}

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

const AUTO_EXPAND_LIMIT = 20

export const FITTING_LEVEL_UNKNOWN_KEY = '—'

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

export function useChartGroups(
  filteredData: ComputedRef<ChartInfo[]>,
  groupBy: Ref<'level' | 'fitting_level' | 'version' | 'album'>
) {
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
        case 'fitting_level':
          // Charts that the fitting microservice abstained from (insufficient
          // samples) have fitting_level === null; see docs/fitting_level.en.md
          // §4.4. They are grouped together under a dedicated sentinel key
          // which later sorts to the end of the group list.
          key = chart.fitting_level == null
            ? FITTING_LEVEL_UNKNOWN_KEY
            : (Math.round(chart.fitting_level * 10) / 10).toFixed(1)
          break
        default:
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
      if (groupBy.value === 'fitting_level') {
        // Within a fitting_level group sort by fitting_level desc; nulls last.
        charts.sort((a, b) => {
          const av = a.fitting_level ?? -Infinity
          const bv = b.fitting_level ?? -Infinity
          return bv - av
        })
      } else {
        charts.sort((a, b) => b.level - a.level)
      }
      groups.push({ key, charts })
    }

    if (groupBy.value === 'level') {
      groups.sort((a, b) => parseFloat(b.key) - parseFloat(a.key))
    } else if (groupBy.value === 'fitting_level') {
      // Numeric keys desc, with the unknown sentinel sinking to the end.
      groups.sort((a, b) => {
        const av = a.key === FITTING_LEVEL_UNKNOWN_KEY ? -Infinity : parseFloat(a.key)
        const bv = b.key === FITTING_LEVEL_UNKNOWN_KEY ? -Infinity : parseFloat(b.key)
        return bv - av
      })
    } else if (groupBy.value === 'version') {
      groups.sort((a, b) => compareVersions(b.key, a.key))
    } else {
      groups.sort((a, b) => a.key.localeCompare(b.key))
    }

    return groups
  })

  watch(filteredData, () => {
    collapsedLevels.value = computeAutoCollapsed(groupedData.value)
  })

  watch(groupBy, () => {
    collapsedLevels.value = computeAutoCollapsed(groupedData.value)
  })

  watch(groupedData, (groups) => {
    if (collapsedLevels.value.size === 0 && groups.length > 0) {
      collapsedLevels.value = computeAutoCollapsed(groups)
    }
  }, { immediate: true })

  return { groupedData, collapsedLevels, toggleLevel }
}
