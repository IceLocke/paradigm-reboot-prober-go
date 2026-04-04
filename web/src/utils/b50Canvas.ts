/**
 * B50 image canvas renderer.
 * Generates a B50 best records image using Canvas 2D API.
 */
import type { PlayRecordInfo, Difficulty } from '@/api/types'

// ─── Render Options ────────────────────────────────────────────

export interface B50RenderOptions {
  b15Records: PlayRecordInfo[]
  b35Records: PlayRecordInfo[]
  username: string
  rating: number   // Player overall rating (avg B50, already /100)
  b15Avg: number   // B15 avg rating (already /100)
  b35Avg: number   // B35 avg rating (already /100)
}

// ─── Layout Constants ──────────────────────────────────────────

const CANVAS_WIDTH = 1440
const PADDING_X = 32
const PADDING_TOP = 32
const PADDING_BOTTOM = 24

const COLS = 5
const CARD_GAP = 8
const CARD_WIDTH = (CANVAS_WIDTH - 2 * PADDING_X - (COLS - 1) * CARD_GAP) / COLS
const CARD_HEIGHT = 160
const CARD_RADIUS = 8

const HEADER_HEIGHT = 68
const SECTION_TITLE_HEIGHT = 24
const GAP_AFTER_HEADER = 18
const GAP_AFTER_SECTION_TITLE = 12
const GAP_BETWEEN_SECTIONS = 22
const FOOTER_HEIGHT = 28

// ─── Colors ────────────────────────────────────────────────────

const DIFF_COLORS: Record<string, string> = {
  detected: '#3b82f6',
  invaded: '#ef4444',
  massive: '#a855f7',
  reboot: '#f97316',
}

const DIFF_NAMES: Record<string, string> = {
  detected: 'DETECTED',
  invaded: 'INVADED',
  massive: 'MASSIVE',
  reboot: 'REBOOT',
}

// ─── Fonts ─────────────────────────────────────────────────────

const FONT_SANS = "'Inter', 'Noto Sans SC', sans-serif"
const FONT_MONO = "'JetBrains Mono', monospace"

// ─── Helpers ───────────────────────────────────────────────────

function formatDiffLevel(level: number): string {
  const base = Math.floor(level)
  const plus = level - base >= 0.5 ? '+' : ''
  return `${base}${plus}`
}

