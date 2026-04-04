import axios from 'axios'
import type { AxiosInstance } from 'axios'

const API_BASE = import.meta.env.VITE_API_ENDPOINT || '/api/v2'

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
