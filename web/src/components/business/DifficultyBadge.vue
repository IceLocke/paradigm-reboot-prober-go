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
  detected: 'DET',
  invaded: 'IVD',
  massive: 'MSV',
  reboot: 'RBT',
}

const diffFullNames: Record<Difficulty, string> = {
  detected: 'DETECTED',
  invaded: 'INVADED',
  massive: 'MASSIVE',
  reboot: 'REBOOT',
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
.diff-badge--detected { background: rgba(59, 130, 246, 0.15); color: var(--diff-detected); }
.diff-badge--invaded  { background: rgba(239, 68, 68, 0.15);  color: var(--diff-invaded); }
.diff-badge--massive  { background: rgba(168, 85, 247, 0.15); color: var(--diff-massive); }
.diff-badge--reboot   { background: rgba(249, 115, 22, 0.15); color: var(--diff-reboot); }
</style>
