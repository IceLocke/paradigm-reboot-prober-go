<template>
  <div class="stat-card">
    <p class="stat-label">{{ label }}</p>
    <p class="stat-value mono">{{ displayValue }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  label: string
  value: number
  precision?: number
}>()

const displayValue = computed(() => {
  const p = props.precision ?? 2
  return props.value.toFixed(p)
})
</script>

<style scoped>
.stat-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-1);
  padding: var(--space-4);
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 10px;
  min-width: 120px;
}
.stat-label {
  font-size: var(--text-xs);
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  font-weight: 500;
}
.stat-value {
  font-family: var(--font-mono);
  font-size: var(--text-2xl);
  font-weight: 600;
  color: var(--accent);
}
</style>