function formatDate(): string {
  const now = new Date()
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${now.getFullYear()}/${pad(now.getMonth() + 1)}/${pad(now.getDate())} ${pad(now.getHours())}:${pad(now.getMinutes())}:${pad(now.getSeconds())}`
}

function roundedRectPath(
  ctx: CanvasRenderingContext2D,
  x: number, y: number, w: number, h: number, r: number,
) {
  ctx.beginPath()
  ctx.moveTo(x + r, y)
  ctx.lineTo(x + w - r, y)
  ctx.quadraticCurveTo(x + w, y, x + w, y + r)
  ctx.lineTo(x + w, y + h - r)
  ctx.quadraticCurveTo(x + w, y + h, x + w - r, y + h)
  ctx.lineTo(x + r, y + h)
  ctx.quadraticCurveTo(x, y + h, x, y + h - r)
  ctx.lineTo(x, y + r)
  ctx.quadraticCurveTo(x, y, x + r, y)
  ctx.closePath()
}

/** Draw image with CSS `object-fit: cover` behaviour */
function drawImageCover(
  ctx: CanvasRenderingContext2D,
  img: HTMLImageElement,
  x: number, y: number, w: number, h: number,
) {
  const imgRatio = img.naturalWidth / img.naturalHeight
  const targetRatio = w / h
  let sx: number, sy: number, sw: number, sh: number
  if (imgRatio > targetRatio) {
    sh = img.naturalHeight
    sw = sh * targetRatio
    sx = (img.naturalWidth - sw) / 2
    sy = 0
  } else {
    sw = img.naturalWidth
    sh = sw / targetRatio
    sx = 0
    sy = (img.naturalHeight - sh) / 2
  }
  ctx.drawImage(img, sx, sy, sw, sh, x, y, w, h)
}

function truncateText(ctx: CanvasRenderingContext2D, text: string, maxWidth: number): string {
  if (ctx.measureText(text).width <= maxWidth) return text
  let t = text
  while (t.length > 0 && ctx.measureText(t + '...').width > maxWidth) {
    t = t.slice(0, -1)
  }
  return t + '...'
}

// ─── Image Loading ─────────────────────────────────────────────

function loadImage(url: string): Promise<HTMLImageElement | null> {
  return new Promise((resolve) => {
    const img = new Image()
    img.crossOrigin = 'anonymous'
    img.onload = () => resolve(img)
    img.onerror = () => resolve(null)
    img.src = url
  })
}

async function preloadImages(
  records: PlayRecordInfo[],
): Promise<Map<string, HTMLImageElement>> {
  const covers = new Set<string>()
  for (const r of records) {
    if (r.chart.cover) covers.add(r.chart.cover)
  }

  const map = new Map<string, HTMLImageElement>()
  const tasks = Array.from(covers).map(async (cover) => {
    const img = await loadImage(`/cover/${cover}`)
    if (img) map.set(cover, img)
  })
  await Promise.all(tasks)
  return map
}

// ─── Drawing Functions ─────────────────────────────────────────

function drawBackground(
  ctx: CanvasRenderingContext2D,
  width: number, height: number,
  bgImage: HTMLImageElement | null,
) {
  // Solid dark base
  ctx.fillStyle = '#0e0e12'
  ctx.fillRect(0, 0, width, height)

  if (bgImage) {
    ctx.save()
    ctx.filter = 'blur(30px)'
    // Draw larger than canvas to prevent white edges from blur
    drawImageCover(ctx, bgImage, -40, -40, width + 80, height + 80)
    ctx.restore()
  }

  // Dark overlay to ensure readability
  ctx.fillStyle = 'rgba(14, 14, 18, 0.65)'
  ctx.fillRect(0, 0, width, height)
}

function drawHeader(
  ctx: CanvasRenderingContext2D,
  y: number,
  options: B50RenderOptions,
) {
  const leftX = PADDING_X
  const rightX = CANVAS_WIDTH - PADDING_X

  // Left: project title
  ctx.font = `bold 28px ${FONT_SANS}`
  ctx.fillStyle = '#ffffff'
  ctx.textAlign = 'left'
  ctx.textBaseline = 'top'
  ctx.fillText('Paradigm: Reboot Player Bests', leftX, y)

  // Left: date/time
  ctx.font = `16px ${FONT_SANS}`
  ctx.fillStyle = '#a1a1aa'
  ctx.fillText(formatDate(), leftX, y + 38)

  // Right: username
  ctx.font = `bold 28px ${FONT_SANS}`
  ctx.fillStyle = '#ffffff'
  ctx.textAlign = 'right'
  ctx.fillText(options.username, rightX, y)

  // Right: rating
  ctx.font = `16px ${FONT_MONO}`
  ctx.fillStyle = '#a1a1aa'
  ctx.fillText(`Rating: ${options.rating.toFixed(4)}`, rightX, y + 38)
}

function drawSectionTitle(
  ctx: CanvasRenderingContext2D,
  centerY: number,
  label: string,
  avg: number,
) {
  const text = `${label} / Avg. ${avg.toFixed(4)}`
  const centerX = CANVAS_WIDTH / 2

  ctx.font = `bold 18px ${FONT_SANS}`
  ctx.fillStyle = '#e4e4e7'
  ctx.textAlign = 'center'
  ctx.textBaseline = 'middle'

  const textWidth = ctx.measureText(text).width
  const lineGap = 16

  // Separator lines
  ctx.strokeStyle = 'rgba(255, 255, 255, 0.2)'
  ctx.lineWidth = 1

  ctx.beginPath()
  ctx.moveTo(PADDING_X, centerY)
  ctx.lineTo(centerX - textWidth / 2 - lineGap, centerY)
  ctx.stroke()

  ctx.beginPath()
  ctx.moveTo(centerX + textWidth / 2 + lineGap, centerY)
  ctx.lineTo(CANVAS_WIDTH - PADDING_X, centerY)
  ctx.stroke()

  // Text
  ctx.fillText(text, centerX, centerY)
}

function drawRecordCard(
  ctx: CanvasRenderingContext2D,
  x: number, y: number,
  w: number, h: number,
  record: PlayRecordInfo,
  rank: number,
  coverImage: HTMLImageElement | null,
) {
  ctx.save()

  // Clip to rounded rect
  roundedRectPath(ctx, x, y, w, h, CARD_RADIUS)
  ctx.clip()

  // ── Background: blurred cover or fallback ──
  if (coverImage) {
    ctx.save()
    ctx.filter = 'blur(6px)'
    const bleed = 20 // extra pixels to avoid blur edge artifacts
    drawImageCover(ctx, coverImage, x - bleed, y - bleed, w + 2 * bleed, h + 2 * bleed)
    ctx.restore()
  } else {
    ctx.fillStyle = '#1a1a22'
    ctx.fillRect(x, y, w, h)
  }

  // Dark overlay for readability
  ctx.fillStyle = 'rgba(0, 0, 0, 0.42)'
  ctx.fillRect(x, y, w, h)

  // ── Text content ──
  const pad = 10
  const contentX = x + pad
  const contentW = w - 2 * pad

  // Enable text shadow for all card text
  ctx.shadowColor = 'rgba(0, 0, 0, 0.6)'
  ctx.shadowBlur = 3
  ctx.shadowOffsetX = 0
  ctx.shadowOffsetY = 1

  // Title (top-left)
  ctx.font = `bold 14px ${FONT_SANS}`
  ctx.fillStyle = '#ffffff'
  ctx.textAlign = 'left'
  ctx.textBaseline = 'top'
  const rankStr = `#${rank}`
  ctx.font = `bold 13px ${FONT_MONO}`
  const rankWidth = ctx.measureText(rankStr).width
  ctx.font = `bold 16px ${FONT_SANS}`
  const title = truncateText(ctx, record.chart.title, contentW - rankWidth - 8)
  ctx.fillText(title, contentX, y + pad)

  // Rank (top-right)
  ctx.font = `bold 13px ${FONT_MONO}`
  ctx.fillStyle = 'rgba(255, 255, 255, 0.8)'
  ctx.textAlign = 'right'
  ctx.textBaseline = 'top'
  ctx.fillText(rankStr, x + w - pad, y + pad)

  // Difficulty label (below title, left)
  const diffName = DIFF_NAMES[record.chart.difficulty] || record.chart.difficulty.toUpperCase()
  const levelStr = formatDiffLevel(record.chart.level)
  const diffLabel = `${diffName} ${levelStr}`
  ctx.font = `bold 13px ${FONT_MONO}`
  ctx.fillStyle = DIFF_COLORS[record.chart.difficulty] || '#ffffff'
  ctx.textAlign = 'left'
  ctx.textBaseline = 'top'
  ctx.fillText(diffLabel, contentX, y + pad + 22)

  // Bottom left: level > rating
  ctx.shadowBlur = 2
  ctx.shadowOffsetY = 1
  const ratingVal = (record.rating / 100).toFixed(2)
  const levelVal = record.chart.level.toFixed(1)
  ctx.font = `13px ${FONT_MONO}`
  ctx.fillStyle = 'rgba(255, 255, 255, 0.8)'
  ctx.textAlign = 'left'
  ctx.textBaseline = 'bottom'
  ctx.fillText(`${levelVal} > ${ratingVal}`, contentX, y + h - pad)

  // Score (left-aligned, vertically centered)
  ctx.shadowBlur = 5
  ctx.shadowOffsetY = 2
  ctx.font = `bold 32px ${FONT_MONO}`
  ctx.fillStyle = '#ffffff'
  ctx.textAlign = 'left'
  ctx.textBaseline = 'bottom'
  const scoreStr = String(record.score).padStart(7, '0')
  ctx.fillText(scoreStr, contentX, y + h - pad - 16)

  // Reset shadow
  ctx.shadowColor = 'transparent'
  ctx.shadowBlur = 0
  ctx.shadowOffsetX = 0
  ctx.shadowOffsetY = 0

  ctx.restore()
}

