/**
 * B50 image canvas renderer.
 * Generates a Best Records image using Canvas 2D API.
 */
import type { PlayRecordInfo } from '@/api/types'
import { coverUrl as coverFullUrl, coverThumbUrl } from '@/utils/cover'
import { formatRating } from '@/utils/rating'

// ─── Render Options ────────────────────────────────────────────

export interface B50Section {
  label: string
  avg: string
  records: PlayRecordInfo[]
  /** Records with index >= cutoff are rendered dimmed (overflow/floor). */
  cutoff?: number
}

export interface B50RenderOptions {
  sections: B50Section[]
  username: string
  nickname: string
  rating: string
  title?: string
}

// ─── Layout Constants ──────────────────────────────────────────

const CANVAS_WIDTH = 1200
const PADDING_X = 32
const PADDING_TOP = 32
const PADDING_BOTTOM = 24

const COLS = 5
const CARD_GAP = 16
const CARD_WIDTH = (CANVAS_WIDTH - 2 * PADDING_X - (COLS - 1) * CARD_GAP) / COLS
const CARD_HEIGHT = 136
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
  detected: "DETECTED",
  invaded: 'INVADED',
  massive: 'MASSIVE',
  reboot: 'REBOOT',
}

// ─── Fonts ─────────────────────────────────────────────────────

const FONT_SANS = "'Inter', 'Noto Sans SC', sans-serif"
const FONT_MONO = "'JetBrains Mono', monospace"
const FONT_SCORE = "'Anta', sans-serif"

// ─── Helpers ───────────────────────────────────────────────────

function formatDiffLevel(level: number): string {
  const base = Math.floor(level)
  const plus = level - base >= 0.6 ? '+' : ''
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

/** Draw image with CSS `object-fit: cover` behaviour, biased slightly upward */
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
    // Bias crop toward the top (0.35) so cover art's visual center is preserved
    sy = (img.naturalHeight - sh) * 0.3
  }
  ctx.drawImage(img, sx, sy, sw, sh, x, y, w, h)
}

/** Detect whether the browser actually renders CanvasRenderingContext2D.filter */
const supportsFilter = (() => {
  try {
    const c = document.createElement('canvas')
    c.width = 3
    c.height = 1
    const ctx = c.getContext('2d')
    if (!ctx) return false
    ctx.filter = 'blur(1px)'
    ctx.fillStyle = '#fff'
    ctx.fillRect(1, 0, 1, 1)
    // If blur actually renders, white bleeds into the neighboring pixel
    return ctx.getImageData(0, 0, 1, 1).data[3] > 0
  } catch {
    return false
  }
})()

/**
 * Draw a cover image with blur.
 * Uses native ctx.filter when available, otherwise falls back to an
 * iterative downscale/upscale trick (fast, pure drawImage, no pixel ops).
 */
