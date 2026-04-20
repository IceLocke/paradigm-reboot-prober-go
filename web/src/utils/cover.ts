/**
 * Cover image URL helpers.
 *
 * Covers live under `/cover/<name>.<ext>` (served either from `web/public/cover/`
 * in dev or from the static file handler in prod). Every full-size cover has a
 * matching thumbnail at `/cover/<name>_thumb.<ext>` that is ~1/10 the byte size,
 * which is what we want to use in list/grid views to keep bandwidth down.
 */

/** Return the URL for the full-size cover. Passes absolute URLs through. */
export function coverUrl(cover: string | null | undefined): string {
  if (!cover) return ''
  if (cover.startsWith('http')) return cover
  return `/cover/${cover}`
}

/**
 * Return the URL for the thumbnail variant.
 *
 * `Cover_abc.jpg` → `/cover/Cover_abc_thumb.jpg`
 * `Cover_abc.png` → `/cover/Cover_abc_thumb.png`
 *
 * Absolute URLs (http/https) are returned as-is — we don't know their layout.
 * If the cover string has no extension, the original cover URL is returned.
 */
export function coverThumbUrl(cover: string | null | undefined): string {
  if (!cover) return ''
  if (cover.startsWith('http')) return cover
  const dot = cover.lastIndexOf('.')
  if (dot === -1) return `/cover/${cover}`
  return `/cover/${cover.substring(0, dot)}_thumb${cover.substring(dot)}`
}
