// Sitemap URL source for @nuxtjs/sitemap (WI-S3).
// DB-backed types proxy the engine cursor API (GET /api/v1/seo/sitemap/{type})
// and walk every page so the module receives the full URL set. Static/guide
// types resolve locally (no engine needed). Cached 1h like the engine.
const TYPES = new Set(['tracks', 'playlists', 'creators', 'landings', 'guides', 'static']);

// NOTE: keep in sync with service/seo.go guideEntries/staticEntries.
const LOCAL_LASTMOD = '2026-11-16T00:00:00Z';
const LOCAL: Record<string, { loc: string; lastmod: string }[]> = {
  guides: [
    'store-music-license',
    'suno-paid-commercial-use',
    'ai-music-youtube-checklist',
  ].map((slug) => ({ loc: `/guides/${slug}`, lastmod: LOCAL_LASTMOD })),
  static: [
    '/', '/explore', '/charts', '/playlists', '/pricing', '/business', '/for-creators', '/trust', '/create', '/guides',
    '/about', '/help', '/contact', '/creators', '/certificate', '/models',
    '/models/avibe', '/models/suno', '/models/udio', '/models/aiva', '/models/stable-audio', '/models/elevenlabs',
    '/models/open-model', '/models/soundraw', '/models/mubert',
  ].map(
    (loc) => ({ loc, lastmod: LOCAL_LASTMOD }),
  ),
};

export default defineCachedEventHandler(
  async (event) => {
    const q = getQuery(event);
    const type = String(q.type ?? '');
    const HREFLANGS = [
      { code: 'ko', hreflang: 'ko-KR', prefix: '' },
      { code: 'en', hreflang: 'en-US', prefix: '/en' },
      { code: 'ja', hreflang: 'ja-JP', prefix: '/ja' },
      { code: 'zh-cn', hreflang: 'zh-CN', prefix: '/zh-cn' },
      { code: 'th', hreflang: 'th-TH', prefix: '/th' },
    ];
    // Expand one unprefixed loc into 5 locale URLs, each carrying the full
    // alternate set (x-default = /en per spec §3). Explicit alternatives
    // replace the module auto-i18n for these sources.
    const expand = (loc: string, lastmod: string) => {
      const clean = loc === '' ? '/' : loc;
      const hrefs = HREFLANGS.map((h) => ({
        hreflang: h.hreflang,
        href: h.prefix === '' ? clean : clean === '/' ? `${h.prefix}/` : `${h.prefix}${clean}`,
      }));
      const enHref = hrefs[1]?.href ?? clean;
      const alternatives = [...hrefs, { hreflang: 'x-default', href: enHref }];
      return hrefs.map((h) => ({ loc: h.href, lastmod, alternatives }));
    };
    if (!TYPES.has(type)) {
      throw createError({ statusCode: 400, statusMessage: 'unknown sitemap type' });
    }
    if (LOCAL[type]) return LOCAL[type].flatMap((it) => expand(it.loc, it.lastmod));
    const { apiOrigin } = useRuntimeConfig(event);
    const base = String(apiOrigin || 'http://localhost:18081');
    const out: { loc: string; lastmod: string; alternatives: { hreflang: string; href: string }[] }[] = [];
    let cursor = '';
    for (let page = 0; page < 25; page++) {
      const url = `${base}/api/v1/seo/sitemap/${type}?limit=2000${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ''}`;
      const res = await $fetch<{ data?: { items?: { loc?: string; lastmod?: string }[]; nextCursor?: string } }>(url).catch(
        () => null,
      );
      const items = res?.data?.items ?? [];
      for (const it of items) {
        if (it?.loc) out.push(...expand(it.loc, it.lastmod ?? ''));
      }
      cursor = res?.data?.nextCursor ?? '';
      if (!cursor || items.length === 0) break;
    }
    return out;
  },
  { maxAge: 3600 },
);
