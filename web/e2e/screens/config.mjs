// Screen-harness route matrix (URL-based: survives the Nuxt migration).
// Placeholders {track} {playlist} {cert} are resolved by setup.mjs into .auth.json.
export const WEB = process.env.HARNESS_WEB || 'http://localhost:15173';
export const API = process.env.HARNESS_API || 'http://localhost:18081/api/v1';

export const VIEWPORTS = {
  m390: { width: 390, height: 844, touch: true },
  t768: { width: 768, height: 1024, touch: false },
  d1440: { width: 1440, height: 900, touch: false },
};

export const ROLES = ['guest', 'listener', 'creator', 'storePartner', 'admin'];

const GUEST_ROUTES = [
  ['home', '/'],
  ['explore', '/explore'],
  ['pricing', '/pricing'],
  ['playlists', '/playlists'],
  ['playlist-public', '/playlists/{playlist}'],
  ['track', '/tracks/{track}'],
  ['login', '/login'],
  ['register-tab', '/login?tab=register'],
  ['partner-tab', '/login?tab=partner'],
  ['reset', '/reset-password'],
  ['legal', '/legal'],
  ['legal-tos', '/legal/TOS'],
  ['cert', '/certificate/{cert}'],
  ['cert-invalid', '/certificate/AVB-0000'],
  ['store-player', '/store/player'],
  ['server-error', '/500'],
  ['not-found', '/no-such-page'],
  ['guard-library', '/my-library'],
  ['guard-partner', '/partner'],
  ['guard-checkout', '/checkout'],
  ['guard-admin', '/admin'],
];

const LISTENER_ROUTES = [
  ['home', '/'],
  ['explore', '/explore'],
  ['library', '/my-library'],
  ['playlists', '/playlists'],
  ['playlist-public', '/playlists/{playlist}'],
  ['track', '/tracks/{track}'],
  ['disputes', '/disputes'],
  ['account-security', '/account/security'],
  ['checkout', '/checkout'],
  ['pricing', '/pricing'],
  ['guard-partner', '/partner'],
  ['guard-admin', '/admin'],
];

const CREATOR_ROUTES = [
  ['home', '/'],
  ['partner', '/partner'],
  ['partner-upload', '/partner/upload'],
  ['partner-subscriptions', '/partner/subscriptions'],
  ['partner-stores', '/partner/stores'],
  ['partner-generations', '/partner/generations'],
  ['guard-admin', '/admin'],
];

const STORE_ROUTES = [
  ['partner-stores', '/partner/stores'],
  ['partner', '/partner'],
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
  ['home', '/'],
];

export const MATRIX = {
  guest: GUEST_ROUTES,
  listener: LISTENER_ROUTES,
  creator: CREATOR_ROUTES,
  storePartner: STORE_ROUTES,
  admin: ADMIN_ROUTES,
};
