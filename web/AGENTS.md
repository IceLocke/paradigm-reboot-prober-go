# AGENTS.md — Web Frontend

## Project Overview

This is the **frontend** for **Paradigm: Reboot Prober**, a rating calculator web app for the rhythm game *Paradigm: Reboot*. It is a complete rewrite of the legacy frontend (`web/legacy/`, Element Plus + JavaScript) using a modern dark-themed design system.

The frontend communicates with the Go backend REST API (`/api/v2`). When the backend is unavailable, it runs in **mock mode** with realistic generated data.

Repository root: `paradigm-reboot-prober-go`
Frontend root: `web/`

---

## Tech Stack

| Component          | Technology                                        |
|--------------------|---------------------------------------------------|
| Language           | TypeScript (strict mode)                          |
| Framework          | Vue 3.5 (Composition API, `<script setup>`)       |
| Build Tool         | Vite 6                                            |
| UI Components      | Naive UI 2.40 (complex) + custom (basic)          |
| State Management   | Pinia 2 + pinia-plugin-persistedstate 3           |
| Routing            | Vue Router 4 (hash mode)                          |
| HTTP Client        | Axios 1.7                                         |
| Charting           | ECharts 5 + vue-echarts 7                         |
| Internationalization | vue-i18n 11 (Composition API mode)              |
| Date Handling      | dayjs                                             |
| File Download      | file-saver                                        |
| Utilities          | @vueuse/core                                      |
| Fonts              | Inter, Noto Sans SC, JetBrains Mono (Google Fonts)|
| Package Manager    | pnpm 10                                           |
| API Type Generation| openapi-typescript 7 + swagger2openapi 7           |
| Design             | Custom dark theme, CSS custom properties           |

---

## Project Structure

```
web/
├── index.html                    # Entry HTML (Google Fonts, viewport meta)
├── package.json                  # Dependencies and scripts
├── tsconfig.json                 # TypeScript config (strict, bundler resolution)
├── tsconfig.node.json            # TS config for vite.config.ts
├── vite.config.ts                # Vite config (alias @, proxy /api → :8080)
├── env.d.ts                      # Global type declarations
├── .gitignore
├── API_DIFF.md                   # v1 → v2 API migration reference
│
├── public/
│   └── favicon.ico
│
├── src/
│   ├── main.ts                   # App entry: Pinia, Router, I18n, CSS imports
│   ├── App.vue                   # Root: NConfigProvider, layout, auth modals
│   │
│   ├── assets/styles/
│   │   ├── variables.css         # Design tokens (colors, spacing, fonts, etc.)
│   │   ├── reset.css             # CSS reset + scrollbar + base styles
│   │   └── global.css            # Utility classes, Vue transitions, page layout
│   │
│   ├── config/
│   │   └── naive-theme.ts        # Naive UI GlobalThemeOverrides (dark theme)
│   │
│   ├── api/                      # API layer
│   │   ├── generated.d.ts        # Auto-generated types (DO NOT EDIT)
│   │   ├── openapi3.json         # Converted OpenAPI 3.0 spec (gitignored)
│   │   ├── types.ts              # Re-exports from generated.d.ts + DeepRequired
│   │   ├── client.ts             # Axios instance (base URL, JWT interceptor)
│   │   ├── user.ts               # User API (login, register, profile, token)
│   │   ├── song.ts               # Song API (list charts, song detail, CRUD)
│   │   ├── record.ts             # Record API (query, upload)
│   │   └── mock.ts               # Mock data generator + USE_MOCK flag
│   │
│   ├── stores/
│   │   ├── user.ts               # User state (username, token, profile) — persisted
│   │   └── app.ts                # App state (charts cache, upload cart, sidebar)
│   │
│   ├── composables/
│   │   └── useBreakpoint.ts      # Reactive viewport width / isMobile / isDesktop
│   │
│   ├── i18n/
│   │   ├── index.ts              # createI18n setup (auto-detect browser language)
│   │   ├── en.ts                 # English translations
│   │   └── zh.ts                 # Chinese translations
│   │
│   ├── router/
│   │   └── index.ts              # Routes: /best50, /songs, /records, /about
│   │
│   ├── components/
│   │   ├── ui/                   # Custom base components (full style control)
│   │   │   ├── BaseButton.vue    # Variants: primary/secondary/ghost/danger
│   │   │   ├── BaseCard.vue      # Card with optional header slot
│   │   │   ├── BaseInput.vue     # Text input with label, v-model support
│   │   │   └── BaseTabs.vue      # Segmented tab selector
│   │   │
│   │   ├── layout/               # App shell
│   │   │   ├── AppHeader.vue     # Top bar: logo, auth buttons, hamburger menu
│   │   │   └── Sidebar.vue       # Desktop sidebar + mobile drawer (Teleport)
│   │   │
│   │   └── business/             # Domain-specific components
│   │       ├── DifficultyBadge.vue    # Colored difficulty tag (DT/IN/MS/RB)
│   │       ├── StatCard.vue           # Numeric stat display (mono font, accent color)
│   │       ├── SongDetailModal.vue    # Song info modal (cover, metadata, charts)
│   │       ├── QuickUploadModal.vue   # Single record upload form
│   │       ├── UploadCartPanel.vue    # Batch upload queue with inline editing
│   │       ├── LoginModal.vue         # Login form modal
│   │       ├── RegisterModal.vue      # Registration form modal
│   │       └── ProfileModal.vue       # User profile editor modal
│   │
│   └── views/                    # Page-level components (one per route)
│       ├── Best50View.vue        # B50 overview: stats, scatter charts, tables
│       ├── SongsView.vue         # Charts list: search, filter tabs, pagination
│       ├── RecordsView.vue       # Play records: scope tabs, table, pagination
│       └── AboutView.vue         # About page: description, GitHub link
│
├── legacy/                       # Old frontend (git submodule, DO NOT MODIFY)
└── styles/                       # Design spec documents (reference only)
    ├── vue-style-pattern.md      # Design philosophy overview
    └── rules/
        ├── tokens.md             # CSS variable definitions
        ├── components.md         # Custom component code specs
        ├── naive-ui.md           # Naive UI integration guide
        └── responsive.md         # Mobile-first responsive rules
```

