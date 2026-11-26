// Locale detection (redefinition §5.1). Order: user setting > stored > browser.
// Never force-redirects; the banner only suggests.
export const SUPPORTED_LOCALES = ['ko', 'en', 'ja', 'zh-cn', 'th'] as const;
export type SupportedLocale = (typeof SUPPORTED_LOCALES)[number];

const STORE_KEY = 'avibe.locale';

/** Map a browser language tag to a supported locale code. */
export function browserToLocale(tag: string | undefined | null): SupportedLocale {
  const t = (tag ?? '').toLowerCase();
  if (t.startsWith('ko')) return 'ko';
  if (t.startsWith('ja')) return 'ja';
  if (t.startsWith('th')) return 'th';
  if (t.startsWith('zh-hant') || t.startsWith('zh-tw') || t.startsWith('zh-hk')) return 'en';
  if (t.startsWith('zh')) return 'zh-cn';
  if (t.startsWith('en')) return 'en';
  return 'en';
}

/** Best browser match from navigator.languages. */
export function detectBrowserLocale(langs: readonly string[] | undefined): SupportedLocale {
  for (const tag of langs ?? []) {
    const t = tag.toLowerCase();
    if (t.startsWith('ko') || t.startsWith('ja') || t.startsWith('th') || t.startsWith('zh') || t.startsWith('en')) {
      return browserToLocale(t);
    }
  }
  return 'en';
}

export function readStoredLocale(): SupportedLocale | null {
  try {
    const v = localStorage.getItem(STORE_KEY);
    return (SUPPORTED_LOCALES as readonly string[]).includes(v ?? '') ? (v as SupportedLocale) : null;
  } catch {
    return null;
  }
}

export function writeStoredLocale(code: string): void {
  try {
    localStorage.setItem(STORE_KEY, code);
  } catch {
    // Private mode: locale just doesn't persist.
  }
}

/** Full resolution: explicit user setting wins, then stored, then browser. */
export function resolveLocale(userLocale: string | null | undefined, langs: readonly string[] | undefined): SupportedLocale {
  if (userLocale && (SUPPORTED_LOCALES as readonly string[]).includes(userLocale)) {
    return userLocale as SupportedLocale;
  }
  return readStoredLocale() ?? detectBrowserLocale(langs);
}

/** Intl locale for date/number formats (route codes differ: zh-cn). */
export function intlLocale(code: string): string {
  if (code === 'zh-cn') return 'zh-CN';
  return code;
}

// Base route names of public pages (prefix or ko-default): the URL wins and
// the server locale must not override them. Keep in sync with routeRules SSR
// list as R3+ adds pages.
export const PUBLIC_BASE_NAMES = new Set([
  'home',
  'explore',
  'charts',
  'playlists',
  'playlist-detail',
  'track-detail',
  'pricing',
  'business',
  'for-creators',
  'trust',
  'create',
  'legal-terms',
  'legal',
  'certificate-verify',
  'certificate',
  'explore-genre-genre',
  'explore-mood-mood',
  'explore-use-useCase',
  'creators',
  'creators-handle',
  'guides',
  'guides-slug',
  'models',
  'models-slug',
  'about',
  'help',
  'contact',
  'faq',
  'support',
  'search',
  'tracks',
  'embed-track-shortId',
  'embed-playlist-shortId',
  'not-found',
]);

/** True when the path carries an explicit locale prefix (/en/… — URL wins). */
export function isPrefixedPath(path: string): boolean {
  const first = path.split('/')[1];
  return ['en', 'ja', 'zh-cn', 'th'].includes(first ?? '');
}

/**
 * Server-locale application (A2, detection order ① users.locale). Returns the
 * locale to switch to, or null to keep the current one. Public pages and
 * prefixed paths keep URL-wins and never switch.
 */
export function shouldApplyServerLocale(
  serverLocale: string | null | undefined,
  current: string,
  path: string,
  baseName: string,
): SupportedLocale | null {
  if (!serverLocale || !(SUPPORTED_LOCALES as readonly string[]).includes(serverLocale)) return null;
  if (serverLocale === current) return null;
  if (PUBLIC_BASE_NAMES.has(baseName) || isPrefixedPath(path)) return null;
  return serverLocale as SupportedLocale;
}
