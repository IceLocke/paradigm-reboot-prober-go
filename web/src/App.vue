<template>
  <Analytics/>
  <SpeedInsights />
  <n-config-provider :theme="darkTheme" :theme-overrides="themeOverrides">
    <n-message-provider>
      <n-notification-provider>
        <div class="app-layout">
          <AppHeader
            @toggle-sidebar="appStore.sidebarOpen = !appStore.sidebarOpen"
            @show-login="showLogin = true"
            @show-register="showRegister = true"
            @show-profile="showProfile = true"
            @show-logout="showLogout = true"
          />
          <div class="app-body">
            <Sidebar v-model="appStore.sidebarOpen" />
            <main class="app-main">
              <router-view />
            </main>
          </div>
        </div>

        <!-- Auth Modals -->
        <LoginModal
          v-model:show="showLogin"
          @success="onLoginSuccess"
          @go-register="showLogin = false; showRegister = true"
        />
        <RegisterModal
          v-model:show="showRegister"
          @success="showRegister = false; showLogin = true"
          @go-login="showRegister = false; showLogin = true"
        />
        <LogoutModal v-model:show="showLogout" />
        <ProfileModal
          v-model:show="showProfile"
        />
      </n-notification-provider>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { darkTheme, NConfigProvider, NMessageProvider, NNotificationProvider } from 'naive-ui'

import { themeOverrides } from '@/config/naive-theme'
import { useUserStore } from '@/stores/user'
import { useAppStore } from '@/stores/app'
import { getMyInfo } from '@/api/user'
import { getAllCharts } from '@/api/song'
import { USE_MOCK, getMockCharts, getMockUser } from '@/api/mock'
import { SpeedInsights } from '@vercel/speed-insights/vue';
import { Analytics } from "@vercel/analytics/vue"

import AppHeader from '@/components/layout/AppHeader.vue'
import Sidebar from '@/components/layout/Sidebar.vue'
import LoginModal from '@/components/business/LoginModal.vue'
import RegisterModal from '@/components/business/RegisterModal.vue'
import LogoutModal from '@/components/business/LogoutModal.vue'
import ProfileModal from '@/components/business/ProfileModal.vue'

const userStore = useUserStore()
const appStore = useAppStore()

const showLogin = ref(false)
const showLogout = ref(false)
const showRegister = ref(false)
const showProfile = ref(false)

const onLoginSuccess = () => {
  showLogin.value = false
  loadUserInfo()
}

const loadUserInfo = async () => {
  if (USE_MOCK) {
    const mockUser = getMockUser()
    userStore.$patch({
      profile: mockUser,
      username: mockUser.username,
      logged_in: true,
      is_admin: mockUser.is_admin,
    })
    return
  }
  try {
    const res = await getMyInfo()
    userStore.$patch({
      profile: res.data,
      username: res.data.username,
      logged_in: true,
      is_admin: res.data.is_admin,
    })
  } catch {
    if (userStore.logged_in) {
      userStore.$reset()
    }
  }
}

const loadCharts = async () => {
  if (USE_MOCK) {
    appStore.charts = getMockCharts()
    return
  }
  try {
    const res = await getAllCharts()
    appStore.charts = res.data
  } catch { /* handled */ }
}

onMounted(() => {
  // Try to restore session
  if (userStore.access_token) {
    loadUserInfo()
  }
  // Load charts data
  loadCharts()
})
</script>

<style scoped>
.app-layout {
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
}
.app-body {
  display: flex;
  flex: 1;
  overflow: hidden;
}
.app-main {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
}
</style>
