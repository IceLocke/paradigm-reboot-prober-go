import { defineStore } from 'pinia'
import type { User } from '@/api/types'

interface UserState {
  username: string
  access_token: string
  is_admin: boolean
  logged_in: boolean
  profile: User | null
}

export const useUserStore = defineStore('userStore', {
  state: (): UserState => ({
    username: '',
    access_token: '',
    is_admin: false,
    logged_in: false,
    profile: null,
  }),
  persist: true,
})
