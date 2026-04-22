import type { ChartInfo, PlayRecordInfo, PlayRecordResponse, User, AllChartsResponse } from './types'
import { DIFFICULTY_ORDER } from '@/utils/difficulty'

// ---- Mock Song Data ----
const difficulties = DIFFICULTY_ORDER

const songTitles = [
  { title: 'Paradigm Shift', artist: 'Zekk', version: '1.0', album: 'Core', genre: 'Hardcore', cover: 'Paradigm Shift.png' },
  { title: 'Protoflicker', artist: 'Frums', version: '1.0', album: 'Core', genre: 'Experimental', cover: 'Protoflicker.png' },
  { title: 'REDRAVE', artist: 'Laur', version: '2.0', album: 'Rave', genre: 'Artcore', cover: 'REDRAVE.png' },
  { title: 'Halcyon', artist: 'xi', version: '1.0', album: 'Core', genre: 'Classical', cover: 'Halcyon.png' },
  { title: 'Chronostasis', artist: 'sakuzyo', version: '2.0', album: 'Time', genre: 'Orchestral', cover: 'Chronostasis.png' },
  { title: 'Hemisphere', artist: 'Feryquitous', version: '1.0', album: 'Core', genre: 'Artcore', cover: 'Hemisphere.png' },
  { title: 'Divergence', artist: 't+pazolite', version: '2.0', album: 'Rave', genre: 'Hardcore', cover: 'Divergence.png' },
  { title: 'Selenotaxis', artist: 'Camellia', version: '1.0', album: 'Core', genre: 'Electronic', cover: 'Selenotaxis.png' },
  { title: 'Kokytos', artist: 'Masashi', version: '2.0', album: 'Inferno', genre: 'Orchestral', cover: 'Kokytos.png' },
  { title: 'Metheus', artist: 'USAO', version: '1.0', album: 'Core', genre: 'Hardcore', cover: 'Metheus.png' },
  { title: 'Hydra', artist: 'Aoi', version: '1.0', album: 'Core', genre: 'Future Bass', cover: 'Hydra.png' },
  { title: 'Kronos', artist: 'Gram', version: '2.0', album: 'Time', genre: 'Orchestral', cover: 'Kronos.png' },
  { title: 'Pulsar', artist: 'LeaF', version: '1.0', album: 'Core', genre: 'Experimental', cover: 'Pulsar.png' },
  { title: 'Iridescence', artist: 'seatrus', version: '2.0', album: 'Spectrum', genre: 'Artcore', cover: 'Iridescence.png' },
  { title: 'Incyde', artist: 'Silentroom', version: '1.0', album: 'Core', genre: 'Electronic', cover: 'Incyde.png' },
  { title: 'Rainmaker', artist: 'Sakuzyo', version: '1.0', album: 'Core', genre: 'Piano', cover: 'Rainmaker.png' },
  { title: 'Downfall', artist: 'Laur', version: '2.0', album: 'Dark', genre: 'Hardcore', cover: 'Downfall.png' },
  { title: 'Overflow', artist: 'Cres.', version: '1.0', album: 'Core', genre: 'Artcore', cover: 'Overflow.png' },
  { title: 'Stasis', artist: 'Frums', version: '2.0', album: 'Time', genre: 'Experimental', cover: 'Stasis.png' },
  { title: 'Cipher', artist: 'Camellia', version: '1.0', album: 'Core', genre: 'Electronic', cover: 'Cipher.png' },
  { title: 'Platinum', artist: 'MisoilePunch', version: '1.0', album: 'Core', genre: 'Pop', cover: 'Platinum.png' },
  { title: 'Verreta', artist: 'Silentroom', version: '2.0', album: 'Spectrum', genre: 'Artcore', cover: 'Verreta.png' },
  { title: 'Disorder', artist: 'Risa Yuzuki', version: '1.0', album: 'Core', genre: 'Gothic', cover: 'Disorder.png' },
  { title: 'Kokoro', artist: 'Mameyudoufu', version: '1.0', album: 'Core', genre: 'Future Bass', cover: 'Kokoro.png' },
  { title: 'Oriens', artist: 'technoplanet', version: '2.0', album: 'Spectrum', genre: 'Trance', cover: 'Oriens.png' },
  { title: 'HYSTERIA', artist: 'Kobaryo', version: '2.0', album: 'Rave', genre: 'Speedcore', cover: 'HYSTERIA.png' },
  { title: 'Dogbite', artist: 'EBIMAYO', version: '1.0', album: 'Core', genre: 'Bass', cover: 'Dogbite.png' },
  { title: 'Burn', artist: 'YUC\'e', version: '1.0', album: 'Core', genre: 'Happy Hardcore', cover: 'Burn.png' },
  { title: 'EVERYTHING', artist: 'PSYQUI', version: '2.0', album: 'Rave', genre: 'Future Bass', cover: 'EVERYTHING.png' },
  { title: 'Avantgarde', artist: 'Feryquitous', version: '1.0', album: 'Core', genre: 'Artcore', cover: 'Avantgarde.png' },
]

