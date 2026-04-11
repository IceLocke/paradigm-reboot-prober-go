import type { ChartInfo } from '@/api/types'

export interface LevelBracket {
  label: string
  minVal: number
  maxVal: number
}

/**
 * Build sorted level bracket options from chart data.
 * - Level ≤ 12: single bracket per integer (e.g. "10" → 10.0~10.9)
 * - Level > 12: split into base and "+" (e.g. "13" → 13.0~13.5, "13+" → 13.6~13.9)
 *   "+" variant only appears if any chart has .6~.9 at that integer level.
 */
export function buildLevelBrackets(charts: ChartInfo[]): LevelBracket[] {
  const intMap = new Map<number, { hasBase: boolean; hasPlus: boolean }>()

  for (const c of charts) {
    const base = Math.floor(c.level)
    if (!intMap.has(base)) intMap.set(base, { hasBase: false, hasPlus: false })
    const entry = intMap.get(base)!
    if (base > 12 && c.level - base >= 0.6) {
      entry.hasPlus = true
    } else {
      entry.hasBase = true
    }
  }

  const result: LevelBracket[] = []
  for (const [base, flags] of 
    Array.from(intMap.entries())
      .sort((a, b) => (b[0] - a[0]))
  ) {
    if (base <= 12) {
      result.push({ label: String(base), minVal: base, maxVal: base + 0.9 })
    } else {
      if (flags.hasPlus) {
        result.push({ label: `${base}+`, minVal: base + 0.6, maxVal: base + 0.9 })
      }
      if (flags.hasBase) {
        result.push({ label: String(base), minVal: base, maxVal: base + 0.5 })
      }
    }
  }
  return result
}