function drawFooter(ctx: CanvasRenderingContext2D, y: number) {
  ctx.font = `14px ${FONT_SANS}`
  ctx.fillStyle = 'rgba(255, 255, 255, 0.35)'
  ctx.textAlign = 'center'
  ctx.textBaseline = 'top'
  ctx.fillText('Generated by PRP Web', CANVAS_WIDTH / 2, y)
}

// ─── Card Grid Renderer ────────────────────────────────────────

function drawCardGrid(
  ctx: CanvasRenderingContext2D,
  startY: number,
  records: PlayRecordInfo[],
  imageMap: Map<string, HTMLImageElement>,
) {
  records.forEach((record, i) => {
    const col = i % COLS
    const row = Math.floor(i / COLS)
    const cx = PADDING_X + col * (CARD_WIDTH + CARD_GAP)
    const cy = startY + row * (CARD_HEIGHT + CARD_GAP)
    const img = imageMap.get(record.chart.cover) ?? null
    drawRecordCard(ctx, cx, cy, CARD_WIDTH, CARD_HEIGHT, record, i + 1, img)
  })
}

function gridBlockHeight(recordCount: number): number {
  const rows = Math.ceil(recordCount / COLS)
  return rows > 0 ? rows * CARD_HEIGHT + (rows - 1) * CARD_GAP : 0
}

