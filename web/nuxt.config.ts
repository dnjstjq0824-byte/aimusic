// Nuxt 4 app config (WI-P1). Public pages SSR, app screens CSR (SEO §2).
// Build targets: node-server (default, local verify) and cloudflare-module
// via NITRO_PRESET=cloudflare-module (SEO §1, WI-P1 dual-build).
const LEGACY_NAMES: Record<string, string> = {
  index: 'home',
  'tracks-id': 'track-detail',
  'playlists-id': 'playlist-detail',
  'studio': 'studio-dashboard',
  'admin-index': 'admin-overview',
  'legal-code': 'legal-terms',
  'certificate-number': 'certificate-verify',
  'oauth-callback': 'oauth-callback',
  'account-security': 'account-security',
  'store-player': 'store-player',
  'checkout-confirm': 'checkout-confirm',
  'my-library': 'my-library',
  'reset-password': 'reset-password',
  slug: 'not-found',
  500: 'server-error',
};

export default defineNuxtConfig({
  srcDir: 'app',
  compatibilityDate: '2025-07-15',
  devtools: { enabled: false },
  modules: [
    '@pinia/nuxt',
    '@nuxtjs/i18n',
    'nuxt-site-config',
    '@nuxtjs/sitemap',
    '@nuxtjs/robots',
    'nuxt-schema-org',
    'nuxt-og-image',
  ],
  site: {
    url: 'https://avibemusic.com',
    name: 'AVIBE',
    description: 'AVIBE — verified commercial-use AI music streaming.',
    defaultLocale: 'ko',
  },
  // WI-S3: locale x type sitemaps from the engine cursor API (S3 §6.1).
  sitemap: {
    autoI18n: true,
    exclude: ['/api/**', '/login', '/signup/**', '/forgot-password', '/reset-password', '/verify-email', '/oauth/**', '/library', '/me/**', '/account/**', '/rights-center', '/checkout/**', '/partner/**', '/studio/**', '/admin/**', '/store/**', '/business/console/**', '/500', '/dev/**'],
    sitemaps: {
      tracks: { sources: ['/__sitemap__/engine-urls?type=tracks'] },
      playlists: { sources: ['/__sitemap__/engine-urls?type=playlists'] },
      creators: { sources: ['/__sitemap__/engine-urls?type=creators'] },
      landings: { sources: ['/__sitemap__/engine-urls?type=landings'] },
      guides: { sources: ['/__sitemap__/engine-urls?type=guides'] },
      static: { sources: ['/__sitemap__/engine-urls?type=static'] },
    },
  },
  // WI-S3: SEO §2 + §6.2 union. /embed stays crawlable (noindex via meta).
  // AI search/answer bots are allowed on public text pages (spec §6.2).
  robots: {
    sitemap: ['/sitemap_index.xml'],
    allow: ['/'],
    disallow: [
      '/api/', '/account', '/me/', '/library', '/rights-center', '/studio', '/partner',
      '/business/console', '/admin', '/store/player', '/checkout', '/login', '/signup',
      '/forgot-password', '/reset-password', '/verify-email', '/oauth/', '/500',
    ],
    blockAiBots: false,
    blockNonSeoBots: false,
  },
  // WI-S2: global WebSite + Organization identity (sameAs deferred to PO SNS).
  schemaOrg: {
    identity: {
      type: 'Organization',
      name: 'AVIBE',
      url: 'https://avibemusic.com',
      logo: 'https://avibemusic.com/favicon.svg',
    },
  },
  // WI-S2: dynamic OG images (Satori + resvg, 1200x630, Thai/CJK textless fallback).
  ogImage: {
    enabled: true,
    defaults: { width: 1200, height: 630, cacheMaxAgeSeconds: 86400 },
  },
  i18n: {
    strategy: 'prefix_except_default',
    defaultLocale: 'ko',
    detectBrowserLanguage: false,
    baseUrl: 'https://avibemusic.com',
    langDir: '../i18n/locales',
    locales: [
      { code: 'ko', language: 'ko-KR', files: ['ko/common.json', 'ko/errors.json', 'ko/player.json', 'ko/auth.json', 'ko/home.json', 'ko/explore.json', 'ko/track.json', 'ko/tags.json', 'ko/library.json', 'ko/playlist.json', 'ko/pricing.json', 'ko/checkout.json', 'ko/account.json', 'ko/legal.json', 'ko/studio.json', 'ko/business.json', 'ko/generate.json', 'ko/admin.json', 'ko/seo.json'] },
      { code: 'en', language: 'en-US', files: ['en/common.json', 'en/errors.json', 'en/player.json', 'en/auth.json', 'en/home.json', 'en/explore.json', 'en/track.json', 'en/tags.json', 'en/library.json', 'en/playlist.json', 'en/pricing.json', 'en/checkout.json', 'en/account.json', 'en/legal.json', 'en/studio.json', 'en/business.json', 'en/generate.json', 'en/admin.json', 'en/seo.json'] },
      { code: 'ja', language: 'ja-JP', files: ['ja/common.json', 'ja/errors.json', 'ja/player.json', 'ja/auth.json', 'ja/home.json', 'ja/explore.json', 'ja/track.json', 'ja/tags.json', 'ja/library.json', 'ja/playlist.json', 'ja/pricing.json', 'ja/checkout.json', 'ja/account.json', 'ja/legal.json', 'ja/studio.json', 'ja/business.json', 'ja/generate.json', 'ja/admin.json', 'ja/seo.json'] },
      { code: 'zh-cn', language: 'zh-CN', files: ['zh-CN/common.json', 'zh-CN/errors.json', 'zh-CN/player.json', 'zh-CN/auth.json', 'zh-CN/home.json', 'zh-CN/explore.json', 'zh-CN/track.json', 'zh-CN/tags.json', 'zh-CN/library.json', 'zh-CN/playlist.json', 'zh-CN/pricing.json', 'zh-CN/checkout.json', 'zh-CN/account.json', 'zh-CN/legal.json', 'zh-CN/studio.json', 'zh-CN/business.json', 'zh-CN/generate.json', 'zh-CN/admin.json', 'zh-CN/seo.json'] },
      { code: 'th', language: 'th-TH', files: ['th/common.json', 'th/errors.json', 'th/player.json', 'th/auth.json', 'th/home.json', 'th/explore.json', 'th/track.json', 'th/tags.json', 'th/library.json', 'th/playlist.json', 'th/pricing.json', 'th/checkout.json', 'th/account.json', 'th/legal.json', 'th/studio.json', 'th/business.json', 'th/generate.json', 'th/admin.json', 'th/seo.json'] },
      ...(process.dev
        ? [{ code: 'en-XA', language: 'en-XA', files: ['en-XA/common.json', 'en-XA/errors.json', 'en-XA/player.json', 'en-XA/auth.json', 'en-XA/home.json', 'en-XA/explore.json', 'en-XA/track.json', 'en-XA/tags.json', 'en-XA/library.json', 'en-XA/playlist.json', 'en-XA/account.json', 'en-XA/legal.json', 'en-XA/pricing.json', 'en-XA/business.json', 'en-XA/studio.json', 'en-XA/seo.json'] }]
        : []),
    ],
    vueI18n: './i18n.config.ts',
  },
  css: ['~/assets/css/fonts.css', '~/assets/css/main.css'],
  postcss: {
    plugins: {
      tailwindcss: {},
      autoprefixer: {},
    },
  },
  devServer: { port: 15173 },
  runtimeConfig: {
    // Server-only origins (WI-S3/S5): SSR data fetch + IndexNow key proof.
    apiOrigin: process.env.NUXT_API_PROXY || process.env.VITE_API_PROXY || 'http://localhost:18081',
    indexnowKey: process.env.INDEXNOW_KEY || '',
    public: {
      apiBaseUrl: process.env.VITE_API_BASE_URL || '/api/v1',
      tossClientKey: process.env.VITE_TOSS_CLIENT_KEY || '',
      // Search-console verification tokens (public by design, PO-provided).
      verifyGoogle: process.env.NUXT_PUBLIC_VERIFY_GOOGLE || '',
      verifyBing: process.env.NUXT_PUBLIC_VERIFY_BING || '',
      verifyNaver: process.env.NUXT_PUBLIC_VERIFY_NAVER || '',
      verifyDaum: process.env.NUXT_PUBLIC_VERIFY_DAUM || '',
      cfWebAnalyticsToken: process.env.NUXT_PUBLIC_CF_WEB_ANALYTICS_TOKEN || '',
      // WI-G5: studio generation panel v2 (PromptBox/StyleTags/LyricsEditor/
      // VariationCard). Default OFF — operational screens stay unchanged.
      generateUi: process.env.NUXT_PUBLIC_GENERATE_UI === '1',
    },
  },
  // CI only: `nuxt preview` (node-server) does not compress, so Lighthouse
  // measured raw JS against the gzip budget. Prod is compressed by Cloudflare,
  // so the shipped image stays unchanged unless this flag is set.
  nitro: {
    compressPublicAssets: process.env.NUXT_COMPRESS_PUBLIC_ASSETS === '1',
  },
  routeRules: {
    // Local/node API origin (build-time env; the edge worker serves /api in prod).
    '/api/**': {
      proxy: `${process.env.NUXT_API_PROXY || process.env.VITE_API_PROXY || 'http://localhost:18081'}/api/**`,
    },
    // SEO §2: public pages SSR+SWR, everything else CSR. Query-based rules
    // (/explore?q noindex) land in S2 via page head.
    '/': { ssr: true, swr: 600 },
    '/explore': { swr: 600 },
    '/explore/**': { swr: 600 },
    '/charts': { swr: 600 },
    '/playlists': { swr: 600 },
    '/playlists/**': { swr: 3600 },
    '/tracks/**': { swr: 3600 },
    '/creators/**': { swr: 3600 },
    '/certificate/**': { swr: 3600 },
    '/legal/**': { swr: 86400 },
    '/business': { swr: 600 },
    '/for-creators': { swr: 600 },
    '/trust': { swr: 600 },
    '/pricing': { swr: 600 },
    '/guides': { swr: 600 },
    '/guides/**': { swr: 600 },
    '/create': { swr: 600 },
    '/embed/**': { swr: 3600, robots: 'noindex, follow' },
    '/login': { ssr: false, robots: 'noindex, follow' },
    '/signup/**': { ssr: false, robots: 'noindex, follow' },
    '/forgot-password': { ssr: false, robots: 'noindex, follow' },
    '/reset-password': { ssr: false, robots: 'noindex, follow' },
    '/verify-email': { ssr: false, robots: 'noindex, follow' },
    '/oauth/**': { ssr: false, robots: 'noindex, follow' },
    '/my-library': { ssr: false, robots: 'noindex, follow' },
    '/library': { robots: 'noindex, follow' },
    '/me/**': { ssr: false, robots: 'noindex, follow' },
    '/account/**': { ssr: false, robots: 'noindex, follow' },
    '/rights-center': { ssr: false, robots: 'noindex, follow' },
    '/checkout/**': { ssr: false, robots: 'noindex, follow' },
    '/partner/**': { ssr: false, robots: 'noindex, follow' },
    '/studio/**': { ssr: false, robots: 'noindex, follow' },
    '/admin/**': { ssr: false, robots: 'noindex, follow' },
    '/store/**': { ssr: false, robots: 'noindex, follow' },
    '/business/console/**': { ssr: false, robots: 'noindex, follow' },
    '/500': { ssr: false, robots: 'noindex, follow' },
    // WI-S5: extensionless Apple association file must serve as JSON.
    '/.well-known/apple-app-site-association': { headers: { 'content-type': 'application/json' } },
    // NOTE: no '/**' catch-all — it overrides specific rules. Default is SSR;
    // every app/console path above is explicitly CSR.
  },
  app: {
    head: {
      // NOTE: <html lang> is managed by @nuxtjs/i18n per route. Do not set statically.
      charset: 'utf-8',
      viewport: 'width=device-width, initial-scale=1.0, viewport-fit=cover',
      title: 'AVIBE — Beyond Your Music',
      meta: [
        { name: 'theme-color', content: '#17161c' },
        {
          name: 'description',
          content:
            'AVIBE는 상업 이용 조건 확인 절차를 거친 AI 음원을 유통하는 스트리밍 플랫폼입니다. Suno·Udio·AIVA 등 7개 AI 프로바이더 음원을 듣고, 플레이리스트에 담으세요.',
        },
        { property: 'og:type', content: 'website' },
        { property: 'og:site_name', content: 'AVIBE' },
        { property: 'og:title', content: 'AVIBE — Beyond Your Music' },
        {
          property: 'og:description',
          content: '상업용 라이선스 4단계 검증 게이트를 통과한 AI 음원만 스트리밍합니다.',
        },
        { property: 'og:image', content: '/favicon.svg' },
        { property: 'og:locale', content: 'ko_KR' },
        { name: 'twitter:card', content: 'summary' },
        { name: 'twitter:title', content: 'AVIBE — Beyond Your Music' },
        { name: 'twitter:description', content: '상업용 라이선스가 검증된 AI 음원 스트리밍 플랫폼' },
      ],
      link: [
        { rel: 'icon', type: 'image/svg+xml', href: '/favicon.svg' },
        { rel: 'canonical', href: 'https://avibemusic.com/' },
        { rel: 'preload', href: '/fonts/pretendard/subset/PretendardVariable.subset.91.woff2', as: 'font', type: 'font/woff2', crossorigin: '' },
      ],
    },
  },
  hooks: {
    'pages:extend'(pages) {
      const rename = (list: { name?: string | null; children?: unknown[] }[]) => {
        for (const p of list) {
          if (p.name && LEGACY_NAMES[p.name]) p.name = LEGACY_NAMES[p.name];
          if (p.children) rename(p.children as { name?: string | null }[]);
        }
      };
      rename(pages);
    },
    // Bundle diet (2026-11-21): no <link rel="prefetch"> for lazy chunks
    // (other pages, player UI, web-vitals). On mobile they competed with the
    // first render for bandwidth; chunks still load on demand.
    'build:manifest'(manifest) {
      for (const key in manifest) {
        const entry = manifest[key];
        if (entry) entry.dynamicImports = [];
      }
    },
  },
});
