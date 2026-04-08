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
        <span class="nav-icon">
          <component :is="item.icon" :size="20" />
        </span>
        <span class="nav-label">{{ item.label }}</span>
      </router-link>
    </nav>
  </aside>

  <!-- Mobile drawer -->
  <Teleport to="body">
    <Transition name="fade">
      <div v-if="!isDesktop && show" class="drawer-overlay" @click="$emit('update:modelValue', false)"></div>
    </Transition>
    <Transition name="slide">
      <div v-if="show" class="drawer" @click.stop>
        <div class="drawer-header">
          <span class="logo-accent">Paradigm</span>
          <span class="logo-text">PROBER</span>
        </div>
        <nav class="sidebar-nav">
          <router-link
            v-for="item in navItems"
            :key="item.path"
            :to="item.path"
            :class="['nav-item', { 'nav-item--active': $route.path === item.path }]"
            @click="$emit('update:modelValue', false)"
          >
            <span class="nav-icon">
              <component :is="item.icon" :size="20" />
            </span>
            <span class="nav-label">{{ item.label }}</span>
          </router-link>
        </nav>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ChartNoAxesColumn, Music2, FileText, Info } from '@lucide/vue';
import { useBreakpoint } from '@/composables/useBreakpoint'

const { t } = useI18n()
const { isDesktop } = useBreakpoint()

const show = defineModel<boolean>({ required: true })

const navItems = computed(() => [
  { path: '/best50', label: t('term.b50'), icon: ChartNoAxesColumn },
  { path: '/songs', label: t('term.charts'), icon: Music2 },
  { path: '/records', label: t('term.records'), icon: FileText },
  { path: '/about', label: t('common.about'), icon: Info },
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

/* Transitions */
.fade-enter-active, .fade-leave-active { transition: opacity var(--transition-base); }
.fade-enter-from, .fade-leave-to { opacity: 0; }

.slide-enter-active { transition: transform var(--transition-slow); }
.slide-leave-active { transition: transform var(--transition-base); }
.slide-enter-from { transform: translateX(-100%); }
.slide-leave-to { transform: translateX(-100%); }
</style>
