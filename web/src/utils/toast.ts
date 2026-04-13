import { message } from './discrete'
import i18n from '@/i18n'

// ---------------------------------------------------------------------------
// Error extraction
// ---------------------------------------------------------------------------

/** Extract the `error` string from an AxiosError-shaped object. */
export function extractApiError(error: unknown): string {
  const e = error as { response?: { data?: { error?: string } } }
  return e?.response?.data?.error ?? ''
}

/**
 * Build a human-readable error string:
 *   "<i18n base text>: <API error detail>"
 * If no detail exists, returns just the base text.
 */
export function formatApiError(key: string, error?: unknown): string {
  const { t } = i18n.global
  const base = t(key)
  const detail = error ? extractApiError(error) : ''
  return detail ? `${base}: ${detail}` : base
}

// ---------------------------------------------------------------------------
// Toast helpers — call from anywhere (components, interceptors, utilities)
// ---------------------------------------------------------------------------

/**
 * Show a **success** toast.
 *
 * @param key   - i18n key (defaults to generic `message.request_success`)
 * @param params - interpolation params for the i18n key
 */
export function toastSuccess(key = 'message.request_success', params?: Record<string, unknown>) {
  const { t } = i18n.global
  message.success(t(key, params ?? {}))
}

/**
 * Show an **error** toast, with optional API error detail appended.
 *
 * @param key   - i18n key (defaults to generic `message.request_failed`)
 * @param error - the caught error; API error detail is extracted automatically
 */
export function toastError(key = 'message.request_failed', error?: unknown) {
  message.error(formatApiError(key, error))
}

/**
 * Show a **warning** toast.
 *
 * @param key   - i18n key
 * @param params - interpolation params
 */
export function toastWarning(key: string, params?: Record<string, unknown>) {
  const { t } = i18n.global
  message.warning(t(key, params ?? {}))
}

/**
 * Show an **info** toast.
 *
 * @param key   - i18n key
 * @param params - interpolation params
 */
export function toastInfo(key: string, params?: Record<string, unknown>) {
  const { t } = i18n.global
  message.info(t(key, params ?? {}))
}
