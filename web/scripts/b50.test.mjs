import assert from 'node:assert/strict'
import { test } from 'node:test'
import { calculateSingleRating, formatAvgRating } from '../src/utils/rating.ts'
import { getRatingLevel, recalculateFittingRecords, selectBest50 } from '../src/utils/b50.ts'

function record(id, b15 = false, level = 16, fittingLevel = level) {
  return {
    id,
    score: 1000000,
    rating: calculateSingleRating(level, 1000000),
    record_time: '2026-01-01T00:00:00Z',
    chart: {
      id,
      song_id: id,
      title: `Chart ${id}`,
      cover: '',
      version: '1.0',
      b15,
      wiki_id: `chart_${id}`,
      difficulty: 'massive',
      level,
      fitting_level: fittingLevel,
    },
  }
}

test('single rating matches backend examples, score caps, and integer truncation', () => {
  const cases = [
    [10, 1010000, 11000],
    [10, 1020000, 11000],
    [10, 1009500, 10817],
    [10, 1009000, 10700],
    [10, 1005000, 10333],
    [10, 1000000, 10000],
    [10, 995000, 9825],
    [10, 989999, 9650],
    [10, 990000, 9750],
    [10, 979999, 9401],
    [10, 980000, 9501],
    [10, 969999, 9153],
    [10, 970000, 9253],
    [10, 949999, 8759],
    [10, 950000, 8859],
    [10, 929999, 8368],
    [10, 930000, 8468],
    [10, 899999, 7638],
    [10, 900000, 7938],
    [10, 500000, 2635],
    [10, 0, 0],
    [16.4, 1008900, 16993],
    [16.123456, 1000000, 16123],
    [16.399999999, 1000000, 16400],
  ]
  for (const [level, score, expected] of cases) {
    assert.equal(calculateSingleRating(level, score), expected, `${level} / ${score}`)
  }
})

test('fitting levels change B35 and B15 membership beyond the official cutoffs', () => {
  const source = [
    ...Array.from({ length: 35 }, (_, i) => record(i + 1, false, 16, 15.8)),
    record(36, false, 15.9, 16.2),
    ...Array.from({ length: 15 }, (_, i) => record(i + 101, true, 17, 16.8)),
    record(116, true, 16.9, 17.2),
  ]
  const original = structuredClone(source)
  const official = selectBest50(source)
  assert.ok(!official.b35Records.some((r) => r.id === 36))
  assert.ok(!official.b15Records.some((r) => r.id === 116))
  const fitted = selectBest50(recalculateFittingRecords(source))
  assert.equal(fitted.b35Records.length, 35)
  assert.equal(fitted.b15Records.length, 15)
  assert.equal(fitted.b35Records[0].id, 36)
  assert.equal(fitted.b15Records[0].id, 116)
  assert.equal(fitted.b35Records[0].rating, 16200)
  assert.equal(fitted.b15Records[0].rating, 17200)
  assert.deepEqual(source, original)
})

test('missing fitting levels use official levels without treating zero as missing', () => {
  const source = [record(1, false, 16.4, null), record(2, false, 16.4, 0)]
  const fitted = recalculateFittingRecords(source)
  assert.equal(getRatingLevel(source[0].chart, true), 16.4)
  assert.equal(getRatingLevel(source[1].chart, true), 0)
  assert.equal(fitted.find((r) => r.id === 1).rating, 16400)
  assert.equal(fitted.find((r) => r.id === 2).rating, 0)
})

test('overflow retains five extra fitted records per season; global ranking ignores seasons', () => {
  const records = recalculateFittingRecords([
    ...Array.from({ length: 60 }, (_, i) => record(i + 1, false, 16, 16.2)),
    ...Array.from({ length: 30 }, (_, i) => record(i + 101, true, 15, 15.2)),
  ])
  const standard = selectBest50(records)
  const overflow = selectBest50(records, 5)
  assert.equal(overflow.b35Records.length, 40)
  assert.equal(overflow.b15Records.length, 20)
  assert.deepEqual(overflow.b35Records.slice(0, 35), standard.b35Records)
  assert.deepEqual(overflow.b15Records.slice(0, 15), standard.b15Records)
  assert.equal(records.slice(0, 50).length, 50)
  assert.ok(records.slice(0, 50).every((r) => !r.chart.b15))
  assert.equal(records[0].id, 60)
})

test('sparse B50 keeps season limits and fixed average denominators', () => {
  const { b35Records, b15Records } = selectBest50([record(1, false, 16, 16)])
  assert.equal(b15Records.length, 0)
  const sum = b35Records.reduce((sum, r) => sum + r.rating, 0)
  assert.equal(formatAvgRating(sum, 50), '3.2000')
  assert.equal(formatAvgRating(sum, 35), '4.5714')
  assert.equal(formatAvgRating(0, 15), '0.0000')
  assert.deepEqual(selectBest50([]), { b35Records: [], b15Records: [] })
})
