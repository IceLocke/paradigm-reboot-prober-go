import type {AxiosInstance, InternalAxiosRequestConfig} from 'axios'
import axios from 'axios'
import pako from 'pako'

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

// Response interceptor: handle 401
client.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // Token expired - the component should handle this
    }
    return Promise.reject(error)
  }
)

export default client
export { API_BASE }
