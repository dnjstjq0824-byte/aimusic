// Global route middleware: layout shell + auth/role guards + tab title (WI-R2 i18n).
import { computed } from 'vue';
import { useAuthStore } from '@/stores/authStore';
import { useToast } from '@/composables/useToast';
import { safeRedirect } from '@/utils/redirect';

const AUTH_REQUIRED = new Set([
  'my-library',
  'checkout',
  'checkout-confirm',
  'playlist-detail',
  'dispute-center',
  'account-security',
]);

function layoutFor(path: string): string | false {
  if (path.startsWith('/partner')) return 'partner';
  if (path.startsWith('/admin')) return 'admin';
  if (path === '/store/player' || path === '/500') return false;
  return 'user';
}

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
  else setPageLayout(layout as 'user' | 'partner' | 'admin');

  const localePath = useLocalePath();
  const auth = useAuthStore();
  if (!auth.initialized && import.meta.client) await auth.hydrate();
  // Already signed in: never show the login form again; continue to the target.
  if (name === 'login' && auth.isAuthenticated) {
    const redirect = safeRedirect(to.query.redirect);
    return redirect === to.fullPath ? localePath({ name: 'home' }) : redirect;
  }
  const needsAuth = AUTH_REQUIRED.has(name) || to.path.startsWith('/partner') || to.path.startsWith('/admin');
  if (needsAuth && !auth.isAuthenticated) {
    return { name: 'login', query: { redirect: to.fullPath } };
  }
  const roles = to.path.startsWith('/admin')
    ? ['ADMIN']
    : to.path.startsWith('/partner')
      ? ['PARTNER', 'ADMIN']
      : undefined;
  if (roles && auth.role && !roles.includes(auth.role)) {
    const needAdmin = roles.length === 1 && roles[0] === 'ADMIN';
    if (import.meta.client) {
      const { $i18n } = useNuxtApp();
      useToast().warning($i18n.t(needAdmin ? 'common.auth.needAdmin' : 'common.auth.needPartner'));
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
