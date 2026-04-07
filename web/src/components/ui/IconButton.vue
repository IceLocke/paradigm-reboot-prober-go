<template>
  <button
    type="button"
    class="icon-btn"
    v-bind="buttonAttrs"
    @click="emit('click')"
  >
    <component
      v-if="icon"
      :is="icon"
      v-bind="iconProps"
    />
    <slot />
  </button>
</template>

<script setup lang="ts">
import { useAttrs } from 'vue'
import type { LucideIcon } from '@lucide/vue'

const { icon, ...iconProps } = defineProps<{
  title?: string
  icon?: LucideIcon
  size?: number
  color?: string
  strokeWidth?: number
  absoluteStrokeWidth?: boolean
}>()

const buttonAttrs = useAttrs()

const emit = defineEmits<{ click: [] }>()
</script>

<style scoped>
.icon-btn {
  position: relative;
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
  transition: background var(--transition-fast);
}
.icon-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
@media (hover: hover) {
  .icon-btn:hover { background: rgba(255,255,255,0.06); color: var(--text-primary); }
}
</style>
