import type { Difficulty } from '@/api/types'

/**
 * Canonical chart-difficulty order used across the app.
 * Matches the backend enum order and the display order on the song detail page.
 */
export const DIFFICULTY_ORDER: readonly Difficulty[] = [
  'detected',
  'invaded',
  'massive',
  'reboot',
] as const

/** Numeric rank for each difficulty, derived from DIFFICULTY_ORDER. */
export const DIFFICULTY_RANK: Record<Difficulty, number> = Object.freeze(
  DIFFICULTY_ORDER.reduce<Record<string, number>>((acc, d, i) => {
    acc[d] = i
    return acc
  }, {}),
) as Record<Difficulty, number>

/** Sort any array of items carrying a `difficulty` field by the canonical order. */
export function sortByDifficulty<T extends { difficulty: Difficulty }>(items: readonly T[]): T[] {
  return [...items].sort(
    (a, b) => DIFFICULTY_RANK[a.difficulty] - DIFFICULTY_RANK[b.difficulty],
  )
}
