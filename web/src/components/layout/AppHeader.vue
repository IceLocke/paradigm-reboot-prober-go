<template>
  <header class="app-header">
    <div class="header-left">
      <button class="menu-btn" @click="$emit('toggleSidebar')" v-if="isMobile">
        <Menu :size="20" />
      </button>
      <div class="logo">
        <span class="logo-accent">Paradigm</span>
        <span class="logo-text">PROBER</span>
      </div>
    </div>
    <div class="header-right">
      <template v-if="userStore.logged_in">
        <span class="welcome-text">{{ t('message.welcome', { username: userStore.username }) }}</span>
        <button class="icon-btn" :title="t('auth.profile')" @click="$emit('showProfile')">
          <UserRound :size="20" />
        </button>
        <button class="icon-btn" :title="t('auth.logout')" @click="$emit('showLogout')">
          <LogOut :size="20" />
        </button>
      </template>
      <template v-else>
        <button class="btn btn--ghost btn--sm" @click="$emit('showLogin')">{{ t('auth.login') }}</button>
        <button class="btn btn--primary btn--sm" @click="$emit('showRegister')">{{ t('auth.register') }}</button>
      </template>
    </div>
  </header>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Menu, UserRound, LogOut } from '@lucide/vue';
import { useUserStore } from '@/stores/user'
import { useBreakpoint } from '@/composables/useBreakpoint'

const { t } = useI18n()
const userStore = useUserStore()
const { isMobile } = useBreakpoint()

defineEmits<{
  toggleSidebar: []
  showLogin: []
  showRegister: []
  showProfile: []
  showLogout: []
}>()
</script>

<style scoped>
.app-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 56px;
  padding: 0 var(--space-4);
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
  z-index: 100;
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
.menu-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: 8px;
}
@media (hover: hover) {
  .menu-btn:hover { background: rgba(255,255,255,0.06); color: var(--text-primary); }
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
.icon-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: 8px;
  transition: background var(--transition-fast), color var(--transition-fast);
}
@media (hover: hover) {
  .icon-btn:hover { background: rgba(255,255,255,0.06); color: var(--text-primary); }
}
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
  font-weight: 500;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  transition: background var(--transition-base);
  white-space: nowrap;
  font-family: inherit;
}
.btn--sm { padding: 6px 12px; font-size: 13px; min-height: 36px; }
.btn--primary { background: var(--accent); color: #fff; }
.btn--ghost { background: transparent; color: var(--text-secondary); }
@media (hover: hover) {
  .btn--primary:hover { background: var(--accent-hover); }
  .btn--ghost:hover { background: rgba(255,255,255,0.06); color: var(--text-primary); }
}
</style>
