// Nuxt 4 app config (WI-P1). Public pages SSR, app screens CSR (SEO §2).
// Build targets: node-server (default, local verify) and cloudflare-module
// via NITRO_PRESET=cloudflare-module (SEO §1, WI-P1 dual-build).
const LEGACY_NAMES: Record<string, string> = {
  index: 'home',
  'tracks-id': 'track-detail',
  'playlists-id': 'playlist-detail',
  'partner-index': 'partner-dashboard',
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
  modules: ['@pinia/nuxt', '@nuxtjs/i18n'],
  i18n: {
    strategy: 'prefix_except_default',
    defaultLocale: 'ko',
    detectBrowserLanguage: false,
    baseUrl: 'https://avibemusic.com',
    langDir: '../i18n/locales',
    locales: [
      { code: 'ko', language: 'ko-KR', files: ['ko/common.json', 'ko/errors.json', 'ko/player.json', 'ko/auth.json', 'ko/home.json', 'ko/explore.json', 'ko/track.json', 'ko/library.json', 'ko/playlist.json', 'ko/pricing.json', 'ko/checkout.json', 'ko/account.json', 'ko/legal.json', 'ko/studio.json', 'ko/business.json', 'ko/generate.json', 'ko/admin.json'] },
      { code: 'en', language: 'en-US', files: ['en/common.json', 'en/errors.json', 'en/player.json', 'en/auth.json', 'en/home.json', 'en/explore.json', 'en/track.json', 'en/library.json', 'en/playlist.json', 'en/pricing.json', 'en/checkout.json', 'en/account.json', 'en/legal.json', 'en/studio.json', 'en/business.json', 'en/generate.json', 'en/admin.json'] },
      { code: 'ja', language: 'ja-JP', files: ['ja/common.json', 'ja/errors.json', 'ja/player.json', 'ja/auth.json', 'ja/home.json', 'ja/explore.json', 'ja/track.json', 'ja/library.json', 'ja/playlist.json', 'ja/pricing.json', 'ja/checkout.json', 'ja/account.json', 'ja/legal.json', 'ja/studio.json', 'ja/business.json', 'ja/generate.json', 'ja/admin.json'] },
      { code: 'zh-cn', language: 'zh-CN', files: ['zh-CN/common.json', 'zh-CN/errors.json', 'zh-CN/player.json', 'zh-CN/auth.json', 'zh-CN/home.json', 'zh-CN/explore.json', 'zh-CN/track.json', 'zh-CN/library.json', 'zh-CN/playlist.json', 'zh-CN/pricing.json', 'zh-CN/checkout.json', 'zh-CN/account.json', 'zh-CN/legal.json', 'zh-CN/studio.json', 'zh-CN/business.json', 'zh-CN/generate.json', 'zh-CN/admin.json'] },
      { code: 'th', language: 'th-TH', files: ['th/common.json', 'th/errors.json', 'th/player.json', 'th/auth.json', 'th/home.json', 'th/explore.json', 'th/library.json', 'th/playlist.json', 'th/pricing.json', 'th/checkout.json', 'th/account.json', 'th/legal.json', 'th/studio.json', 'th/business.json', 'th/generate.json', 'th/admin.json'] },
      ...(process.dev
        ? [{ code: 'en-XA', language: 'en-XA', files: ['en-XA/common.json', 'en-XA/errors.json', 'en-XA/player.json'] }]
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
    public: {
      apiBaseUrl: process.env.VITE_API_BASE_URL || '/api/v1',
      tossClientKey: process.env.VITE_TOSS_CLIENT_KEY || '',
    },
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
    '/embed/**': { swr: 3600 },
    '/login': { ssr: false },
    '/signup/**': { ssr: false },
    '/forgot-password': { ssr: false },
    '/reset-password': { ssr: false },
    '/verify-email': { ssr: false },
    '/oauth/**': { ssr: false },
    '/my-library': { ssr: false },
    '/me/**': { ssr: false },
    '/account/**': { ssr: false },
    '/disputes': { ssr: false },
    '/rights-center': { ssr: false },
    '/checkout/**': { ssr: false },
    '/partner/**': { ssr: false },
    '/studio/**': { ssr: false },
    '/admin/**': { ssr: false },
    '/store/**': { ssr: false },
    '/business/console/**': { ssr: false },
    '/500': { ssr: false },
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
        { name: 'theme-color', content: '#08080d' },
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
  },
});