function generateChartInfoList(): ChartInfo[] {
  const charts: ChartInfo[] = []
  let chartId = 1

  songTitles.forEach((song, songIdx) => {
    const songId = songIdx + 1
    const isB15 = song.version === '2.0'
    const diffCount = Math.random() > 0.7 ? 4 : 3
    const baseLevels = [
      8 + Math.random() * 3,
      10 + Math.random() * 2,
      12 + Math.random() * 2,
      13 + Math.random() * 2,
    ]

    for (let d = 0; d < diffCount; d++) {
      const level = Math.round(baseLevels[d] * 10) / 10
      charts.push({
        id: chartId++,
        song_id: songId,
        title: song.title,
        artist: song.artist,
        bpm: String(120 + Math.floor(Math.random() * 80)),
        cover: song.cover,
        illustrator: 'Illustrator',
        version: song.version,
        album: song.album,
        genre: song.genre,
        length: `${2 + Math.floor(Math.random() * 2)}:${String(Math.floor(Math.random() * 60)).padStart(2, '0')}`,
        b15: isB15,
        wiki_id: `w${songId}`,
        difficulty: difficulties[d],
        level,
        fitting_level: Math.round((level + (Math.random() - 0.5) * 0.4) * 10) / 10,
        level_design: ['Chart_A', 'Chart_B', 'Chart_C', 'Chart_D'][d],
        notes: 400 + Math.floor(Math.random() * 800),
      })
    }
  })

  return charts
}

function calculateRating(level: number, score: number): number {
  if (score >= 1009000) return Math.round((level + 2) * 100)
  if (score >= 1007000) return Math.round((level + 1.5 + (score - 1007000) / 4000) * 100)
  if (score >= 1005000) return Math.round((level + 1 + (score - 1005000) / 4000) * 100)
  if (score >= 1000000) return Math.round((level + (score - 1000000) / 5000) * 100)
  if (score >= 990000) return Math.round((level - 1 + (score - 990000) / 10000) * 100)
  if (score >= 970000) return Math.round((level - 3 + (score - 970000) / 10000) * 100)
  if (score >= 900000) return Math.round((level - 5 + (score - 900000) * 2 / 70000) * 100)
  return Math.round(Math.max(0, (level - 5) * (score / 900000)) * 100)
}

function generateRecords(charts: ChartInfo[], count: number): PlayRecordInfo[] {
  const records: PlayRecordInfo[] = []
  for (let i = 0; i < count; i++) {
    const chart = charts[Math.floor(Math.random() * charts.length)]
    const score = 900000 + Math.floor(Math.random() * 110000)
    const rating = calculateRating(chart.level, score)
    const daysAgo = Math.floor(Math.random() * 90)
    const date = new Date()
    date.setDate(date.getDate() - daysAgo)

    records.push({
      id: i + 1,
      score,
      rating,
      record_time: date.toISOString(),
      chart: {
        id: chart.id,
        song_id: chart.song_id,
        title: chart.title,
        cover: chart.cover,
        version: chart.version,
        b15: chart.b15,
        wiki_id: chart.wiki_id,
        difficulty: chart.difficulty,
        level: chart.level,
        fitting_level: chart.fitting_level,
      },
    })
  }
  return records
}

