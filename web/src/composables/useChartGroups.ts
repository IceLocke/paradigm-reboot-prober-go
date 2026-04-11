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
  groupBy: Ref<'level' | 'version' | 'album'>
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
      groups.push({ key, charts })
    }

    if (groupBy.value === 'level') {
      groups.sort((a, b) => parseFloat(b.key) - parseFloat(a.key))
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
