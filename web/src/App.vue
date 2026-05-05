<template>
  <n-config-provider :theme="darkTheme" :theme-overrides="themeOverrides">
    <div :class="['app-layout', { 'app-layout--desktop': isDesktop }]">
      <AppHeader
        @toggle-sidebar="appStore.sidebarOpen = !appStore.sidebarOpen"
        @show-login="showLogin = true"
        @show-register="showRegister = true"
      />
      <Sidebar v-model="appStore.sidebarOpen" />
      <main class="app-main">
        <router-view />
      </main>
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
  </n-config-provider>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { darkTheme, NConfigProvider } from 'naive-ui'

import { themeOverrides } from '@/config/naive-theme'
import { useUserStore } from '@/stores/user'
import { useAppStore } from '@/stores/app'
import { getMyInfo } from '@/api/user'
import { getAllCharts } from '@/api/song'
import { USE_MOCK, getMockCharts, getMockUser } from '@/api/mock'
import { useBreakpoint } from '@/composables/useBreakpoint'

import AppHeader from '@/components/layout/AppHeader.vue'
import Sidebar from '@/components/layout/Sidebar.vue'
import LoginModal from '@/components/business/LoginModal.vue'
import RegisterModal from '@/components/business/RegisterModal.vue'

const userStore = useUserStore()
const appStore = useAppStore()
const { isDesktop } = useBreakpoint()

const showLogin = ref(false)
const showRegister = ref(false)

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

  // Stale-while-revalidate: show cached data immediately, then refresh in background
  const hasCache = appStore.charts !== null && appStore.chartsETag !== null

  try {
    const res = await getAllCharts(appStore.chartsETag ?? undefined)

    if (res.status === 304) {
      // Cache still valid — nothing to do
      return
    }

    // 200 OK — update store and cache
    appStore.charts = res.data
    appStore.chartsETag = res.headers.etag ?? null
  } catch {
    // On network error, keep stale cache so the UI stays usable
    if (!hasCache) {
      appStore.charts = null
    }
  }
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
  display: grid;
  grid-template-areas:
    "header"
    "content";
  grid-template-columns: 1fr;
  grid-template-rows: var(--app-header-height) 1fr;
  height: 100vh;
}
.app-layout--desktop {
  grid-template-areas:
    "header header"
    "sidebar content";
  grid-template-columns: var(--app-sidebar-width) 1fr;
}
:deep(.app-header) {
  grid-area: header;
}
:deep(.app-sidebar) {
  grid-area: sidebar;
}
.app-main {
  grid-area: content;
  overflow-x: hidden;
}
</style>
