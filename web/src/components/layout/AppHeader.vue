<template>
  <header class="app-header">
    <div class="header-left">
      <button class="menu-btn" @click="$emit('toggleSidebar')" v-if="isMobile">
        <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
          <path d="M3 5h14M3 10h14M3 15h14" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
        </svg>
      </button>
      <div class="logo">
        <span class="logo-text">Paradigm</span>
        <span class="logo-accent">Prober</span>
      </div>
    </div>
    <div class="header-right">
      <template v-if="userStore.logged_in">
        <span class="welcome-text">{{ t('message.welcome', { username: userStore.username }) }}</span>
        <button class="icon-btn" :title="t('auth.profile')" @click="$emit('showProfile')">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
        </button>
        <button class="icon-btn" :title="t('auth.logout')" @click="$emit('showLogout')">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><polyline points="16 17 21 12 16 7"/><line x1="21" y1="12" x2="9" y2="12"/></svg>
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
  gap: var(--space-2);
}
.logo-text {
  font-size: var(--text-lg);
  font-weight: 600;
  color: var(--text-primary);
}
.logo-accent {
  font-size: var(--text-lg);
  font-weight: 600;
  color: var(--accent);
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
