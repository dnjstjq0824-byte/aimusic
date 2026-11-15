// Applies the resolved locale on app start (client only) for UNPREFIXED
// non-public screens. Public pages resolve from the URL (or ko default);
// this plugin must not override them.
import { resolveLocale, writeStoredLocale } from '@/utils/locale';

// Base route names of public pages (prefix or ko-default). Keep in sync
// with routeRules SSR list as R3+ adds pages.
const PUBLIC_BASE_NAMES = new Set([
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
  'certificate-verify',
]);

export default defineNuxtPlugin((nuxtApp) => {
  const route = useRoute();
  const base = String(route.name ?? '').replace(/___.+$/, '');
  if (PUBLIC_BASE_NAMES.has(base)) return;
  // NOTE: useI18n() is setup-only; in plugins use nuxtApp.$i18n (cf. app.global.ts).
  const code = resolveLocale(null, navigator.languages ? [...navigator.languages] : []);
  writeStoredLocale(code);
  void nuxtApp.$i18n.setLocale(code);
});
