<template>
  <!-- Desktop sidebar -->
  <aside v-if="isDesktop" class="sidebar">
    <nav class="sidebar-nav">
      <router-link
        v-for="item in navItems"
        :key="item.path"
        :to="item.path"
        :class="['nav-item', { 'nav-item--active': $route.path === item.path }]"
      >
        <span class="nav-icon" v-html="item.icon"></span>
        <span class="nav-label">{{ item.label }}</span>
      </router-link>
    </nav>
  </aside>

  <!-- Mobile drawer -->
  <Teleport to="body">
    <Transition name="fade">
      <div v-if="!isDesktop && modelValue" class="drawer-overlay" @click="$emit('update:modelValue', false)">
        <Transition name="slide">
          <div v-if="modelValue" class="drawer" @click.stop>
            <div class="drawer-header">
              <span class="logo-text">Paradigm</span>
              <span class="logo-accent">Prober</span>
            </div>
            <nav class="sidebar-nav">
              <router-link
                v-for="item in navItems"
                :key="item.path"
                :to="item.path"
                :class="['nav-item', { 'nav-item--active': $route.path === item.path }]"
                @click="$emit('update:modelValue', false)"
              >
                <span class="nav-icon" v-html="item.icon"></span>
                <span class="nav-label">{{ item.label }}</span>
              </router-link>
            </nav>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useBreakpoint } from '@/composables/useBreakpoint'

const { t } = useI18n()
const { isDesktop } = useBreakpoint()

defineProps<{ modelValue: boolean }>()
defineEmits<{ 'update:modelValue': [value: boolean] }>()

const icons = {
  b50: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12V7H5a2 2 0 0 1 0-4h14v4"/><path d="M3 5v14a2 2 0 0 0 2 2h16v-5"/><path d="M18 12a2 2 0 0 0 0 4h4v-4Z"/></svg>',
  songs: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="8" cy="18" r="4"/><path d="M12 18V2l7 4"/></svg>',
  records: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8Z"/><path d="M14 2v6h6"/><path d="M16 13H8"/><path d="M16 17H8"/><path d="M10 9H8"/></svg>',
  about: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M12 16v-4"/><path d="M12 8h.01"/></svg>',
}

const navItems = computed(() => [
  { path: '/best50', label: t('term.b50'), icon: icons.b50 },
  { path: '/songs', label: t('term.song_levels'), icon: icons.songs },
  { path: '/records', label: t('term.records'), icon: icons.records },
  { path: '/about', label: t('common.about'), icon: icons.about },
])
</script>

<style scoped>
.sidebar {
  width: 220px;
  background: var(--bg-secondary);
  border-right: 1px solid var(--border);
  padding: var(--space-3);
  flex-shrink: 0;
  overflow-y: auto;
}
.sidebar-nav {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.nav-item {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  border-radius: 8px;
  color: var(--text-secondary);
  text-decoration: none;
  font-size: var(--text-base);
  font-weight: 500;
  transition: background var(--transition-fast), color var(--transition-fast);
  min-height: 44px;
}
@media (hover: hover) {
  .nav-item:hover {
    background: rgba(255, 255, 255, 0.04);
    color: var(--text-primary);
  }
}
.nav-item--active {
  background: var(--accent-muted);
  color: var(--accent);
}
.nav-icon {
  display: flex;
  align-items: center;
  flex-shrink: 0;
}
.nav-label {
  white-space: nowrap;
}

/* Drawer overlay */
.drawer-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  z-index: 200;
}
.drawer {
  position: fixed;
  top: 0;
  left: 0;
  bottom: 0;
  width: 280px;
  max-width: 80vw;
  background: var(--bg-secondary);
  border-right: 1px solid var(--border);
  padding: var(--space-3);
  z-index: 201;
  overflow-y: auto;
}
.drawer-header {
  display: flex;
  align-items: baseline;
  gap: var(--space-2);
  padding: var(--space-4);
  margin-bottom: var(--space-3);
}
.logo-text { font-size: var(--text-lg); font-weight: 600; color: var(--text-primary); }
.logo-accent { font-size: var(--text-lg); font-weight: 600; color: var(--accent); }

/* Transitions */
.fade-enter-active, .fade-leave-active { transition: opacity var(--transition-base); }
.fade-enter-from, .fade-leave-to { opacity: 0; }

.slide-enter-active { transition: transform var(--transition-slow); }
.slide-leave-active { transition: transform var(--transition-base); }
.slide-enter-from { transform: translateX(-100%); }
.slide-leave-to { transform: translateX(-100%); }
</style>
