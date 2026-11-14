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
  modules: ['@pinia/nuxt'],
  css: ['~/assets/css/main.css'],
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
      htmlAttrs: { lang: 'ko' },
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
        { rel: 'preconnect', href: 'https://fonts.googleapis.com' },
        { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: '' },
        { rel: 'preconnect', href: 'https://cdn.jsdelivr.net', crossorigin: '' },
        {
          rel: 'stylesheet',
          href: 'https://cdn.jsdelivr.net/gh/orioncactus/pretendard@v1.3.9/dist/web/variable/pretendardvariable-dynamic-subset.min.css',
        },
        {
          rel: 'stylesheet',
          href: 'https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800;900&family=JetBrains+Mono:wght@400;500;600&display=swap',
        },
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
