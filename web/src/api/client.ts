import type {AxiosInstance, InternalAxiosRequestConfig} from 'axios'
import axios from 'axios'
import pako from 'pako'
import { toastWarning } from '@/utils/toast'
import type { Token, RefreshTokenRequest } from './types'

const API_BASE = import.meta.env.VITE_API_ENDPOINT || '/api/v2'

/**
 * Minimum request body size (in bytes) before gzip compression kicks in.
 * Bodies smaller than this threshold are sent as-is because the compression
 * overhead would outweigh the bandwidth savings.
 */
const GZIP_THRESHOLD = 1024

const client: AxiosInstance = axios.create({
  baseURL: API_BASE,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor: attach JWT token
client.interceptors.request.use((config) => {
  const raw = localStorage.getItem('userStore')
  if (raw) {
    try {
      const store = JSON.parse(raw)
      if (store.access_token) {
        config.headers.Authorization = store.access_token
      }
    } catch {
      // ignore parse errors
    }
  }
  return config
})

// Request interceptor: gzip-compress large request bodies
client.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  if (!config.data) return config

  // Only compress JSON payloads (objects / arrays that axios will serialize)
  const isJsonPayload =
    typeof config.data === 'object' &&
    !(config.data instanceof FormData) &&
    !(config.data instanceof Blob) &&
    !(config.data instanceof ArrayBuffer)

  if (!isJsonPayload) return config

  const json = JSON.stringify(config.data)
  const encoded = new TextEncoder().encode(json)
  if (encoded.byteLength < GZIP_THRESHOLD) return config

  config.data = pako.gzip(encoded)
  config.headers['Content-Encoding'] = 'gzip'
  config.headers['Content-Type'] = 'application/json'

  return config
})

// ─── Token refresh logic ───────────────────────────────────────────
// Prevents multiple concurrent refresh calls: when a 401 triggers a
// refresh, subsequent 401s queue behind the same promise.

let isRefreshing = false
let refreshSubscribers: Array<(token: string) => void> = []

function subscribeTokenRefresh(cb: (token: string) => void) {
  refreshSubscribers.push(cb)
}

function onTokenRefreshed(token: string) {
  refreshSubscribers.forEach((cb) => cb(token))
  refreshSubscribers = []
}

// Endpoints where 401 means "wrong credentials", not "token expired"
const AUTH_401_SKIP_PATHS = ['/user/login', '/user/me/password', '/user/refresh']

// Response interceptor: handle 401 with automatic token refresh
client.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config
    if (!error.response || error.response.status !== 401) {
      return Promise.reject(error)
    }

    const reqUrl: string = originalRequest?.url ?? ''
    const isCredentialCheck = AUTH_401_SKIP_PATHS.some((p) => reqUrl.endsWith(p))

    // For credential-check endpoints, just reject without refresh attempt
    if (isCredentialCheck) {
      return Promise.reject(error)
    }

    // Check if we have a refresh token
    let storedRefreshToken = ''
    try {
      const raw = localStorage.getItem('userStore')
      if (raw) {
        const store = JSON.parse(raw)
        storedRefreshToken = store.refresh_token || ''
      }
    } catch { /* ignore */ }

    if (!storedRefreshToken) {
      // No refresh token — show expiry toast and clear session
      showExpiredToastAndClear()
      return Promise.reject(error)
    }

    // If already refreshing, queue this request
    if (isRefreshing) {
      return new Promise((resolve) => {
        subscribeTokenRefresh((newAccessToken: string) => {
          originalRequest.headers.Authorization = newAccessToken
          resolve(client(originalRequest))
        })
      })
    }

    isRefreshing = true

    try {
      // Call refresh endpoint directly with axios to avoid interceptor loops
      const res = await axios.post<Token>(`${API_BASE}/user/refresh`, {
        refresh_token: storedRefreshToken,
      } satisfies RefreshTokenRequest, {
        headers: { 'Content-Type': 'application/json' },
      })

      const newAccessToken = `Bearer ${res.data.access_token}`
      const newRefreshToken = res.data.refresh_token

      // Update localStorage
      try {
        const raw = localStorage.getItem('userStore')
        if (raw) {
          const store = JSON.parse(raw)
          store.access_token = newAccessToken
          store.refresh_token = newRefreshToken
          localStorage.setItem('userStore', JSON.stringify(store))
        }
      } catch { /* ignore */ }

      isRefreshing = false
      onTokenRefreshed(newAccessToken)

      // Retry the original request with new token
      originalRequest.headers.Authorization = newAccessToken
      return client(originalRequest)
    } catch {
      // Refresh failed — clear session and notify
      isRefreshing = false
      refreshSubscribers = []
      showExpiredToastAndClear()
      return Promise.reject(error)
    }
  }
)

function showExpiredToastAndClear() {
  try {
    const raw = localStorage.getItem('userStore')
    if (raw) {
      const store = JSON.parse(raw)
      if (store.access_token) {
        toastWarning('message.token_expired')
      }
    }
  } catch { /* ignore */ }

  // Clear auth state in localStorage
  try {
    const raw = localStorage.getItem('userStore')
    if (raw) {
      const store = JSON.parse(raw)
      store.access_token = ''
      store.refresh_token = ''
      store.logged_in = false
      store.username = ''
      store.is_admin = false
      store.profile = null
      localStorage.setItem('userStore', JSON.stringify(store))
    }
  } catch { /* ignore */ }
}

export default client
export { API_BASE }
