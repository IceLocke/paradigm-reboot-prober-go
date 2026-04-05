import { defineStore } from 'pinia'
import type { ChartInfo } from '@/api/types'

interface UploadItem {
  title: string
  difficulty: string
  level: number
  chart_id: number
  score?: number
}

interface AppState {
  charts: ChartInfo[] | null
  uploadList: UploadItem[]
  sidebarOpen: boolean
}

export const useAppStore = defineStore('appStore', {
  state: (): AppState => ({
    charts: null,
    uploadList: [],
    sidebarOpen: false,
  }),
})
