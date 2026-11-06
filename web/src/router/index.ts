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
      { path: 'login', name: 'login', component: () => import('@/views/user/Login.vue') },
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
    ],
  },
  {
    path: '/admin',
    component: AdminLayout,
    meta: { requiresAuth: true, roles: ['ADMIN'] },
    children: [
      { path: '', name: 'admin-reviews', component: () => import('@/views/admin/Reviews.vue') },
      { path: 'policies', name: 'admin-policies', component: () => import('@/views/admin/LicensePolicies.vue') },
      { path: 'disputes', name: 'admin-disputes', component: () => import('@/views/admin/Disputes.vue') },
      { path: 'curation', name: 'admin-curation', component: () => import('@/views/admin/Curation.vue') },
    ],
  },
  { path: '/:pathMatch(.*)*', redirect: '/' },
];

export const router = createRouter({
  history: createWebHistory(),
  routes,
});

router.beforeEach(async (to) => {
  const auth = useAuthStore();
  if (!auth.initialized) await auth.hydrate();
  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    return { name: 'login', query: { redirect: to.fullPath } };
  }
  const roles = to.meta.roles as string[] | undefined;
  if (roles && auth.role && !roles.includes(auth.role)) {
    return { name: 'home' };
  }
  return true;
});
