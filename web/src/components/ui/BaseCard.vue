<template>
  <div :class="['card', { 'card--hoverable': hoverable }]">
    <div v-if="$slots.header" class="card__header">
      <slot name="header" />
    </div>
    <div class="card__body">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  hoverable?: boolean
}>(), {
  hoverable: false,
})
</script>

<style scoped>
.card {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 10px;
  overflow: hidden;
}
.card--hoverable {
  transition: border-color var(--transition-base);
  cursor: pointer;
}
@media (hover: hover) {
  .card--hoverable:hover { border-color: var(--border-hover); }
}
.card__header {
  padding: var(--space-4) var(--space-5);
  border-bottom: 1px solid var(--border);
}
.card__body {
  padding: var(--space-5);
}
@media (max-width: 639px) {
  .card__header { padding: var(--space-3) var(--space-4); }
  .card__body { padding: var(--space-4); }
}
</style>
