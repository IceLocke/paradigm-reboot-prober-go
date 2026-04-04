import type { GlobalThemeOverrides } from 'naive-ui'

export const themeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#3b82f6',
    primaryColorHover: '#2563eb',
    primaryColorPressed: '#1d4ed8',
    primaryColorSuppl: '#3b82f6',

    bodyColor: '#0e0e12',
    cardColor: '#1a1a22',
    modalColor: '#1a1a22',
    popoverColor: '#1e1e26',
    tableColor: '#1a1a22',
    tableColorStriped: '#16161c',
    inputColor: '#16161c',

    borderColor: '#27272a',
    dividerColor: '#27272a',

    textColorBase: '#e4e4e7',
    textColor1: '#e4e4e7',
    textColor2: '#a1a1aa',
    textColor3: '#52525b',
    placeholderColor: '#52525b',

    borderRadius: '8px',
    borderRadiusSmall: '6px',

    successColor: '#22c55e',
    warningColor: '#eab308',
    errorColor: '#ef4444',
    infoColor: '#3b82f6',

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
  Tag: {
    borderRadius: '4px',
  },
  Pagination: {
    itemBorderRadius: '6px',
  },
}
