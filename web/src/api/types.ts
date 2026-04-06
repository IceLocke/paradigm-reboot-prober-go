/**
 * API type definitions — re-exported from auto-generated OpenAPI types.
 *
 * Source of truth: `docs/swagger.json` (backend)
 * Generated via:  `pnpm generate:api`
 *
 * DO NOT define API types manually here. If the backend schema changes:
 *   1. Regenerate: `pnpm generate:api`
 *   2. Fix any type errors surfaced by the compiler.
 *
 * Response model types are wrapped with `DeepRequired` because the backend
 * always returns complete objects — the optionality in the OpenAPI spec is
 * just a side-effect of Go struct tags not marking every field `required`.
 */
import type { components } from './generated'

/** Shorthand for the schemas namespace */
type Schemas = components['schemas']

/**
 * Recursively make all properties required and non-nullable.
 * Used for API *response* models where the backend always returns full objects.
 */
type DeepRequired<T> = T extends (infer U)[]
  ? DeepRequired<U>[]
  : T extends object
    ? { [K in keyof T]-?: DeepRequired<NonNullable<T[K]>> }
    : T

// ─── Enums ────────────────────────────────────────────────────────
export type Difficulty = Schemas['model.Difficulty']

// ─── Generic API Response ─────────────────────────────────────────
export type Response = Schemas['model.Response']
export type Token = DeepRequired<Schemas['model.Token']>
export type UploadToken = DeepRequired<Schemas['model.UploadToken']>
// ─── Domain Models (response — always complete) ───────────────────
export type User = DeepRequired<Schemas['model.User']>

/** Fields from SongBaseOverride — genuinely optional (Go `*string`), excluded from DeepRequired. */
type SongBaseOverrideKeys = 'override_title' | 'override_artist' | 'override_version' | 'override_cover'

export type Chart = Omit<DeepRequired<Schemas['model.Chart']>, 'song' | SongBaseOverrideKeys> &
  Partial<Pick<Schemas['model.Chart'], SongBaseOverrideKeys>> & {
  song?: Song
}
export type Song = Omit<DeepRequired<Schemas['model.Song']>, 'charts'> & {
  charts: Chart[]
}
export type ChartInfo = DeepRequired<Schemas['model.ChartInfo']>
export type ChartInfoSimple = DeepRequired<Schemas['model.ChartInfoSimple']>
export type ChartInput = Omit<DeepRequired<Schemas['model.ChartInput']>, SongBaseOverrideKeys> &
  Partial<Pick<Schemas['model.ChartInput'], SongBaseOverrideKeys>>
export type PlayRecord = Omit<DeepRequired<Schemas['model.PlayRecord']>, 'chart'> & {
  chart?: Chart
}
export type PlayRecordBase = DeepRequired<Schemas['model.PlayRecordBase']>
export type PlayRecordInfo = DeepRequired<Schemas['model.PlayRecordInfo']>
export type PlayRecordResponse = DeepRequired<Schemas['model.PlayRecordResponse']>

// ─── Request DTOs (keep optional fields as-is) ────────────────────
export type BatchCreatePlayRecordRequest = Schemas['request.BatchCreatePlayRecordRequest']
export type CreateUserRequest = Schemas['request.CreateUserRequest']
export type UpdateUserRequest = Schemas['request.UpdateUserRequest']
export type ChangePasswordRequest = Schemas['request.ChangePasswordRequest']
export type CreateSongRequest = Schemas['request.CreateSongRequest']
export type UpdateSongRequest = Schemas['request.UpdateSongRequest']
export type ResetPasswordRequest = Schemas['request.ResetPasswordRequest']
