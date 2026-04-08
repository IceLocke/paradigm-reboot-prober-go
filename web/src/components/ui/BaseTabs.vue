<template>
  <div class="tabs">
    <div class="tabs__header">
      <button
        v-for="tab in tabs"
        :key="tab.key"
        :class="['tabs__tab', { 'tabs__tab--active': active === tab.key }]"
        @click="$emit('update:modelValue', tab.key)"
      >
        {{ tab.label }}
      </button>
    </div>
    <div class="tabs__content">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
const active = defineModel<string>({ required: true })

interface Tab {
  key: string
  label: string
}

defineProps<{
  tabs: Tab[]
}>()
</script>

<style scoped>
.tabs__header {
  display: flex;
  gap: 2px;
  padding: 3px;
  background: var(--bg-secondary);
  border-radius: 8px;
  width: fit-content;
  max-width: 100%;
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
}
.tabs__header::-webkit-scrollbar { display: none; }
.tabs__header { scrollbar-width: none; }

.tabs__tab {
  padding: 7px 16px;
  background: transparent;
  border: none;
  color: var(--text-muted);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  border-radius: 6px;
  transition: color var(--transition-base), background var(--transition-base);
  white-space: nowrap;
  min-height: 44px;
  font-family: inherit;
}
@media (hover: hover) {
  .tabs__tab:hover { color: var(--text-secondary); }
}
.tabs__tab--active {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}
.tabs__content {
  margin-top: var(--space-5);
}
</style>
