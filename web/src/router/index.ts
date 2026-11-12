import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router';
import { useAuthStore } from '@/stores/authStore';
import { useToast } from '@/composables/useToast';
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
      {
        path: 'certificate/:number',
        name: 'certificate-verify',
        component: () => import('@/views/legal/CertificateVerify.vue'),
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

// F-04: per-route tab titles (bookmarks, history, screen readers).
const ROUTE_TITLES: Record<string, string> = {
  home: '홈',
  explore: '음원 탐색',
  'my-library': '내 라이브러리',
  playlists: '플레이리스트',
  login: '로그인',
  'reset-password': '비밀번호 찾기',
  pricing: '요금제',
  checkout: '결제',
  'checkout-confirm': '결제 완료',
  'track-detail': '음원 상세',
  'playlist-detail': '플레이리스트',
  'dispute-center': '권리보호센터',
  'account-security': '계정 보안',
  'legal-root': '약관',
  'legal-terms': '약관',
  'certificate-verify': '인증서 확인',
  'partner-dashboard': '파트너 대시보드',
  'partner-upload': '음원 업로드',
  'partner-subscriptions': 'AI 구독 증빙',
  'partner-stores': '매장 관리',
  'partner-generations': '음원 생성',
  'admin-overview': '관리자 개요',
  'admin-reviews': '심사',
  'admin-policies': '라이선스 정책',
  'admin-disputes': '분쟁 처리',
  'admin-curation': '큐레이션',
  'admin-audit': '감사 로그',
  'admin-users': '사용자 관리',
  'admin-stores': '매장 승인',
  'admin-distribution': '유통',
  'admin-settlements': '정산',
  'store-player': '매장 플레이어',
  'server-error': '오류',
  'not-found': '페이지를 찾을 수 없음',
};

router.afterEach((to) => {
  if (typeof document === 'undefined') return;
  const title = ROUTE_TITLES[String(to.name ?? '')];
  document.title = title ? title + ' · AVIBE' : 'AVIBE — Beyond Your Music';
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
    const needAdmin = roles.length === 1 && roles[0] === 'ADMIN';
    useToast().warning(
      needAdmin ? '관리자 전용 페이지입니다.' : '파트너 전용 페이지입니다. 파트너 승인 후 이용하세요.',
    );
    return { name: 'home' };
  }
  return true;
});
