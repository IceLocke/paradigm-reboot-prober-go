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
        <IconButton :icon="UserRound" :size="20" :title="t('auth.profile')" @click="$emit('showProfile')" />
        <IconButton :icon="LogOut" :size="20" :title="t('auth.logout')" @click="$emit('showLogout')" />
      </template>
      <template v-else>
        <BaseButton variant="ghost" size="sm" @click="$emit('showRegister')" :text="t('auth.register')" />
        <BaseButton size="sm" @click="$emit('showLogin')" :text="t('auth.login')" />
      </template>
    </div>
  </header>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Menu, UserRound, LogOut } from '@lucide/vue';
import { useUserStore } from '@/stores/user'
import { useBreakpoint } from '@/composables/useBreakpoint'
import BaseButton from '@/components/ui/BaseButton.vue'
import IconButton from '@/components/ui/IconButton.vue'

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
</style>
