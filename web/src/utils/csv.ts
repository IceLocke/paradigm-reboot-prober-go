import type { ChartWithScore } from '@/api/types'

/**
 * Generate a CSV string from an array of ChartWithScore.
 * Includes UTF-8 BOM for Excel compatibility.
 */
export function exportCsv(charts: ChartWithScore[]): string {
  const header = 'id,title,version,difficulty,level,score'
  const sorted = [...charts].sort((a, b) => b.level - a.level)
  const rows = sorted.map((c) => {
    const title = c.title.includes(',') || c.title.includes('"')
      ? `"${c.title.replace(/"/g, '""')}"`
      : c.title
    return `${c.id},${title},${c.version},${c.difficulty},${c.level},${c.score}`
  })
  return '\uFEFF' + header + '\n' + rows.join('\n') + '\n'
}

/**
 * Decode an ArrayBuffer to string, auto-detecting UTF-8 vs GBK.
 * Tries UTF-8 first (with fatal: true); falls back to GBK on failure.
 */
export function decodeFileBuffer(buffer: ArrayBuffer): string {
  try {
    return new TextDecoder('utf-8', { fatal: true }).decode(buffer)
  } catch {
    return new TextDecoder('gbk').decode(buffer)
  }
}

/**
 * Parse a CSV row respecting quoted fields.
 */
function parseCsvRow(line: string): string[] {
  const fields: string[] = []
  let current = ''
  let inQuotes = false

  for (let i = 0; i < line.length; i++) {
    const ch = line[i]
    if (inQuotes) {
      if (ch === '"') {
        if (i + 1 < line.length && line[i + 1] === '"') {
          current += '"'
          i++
        } else {
          inQuotes = false
        }
      } else {
        current += ch
      }
    } else {
      if (ch === '"') {
        inQuotes = true
      } else if (ch === ',') {
        fields.push(current.trim())
        current = ''
      } else {
        current += ch
      }
    }
  }
  fields.push(current.trim())
  return fields
}

export interface CsvRecord {
  chartId: number
  score: number
}

/**
 * Parse CSV text into an array of {chartId, score} records.
 * Matches columns by header name: "id" → chart_id, "score" → score.
 * Filters out rows with score <= 0 or score > 1010000.
 */
export function parseCsvToRecords(csvText: string): CsvRecord[] {
  // Strip BOM if present
  const text = csvText.startsWith('\uFEFF') ? csvText.slice(1) : csvText
  const lines = text.split(/\r?\n/).filter((l) => l.trim() !== '')
  if (lines.length < 2) return []

  const headers = parseCsvRow(lines[0]).map((h) => h.toLowerCase())
  let idIdx = headers.indexOf('id')
  if (idIdx === -1) idIdx = headers.indexOf('chart_id')
  if (idIdx === -1) idIdx = headers.indexOf('song_level_id')
  const scoreIdx = headers.indexOf('score')
  if (idIdx === -1 || scoreIdx === -1) return []

  const records: CsvRecord[] = []
  for (let i = 1; i < lines.length; i++) {
    const fields = parseCsvRow(lines[i])
    const chartId = parseInt(fields[idIdx], 10)
    const score = parseInt(fields[scoreIdx], 10)
    if (isNaN(chartId) || isNaN(score)) continue
    if (score <= 0 || score > 1010000) continue
    records.push({ chartId, score })
  }
  return records
}

/**
 * Filter out records whose score matches the current best score exactly.
 * @param records Parsed CSV records
 * @param currentBestMap Map from chart_id to current best score (from all-charts API)
 * @returns Records that actually need uploading
 */
export function filterUnchangedRecords(
  records: CsvRecord[],
  currentBestMap: Map<number, number>,
): CsvRecord[] {
  return records.filter((r) => r.score !== currentBestMap.get(r.chartId))
}