function generateBest50(charts: ChartInfo[]): PlayRecordInfo[] {
  const oldCharts = charts.filter((c) => !c.b15)
  const newCharts = charts.filter((c) => c.b15)

  const b35Records: PlayRecordInfo[] = []
  const b15Records: PlayRecordInfo[] = []

  // Unique charts for best records
  const usedOld = new Set<number>()
  const usedNew = new Set<number>()

  for (const chart of oldCharts) {
    if (usedOld.has(chart.id)) continue
    usedOld.add(chart.id)
    const score = 950000 + Math.floor(Math.random() * 60000)
    const daysAgo = Math.floor(Math.random() * 60)
    const date = new Date()
    date.setDate(date.getDate() - daysAgo)
    b35Records.push({
      id: 1000 + b35Records.length,
      score,
      rating: calculateRating(chart.level, score),
      record_time: date.toISOString(),
      chart: {
        id: chart.id,
        song_id: chart.song_id,
        title: chart.title,
        cover: chart.cover,
        version: chart.version,
        b15: chart.b15,
        wiki_id: chart.wiki_id,
        difficulty: chart.difficulty,
        level: chart.level,
        fitting_level: chart.fitting_level,
      },
    })
    if (b35Records.length >= 35) break
  }

  for (const chart of newCharts) {
    if (usedNew.has(chart.id)) continue
    usedNew.add(chart.id)
    const score = 950000 + Math.floor(Math.random() * 60000)
    const daysAgo = Math.floor(Math.random() * 30)
    const date = new Date()
    date.setDate(date.getDate() - daysAgo)
    b15Records.push({
      id: 2000 + b15Records.length,
      score,
      rating: calculateRating(chart.level, score),
      record_time: date.toISOString(),
      chart: {
        id: chart.id,
        song_id: chart.song_id,
        title: chart.title,
        cover: chart.cover,
        version: chart.version,
        b15: chart.b15,
        wiki_id: chart.wiki_id,
        difficulty: chart.difficulty,
        level: chart.level,
        fitting_level: chart.fitting_level,
      },
    })
    if (b15Records.length >= 15) break
  }

  // Sort by rating desc
  b35Records.sort((a, b) => b.rating - a.rating)
  b15Records.sort((a, b) => b.rating - a.rating)

  return [...b35Records, ...b15Records]
}

// Cached mock data
let _charts: ChartInfo[] | null = null
let _b50Records: PlayRecordInfo[] | null = null

export function getMockCharts(): ChartInfo[] {
  if (!_charts) _charts = generateChartInfoList()
  return _charts
}

export function getMockB50(): PlayRecordResponse {
  const charts = getMockCharts()
  if (!_b50Records) _b50Records = generateBest50(charts)
  return {
    username: 'demo_user',
    nickname: 'Demo',
    total: _b50Records.length,
    records: _b50Records,
  }
}

export function getMockRecords(
  scope: string,
  pageSize: number,
  pageIndex: number
): PlayRecordResponse {
  const charts = getMockCharts()

  if (scope === 'b50') return getMockB50()

  const allRecords = generateRecords(charts, 200)
  allRecords.sort((a, b) => b.rating - a.rating)

  if (scope === 'best') {
    // One record per chart, highest score
    const bestMap = new Map<number, PlayRecordInfo>()
    for (const rec of allRecords) {
      const existing = bestMap.get(rec.chart.id)
      if (!existing || rec.score > existing.score) {
        bestMap.set(rec.chart.id, rec)
      }
    }
    const bestRecords = Array.from(bestMap.values()).sort((a, b) => b.rating - a.rating)
    const start = (pageIndex - 1) * pageSize
    return {
      username: 'demo_user',
      nickname: 'Demo',
      total: bestRecords.length,
      records: bestRecords.slice(start, start + pageSize),
    }
  }

  // scope === 'all'
  const start = (pageIndex - 1) * pageSize
  return {
    username: 'demo_user',
    nickname: 'Demo',
    total: allRecords.length,
    records: allRecords.slice(start, start + pageSize),
  }
}

export function getMockUser(): User {
  return {
    id: 1,
    username: 'demo_user',
    nickname: 'Demo',
    email: 'demo@example.com',
    qq_account: '123456789',
    account: 'demo_account',
    account_number: 1001,
    uuid: '550e8400-e29b-41d4-a716-446655440000',
    anonymous_probe: false,
    is_admin: false,
    is_active: true,
    upload_token: 'mock-upload-token-abc123',
    created_at: '',
    updated_at: '',
  }
}

// Flag for mock mode
export const USE_MOCK = false

export function getMockAllCharts(): AllChartsResponse {
  const charts = getMockCharts()
  return {
    username: 'demo_user',
    nickname: 'Demo',
    charts: charts.map((c) => ({
      id: c.id,
      title: c.title,
      version: c.version,
      difficulty: c.difficulty,
      level: c.level,
      score: Math.random() > 0.3 ? 900000 + Math.floor(Math.random() * 110000) : 0,
    })),
  }
}
