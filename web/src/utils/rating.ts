/**
 * Rating formatting utilities.
 *
 * Backend stores ratings as integers (rating × 100). Formatting stays in the
 * integer domain; single-chart calculations mirror pkg/rating.SingleRating.
 */

/** Calculate an integer rating (×100), using the backend's formula and rounding. */
export function calculateSingleRating(level: number, score: number): number {
  score = Math.min(score, 1010000)
  let rating: number

  if (score >= 1009000) {
    rating = level * 10 + 7 + 3 * ((score - 1009000) / 1000) ** 1.35
  } else if (score >= 1000000) {
    rating = 10 * (level + 2 * (score - 1000000) / 30000)
  } else {
    const bounds = [900000, 930000, 950000, 970000, 980000, 990000]
    const rewards = [3, 1, 1, 1, 1, 1]
    rating = bounds.reduce((sum, bound, i) => sum + (score >= bound ? rewards[i] : 0), 0)
    rating += 10 * (level * (score / 1000000) ** 1.5 - 0.9)
  }

  return Math.trunc(Math.max(0, rating) * 100 + 0.00002)
}

/**
 * Format a single integer rating (×100) as a fixed-2-decimal string.
 * e.g. 12345 → "123.45", 5 → "0.05", 100 → "1.00"
 */
export function formatRating(rating: number): string {
  const s = String(rating).padStart(3, '0')
  const intPart = s.slice(0, -2) || '0'
  const fracPart = s.slice(-2)
  return `${intPart}.${fracPart}`
}

/**
 * Compute an average rating from a sum of integer ratings and a record count.
 * All arithmetic is integer-only; the decimal point is inserted manually.
 *
 * @param sum   Sum of `record.rating` values (each is rating × 100)
 * @param count Number of records (e.g. 50, 35, 15)
 * @param decimals Number of decimal places to keep (default 4)
 */
export function formatAvgRating(sum: number, count: number, decimals = 4): string {
  if (count === 0) {
    return (0).toFixed(decimals)
  }
  const divisor = count * 100
  // Integer-only scaled value: round(sum * 10^decimals / divisor)
  const scaled = Math.floor((sum * (10 ** decimals) + divisor / 2) / divisor)
  const s = String(scaled).padStart(decimals + 1, '0')
  const intPart = s.slice(0, -decimals) || '0'
  const fracPart = s.slice(-decimals)
  return `${intPart}.${fracPart}`
}
