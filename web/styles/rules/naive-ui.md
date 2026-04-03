# Naive UI 集成方案

> 返回纲要 → [../vue-style-pattern.md](../vue-style-pattern.md)

---

## 安装

```bash
npm i naive-ui
# 按需引入（推荐）
npm i -D unplugin-auto-import unplugin-naive-ui
```

---

## Vite 配置

```ts
// vite.config.ts
import AutoImport from 'unplugin-auto-import/vite'
import NaiveUI from 'unplugin-naive-ui/vite'

export default defineConfig({
  plugins: [
    vue(),
    NaiveUI(),
    AutoImport({
      imports: [
        'vue',
        {
          'naive-ui': ['useMessage', 'useNotification', 'useDialog'],
        },
      ],
    }),
  ],
})
```

---

## 主题对接

在入口处通过 `n-config-provider` 将项目色彩系统注入 Naive UI。
建议将 `themeOverrides` 抽离到 `src/config/naive-theme.ts` 集中维护。

### App.vue

```vue
<template>
  <n-config-provider :theme="darkTheme" :theme-overrides="themeOverrides">
    <n-message-provider>
      <n-notification-provider>
        <RouterView />
      </n-notification-provider>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import { darkTheme } from 'naive-ui'
import { themeOverrides } from '@/config/naive-theme'
</script>
```

### naive-theme.ts

```ts
// src/config/naive-theme.ts
import type { GlobalThemeOverrides } from 'naive-ui'

export const themeOverrides: GlobalThemeOverrides = {
  common: {
    // 强调色
    primaryColor: '#3b82f6',
    primaryColorHover: '#2563eb',
    primaryColorPressed: '#1d4ed8',
    primaryColorSuppl: '#3b82f6',

    // 背景
    bodyColor: '#0e0e12',
    cardColor: '#1a1a22',
    modalColor: '#1a1a22',
    popoverColor: '#1e1e26',
    tableColor: '#1a1a22',
    tableColorStriped: '#16161c',
    inputColor: '#16161c',

    // 边框
    borderColor: '#27272a',
    dividerColor: '#27272a',

    // 文字
    textColorBase: '#e4e4e7',
    textColor1: '#e4e4e7',
    textColor2: '#a1a1aa',
    textColor3: '#52525b',
    placeholderColor: '#52525b',

    // 圆角
    borderRadius: '8px',
    borderRadiusSmall: '6px',

    // 功能色
    successColor: '#22c55e',
    warningColor: '#eab308',
    errorColor: '#ef4444',
    infoColor: '#3b82f6',

    // 字体
    fontFamily: "'Inter', 'Noto Sans SC', -apple-system, BlinkMacSystemFont, sans-serif",
    fontFamilyMono: "'JetBrains Mono', 'Fira Code', 'Consolas', monospace",
  },
  DataTable: {
    borderColor: '#27272a',
    thColor: '#16161c',
    thColorHover: '#1e1e26',
    tdColor: '#1a1a22',
    tdColorHover: '#1e1e26',
    thTextColor: '#a1a1aa',
    tdTextColor: '#e4e4e7',
    thFontWeight: '500',
    borderRadius: '10px',
  },
  Button: {
    borderRadiusMedium: '8px',
  },
  Card: {
    borderRadius: '10px',
    borderColor: '#27272a',
  },
  Tag: {
    borderRadius: '4px',
  },
}
```

---

## 常用组件速查

| 场景 | 组件 | 说明 |
|------|------|------|
| 数据表格 | `n-data-table` | 排序、筛选、分页、虚拟滚动、固定列、自定义渲染 |
| 下拉选择 | `n-select` | 单选 / 多选、搜索、远程加载、分组 |
| 日期选择 | `n-date-picker` | 日期 / 范围 / 月份 |
| 弹窗 | `n-modal` | 对话框、抽屉式弹窗 |
| 通知 | `useMessage` / `useNotification` | 轻提示 / 详细通知 |
| 气泡确认 | `n-popconfirm` | 删除等危险操作的二次确认 |
| 下拉菜单 | `n-dropdown` | 右键菜单、操作菜单 |
| 分页 | `n-pagination` | 表格外置分页 |
| 加载 | `n-spin` / `n-skeleton` | 加载态 / 骨架屏 |
| 空状态 | `n-empty` | 无数据提示 |

---

## DataTable 使用示例

```vue
<template>
  <div class="table-wrapper">
    <n-data-table
      :columns="columns"
      :data="records"
      :pagination="{ pageSize: 20 }"
      :bordered="false"
      :scroll-x="600"
      striped
      size="small"
    />
  </div>
</template>

<script setup lang="ts">
import { h, ref } from 'vue'
import { NTag } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'

interface Record {
  songName: string
  difficulty: string
  score: number
  rating: number
}

const columns: DataTableColumns<Record> = [
  { title: '曲名', key: 'songName', ellipsis: { tooltip: true }, minWidth: 150 },
  {
    title: '难度',
    key: 'difficulty',
    width: 100,
    render(row) {
      const colorMap: Record<string, string> = {
        detected: 'success',
        invaded: 'warning',
        massive: 'warning',
        reboot: 'error',
      }
      return h(NTag, { size: 'small', type: colorMap[row.difficulty] ?? 'default' }, () => row.difficulty)
    },
  },
  { title: '分数', key: 'score', sorter: (a, b) => a.score - b.score, width: 120 },
  { title: 'Rating', key: 'rating', sorter: (a, b) => a.rating - b.rating, width: 100 },
]

const records = ref<Record[]>([])
</script>

<style scoped>
/* 移动端表格水平滚动 */
.table-wrapper {
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
}
</style>
```

> **移动端注意**：设置 `:scroll-x` 为列最小宽度之和，确保窄屏下可横向滚动而非挤压列宽。

---

## 使用约束

1. **不用** Naive UI 的 `n-button` / `n-card` / `n-input` / `n-tabs`，用项目自定义版本
2. 所有 `themeOverrides` 集中在 `src/config/naive-theme.ts`
3. 修改色值时同步更新 `variables.css` 和 `naive-theme.ts`，保持单一来源
4. 移动端弹窗使用 `n-modal` 时设置 `style="width: 90vw; max-width: 480px"` 避免溢出
