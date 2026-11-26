// Applies the resolved locale on app start (client only) for UNPREFIXED
// non-public screens. Public pages resolve from the URL (or ko default);
// this plugin must not override them.
import { PUBLIC_BASE_NAMES, isPrefixedPath, resolveLocale, writeStoredLocale } from '@/utils/locale';

export default defineNuxtPlugin((nuxtApp) => {
  const route = useRoute();
  const base = String(route.name ?? '').replace(/___.+$/, '');
  // The URL prefix always wins (/en/… stays English), even on pages added
  // after this list was written.
  if (PUBLIC_BASE_NAMES.has(base) || isPrefixedPath(route.path)) return;
  // NOTE: useI18n() is setup-only; in plugins use nuxtApp.$i18n (cf. app.global.ts).
  const code = resolveLocale(null, navigator.languages ? [...navigator.languages] : []);
  writeStoredLocale(code);
  void nuxtApp.$i18n.setLocale(code);
});
