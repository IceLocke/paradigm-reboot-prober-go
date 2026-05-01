/**
 * Rating formatting utilities.
 *
 * Backend stores ratings as integers (rating × 100). All calculations here
 * stay in the integer domain and insert the decimal point manually so that
 * no floating-point errors can creep in.
 */

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
