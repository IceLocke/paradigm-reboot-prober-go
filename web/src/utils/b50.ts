import type { ChartInfoSimple, PlayRecordInfo } from '@/api/types'

/** Use the published fitting level, falling back to the official level when absent. */
export function getRatingLevel(chart: ChartInfoSimple, fitting = false): number {
  return fitting ? chart.fitting_level ?? chart.level : chart.level
}

/** Display the server's fitted rating without mutating API records or reselecting B50. */
export function withFittingRatings(records: PlayRecordInfo[]): PlayRecordInfo[] {
  return records.map((record) => ({
    ...record,
    rating: record.fitting_rating,
  }))
}

function compareRecords(a: PlayRecordInfo, b: PlayRecordInfo): number {
  return b.rating - a.rating || b.id - a.id
}

/** Select B35 + B15, optionally keeping extra records below each cutoff for image export. */
export function selectBest50(records: PlayRecordInfo[], underflow = 0) {
  const ranked = [...records].sort(compareRecords)
  return {
    b35Records: ranked.filter((record) => !record.chart.b15).slice(0, 35 + underflow),
    b15Records: ranked.filter((record) => record.chart.b15).slice(0, 15 + underflow),
  }
}