---

## Architecture

### Data Flow

```
User Interaction → View → (API / Mock) → Store → View re-renders
```

### Layers

| Layer | Location | Responsibility |
|-------|----------|----------------|
| **Views** | `src/views/` | Page-level components, compose business components, call API/mock |
| **Business Components** | `src/components/business/` | Domain logic UI (modals, badges, upload cart) |
| **Base UI Components** | `src/components/ui/` | Styled primitives (button, card, input, tabs) |
| **Layout Components** | `src/components/layout/` | App shell (header, sidebar/drawer) |
| **API Layer** | `src/api/` | Axios client, typed API functions, mock data |
| **Stores** | `src/stores/` | Pinia reactive state (user session, app cache) |
| **Composables** | `src/composables/` | Reusable reactive logic (breakpoint detection) |
| **Design System** | `src/assets/styles/` + `src/config/` | CSS variables + Naive UI theme overrides |

### Mock Mode

The file `src/api/mock.ts` exports a flag:

```ts
export const USE_MOCK = true
```

When `true`, all views call `getMockCharts()`, `getMockB50()`, `getMockRecords()`, `getMockUser()` instead of the real API. To switch to the real backend, set `USE_MOCK = false`. The mock data generator mirrors the backend's rating formula.

---

## Design System

### Design Philosophy

- **Dark theme only** — deep backgrounds, single accent color, subtle layering
- **Single accent color** — `#3b82f6` (blue) for all interactive elements
- **Semantic colors** only for success/warning/danger, never decorative
- **Restrained animation** — only `opacity` and `transform`, respect `prefers-reduced-motion`

### Key Design Tokens (`src/assets/styles/variables.css`)

| Token              | Value     | Usage                       |
|--------------------|-----------|-----------------------------|
| `--bg-primary`     | `#0e0e12` | Page background             |
| `--bg-secondary`   | `#16161c` | Header, sidebar, input bg   |
| `--bg-card`        | `#1a1a22` | Card surfaces               |
| `--accent`         | `#3b82f6` | Primary interactive color   |
| `--text-primary`   | `#e4e4e7` | Main text                   |
| `--text-secondary` | `#a1a1aa` | Secondary text              |
| `--text-muted`     | `#52525b` | Disabled / caption text     |
| `--border`         | `#27272a` | Default border color        |
| `--diff-detected`  | `#3b82f6` | Detected difficulty (blue)  |
| `--diff-invaded`   | `#ef4444` | Invaded difficulty (red)    |
| `--diff-massive`   | `#a855f7` | Massive difficulty (purple) |
| `--diff-reboot`    | `#f97316` | Reboot difficulty (orange)  |

