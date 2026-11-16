<template>
  <div class="min-h-screen overflow-x-clip bg-surface-0 text-text-primary flex flex-col font-sans selection:bg-brand-500 selection:text-on-brand">
    <header class="sticky top-0 z-40 border-b border-surface-border bg-surface-1/90 backdrop-blur-xl">
      <div class="mx-auto flex max-w-7xl items-center justify-between gap-3 px-4 py-3 sm:px-6 lg:px-8">
        <div class="flex min-w-0 items-center gap-3">
          <button
            type="button"
            class="lg:hidden flex h-9 w-9 items-center justify-center rounded-lg border border-surface-border bg-surface-1 text-text-primary hover:text-white"
            :aria-expanded="mobileNavOpen"
            aria-controls="studio-mobile-nav"
            :aria-label="mobileNavOpen ? t('studio.nav.close') : t('studio.nav.open')"
            @click="mobileNavOpen = !mobileNavOpen"
          >
            <svg v-if="!mobileNavOpen" class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
              <path d="M4 7h16M4 12h16M4 17h16" />
            </svg>
            <svg v-else class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
              <path d="M18 6 6 18M6 6l12 12" />
            </svg>
          </button>
          <RouterLink to="/studio" class="flex items-center" :aria-label="t('studio.nav.home')">
            <BrandLogo :size="32" tagline="Studio Console" />
          </RouterLink>
        </div>

        <div class="flex shrink-0 items-center gap-3 text-xs">
          <RouterLink to="/" class="text-text-secondary hover:text-white transition-colors">
            {{ t('studio.nav.listenerHome') }}
          </RouterLink>
          <button
            type="button"
            class="text-text-secondary hover:text-white transition-colors"
            @click="auth.logout()"
          >
            {{ t('studio.nav.logout') }}
          </button>
        </div>
      </div>

      <nav
        v-if="mobileNavOpen"
        id="studio-mobile-nav"
        :aria-label="t('studio.nav.menu')"
        class="lg:hidden border-t border-surface-border bg-surface-1/95 px-4 py-4 text-sm sm:grid sm:grid-cols-2 sm:gap-1"
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
      </nav>
    </header>

    <div class="mx-auto flex w-full max-w-7xl flex-1 gap-6 px-4 py-6 sm:px-6 sm:py-8 lg:px-8">
      <aside class="hidden w-52 shrink-0 lg:block">
        <nav :aria-label="t('studio.nav.menu')" class="sticky top-20 space-y-1 text-sm">
          <RouterLink
            v-for="item in menu"
            :key="item.to"
            :to="item.to"
            class="block rounded-lg px-3 py-2 transition-colors"
            :class="routeName === item.name ? 'bg-surface-2 text-white font-bold' : 'text-text-secondary hover:text-white'"
          >
            {{ item.label }}
          </RouterLink>
        </nav>
      </aside>

      <main class="min-w-0 flex-1 pb-20">
        <slot />
      </main>
    </div>

    <UiFooter />
    <UiToastContainer />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useRoute } from 'vue-router';
import { useAuthStore } from '@/stores/authStore';
import UiFooter from '@/components/ui/UiFooter.vue';
import UiToastContainer from '@/components/ui/UiToastContainer.vue';
import BrandLogo from '@/components/BrandLogo.vue';

const { t } = useI18n();
const auth = useAuthStore();
const route = useRoute();
const routeName = computed(() => String(route.name ?? '').replace(/___.+$/, ''));
const mobileNavOpen = ref(false);
watch(() => route.fullPath, () => (mobileNavOpen.value = false));

const menu = computed(() => [
  { to: '/studio', name: 'studio-dashboard', label: t('studio.nav.dashboard') },
  { to: '/studio/create', name: 'studio-create', label: t('studio.nav.create') },
  { to: '/studio/upload', name: 'studio-upload', label: t('studio.nav.upload') },
  { to: '/studio/catalog', name: 'studio-catalog', label: t('studio.nav.catalog') },
  { to: '/studio/proofs', name: 'studio-proofs', label: t('studio.nav.proofs') },
  { to: '/studio/earnings', name: 'studio-earnings', label: t('studio.nav.earnings') },
  { to: '/studio/settings', name: 'studio-settings', label: t('studio.nav.settings') },
]);
</script>
