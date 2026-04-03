# 自定义组件规范

> 返回纲要 → [../vue-style-pattern.md](../vue-style-pattern.md)

以下组件由项目自行实现，保持对视觉风格的完全控制。
复杂交互组件（DataTable, Select, Modal 等）使用 Naive UI，见 [naive-ui.md](naive-ui.md)。

---

## BaseButton

```vue
<!-- BaseButton.vue -->
<template>
  <button
    :class="['btn', `btn--${variant}`, `btn--${size}`]"
    :disabled="disabled"
    @click="$emit('click')"
  >
    <slot />
  </button>
</template>

<script setup>
defineProps({
  variant: { type: String, default: 'primary' },   // primary | secondary | ghost | danger
  size: { type: String, default: 'md' },            // sm | md | lg
  disabled: { type: Boolean, default: false }
})

defineEmits(['click'])
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
  transition: background 0.2s, opacity 0.2s;
  white-space: nowrap;
  min-height: 44px; /* 移动端触摸目标 */
}

.btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

/* 尺寸 */
.btn--sm { padding: 6px 12px; font-size: 13px; min-height: 36px; }
.btn--md { padding: 8px 16px; font-size: 14px; }
.btn--lg { padding: 10px 20px; font-size: 15px; }

/* 移动端 sm 按钮也保证最小触摸区域 */
@media (pointer: coarse) {
  .btn--sm { min-height: 44px; }
}

/* Primary */
.btn--primary {
  background: var(--accent);
  color: #fff;
}
@media (hover: hover) {
  .btn--primary:hover:not(:disabled) {
    background: var(--accent-hover);
  }
}

/* Secondary */
.btn--secondary {
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-primary);
}
@media (hover: hover) {
  .btn--secondary:hover:not(:disabled) {
    border-color: var(--border-hover);
    background: var(--bg-tertiary);
  }
}

/* Ghost */
.btn--ghost {
  background: transparent;
  color: var(--text-secondary);
}
@media (hover: hover) {
  .btn--ghost:hover:not(:disabled) {
    background: rgba(255, 255, 255, 0.06);
    color: var(--text-primary);
  }
}

/* Danger */
.btn--danger {
  background: var(--color-danger);
  color: #fff;
}
@media (hover: hover) {
  .btn--danger:hover:not(:disabled) {
    opacity: 0.85;
  }
}
</style>
```

---

## BaseCard

```vue
<!-- BaseCard.vue -->
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

<script setup>
defineProps({
  hoverable: { type: Boolean, default: false }
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
  transition: border-color 0.2s;
  cursor: pointer;
}

@media (hover: hover) {
  .card--hoverable:hover {
    border-color: var(--border-hover);
  }
}

.card__header {
  padding: var(--space-4) var(--space-5);
  border-bottom: 1px solid var(--border);
}

.card__body {
  padding: var(--space-5);
}

/* 移动端缩减内边距 */
@media (max-width: 639px) {
  .card__header { padding: var(--space-3) var(--space-4); }
  .card__body   { padding: var(--space-4); }
}
</style>
```

---

## BaseTabs

```vue
<!-- BaseTabs.vue -->
<template>
  <div class="tabs">
    <div class="tabs__header">
      <button
        v-for="tab in tabs"
        :key="tab.key"
        :class="['tabs__tab', { 'tabs__tab--active': modelValue === tab.key }]"
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

<script setup>
defineProps({
  tabs: { type: Array, required: true },
  modelValue: { type: String, required: true }
})
defineEmits(['update:modelValue'])
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
  overflow-x: auto;       /* 移动端标签过多时可横滑 */
  -webkit-overflow-scrolling: touch;
}

/* 隐藏横向滚动条 */
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
  transition: color 0.2s, background 0.2s;
  white-space: nowrap;
  min-height: 44px;       /* 触摸目标 */
}

.tabs__tab:hover {
  color: var(--text-secondary);
}

.tabs__tab--active {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}

.tabs__content {
  margin-top: var(--space-5);
}
</style>
```

---

## BaseInput

```vue
<!-- BaseInput.vue -->
<template>
  <div class="input-group">
    <label v-if="label" class="input-label">{{ label }}</label>
    <input
      class="input"
      :type="type"
      :value="modelValue"
      :placeholder="placeholder"
      @input="$emit('update:modelValue', $event.target.value)"
    />
  </div>
</template>

<script setup>
defineProps({
  modelValue: { type: [String, Number], default: '' },
  label: { type: String, default: '' },
  type: { type: String, default: 'text' },
  placeholder: { type: String, default: '' }
})
defineEmits(['update:modelValue'])
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
  font-size: 16px;          /* ≥16px 防止 iOS 缩放 */
  outline: none;
  transition: border-color 0.2s;
  min-height: 44px;          /* 触摸目标 */
}

.input::placeholder {
  color: var(--text-muted);
}

.input:focus {
  border-color: var(--accent);
}
</style>
```

---

## SongListItem（业务组件示例）

```vue
<!-- SongListItem.vue -->
<template>
  <div :class="['song-item', { 'song-item--active': active }]"
       @click="$emit('select')">
    <div class="song-item__cover">
      <img :src="song.cover" :alt="song.name" loading="lazy" />
    </div>
    <div class="song-item__info">
      <p class="song-item__name">{{ song.name }}</p>
      <span class="song-item__artist">{{ song.artist }}</span>
    </div>
    <div class="song-item__badges">
      <span v-for="diff in song.difficulties"
            :key="diff.type"
            :class="['diff-badge', `diff-badge--${diff.type}`]">
        {{ diff.level }}
      </span>
    </div>
  </div>
</template>

<script setup>
defineProps({
  song: { type: Object, required: true },
  active: { type: Boolean, default: false }
})
defineEmits(['select'])
</script>

<style scoped>
.song-item {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.15s;
  min-height: 44px;
}

@media (hover: hover) {
  .song-item:hover {
    background: rgba(255, 255, 255, 0.04);
  }
}

.song-item--active {
  background: var(--accent-muted);
}

.song-item__cover {
  width: 48px;
  height: 48px;
  border-radius: 6px;
  overflow: hidden;
  flex-shrink: 0;
}

.song-item__cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.song-item__info {
  flex: 1;
  min-width: 0;
}

.song-item__name {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
  margin: 0 0 2px 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.song-item__artist {
  font-size: 12px;
  color: var(--text-muted);
}

.song-item__badges {
  display: flex;
  gap: 6px;
  flex-shrink: 0;
}

/* 移动端隐藏 badge，节省空间 */
@media (max-width: 479px) {
  .song-item__badges { display: none; }
}

.diff-badge {
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 600;
  font-family: var(--font-mono);
}

.diff-badge--detected { background: rgba(34, 197, 94, 0.15); color: var(--diff-detected); }
.diff-badge--invaded  { background: rgba(234, 179, 8, 0.15); color: var(--diff-invaded); }
.diff-badge--massive  { background: rgba(249, 115, 22, 0.15); color: var(--diff-massive); }
.diff-badge--reboot   { background: rgba(236, 72, 153, 0.15); color: var(--diff-reboot); }
</style>
```
