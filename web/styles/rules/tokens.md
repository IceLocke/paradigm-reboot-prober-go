# 设计令牌（Design Tokens）

> 返回纲要 → [../vue-style-pattern.md](../vue-style-pattern.md)

本文件包含所有 CSS 自定义属性的完整定义。修改色值时须同步更新 `src/config/naive-theme.ts`。

---

## 色彩系统

```css
:root {
  color-scheme: dark;

  /* 背景色 - 三层递进 */
  --bg-primary: #0e0e12;
  --bg-secondary: #16161c;
  --bg-tertiary: #1e1e26;
  --bg-card: #1a1a22;

  /* 强调色 */
  --accent: #3b82f6;
  --accent-hover: #2563eb;
  --accent-muted: rgba(59, 130, 246, 0.15);

  /* 功能色 - 仅用于语义场景 */
  --color-success: #22c55e;
  --color-warning: #eab308;
  --color-danger: #ef4444;
  --color-info: #3b82f6;

  /* 难度色（业务） */
  --diff-detected: #22c55e;
  --diff-invaded: #eab308;
  --diff-massive: #f97316;
  --diff-reboot: #ec4899;

  /* 文字色 */
  --text-primary: #e4e4e7;
  --text-secondary: #a1a1aa;
  --text-muted: #52525b;
  --text-accent: var(--accent);

  /* 边框 */
  --border: #27272a;
  --border-hover: #3f3f46;
  --border-accent: var(--accent);
}
```

---

## 间距系统

```css
:root {
  --space-1: 4px;
  --space-2: 8px;
  --space-3: 12px;
  --space-4: 16px;
  --space-5: 20px;
  --space-6: 24px;
  --space-8: 32px;
  --space-10: 40px;
}
```

---

## 字体

### 字体栈

```css
:root {
  --font-sans: 'Inter', 'Noto Sans SC', -apple-system, BlinkMacSystemFont, sans-serif;
  --font-mono: 'JetBrains Mono', 'Fira Code', 'Consolas', monospace;
}

body {
  font-family: var(--font-sans);
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}
```

### 字号层级

```css
:root {
  --text-xs: 12px;
  --text-sm: 13px;
  --text-base: 14px;
  --text-lg: 16px;
  --text-xl: 18px;
  --text-2xl: 22px;
  --text-3xl: 28px;
}
```

### 排版规范

```css
h1 { font-size: var(--text-3xl); font-weight: 600; color: var(--text-primary); margin: 0; }
h2 { font-size: var(--text-2xl); font-weight: 600; color: var(--text-primary); margin: 0; }
h3 { font-size: var(--text-xl);  font-weight: 500; color: var(--text-primary); margin: 0; }

p  { font-size: var(--text-base); color: var(--text-secondary); line-height: 1.6; margin: 0; }

/* 数值展示 */
.stat-value {
  font-family: var(--font-mono);
  font-size: var(--text-2xl);
  font-weight: 600;
  color: var(--accent);
}

.caption {
  font-size: var(--text-xs);
  color: var(--text-muted);
  letter-spacing: 0.02em;
}
```

---

## 过渡

```css
:root {
  --transition-fast: 0.15s ease;
  --transition-base: 0.2s ease;
  --transition-slow: 0.3s ease;
}
```

---

## 圆角速查

| 场景 | 值 |
|------|----|
| 大容器 / 卡片 / 弹窗 | `10px` |
| 按钮 / 输入框 / Tabs | `8px` |
| 小元素 / 列表项 | `6px` |
| Badge / Tag | `4px` |
