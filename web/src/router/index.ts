import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router';
import { useAuthStore } from '@/stores/authStore';
import UserLayout from '@/layouts/UserLayout.vue';
import PartnerLayout from '@/layouts/PartnerLayout.vue';
import AdminLayout from '@/layouts/AdminLayout.vue';

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    component: UserLayout,
    children: [
      { path: '', name: 'home', component: () => import('@/views/user/Home.vue') },
      { path: 'explore', name: 'explore', component: () => import('@/views/user/Explore.vue') },
      {
        path: 'my-library',
        name: 'my-library',
        component: () => import('@/views/user/MyLibrary.vue'),
        meta: { requiresAuth: true },
      },
      { path: 'playlists', name: 'playlists', component: () => import('@/views/user/Playlists.vue') },
      { path: 'login', name: 'login', component: () => import('@/views/user/Login.vue') },
      {
        path: 'reset-password',
        name: 'reset-password',
        component: () => import('@/views/user/ResetPassword.vue'),
      },
      {
        path: 'oauth/callback',
        name: 'oauth-callback',
        component: () => import('@/views/user/OAuthCallback.vue'),
      },
      { path: 'pricing', name: 'pricing', component: () => import('@/views/user/Pricing.vue') },
      {
        path: 'checkout',
        name: 'checkout',
        component: () => import('@/views/user/Checkout.vue'),
        meta: { requiresAuth: true },
      },
      {
        path: 'checkout/confirm',
        name: 'checkout-confirm',
        component: () => import('@/views/user/CheckoutConfirm.vue'),
        meta: { requiresAuth: true },
      },
      {
        path: 'tracks/:id',
        name: 'track-detail',
        component: () => import('@/views/user/TrackDetail.vue'),
      },
      {
        path: 'playlists/:id',
        name: 'playlist-detail',
        component: () => import('@/views/user/PlaylistDetail.vue'),
        meta: { requiresAuth: true },
      },
      {
        path: 'disputes',
        name: 'dispute-center',
        component: () => import('@/views/user/DisputeCenter.vue'),
        meta: { requiresAuth: true },
      },
      {
        path: 'account/security',
        name: 'account-security',
        component: () => import('@/views/user/Security.vue'),
        meta: { requiresAuth: true },
      },
      {
        path: 'legal',
        name: 'legal-root',
        component: () => import('@/views/legal/LegalTerms.vue'),
      },
      {
        path: 'legal/:code',
        name: 'legal-terms',
        component: () => import('@/views/legal/LegalTerms.vue'),
      },
    ],
  },
  {
    path: '/partner',
    component: PartnerLayout,
    meta: { requiresAuth: true, roles: ['PARTNER', 'ADMIN'] },
    children: [
      { path: '', name: 'partner-dashboard', component: () => import('@/views/partner/Dashboard.vue') },
      { path: 'upload', name: 'partner-upload', component: () => import('@/views/partner/Upload.vue') },
      { path: 'subscriptions', name: 'partner-subscriptions', component: () => import('@/views/partner/Subscriptions.vue') },
      { path: 'stores', name: 'partner-stores', component: () => import('@/views/partner/Stores.vue') },
      { path: 'generations', name: 'partner-generations', component: () => import('@/views/partner/Generations.vue') },
    ],
  },
  {
    path: '/admin',
    component: AdminLayout,
    meta: { requiresAuth: true, roles: ['ADMIN'] },
    children: [
      { path: '', name: 'admin-overview', component: () => import('@/views/admin/Overview.vue') },
      { path: 'reviews', name: 'admin-reviews', component: () => import('@/views/admin/Reviews.vue') },
      { path: 'policies', name: 'admin-policies', component: () => import('@/views/admin/LicensePolicies.vue') },
      { path: 'disputes', name: 'admin-disputes', component: () => import('@/views/admin/Disputes.vue') },
      { path: 'curation', name: 'admin-curation', component: () => import('@/views/admin/Curation.vue') },
      { path: 'audit', name: 'admin-audit', component: () => import('@/views/admin/Audit.vue') },
      { path: 'users', name: 'admin-users', component: () => import('@/views/admin/Users.vue') },
      { path: 'stores', name: 'admin-stores', component: () => import('@/views/admin/Stores.vue') },
      { path: 'distribution', name: 'admin-distribution', component: () => import('@/views/admin/Distribution.vue') },
      { path: 'settlements', name: 'admin-settlements', component: () => import('@/views/admin/Settlements.vue') },
    ],
  },
  {
    path: '/store/player',
    name: 'store-player',
    component: () => import('@/views/store/Player.vue'),
  },
  {
    path: '/500',
    name: 'server-error',
    component: () => import('@/views/errors/ServerError.vue'),
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'not-found',
    component: () => import('@/views/errors/NotFound.vue'),
  },
];

export const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior() {
    return { top: 0 };
  },
});

/** In-app redirects only: must be a same-origin absolute path, never `//host` or a scheme. */
export function safeRedirect(raw: unknown): string {
  if (typeof raw !== 'string' || !raw.startsWith('/') || raw.startsWith('//') || raw.includes('\\')) {
    return '/';
  }
  return raw;
}

router.beforeEach(async (to) => {
  const auth = useAuthStore();
  if (!auth.initialized) await auth.hydrate();
  // Already signed in: never show the login form again; continue to the target.
  if (to.name === 'login' && auth.isAuthenticated) {
    const redirect = safeRedirect(to.query.redirect);
    return redirect === to.fullPath ? { name: 'home' } : redirect;
  }
  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    return { name: 'login', query: { redirect: to.fullPath } };
  }
  const roles = to.meta.roles as string[] | undefined;
  if (roles && auth.role && !roles.includes(auth.role)) {
    return { name: 'home' };
  }
  return true;
});
