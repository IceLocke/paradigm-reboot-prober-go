import { defineStore } from 'pinia'
import type { ChartInfo } from '@/api/types'

interface UploadItem {
  title: string
  difficulty: string
  level: number
  chart_id: number
  score?: number
  new_score?: number
}

interface AppState {
  charts: ChartInfo[] | null
  uploadList: UploadItem[]
  sidebarOpen: boolean
  dismissedVersion: string
  songsViewMode: 'grid' | 'table'
  b50ChartIds: number[] | null
}

export const useAppStore = defineStore('appStore', {
  state: (): AppState => ({
    charts: null,
    uploadList: [],
    sidebarOpen: false,
    dismissedVersion: '',
    songsViewMode: 'grid',
    b50ChartIds: null,
  }),
  persist: {
    paths: ['dismissedVersion', 'songsViewMode'],
  },
})
