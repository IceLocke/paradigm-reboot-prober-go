<template>
  <div class="input-group">
    <label v-if="label" class="input-label">{{ label }}</label>
    <input
      class="input"
      :type="type"
      :value="modelValue"
      :placeholder="placeholder"
      :readonly="readonly"
      @input="$emit('update:modelValue', ($event.target as HTMLInputElement).value)"
    />
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  modelValue?: string | number
  label?: string
  type?: string
  placeholder?: string
  readonly?: boolean
}>(), {
  modelValue: '',
  label: '',
  type: 'text',
  placeholder: '',
  readonly: false,
})

defineEmits<{ 'update:modelValue': [value: string] }>()
</script>

<style scoped>
.input-group {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}
.input-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary);
}
.input {
  padding: 8px 12px;
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 8px;
  color: var(--text-primary);
  font-size: 16px;
  outline: none;
  transition: border-color var(--transition-base);
  min-height: 44px;
  font-family: inherit;
}
.input::placeholder {
  color: var(--text-muted);
}
.input:focus {
  border-color: var(--accent);
}
.input[readonly] {
  opacity: 0.7;
  cursor: default;
}
</style>
