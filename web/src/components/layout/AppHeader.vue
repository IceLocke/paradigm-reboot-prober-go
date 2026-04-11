<template>
  <header class="app-header">
    <div class="header-left">
      <IconButton v-if="isMobile" :icon="Menu" :size="20" @click="$emit('toggleSidebar')" />
      <div class="logo">
        <span class="logo-accent">Paradigm</span>
        <span class="logo-text">PROBER</span>
      </div>
    </div>
    <div class="header-right">
      <template v-if="userStore.logged_in">
        <span class="welcome-text">{{ t('message.welcome', { username: userStore.username }) }}</span>
        <UploadCart />
        <n-popover
          trigger="manual"
          placement="bottom-end"
          :show="showDropdown"
          :show-arrow="false"
          style="padding: var(--space-1) var(--space-2);"
          @clickoutside="(($event.target as HTMLElement)?.id !== 'showDropdown') && (showDropdown = false)"
        >
          <template #trigger>
            <IconButton
              id="showDropdown"
              :icon="UserRound"
              :size="20"
              :title="t('auth.profile')"
              @click="showDropdown = !showDropdown"
            />
          </template>
          <div class="dropdown-list">
            <div
              v-for="item in profileItems"
              :key="item.key"
              class="dropdown-item"
              @click="() => (showDropdown = false, item.onClick())"
            >
              <component :is="item.icon" :size="16" @click="item.onClick" />
              <span>{{ item.label }}</span>
            </div>
          </div>
        </n-popover>
      </template>
      <template v-else>
        <BaseButton variant="ghost" size="sm" @click="$emit('showRegister')" :text="t('auth.register')" />
        <BaseButton size="sm" @click="$emit('showLogin')" :text="t('auth.login')" />
      </template>
    </div>
  </header>
  <LogoutModal v-model:show="showLogout" />
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { NPopover } from 'naive-ui'
import { Menu, UserRound, FilePenLine, LogOut } from '@lucide/vue'
import { useUserStore } from '@/stores/user'
import { useBreakpoint } from '@/composables/useBreakpoint'
import BaseButton from '@/components/ui/BaseButton.vue'
import IconButton from '@/components/ui/IconButton.vue'
import UploadCart from '@/components/business/UploadCart.vue'
import LogoutModal from '@/components/business/LogoutModal.vue'

const router = useRouter()
const { t } = useI18n()
const userStore = useUserStore()
const { isMobile } = useBreakpoint()

const showDropdown = ref(false)
const showLogout = ref(false)

defineEmits<{
  toggleSidebar: []
  showLogin: []
  showRegister: []
}>()

const profileItems = computed(() => [
  {
    key: 'profile',
    label: t('auth.profile'),
    icon: FilePenLine,
    onClick: () => router.push('/profile')
  },
  {
    key: 'logout',
    label: t('auth.logout'),
    icon: LogOut,
    onClick: () => showLogout.value = true
  },
])
</script>

<style scoped>
.app-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: var(--app-header-height);
  padding: 0 var(--space-4);
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border);
  width: 100%;
  z-index: 100;
  position: fixed;
  top: 0;
}
@media (min-width: 768px) {
  .app-header { padding: 0 var(--space-6); }
}
.header-left {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}
.header-right {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}
.logo {
  display: flex;
  align-items: baseline;
  gap: 6px;
  user-select: none;
}
.logo-text {
  font-size: var(--text-base);
  font-weight: 700;
  color: var(--text-primary);
  letter-spacing: 0.1em;
}
.logo-accent {
  font-size: var(--text-lg);
  font-weight: 600;
  color: var(--accent);
  text-shadow: 0 0 12px rgba(59, 130, 246, 0.35);
}
.welcome-text {
  font-size: var(--text-sm);
  color: var(--text-secondary);
  display: none;
}
@media (min-width: 640px) {
  .welcome-text { display: inline; }
}
.dropdown-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}
.dropdown-item {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-2);
  border-radius: 8px;
  cursor: pointer;
  transition: background var(--transition-fast), color var(--transition-fast);
}
@media (hover: hover) {
  .dropdown-item:hover {
    background: rgba(255, 255, 255, 0.06);
    color: var(--text-primary);
  }
}
</style>
