# Vue 前端 Style Pattern - 工具类应用 UI 设计规范

> **阅读方式**：本文件是设计规范的**纲要**，只包含核心原则和速查表。
> 每个主题的完整代码与详细说明见对应子文件。

---

## 1. 设计哲学

- **风格定位**: 现代深色 / 简约克制 / 专业精致
- **视觉基调**: 深色背景 + 单一强调色 + 微妙层次感
- **交互理念**: 高效操作、轻量反馈、清晰信息层级
- **核心特征**: 统一圆角、单色强调、克制动画、充足留白

---

## 2. 设计令牌速查

> 完整定义与代码 → [rules/tokens.md](rules/tokens.md)

### 色彩

| 用途 | 变量 | 值 |
|------|------|----|
| 主背景 | `--bg-primary` | `#0e0e12` |
| 卡片背景 | `--bg-card` | `#1a1a22` |
| 强调色 | `--accent` | `#3b82f6` |
| 主文字 | `--text-primary` | `#e4e4e7` |
| 次要文字 | `--text-secondary` | `#a1a1aa` |
| 边框 | `--border` | `#27272a` |

### 间距

`--space-{1,2,3,4,5,6,8,10}` → 4px 递增至 40px

### 字体

| 用途 | 变量 | 首选字体 |
|------|------|----------|
| 正文 | `--font-sans` | Inter, Noto Sans SC |
| 数值 | `--font-mono` | JetBrains Mono |

### 圆角

| 场景 | 值 |
|------|----|
| 大容器 / 卡片 | `10px` |
| 按钮 / 输入框 | `8px` |
| 小元素 | `6px` |
| Badge | `4px` |

---

## 3. 组件策略

> 自定义组件完整代码 → [rules/components.md](rules/components.md)
> Naive UI 集成方案 → [rules/naive-ui.md](rules/naive-ui.md)

| 层级 | 来源 | 组件 | 理由 |
|------|------|------|------|
| 基础风格 | **自定义** | Button, Card, Input, Tabs | 视觉灵魂，需完全掌控 |
| 复杂交互 | **Naive UI** | DataTable, Select, DatePicker, Modal, Dropdown, Popconfirm, Notification | 交互逻辑重，自造 ROI 低 |
| 布局 | **自定义** | AppHeader, Sidebar, MainContent | 结构性组件，必须自控 |
| 业务 | **自定义** | SongListItem, DifficultyBadge 等 | 业务定制，无法复用 |

**原则**：3 行 CSS 能写清楚的自己写，需要键盘导航 / 虚拟滚动 / 复杂下拉的用 Naive UI。

---

## 4. 移动端适配

> 完整断点定义、布局策略与检查清单 → [rules/responsive.md](rules/responsive.md)

### 核心约束

1. **Mobile-first 媒体查询**：默认样式为移动端，`min-width` 向上扩展
2. **触摸目标 ≥ 44×44px**：所有可点击元素的最小尺寸
3. **表单 font-size ≥ 16px**：防止 iOS Safari 自动缩放
4. **安全区域适配**：底部操作栏必须处理 `safe-area-inset-bottom`
5. **表格水平滚动**：移动端 DataTable 使用 `overflow-x: auto` 包裹，不强制卡片化
6. **禁用 hover 粘滞**：移动端通过 `@media (hover: hover)` 隔离 hover 效果
7. **侧边栏 → 抽屉**：`< 768px` 时侧边栏切换为抽屉式或底部导航
8. **viewport 配置**：`<meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">`

### 断点

| 名称 | 值 | 典型设备 |
|------|----|----------|
| `sm` | `640px` | 大屏手机 |
| `md` | `768px` | 平板竖屏 |
| `lg` | `1024px` | 平板横屏 / 小笔记本 |
| `xl` | `1280px` | 桌面 |

---

## 5. 动画规范

仅使用克制的过渡，不使用循环 / 脉冲 / 闪烁等装饰性动画。

```css
:root {
  --transition-fast: 0.15s ease;
  --transition-base: 0.2s ease;
  --transition-slow: 0.3s ease;
}
```

Vue 过渡只用两种：

- **`fade`**：透明度渐变，用于内容切换
- **`list`**：带 `translateY(-8px)` 的列表增删

动画仅使用 `transform` 和 `opacity`（GPU 加速），尊重 `prefers-reduced-motion`。

---

## 6. 项目结构

```
src/
├── assets/styles/
│   ├── variables.css            # CSS 变量（与 naive-theme.ts 同步维护）
│   ├── reset.css                # 样式重置
│   └── global.css               # 全局样式、过渡类
├── config/
│   └── naive-theme.ts           # Naive UI 主题覆盖配置
├── components/
│   ├── ui/                      # 自定义基础组件
│   │   ├── BaseButton.vue
│   │   ├── BaseCard.vue
│   │   ├── BaseInput.vue
│   │   └── BaseTabs.vue
│   ├── layout/                  # 布局组件
│   │   ├── AppHeader.vue
│   │   ├── Sidebar.vue          # 桌面侧边栏 + 移动端抽屉
│   │   └── MainContent.vue
│   └── business/                # 业务组件
├── composables/
├── views/
└── App.vue                      # 挂载 n-config-provider
```

---

## 7. 开发红线

### 风格
- **一个强调色**：全局只用 `--accent` 蓝，不多色混用
- **功能色只语义化**：成功绿 / 警告黄 / 危险红，不做装饰
- **少即是多**：能用边框表达不用阴影，能用透明度区分不用颜色

### Naive UI
- **不用** Naive UI 的 Button / Card / Input / Tabs，这些用自定义组件
- 主题配置集中在 `naive-theme.ts`，修改色值时同步更新 `variables.css`

### 移动端
- 新增任何交互组件必须在 375px 宽度下验证可用性
- 弹窗在移动端必须接近全屏（`width: 100%; max-height: 90vh`）
- 不允许出现横向溢出导致页面可横滑的情况

### 性能
- 图片使用 `loading="lazy"`
- 避免 `box-shadow` 堆叠
- 尊重 `prefers-reduced-motion`

---

*详细规范子文件：[tokens](rules/tokens.md) · [components](rules/components.md) · [naive-ui](rules/naive-ui.md) · [responsive](rules/responsive.md)*
