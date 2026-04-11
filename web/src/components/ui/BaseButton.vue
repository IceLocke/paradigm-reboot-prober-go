<template>
  <button
    :class="['btn', `btn--${variant}`, `btn--${size}`, { 'btn--full': full }]"
    :disabled="disabled"
    @click="$emit('click')"
  >
    <slot>{{ text }}</slot>
  </button>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  text?: string
  variant?: 'primary' | 'secondary' | 'ghost' | 'danger'
  size?: 'sm' | 'md' | 'lg'
  disabled?: boolean
  full?: boolean
}>(), {
  variant: 'primary',
  size: 'md',
  disabled: false,
  full: false,
})

defineEmits<{ click: [] }>()
</script>

<style scoped>
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
  font-weight: 500;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  transition: background var(--transition-base), opacity var(--transition-base);
  white-space: nowrap;
  min-height: 44px;
  font-family: inherit;
  line-height: 1;
}
.btn:disabled { opacity: 0.4; cursor: not-allowed; }
.btn--sm { padding: 6px 12px; font-size: 13px; min-height: 36px; }
.btn--md { padding: 8px 16px; font-size: 14px; }
.btn--lg { padding: 10px 20px; font-size: 15px; }
@media (pointer: coarse) { .btn--sm { min-height: 44px; } }

.btn--primary { background: var(--accent); color: #fff; }
@media (hover: hover) { .btn--primary:hover:not(:disabled) { background: var(--accent-hover); } }

.btn--secondary { background: var(--bg-card); border: 1px solid var(--border); color: var(--text-primary); }
@media (hover: hover) { .btn--secondary:hover:not(:disabled) { border-color: var(--border-hover); background: var(--bg-tertiary); } }

.btn--ghost { background: transparent; color: var(--text-secondary); }
@media (hover: hover) { .btn--ghost:hover:not(:disabled) { background: rgba(255, 255, 255, 0.06); color: var(--text-primary); } }

.btn--danger { background: var(--color-danger); color: #fff; }
@media (hover: hover) { .btn--danger:hover:not(:disabled) { opacity: 0.85; } }

.btn--full { width: 100% }
</style>
