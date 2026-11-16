// Global route middleware: layout shell + auth/role guards + tab title (WI-R2 i18n).
import { computed } from 'vue';
import { useAuthStore } from '@/stores/authStore';
import { useToast } from '@/composables/useToast';
import { safeRedirect } from '@/utils/redirect';
import { shouldApplyServerLocale } from '@/utils/locale';

const AUTH_REQUIRED = new Set([
  'library',
  'me-playlists',
  'checkout',
  'checkout-confirm',
  'account-index',
  'account-profile',
  'account-security',
  'account-billing',
  'account-language',
  'account-privacy',
  'account-connections',
]);

function layoutFor(path: string): string | false {
  if (path.startsWith('/studio')) return 'studio';
  if (path.startsWith('/business/console')) return 'business';
  if (path.startsWith('/admin')) return 'admin';
  if (path.startsWith('/embed')) return 'embed';
  if (path === '/store/player' || path === '/500') return false;
  if (
    path === '/login' || path.startsWith('/signup') || path === '/forgot-password' ||
    path === '/reset-password' || path === '/verify-email' || path.startsWith('/oauth/')
  ) {
    return 'auth';
  }
  return 'app';
}

// Entry screens that make no sense once signed in (R3: login + signup + forgot).
const GUEST_ONLY = new Set([
  'login',
  'signup',
  'signup-listener',
  'signup-creator',
  'signup-business',
  'forgot-password',
]);

/** Strip the i18n route-name suffix (home___en -> home). */
export function baseRouteName(name: unknown): string {
  return String(name ?? '').replace(/___.+$/, '');
}

export default defineNuxtRouteMiddleware(async (to) => {
  // Trailing slash: single canonical without it (SEO §3).
  // Locale roots (/en/ …) keep their slash — they have no shorter form.
  const isLocaleRoot = /^\/(en|ja|zh-cn|th)\/$/.test(to.path);
  if (to.path !== '/' && to.path.endsWith('/') && !isLocaleRoot) {
    const q = to.fullPath.slice(to.path.length);
    return navigateTo(`${to.path.slice(0, -1)}${q}`, { redirectCode: 301 });
  }

  const name = baseRouteName(to.name);
  // [...slug] catch-all renders bare.
  const layout = name === 'not-found' ? false : layoutFor(to.path);
  if (layout === false) setPageLayout(false);
  else setPageLayout(layout as 'app' | 'studio' | 'business' | 'admin' | 'auth' | 'embed');

  // Legacy tab URLs keep working (R3 split): single-URL 3-tab login is gone.
  if (name === 'login' && typeof to.query.tab === 'string') {
    if (to.query.tab === 'register') return navigateTo('/signup/listener', { redirectCode: 301 });
    if (to.query.tab === 'partner') return navigateTo('/signup/creator', { redirectCode: 301 });
  }
  // Tokenless reset is the forgot screen (R3 split).
  if (name === 'reset-password' && (typeof to.query.token !== 'string' || to.query.token === '')) {
    return navigateTo('/forgot-password', { redirectCode: 301 });
  }
  // R6 split: the old combined library lives at /library now.
  // Path-based: the my-library route no longer exists ([...slug] would 404).
  if (to.path === '/my-library') {
    return navigateTo('/library', { redirectCode: 301 });
  }
  // R8 split: the disputes page lives at /rights-center now.
  if (to.path === '/disputes') {
    return navigateTo('/rights-center', { redirectCode: 301 });
  }
  // R9/R10 split: the partner console moved to /studio/** (creator) and
  // /business/console/** (stores). Old bookmarks keep working (R-3, R-S).
  if (to.path === '/partner' || to.path.startsWith('/partner/')) {
    const rest = to.path === '/partner' ? '' : to.path.slice('/partner'.length);
    const map: Record<string, string> = {
      '': '/studio',
      '/upload': '/studio/upload',
      '/subscriptions': '/studio/proofs',
      '/stores': '/business/console/stores',
      '/generations': '/studio/create',
    };
    const target = map[rest] ?? `/studio${rest}`;
    const q = to.fullPath.slice(to.path.length);
    return navigateTo(`${target}${q}`, { redirectCode: 301 });
  }

  const localePath = useLocalePath();
  const auth = useAuthStore();
  if (!auth.initialized && import.meta.client) {
    await auth.hydrate();
    // A2: a fresh principal applies the server locale (detection order ①).
    const { $i18n } = useNuxtApp();
    const apply = shouldApplyServerLocale(auth.locale, $i18n.locale.value, to.path, name);
    if (apply) await $i18n.setLocale(apply);
  }
  // Already signed in: never show entry forms again; continue to the target.
  if (GUEST_ONLY.has(name) && auth.isAuthenticated) {
    const redirect = safeRedirect(to.query.redirect);
    return redirect === to.fullPath ? localePath({ name: 'home' }) : redirect;
  }
  const needsAuth = AUTH_REQUIRED.has(name) || to.path.startsWith('/studio') || to.path.startsWith('/business/console') || to.path.startsWith('/admin');
  if (needsAuth && !auth.isAuthenticated) {
    return { name: 'login', query: { redirect: to.fullPath } };
  }
  // R-3: listeners reach neither console; migrated accounts hold both grants.
  const guard = to.path.startsWith('/admin')
    ? { roles: ['ADMIN'], key: 'common.auth.needAdmin' }
    : to.path.startsWith('/studio')
      ? { roles: ['PARTNER', 'ADMIN'], key: 'common.auth.needPartner' }
      : to.path.startsWith('/business/console')
        ? { roles: ['BUSINESS', 'ADMIN'], key: 'common.auth.needBusiness' }
        : undefined;
  if (guard && auth.role && !guard.roles.some((r) => auth.hasRole(r as 'ADMIN' | 'PARTNER' | 'BUSINESS'))) {
    if (import.meta.client) {
      const { $i18n } = useNuxtApp();
      useToast().warning($i18n.t(guard.key));
    }
    return localePath({ name: 'home' });
  }

  const { $i18n } = useNuxtApp();
  const key = `common.titles.${name}`;
  // Reactive: client-side locale switches (after this middleware runs) refresh title + lang.
  const pageTitle = computed(() => ($i18n.te(key) ? `${$i18n.t(key)} · AVIBE` : 'AVIBE — Beyond Your Music'));
  const htmlLang = computed(() => ($i18n.locale.value === 'zh-cn' ? 'zh-CN' : $i18n.locale.value));
  useHead({ htmlAttrs: { lang: htmlLang }, title: pageTitle });
});
