# 响应式与移动端适配

> 返回纲要 → [../vue-style-pattern.md](../vue-style-pattern.md)

---

## 断点定义

采用 **Mobile-first** 策略：默认样式面向移动端，通过 `min-width` 逐级向上扩展。

```css
/* 移动端（默认） → 无需媒体查询 */

/* 大屏手机 / 小平板 */
@media (min-width: 640px)  { /* sm */ }

/* 平板竖屏 */
@media (min-width: 768px)  { /* md - 侧边栏出现 */ }

/* 平板横屏 / 小笔记本 */
@media (min-width: 1024px) { /* lg */ }

/* 桌面 */
@media (min-width: 1280px) { /* xl */ }
```

---

## Viewport 配置

```html
<meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
```

- `viewport-fit=cover`：配合 `safe-area-inset-*` 支持刘海屏和底部横条。

---

## 移动端核心约束

### 1. 触摸目标

所有可交互元素（按钮、链接、列表项、表单控件）最小尺寸 **44×44px**。

```css
.interactive-element {
  min-height: 44px;
  min-width: 44px;
}
```

对于视觉上更小的元素（如图标按钮），使用 `padding` 或透明 `::before` 伪元素扩大点击区域。

### 2. 表单输入防缩放

iOS Safari 在 `font-size < 16px` 的输入框获得焦点时会自动缩放页面。

```css
input, select, textarea {
  font-size: 16px; /* 必须 ≥ 16px */
}
```

### 3. 隔离 Hover 效果

触摸设备上 hover 会"粘滞"（点击后保持 hover 态）。用 `@media (hover: hover)` 隔离：

```css
/* ✅ 正确：只在支持 hover 的设备上应用 */
@media (hover: hover) {
  .card:hover {
    border-color: var(--border-hover);
  }
}

/* ❌ 错误：移动端点击后会一直保持 hover 背景 */
.card:hover {
  border-color: var(--border-hover);
}
```

### 4. 安全区域

底部固定栏（操作按钮、导航栏）必须处理安全区域：

```css
.bottom-bar {
  padding-bottom: calc(var(--space-3) + env(safe-area-inset-bottom));
}
```

### 5. 禁止意外横滑

页面不允许出现横向溢出。在根容器上设置：

```css
html, body {
  overflow-x: hidden;
}
```

表格等确实需要横滑的内容，在**内层容器**上单独开启 `overflow-x: auto`。

### 6. 文字最小字号

移动端可读性底线：正文 ≥ 14px，辅助文字 ≥ 12px。不使用小于 12px 的字号。

---

## 布局适配策略

### 侧边栏

| 屏幕 | 行为 |
|------|------|
| `≥ 768px` | 常驻侧边栏，宽 240px |
| `< 768px` | 隐藏侧边栏，通过汉堡菜单触发抽屉覆盖层 |

```vue
<!-- Sidebar.vue 适配思路 -->
<template>
  <!-- 桌面：常驻 -->
  <aside v-if="isDesktop" class="sidebar">
    <SidebarContent />
  </aside>

  <!-- 移动端：抽屉 -->
  <n-drawer v-else v-model:show="drawerVisible" placement="left" :width="280">
    <n-drawer-content>
      <SidebarContent />
    </n-drawer-content>
  </n-drawer>
</template>

<script setup>
import { useBreakpoint } from '@/composables/useBreakpoint'
const { isDesktop } = useBreakpoint() // 基于 768px 判断
const drawerVisible = ref(false)
</script>
```

### 主布局

```css
/* 默认：单列 */
.main-layout {
  display: flex;
  flex-direction: column;
  padding: var(--space-3);
  gap: var(--space-3);
}

/* md 及以上：侧边栏 + 内容 */
@media (min-width: 768px) {
  .main-layout {
    flex-direction: row;
    padding: var(--space-5);
    gap: var(--space-5);
  }
}
```

### 弹窗 / Modal

| 屏幕 | 行为 |
|------|------|
| `≥ 640px` | 居中弹窗，`max-width: 480px` |
| `< 640px` | 接近全屏，`width: 100%; max-height: 90vh; border-radius: 12px 12px 0 0` 从底部弹出 |

```css
.modal-content {
  width: 90vw;
  max-width: 480px;
}

@media (max-width: 639px) {
  .modal-content {
    width: 100%;
    max-width: none;
    max-height: 90vh;
    border-radius: 12px 12px 0 0;
  }
}
```

---

## 表格移动端策略

DataTable 在窄屏下的处理方式：

```vue
<template>
  <div class="table-wrapper">
    <n-data-table
      :columns="columns"
      :data="data"
      :scroll-x="600"
      :bordered="false"
      size="small"
    />
  </div>
</template>

<style scoped>
.table-wrapper {
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
  margin: 0 calc(-1 * var(--space-3)); /* 出血到容器边缘 */
  padding: 0 var(--space-3);
}
</style>
```

要点：
- 设置 `:scroll-x` 为所有列最小宽度之和，防止列被过度压缩
- 外层容器 `overflow-x: auto` 允许横滑
- 不强制做卡片化改造，保持表格的信息密度优势
- 使用负 margin 让表格出血到屏幕边缘，最大化利用宽度

---

## Naive UI 移动端注意

| 组件 | 注意事项 |
|------|----------|
| `n-data-table` | 必须设置 `:scroll-x`，移动端靠横滑查看全部列 |
| `n-modal` | 移动端设置接近全屏尺寸，避免内容溢出 |
| `n-select` | 移动端下拉面板会自动适配，无需额外处理 |
| `n-date-picker` | 面板较宽，移动端考虑使用 `type="month"` 简化 |
| `n-dropdown` | 移动端作为操作菜单时注意定位不超出屏幕 |
| `n-popconfirm` | 气泡方向设置为 `top` 或 `bottom`，避免侧向溢出 |

---

## 验证清单

新增页面或组件时，确认以下事项：

- [ ] 在 375px 宽度（iPhone SE）下无横向溢出
- [ ] 所有可点击元素 ≥ 44×44px
- [ ] 输入框 font-size ≥ 16px
- [ ] hover 效果包裹在 `@media (hover: hover)` 中
- [ ] 底部固定元素处理了 `safe-area-inset-bottom`
- [ ] 表格设置了 `:scroll-x` 并有横滑容器
- [ ] 弹窗在移动端不溢出屏幕
- [ ] `prefers-reduced-motion` 下动画被禁用
