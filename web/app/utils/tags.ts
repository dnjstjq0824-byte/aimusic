// Canonical tag display (R5). The engine stores English keys; the UI shows
// tags.* translations. Unknown/legacy values render as-is, never blank.
export function tagDisplayKey(raw: string | null | undefined): string | null {
  const key = (raw ?? '').trim().toLowerCase();
  return key === '' ? null : key;
}

/** Resolve with a t/te pair (useI18n) so SSR + locale switches stay reactive. */
export function tagDisplayName(
  raw: string | null | undefined,
  t: (key: string) => string,
  te: (key: string) => boolean,
): string {
  const key = tagDisplayKey(raw);
  if (!key) return '';
  const path = `tags.${key}`;
  return te(path) ? String(t(path)) : (raw ?? '').trim();
}