function drawImageCoverBlurred(
  ctx: CanvasRenderingContext2D,
  img: HTMLImageElement,
  x: number, y: number, w: number, h: number,
  blur: number,
) {
  if (supportsFilter) {
    ctx.save()
    ctx.filter = `blur(${blur}px)`
    drawImageCover(ctx, img, x, y, w, h)
    ctx.restore()
    return
  }

  // Fallback: iterative downscale/upscale.
  // Each pass roughly doubles the effective blur radius.
  const offscreen = document.createElement('canvas')
  offscreen.width = w
  offscreen.height = h
  const octx = offscreen.getContext('2d')!
  drawImageCover(octx, img, 0, 0, w, h)

  const passes = Math.max(1, Math.ceil(Math.log2(blur)))
  for (let i = 0; i < passes; i++) {
    const hw = Math.max(1, (offscreen.width / 2) | 0)
    const hh = Math.max(1, (offscreen.height / 2) | 0)
    octx.drawImage(offscreen, 0, 0, offscreen.width, offscreen.height, 0, 0, hw, hh)
    octx.drawImage(offscreen, 0, 0, hw, hh, 0, 0, offscreen.width, offscreen.height)
  }

  ctx.drawImage(offscreen, 0, 0, w, h, x, y, w, h)
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

/** Build the URL for a cover thumbnail (Cover_xxx.jpg → Cover_xxx_thumb.jpg). */
const coverThumbPath = coverThumbUrl

async function preloadImages(
  records: PlayRecordInfo[],
): Promise<Map<string, HTMLImageElement>> {
  const covers = new Set<string>()
  for (const r of records) {
    if (r.chart.cover) covers.add(r.chart.cover)
  }

  const map = new Map<string, HTMLImageElement>()
  const tasks = Array.from(covers).map(async (cover) => {
    // Load thumbnail variant for B50 cards (each card is ~268×160px, no need for full-size)
    const img = await loadImage(coverThumbPath(cover))
      ?? await loadImage(coverFullUrl(cover)) // fallback to full-size if thumb not found
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
  if (bgImage) {
    // Dark base beneath the blurred cover
    ctx.fillStyle = '#0e0e12'
    ctx.fillRect(0, 0, width, height)

    drawImageCover(ctx, bgImage, 0, 0, width, height)

    // Semi-transparent overlay to ensure text readability
    ctx.fillStyle = 'rgba(14, 14, 18, 0.45)'
    ctx.fillRect(0, 0, width, height)
  } else {
    // No cover image — use a slightly lighter tinted background
    ctx.fillStyle = '#1a1a2e'
    ctx.fillRect(0, 0, width, height)
  }
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
  ctx.fillText(options.title || 'Paradigm: Reboot Best Records', leftX, y)

  // Left: date/time
  ctx.font = `16px ${FONT_SANS}`
  ctx.fillStyle = '#a1a1aa'
  ctx.fillText(formatDate(), leftX, y + 38)

  // Right: nickname (display name)
  ctx.font = `bold 28px ${FONT_SANS}`
  ctx.fillStyle = '#ffffff'
  ctx.textAlign = 'right'
  ctx.fillText(options.nickname || options.username, rightX, y)

  // Right: rating
  ctx.font = `20px ${FONT_MONO}`
  ctx.fillStyle = '#ffffff'
  ctx.fillText(`Rating: ${options.rating}`, rightX, y + 38)
}

function drawSectionTitle(
  ctx: CanvasRenderingContext2D,
  centerY: number,
  label: string,
  avg: string,
) {
  const text = `${label} / Avg. ${avg}`
  const centerX = CANVAS_WIDTH / 2

  ctx.font = `18px ${FONT_SANS}`
  ctx.fillStyle = '#ffffff'
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
  isOverflow = false,
) {
  // ── Card shadow (drawn before clip so it's visible outside) ──
  ctx.save()
  ctx.shadowColor = 'rgba(0, 0, 0, 0.45)'
  ctx.shadowBlur = 10
  ctx.shadowOffsetX = 0
  ctx.shadowOffsetY = 3
  roundedRectPath(ctx, x, y, w, h, CARD_RADIUS)
  ctx.fillStyle = '#000'
  ctx.fill()
  ctx.restore()

  // ── Clip to rounded rect for content ──
  ctx.save()
  roundedRectPath(ctx, x, y, w, h, CARD_RADIUS)
  ctx.clip()

  // ── Background: blurred cover or fallback ──
  if (coverImage) {
    const bleed = 20 // extra pixels to avoid blur edge artifacts
    drawImageCoverBlurred(ctx, coverImage, x - bleed, y - bleed, w + 2 * bleed, h + 2 * bleed, 5)
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
  ctx.font = `bold 20px ${FONT_SANS}`
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
  ctx.fillText(diffLabel, contentX, y + pad + 24)

  // Bottom left: level > rating
  ctx.shadowBlur = 2
  ctx.shadowOffsetY = 1
  const ratingVal = formatRating(record.rating)
  const levelVal = record.chart.level.toFixed(1)
  ctx.font = `bold 14px ${FONT_MONO}`
  ctx.fillStyle = '#ffffff'
  ctx.textAlign = 'left'
  ctx.textBaseline = 'bottom'
  ctx.fillText(`${levelVal} → ${ratingVal}`, contentX, y + h - pad)

  // Score (left-aligned, vertically centered)
  ctx.shadowBlur = 5
  ctx.shadowOffsetY = 2
  ctx.font = `bold 28px ${FONT_SCORE}`
  ctx.fillStyle = '#ffffff'
  ctx.textAlign = 'left'
  ctx.textBaseline = 'bottom'
  const scoreStr = String(record.score).padStart(7, '0')
  ctx.fillText(scoreStr, contentX, y + h - pad - 17)

  // Reset shadow
  ctx.shadowColor = 'transparent'
  ctx.shadowBlur = 0
  ctx.shadowOffsetX = 0
  ctx.shadowOffsetY = 0

  // Dim overflow (floor) cards so the B50 boundary is visually obvious
  if (isOverflow) {
    ctx.fillStyle = 'rgba(0, 0, 0, 0.38)'
    ctx.fillRect(x, y, w, h)
  }

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
  cutoff?: number,
) {
  records.forEach((record, i) => {
    const col = i % COLS
    const row = Math.floor(i / COLS)
    const cx = PADDING_X + col * (CARD_WIDTH + CARD_GAP)
    const cy = startY + row * (CARD_HEIGHT + CARD_GAP)
    const img = imageMap.get(record.chart.cover) ?? null
    const isOverflow = cutoff != null && i >= cutoff
    drawRecordCard(ctx, cx, cy, CARD_WIDTH, CARD_HEIGHT, record, i + 1, img, isOverflow)
  })
}

function gridBlockHeight(recordCount: number): number {
  const rows = Math.ceil(recordCount / COLS)
  return rows > 0 ? rows * CARD_HEIGHT + (rows - 1) * CARD_GAP : 0
}

// ─── Main Render Function ──────────────────────────────────────

export async function renderB50Image(options: B50RenderOptions): Promise<Blob> {
  // Eagerly load all fonts used on the canvas.
  // document.fonts.ready alone is insufficient — the browser won't download a
  // font that isn't referenced by any DOM element, so Anta (used only on the
  // canvas for scores) would be missing on the first render.
  await Promise.all([
    document.fonts.load(`bold 28px ${FONT_SCORE}`),
    document.fonts.load(`16px ${FONT_SANS}`),
    document.fonts.load(`16px ${FONT_MONO}`),
  ])

  const { sections } = options

  // ── Calculate total canvas height ──
  let contentHeight = 0
  for (let i = 0; i < sections.length; i++) {
    const section = sections[i]
    contentHeight += SECTION_TITLE_HEIGHT + GAP_AFTER_SECTION_TITLE + gridBlockHeight(section.records.length)
    if (i < sections.length - 1) {
      contentHeight += GAP_BETWEEN_SECTIONS
    }
  }

  const canvasHeight =
    PADDING_TOP +
    HEADER_HEIGHT +
    GAP_AFTER_HEADER +
    contentHeight +
    PADDING_BOTTOM +
    FOOTER_HEIGHT +
    PADDING_BOTTOM

  // ── Create canvas ──
  const canvas = document.createElement('canvas')
  canvas.width = CANVAS_WIDTH
  canvas.height = canvasHeight
  const ctx = canvas.getContext('2d')!

  // ── Preload cover images ──
  const allRecords = sections.flatMap((s) => s.records)
  const imageMap = await preloadImages(allRecords)

  // ── Background (fixed image) ──
  const bgImage = await loadImage('/b50-bg.jpg')
  drawBackground(ctx, CANVAS_WIDTH, canvasHeight, bgImage)

  // ── Header ──
  let curY = PADDING_TOP
  drawHeader(ctx, curY, options)
  curY += HEADER_HEIGHT + GAP_AFTER_HEADER

  // ── Sections ──
  for (let i = 0; i < sections.length; i++) {
    const section = sections[i]
    const titleY = curY + SECTION_TITLE_HEIGHT / 2
    drawSectionTitle(ctx, titleY, section.label, section.avg)
    curY += SECTION_TITLE_HEIGHT + GAP_AFTER_SECTION_TITLE

    drawCardGrid(ctx, curY, section.records, imageMap, section.cutoff)
    curY += gridBlockHeight(section.records.length)

    if (i < sections.length - 1) {
      curY += GAP_BETWEEN_SECTIONS
    }
  }

  // ── Footer ──
  curY += PADDING_BOTTOM
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
