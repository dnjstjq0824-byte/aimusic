<template>
  <div class="min-h-screen overflow-x-clip bg-surface-0 text-text-primary flex flex-col font-sans selection:bg-brand-500 selection:text-on-brand">
    <header class="sticky top-0 z-40 border-b border-surface-border bg-surface-1/90 backdrop-blur-xl">
      <div class="mx-auto flex max-w-7xl items-center justify-between gap-3 px-4 py-3 sm:px-6 lg:px-8">
        <div class="flex min-w-0 items-center gap-6">
          <RouterLink to="/admin" class="flex items-center" :aria-label="t('admin.nav.home')">
            <BrandLogo :size="32" tagline="Audit Console" />
          </RouterLink>

          <nav class="hidden lg:flex items-center gap-1 whitespace-nowrap text-xs font-semibold text-text-secondary" :aria-label="t('admin.nav.menu')">
            <RouterLink
              v-for="item in menu"
              :key="item.to"
              :to="item.to"
              class="rounded-lg px-3 py-1.5 transition-colors hover:text-white"
              :class="routeName === item.name ? 'bg-surface-2 text-white font-bold' : ''"
            >
              {{ item.label }}
            </RouterLink>
            <RouterLink
              to="/"
              class="rounded-lg px-3 py-1.5 text-text-muted hover:text-text-primary ml-2"
            >
              {{ t('admin.nav.listenerHome') }}
            </RouterLink>
          </nav>
        </div>

        <div class="flex shrink-0 items-center gap-3 text-xs">
          <span class="rounded-full bg-brand-950 border border-brand-500/30 px-2.5 py-1 text-brand-300 font-semibold hidden sm:inline">
            {{ t('admin.nav.badge') }}
          </span>
          <button
            type="button"
            class="hidden sm:inline text-text-secondary hover:text-white transition-colors"
            @click="auth.logout()"
          >
            {{ t('admin.nav.logout') }}
          </button>
          <button
            type="button"
            class="lg:hidden flex h-9 w-9 items-center justify-center rounded-lg border border-surface-border bg-surface-1 text-text-primary hover:text-white"
            :aria-expanded="mobileNavOpen"
            aria-controls="admin-mobile-nav"
            :aria-label="mobileNavOpen ? t('admin.nav.close') : t('admin.nav.open')"
            @click="mobileNavOpen = !mobileNavOpen"
          >
            <svg v-if="!mobileNavOpen" class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
              <path d="M4 7h16M4 12h16M4 17h16" />
            </svg>
            <svg v-else class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
              <path d="M18 6 6 18M6 6l12 12" />
            </svg>
          </button>
        </div>
      </div>

      <nav
        v-if="mobileNavOpen"
        id="admin-mobile-nav"
        :aria-label="t('admin.nav.menu')"
        class="lg:hidden border-t border-surface-border bg-surface-1/95 px-4 py-4 space-y-1 text-sm sm:grid sm:grid-cols-2 sm:gap-1 sm:space-y-0"
      >
        <RouterLink
          v-for="item in menu"
          :key="item.to"
          :to="item.to"
          class="block rounded-lg px-3 py-2 text-text-primary hover:bg-surface-2 hover:text-white"
          @click="mobileNavOpen = false"
        >
          {{ item.label }}
        </RouterLink>
        <RouterLink
          to="/"
          class="block rounded-lg px-3 py-2 text-text-muted hover:bg-surface-2 hover:text-text-primary"
          @click="mobileNavOpen = false"
        >
          {{ t('admin.nav.listenerHome') }}
        </RouterLink>
        <button
          type="button"
          class="block w-full rounded-lg px-3 py-2 text-left text-text-secondary hover:bg-surface-2 hover:text-white"
          @click="auth.logout()"
        >
          {{ t('admin.nav.logout') }}
        </button>
      </nav>
    </header>

    <main class="mx-auto w-full max-w-7xl flex-1 px-4 py-6 pb-20 sm:px-6 sm:py-8 lg:px-8">
      <slot />
    </main>

    <UiFooter />
    <UiToastContainer />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute } from 'vue-router';
import { useAuthStore } from '@/stores/authStore';
import UiFooter from '@/components/ui/UiFooter.vue';
import UiToastContainer from '@/components/ui/UiToastContainer.vue';
import BrandLogo from '@/components/BrandLogo.vue';

const { t, locale, setLocale } = useI18n();
const auth = useAuthStore();
const route = useRoute();
const routeName = computed(() => String(route.name ?? '').replace(/___.+$/, ''));
const mobileNavOpen = ref(false);
watch(() => route.fullPath, () => (mobileNavOpen.value = false));

// R-2: the admin console ships ko/en only. Any other saved locale falls
// back to English while inside /admin/**.
function enforceAdminLocale(): void {
  if (locale.value !== 'ko' && locale.value !== 'en') {
    void setLocale('en');
  }
}
watch(() => route.fullPath, () => enforceAdminLocale());
onMounted(() => enforceAdminLocale());

const menu = computed(() => [
  { to: '/admin', name: 'admin-overview', label: t('admin.nav.overview') },
  { to: '/admin/reviews', name: 'admin-reviews', label: t('admin.nav.reviews') },
  { to: '/admin/policies', name: 'admin-policies', label: t('admin.nav.policies') },
  { to: '/admin/disputes', name: 'admin-disputes', label: t('admin.nav.disputes') },
  { to: '/admin/curation', name: 'admin-curation', label: t('admin.nav.curation') },
  { to: '/admin/audit', name: 'admin-audit', label: t('admin.nav.audit') },
  { to: '/admin/users', name: 'admin-users', label: t('admin.nav.users') },
  { to: '/admin/stores', name: 'admin-stores', label: t('admin.nav.stores') },
  { to: '/admin/distribution', name: 'admin-distribution', label: t('admin.nav.distribution') },
  { to: '/admin/settlements', name: 'admin-settlements', label: t('admin.nav.settlements') },
  { to: '/admin/inquiries', name: 'admin-inquiries', label: t('admin.nav.inquiries') },
  { to: '/admin/role-grants', name: 'admin-role-grants', label: t('admin.nav.roles') },
  { to: '/admin/social', name: 'admin-social', label: t('admin.nav.social') },
]);
</script>
