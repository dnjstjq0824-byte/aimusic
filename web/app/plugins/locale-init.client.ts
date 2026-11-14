// Applies the resolved locale on app start (client only) for UNPREFIXED
// non-public screens. Public pages resolve from the URL (or ko default);
// this plugin must not override them.
import { resolveLocale, writeStoredLocale } from '@/utils/locale';

// Base route names of public pages (prefix or ko-default). Keep in sync
// with routeRules SSR list as R3+ adds pages.
const PUBLIC_BASE_NAMES = new Set([
  'home',
  'explore',
  'playlists',
  'playlist-detail',
  'track-detail',
  'pricing',
  'legal-terms',
  'certificate-verify',
]);

export default defineNuxtPlugin(() => {
  const route = useRoute();
  const base = String(route.name ?? '').replace(/___.+$/, '');
  if (PUBLIC_BASE_NAMES.has(base)) return;
  const { setLocale } = useI18n();
  const code = resolveLocale(null, navigator.languages ? [...navigator.languages] : []);
  writeStoredLocale(code);
  void setLocale(code);
});