### Typography

| Usage          | Font Family         | Variable      |
|----------------|---------------------|---------------|
| Body text      | Inter, Noto Sans SC | `--font-sans` |
| Numeric / code | JetBrains Mono      | `--font-mono` |

### Border Radius

| Context                    | Value  |
|----------------------------|--------|
| Cards, modals              | `10px` |
| Buttons, inputs, tabs      | `8px`  |
| Small elements, list items | `6px`  |
| Badges, tags               | `4px`  |

### Theme Sync

The **same color palette** must be maintained in two places:
1. `src/assets/styles/variables.css` — CSS custom properties
2. `src/config/naive-theme.ts` — Naive UI `GlobalThemeOverrides`

When changing any color, **update both files** to keep them in sync.

---

## Component Strategy

| Category | Source | Components | Reason |
|----------|--------|------------|--------|
| Base styling | **Custom** | BaseButton, BaseCard, BaseInput, BaseTabs | Full visual control, design soul |
| Complex interaction | **Naive UI** | NDataTable, NModal, NPagination, NPopover, NSelect, NSpin | Keyboard nav, virtual scroll, complex state |
| Layout | **Custom** | AppHeader, Sidebar | Structural, must be fully customizable |
| Business | **Custom** | DifficultyBadge, StatCard, SongDetailModal, etc. | Domain-specific, no reuse from libraries |

### Rules

- **DO NOT** use Naive UI's `n-button`, `n-card`, `n-input`, `n-tabs` — use the custom `Base*` components
- **DO** use Naive UI for `n-data-table`, `n-modal`, `n-pagination`, `n-popover`, `n-select`, `n-spin`, `n-empty`
- All Naive UI component theming goes through `src/config/naive-theme.ts`

---

## Build and Run Commands

```bash
# Install dependencies
pnpm install

# Start dev server (http://localhost:3000)
pnpm dev

# Type-check and build for production
pnpm build

# Preview production build
pnpm preview

# Regenerate API types from backend Swagger spec
pnpm generate:api
```

### API Type Generation Pipeline

```
docs/swagger.json  ──swagger2openapi──▶  src/api/openapi3.json  ──openapi-typescript──▶  src/api/generated.d.ts
                    (Swagger 2.0 → 3.0)                          (OpenAPI 3.0 → TypeScript)
```

