// Screen-harness route matrix (URL-based: survives the Nuxt migration).
// Placeholders {track} {playlist} {cert} are resolved by setup.mjs into .auth.json.
export const WEB = process.env.HARNESS_WEB || 'http://localhost:15173';
export const API = process.env.HARNESS_API || 'http://localhost:18081/api/v1';

export const VIEWPORTS = {
  m390: { width: 390, height: 844, touch: true },
  t768: { width: 768, height: 1024, touch: false },
  d1440: { width: 1440, height: 900, touch: false },
};

export const ROLES = ['guest', 'listener', 'creator', 'storePartner', 'business', 'admin'];

// R13 language dimension (web locale codes; ko = unprefixed default).
// Override with HARNESS_LANGS=ko,en for a faster subset run.
export const LANGS = (process.env.HARNESS_LANGS || 'ko,en,ja,zh-cn,th')
  .split(',')
  .map((s) => s.trim().toLowerCase())
  .filter((s) => ['ko', 'en', 'ja', 'zh-cn', 'th'].includes(s));

const GUEST_ROUTES = [
  ['home', '/'],
  ['explore', '/explore'],
  ['charts', '/charts'],
  ['pricing', '/pricing'],
  ['business', '/business'],
  ['for-creators', '/for-creators'],
  ['trust', '/trust'],
  ['create', '/create'],
  ['playlists', '/playlists'],
  ['playlist-public', '/playlists/{playlist}'],
  ['track', '/tracks/{track}'],
  ['creator', '/creators/{creator}'],
  ['embed-track', '/embed/track/{trackShort}'],
  ['embed-playlist', '/embed/playlist/{playlistShort}'],
  ['landing-mood', '/explore/mood/cafe'],
  ['landing-genre', '/explore/genre/electronic'],
  ['landing-use', '/explore/use/youtube'],
  ['guides', '/guides'],
  ['guide-article', '/guides/store-music-license'],
  ['login', '/login'],
  ['register-tab', '/login?tab=register'],
  ['partner-tab', '/login?tab=partner'],
  ['signup', '/signup'],
  ['signup-listener', '/signup/listener'],
  ['signup-creator', '/signup/creator'],
  ['signup-business', '/signup/business'],
  ['signup-complete-email', '/signup/complete-email'],
  ['forgot-password', '/forgot-password'],
  ['reset', '/reset-password'],
  ['verify-email', '/verify-email'],
  ['oauth-callback', '/oauth/callback'],
  ['legal', '/legal'],
  ['legal-tos', '/legal/TOS'],
  ['cert', '/certificate/{cert}'],
  ['cert-invalid', '/certificate/AVB-0000'],
  ['store-player', '/store/player'],
  ['server-error', '/500'],
  ['not-found', '/no-such-page'],
  ['guard-library', '/library'],
  ['legacy-library', '/my-library'],
  ['legacy-disputes', '/disputes'],
  ['legacy-partner', '/partner'],
  ['legacy-partner-stores', '/partner/stores'],
  ['guard-account', '/account/security'],
  ['guard-studio', '/studio'],
  ['guard-console', '/business/console'],
  ['guard-checkout', '/checkout'],
  ['guard-admin', '/admin'],
];

const LISTENER_ROUTES = [
  ['home', '/'],
  ['explore', '/explore'],
  ['library', '/library'],
  ['me-playlists', '/me/playlists'],
  ['playlists', '/playlists'],
  ['playlist-public', '/playlists/{playlist}'],
  ['track', '/tracks/{track}'],
  ['creator', '/creators/{creator}'],
  ['guides', '/guides'],
  ['rights-center', '/rights-center'],
  ['account', '/account'],
  ['account-profile', '/account/profile'],
  ['account-security', '/account/security'],
  ['account-billing', '/account/billing'],
  ['account-language', '/account/language'],
  ['account-privacy', '/account/privacy'],
  ['account-connections', '/account/connections'],
  ['checkout', '/checkout'],
  ['pricing', '/pricing'],
  ['guard-studio', '/studio'],
  ['guard-console', '/business/console'],
  ['guard-admin', '/admin'],
];

const CREATOR_ROUTES = [
  ['home', '/'],
  ['studio', '/studio'],
  ['studio-create', '/studio/create'],
  ['studio-upload', '/studio/upload'],
  ['studio-catalog', '/studio/catalog'],
  ['studio-proofs', '/studio/proofs'],
  ['studio-earnings', '/studio/earnings'],
  ['studio-settings', '/studio/settings'],
  ['guard-console', '/business/console'],
  ['guard-admin', '/admin'],
];

const STORE_ROUTES = [
  ['console', '/business/console'],
  ['console-stores', '/business/console/stores'],
  ['console-players', '/business/console/players'],
  ['console-licenses', '/business/console/licenses'],
  ['console-billing', '/business/console/billing'],
  ['studio', '/studio'],
  ['store-player', '/store/player'],
  ['home', '/'],
];

const ADMIN_ROUTES = [
  ['admin', '/admin'],
  ['admin-reviews', '/admin/reviews'],
  ['admin-policies', '/admin/policies'],
  ['admin-disputes', '/admin/disputes'],
  ['admin-curation', '/admin/curation'],
  ['admin-audit', '/admin/audit'],
  ['admin-users', '/admin/users'],
  ['admin-stores', '/admin/stores'],
  ['admin-distribution', '/admin/distribution'],
  ['admin-settlements', '/admin/settlements'],
  ['admin-inquiries', '/admin/inquiries'],
  ['admin-role-grants', '/admin/role-grants'],
  ['admin-social', '/admin/social'],
  ['home', '/'],
];

export const MATRIX = {
  guest: GUEST_ROUTES,
  listener: LISTENER_ROUTES,
  creator: CREATOR_ROUTES,
  storePartner: STORE_ROUTES,
  // WI-N6: BUSINESS role + APPROVED profile captures the real console
  // (storePartner is PARTNER and gets redirected home).
  business: STORE_ROUTES,
  admin: ADMIN_ROUTES,
};