// ─── Main Render Function ──────────────────────────────────────

export async function renderB50Image(options: B50RenderOptions): Promise<Blob> {
  // Ensure web fonts are loaded before rendering
  await document.fonts.ready

  const { b15Records, b35Records } = options

  // ── Calculate total canvas height ──
  const b15BlockH = gridBlockHeight(b15Records.length)
  const b35BlockH = gridBlockHeight(b35Records.length)

  const canvasHeight =
    PADDING_TOP +
    HEADER_HEIGHT +
    GAP_AFTER_HEADER +
    SECTION_TITLE_HEIGHT +
    GAP_AFTER_SECTION_TITLE +
    b15BlockH +
    GAP_BETWEEN_SECTIONS +
    SECTION_TITLE_HEIGHT +
    GAP_AFTER_SECTION_TITLE +
    b35BlockH +
    GAP_BETWEEN_SECTIONS +
    FOOTER_HEIGHT +
    PADDING_BOTTOM

  // ── Create canvas ──
  const canvas = document.createElement('canvas')
  canvas.width = CANVAS_WIDTH
  canvas.height = canvasHeight
  const ctx = canvas.getContext('2d')!

  // ── Preload cover images ──
  const allRecords = [...b15Records, ...b35Records]
  const imageMap = await preloadImages(allRecords)

  // ── Background ──
  const bgCover = b15Records[0]?.chart.cover || b35Records[0]?.chart.cover
  const bgImage = bgCover ? imageMap.get(bgCover) ?? null : null
  drawBackground(ctx, CANVAS_WIDTH, canvasHeight, bgImage)

  // ── Header ──
  let curY = PADDING_TOP
  drawHeader(ctx, curY, options)
  curY += HEADER_HEIGHT + GAP_AFTER_HEADER

  // ── Best 15 section ──
  const b15TitleY = curY + SECTION_TITLE_HEIGHT / 2
  drawSectionTitle(ctx, b15TitleY, 'Best 15', options.b15Avg)
  curY += SECTION_TITLE_HEIGHT + GAP_AFTER_SECTION_TITLE

  drawCardGrid(ctx, curY, b15Records, imageMap)
  curY += b15BlockH + GAP_BETWEEN_SECTIONS

  // ── Best 35 section ──
  const b35TitleY = curY + SECTION_TITLE_HEIGHT / 2
  drawSectionTitle(ctx, b35TitleY, 'Best 35', options.b35Avg)
  curY += SECTION_TITLE_HEIGHT + GAP_AFTER_SECTION_TITLE

  drawCardGrid(ctx, curY, b35Records, imageMap)
  curY += b35BlockH + GAP_BETWEEN_SECTIONS

  // ── Footer ──
  drawFooter(ctx, curY)

  // ── Export as JPEG blob ──
  return new Promise<Blob>((resolve, reject) => {
    canvas.toBlob(
      (blob) => (blob ? resolve(blob) : reject(new Error('Canvas toBlob failed'))),
      'image/jpeg',
      0.92,
    )
  })
}