- **`generated.d.ts`** is auto-generated and committed to the repo. DO NOT edit manually.
- **`openapi3.json`** is a build intermediate, gitignored.
- **`types.ts`** is a hand-maintained thin adapter that re-exports from `generated.d.ts` with `DeepRequired<>` wrappers for response models (because the backend always returns complete objects, but Go struct tags don't mark every field `required`).
- When the backend Swagger spec changes, run `pnpm generate:api` and fix any resulting type errors.

### Dev Server Proxy

Vite proxies `/api` → `http://localhost:8080` so the frontend can call the Go backend during development. In mock mode (`USE_MOCK = true`), no backend is needed.

---

## Routes

| Path       | View Component    | Description                                                    |
|------------|-------------------|----------------------------------------------------------------|
| `/`        | —                 | Redirects to `/best50`                                         |
| `/best50`  | `Best50View.vue`  | B50 stats, scatter charts, B35/B15 record tables               |
| `/songs`   | `SongsView.vue`   | All charts with search, difficulty/version filters, pagination |
| `/records` | `RecordsView.vue` | Play records with scope tabs (best/all), pagination            |
| `/about`   | `AboutView.vue`   | Project description and GitHub link                            |

All view components are **lazy-loaded** via `() => import(...)`.

---

## State Management (Pinia)

### `userStore` (persisted to localStorage)

| Field          | Type           | Description               |
|----------------|----------------|---------------------------|
| `username`     | `string`       | Current username          |
| `access_token` | `string`       | Full JWT (`"Bearer xxx"`) |
| `is_admin`     | `boolean`      | Admin flag                |
| `logged_in`    | `boolean`      | Auth state                |
| `profile`      | `User \| null` | Full user profile object  |

### `appStore` (not persisted)

| Field | Type | Description |
|-------|------|-------------|
| `charts` | `ChartInfo[] \| null` | Cached chart list from `/songs` |
| `uploadList` | `UploadItem[]` | Batch upload cart |
| `sidebarOpen` | `boolean` | Mobile drawer visibility |

---

## API Layer

### Client (`src/api/client.ts`)

- Base URL: `/api/v2`
- Request interceptor: reads `userStore` from `localStorage` and attaches `Authorization` header
- The token format is `"Bearer <jwt>"`, stored as-is in `userStore.access_token`

### Type Definitions (`src/api/types.ts`)

Types are **auto-generated** from `docs/swagger.json` via `pnpm generate:api`, then re-exported through `types.ts` with convenience aliases.

| Category | Types |
|----------|-------|
| **Enums** | `Difficulty` |
| **Response models** (wrapped with `DeepRequired`) | `User`, `Song`, `Chart`, `ChartInfo`, `ChartInfoSimple`, `ChartInput`, `PlayRecord`, `PlayRecordBase`, `PlayRecordInfo`, `PlayRecordResponse`, `Token`, `UploadToken` |
| **Request DTOs** (keep optional fields) | `CreateUserRequest`, `UpdateUserRequest`, `ChangePasswordRequest`, `ResetPasswordRequest`, `BatchCreatePlayRecordRequest`, `CreateSongRequest`, `UpdateSongRequest` |
| **API response wrapper** | `Response` |

### API Modules

| Module | Functions |
|--------|-----------|
| `user.ts` | `login`, `register`, `getMyInfo`, `updateMyInfo`, `changePassword`, `refreshUploadToken`, `resetPassword` |
| `song.ts` | `getAllSongLevels`, `getSingleSongInfo`, `createSong`, `updateSong` |
| `record.ts` | `getRecords`, `uploadRecords` |

### v1 → v2 Breaking Changes

See `web/API_DIFF.md` for the full migration reference. Key changes:
- `song_level_id` → `chart_id`
- `difficulty_id` (int) → `difficulty` (string enum)
- `PATCH` → `PUT` for user/song updates
- Records `scope` now supports `b50`, `best`, `all`, `all-charts`

---

## Internationalization

- Auto-detects browser language (`navigator.language`)
- Supported: `en` (English), `zh` (Simplified Chinese)
- Fallback: English
- Translation files: `src/i18n/en.ts`, `src/i18n/zh.ts`
- Usage in components: `const { t } = useI18n()` + `t('key.path')`
- All user-facing strings MUST go through `t()`, never hardcode

### Translation Structure

```
common.*     — universal actions (submit, save, cancel, refresh)
auth.*       — authentication labels (login, register, username, password)
term.*       — domain terms (b50, records, difficulty, level, rating)
message.*    — notifications and validation messages
about.*      — about page content
```

---

## Mobile-First Responsive Design

### Breakpoints

| Name | Value | Typical Device |
|------|-------|----------------|
| `sm` | `640px` | Large phone |
| `md` | `768px` | Tablet portrait — **sidebar breakpoint** |
| `lg` | `1024px` | Tablet landscape / small laptop |
| `xl` | `1280px` | Desktop |

### Key Behaviors

| Screen | Navigation | Layout |
|--------|------------|--------|
| `≥ 768px` | Persistent sidebar (220px) | Side-by-side |
| `< 768px` | Hamburger → slide-in drawer (Teleport to body) | Single column |

### Constraints

- **Touch targets**: All interactive elements ≥ `44×44px` (`min-height: 44px`)
- **Input font size**: Always `≥ 16px` (prevents iOS Safari auto-zoom)
- **Hover isolation**: All `:hover` styles wrapped in `@media (hover: hover)`
- **No horizontal overflow**: `html, body { overflow-x: hidden }`
- **Table strategy**: `overflow-x: auto` wrapper + `:scroll-x` on NDataTable
- **Modal on mobile**: Near-fullscreen width (`95vw`)

### Composable

```ts
import { useBreakpoint } from '@/composables/useBreakpoint'
const { isMobile, isDesktop, width } = useBreakpoint()
```

---

## Code Style Guidelines

- **Vue SFC order**: `<template>` → `<script setup lang="ts">` → `<style scoped>`
- **Composition API only**: Use `<script setup>`, no Options API
- **TypeScript strict mode**: All props/emits must be typed with generics
- **Props**: Use `defineProps<{...}>()` with `withDefaults()` when needed
- **Emits**: Use `defineEmits<{...}>()` with typed event signatures
- **CSS**: Scoped styles, use CSS custom properties from `variables.css`
- **Naming**: PascalCase for components, camelCase for variables/functions
- **Files**: PascalCase for `.vue` files, camelCase for `.ts` files
- **Imports**: Use `@/` path alias, group by (vue → libraries → local)
- **No `any`**: Avoid `any` type; use `unknown` or proper interfaces
- **Icons**: Inline SVG (no icon library dependency), kept minimal

### CSS Conventions

```css
/* ✅ Correct: use design tokens */
background: var(--bg-card);
color: var(--text-primary);
border-radius: 8px;
transition: background var(--transition-base);

/* ✅ Correct: isolate hover */
@media (hover: hover) {
  .item:hover { background: rgba(255, 255, 255, 0.06); }
}

/* ❌ Wrong: hardcoded colors */
background: #1a1a22;
color: white;

/* ❌ Wrong: unguarded hover */
.item:hover { background: rgba(255, 255, 255, 0.06); }
```

---

## Development Red Lines

### Style
- **One accent color**: Only `--accent` blue. No multi-color decorations
- **Semantic colors only for meaning**: green=success, yellow=warning, red=danger
- **No box-shadow stacking**: Use borders and backgrounds for layering
- **Difficulty colors are the ONLY exception**: 4 fixed colors for game difficulties

### Naive UI
- **NEVER** use `n-button`, `n-card`, `n-input`, `n-tabs` — use `Base*` components
- Theme config lives ONLY in `src/config/naive-theme.ts`
- Color changes must sync `variables.css` ↔ `naive-theme.ts`

### Mobile
- Every new component MUST work at 375px width
- Modals on mobile: `width: 95vw; max-width: none`
- No horizontal page overflow
- `prefers-reduced-motion` must disable all animations

### API
- All API types are auto-generated — run `pnpm generate:api` after backend schema changes
- NEVER edit `src/api/generated.d.ts` manually
- If you add a new response model, add a `DeepRequired<>` re-export in `types.ts`
- If you add a new request DTO, re-export it as-is (no `DeepRequired`) in `types.ts`
- Mock data in `src/api/mock.ts` should mirror real API response structure exactly

---

## Key Domain Concepts

| Concept | Description |
|---------|-------------|
| **Song** | A music track with metadata (title, artist, BPM, cover, etc.) |
| **Chart** | A specific difficulty of a song (`detected` / `invaded` / `massive` / `reboot`) |
| **chart_id** | Unique identifier for a chart (was `song_level_id` in v1) |
| **PlayRecord** | A single play attempt with score, linked to a chart and user |
| **Rating** | Calculated from chart level × score using a piecewise formula. Stored as `int` (×100) |
| **B50** | Best 50 = B35 (top 35 from old songs, `b15=false`) + B15 (top 15 from new songs, `b15=true`) |
| **Upload Cart** | A client-side queue for batching multiple score uploads |
| **Upload Token** | An alternative auth method allowing third-party tools to upload scores |

---

## Files That Must Stay in Sync

| Change | Files to Update |
|--------|-----------------|
| Color palette | `src/assets/styles/variables.css` + `src/config/naive-theme.ts` |
| API schema | Run `pnpm generate:api` → update `src/api/types.ts` re-exports → fix affected views |
| New route | `src/router/index.ts` + sidebar nav items in `Sidebar.vue` |
| New translation key | `src/i18n/en.ts` + `src/i18n/zh.ts` |
| New difficulty type | `src/api/types.ts` (Difficulty union) + `DifficultyBadge.vue` + `variables.css` |
