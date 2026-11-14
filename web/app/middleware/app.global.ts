// Global route middleware: layout shell + auth/role guards + tab title.
// Port of the SPA router guards (WI-P1). Client-side while all routes are ssr:false.
import { useAuthStore } from '@/stores/authStore';
import { useToast } from '@/composables/useToast';
import { safeRedirect } from '@/utils/redirect';
import { ROUTE_TITLES } from '@/utils/routeTitles';

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

export default defineNuxtRouteMiddleware(async (to) => {
  const name = String(to.name ?? '');
  // [...slug] catch-all renders bare.
  const layout = name === 'not-found' ? false : layoutFor(to.path);
  if (layout === false) setPageLayout(false);
  else setPageLayout(layout as 'user' | 'partner' | 'admin');

  const auth = useAuthStore();
  if (!auth.initialized && import.meta.client) await auth.hydrate();
  // Already signed in: never show the login form again; continue to the target.
  if (name === 'login' && auth.isAuthenticated) {
    const redirect = safeRedirect(to.query.redirect);
    return redirect === to.fullPath ? { name: 'home' } : redirect;
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
      useToast().warning(
        needAdmin ? '관리자 전용 페이지입니다.' : '파트너 전용 페이지입니다. 파트너 승인 후 이용하세요.',
      );
    }
    return { name: 'home' };
  }

  if (import.meta.client) {
    const title = ROUTE_TITLES[name];
    document.title = title ? `${title} · AVIBE` : 'AVIBE — Beyond Your Music';
  }
});
