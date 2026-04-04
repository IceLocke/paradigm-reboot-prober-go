<template>
  <span :class="['diff-badge', `diff-badge--${difficulty}`]">
    {{ label }}
  </span>
</template>

<script setup lang="ts">
import type { Difficulty } from '@/api/types'

const props = withDefaults(defineProps<{
  difficulty: Difficulty
  level?: number | null
  short?: boolean
}>(), {
  level: null,
  short: false,
})

const diffNames: Record<Difficulty, string> = {
  detected: 'DT',
  invaded: 'IN',
  massive: 'MS',
  reboot: 'RB',
}

const diffFullNames: Record<Difficulty, string> = {
  detected: 'Detected',
  invaded: 'Invaded',
  massive: 'Massive',
  reboot: 'Reboot',
}

const label = props.short
  ? (props.level != null ? `${diffNames[props.difficulty]} ${props.level}` : diffNames[props.difficulty])
  : (props.level != null ? `${diffFullNames[props.difficulty]} ${props.level}` : diffFullNames[props.difficulty])
</script>

<style scoped>
.diff-badge {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 600;
  font-family: var(--font-mono);
  white-space: nowrap;
}
.diff-badge--detected { background: rgba(34, 197, 94, 0.15); color: var(--diff-detected); }
.diff-badge--invaded { background: rgba(234, 179, 8, 0.15); color: var(--diff-invaded); }
.diff-badge--massive { background: rgba(249, 115, 22, 0.15); color: var(--diff-massive); }
.diff-badge--reboot { background: rgba(236, 72, 153, 0.15); color: var(--diff-reboot); }
</style>
