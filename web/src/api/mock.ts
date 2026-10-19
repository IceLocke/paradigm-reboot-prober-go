import type { ChartInfo, PlayRecordInfo, PlayRecordResponse, User, AllChartsResponse } from './types'
import { DIFFICULTY_ORDER } from '@/utils/difficulty'
import { calculateSingleRating as calculateRating } from '@/utils/rating'

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
      // Simulate the fitting microservice abstaining for ~10% of charts
      // (insufficient samples → fitting_level === null). See
      // docs/fitting_level.en.md §4.4.
      const fittingLevel = Math.random() < 0.1
        ? null
        : Math.round((level + (Math.random() - 0.5) * 0.4) * 10) / 10
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
        fitting_level: fittingLevel,
        level_design: ['Chart_A', 'Chart_B', 'Chart_C', 'Chart_D'][d],
        notes: 400 + Math.floor(Math.random() * 800),
      })
    }
  })

  return charts
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
      fitting_rating: calculateRating(chart.fitting_level ?? chart.level, score),
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

// Cached mock data
let _charts: ChartInfo[] | null = null
let _records: PlayRecordInfo[] | null = null

export function getMockCharts(): ChartInfo[] {
  if (!_charts) _charts = generateChartInfoList()
  return _charts
}

export function getMockB50(fitting = false, underflow = 0): PlayRecordResponse {
  const data = getMockRecords('best', Number.MAX_SAFE_INTEGER, 1, fitting ? 'fitting_rating' : 'rating')
  const records = [
    ...data.records.filter((record) => !record.chart.b15).slice(0, 35 + underflow),
    ...data.records.filter((record) => record.chart.b15).slice(0, 15 + underflow),
  ]
  return { ...data, total: records.length, records }
}

export function getMockRecords(
  scope: string,
  pageSize: number,
  pageIndex: number,
  sortBy = 'rating',
  order = 'desc',
): PlayRecordResponse {
  if (scope === 'b50') return getMockB50(sortBy === 'fitting_rating')
  if (!_records) _records = generateRecords(getMockCharts(), 200)
  let records = [..._records]

  if (scope === 'best') {
    const bestMap = new Map<number, PlayRecordInfo>()
    for (const record of records) {
      const existing = bestMap.get(record.chart.id)
      if (!existing || record.score > existing.score) bestMap.set(record.chart.id, record)
    }
    records = Array.from(bestMap.values())
  }

  const ratingKey = sortBy === 'fitting_rating' ? 'fitting_rating' : 'rating'
  const direction = order === 'asc' ? -1 : 1
  records.sort((a, b) => direction * (b[ratingKey] - a[ratingKey] || b.id - a.id))
  const start = (pageIndex - 1) * pageSize
  return {
    username: 'demo_user',
    nickname: 'Demo',
    total: records.length,
    records: records.slice(start, start + pageSize),
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
