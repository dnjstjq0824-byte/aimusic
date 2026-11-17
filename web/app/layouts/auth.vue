<template>
  <div class="relative min-h-screen overflow-x-clip bg-surface-0 text-text-primary flex flex-col font-sans selection:bg-brand-500 selection:text-on-brand">
    <!-- v3 flow: the auth screens sit in the same soft hero light. -->
    <div class="hero-glow pointer-events-none absolute left-1/2 top-0 h-[36rem] w-screen -translate-x-1/2" aria-hidden="true" />
    <header class="relative mx-auto w-full max-w-7xl flex items-center justify-between px-4 py-4 sm:px-6">
      <NuxtLink :to="localePath({ name: 'home' })" :aria-label="t('common.nav.home')">
        <BrandLogo :size="30" tagline="" />
      </NuxtLink>
      <LocaleSwitcher />
    </header>

    <main class="relative w-full flex-1 px-4 py-6 sm:py-12">
      <slot />
    </main>

    <footer class="mx-auto w-full max-w-7xl px-4 pb-8 sm:px-6">
      <div class="flex flex-col items-center gap-2 border-t border-white/[0.06] pt-4 text-xs text-text-muted sm:flex-row sm:justify-between">
        <span>© 2026 AVIBE</span>
        <nav class="flex items-center gap-4" :aria-label="t('common.nav.legal')">
          <NuxtLink :to="localePath('/legal/TOS')" class="hover:text-text-primary hover:underline">
            {{ t('auth.common.tos') }}
          </NuxtLink>
          <NuxtLink :to="localePath('/legal/PRIVACY')" class="hover:text-text-primary hover:underline">
            {{ t('auth.common.privacy') }}
          </NuxtLink>
        </nav>
      </div>
    </footer>

    <UiToastContainer />
  </div>
</template>

<script setup lang="ts">
import BrandLogo from '@/components/BrandLogo.vue';
import LocaleSwitcher from '@/components/ui/LocaleSwitcher.vue';
import UiToastContainer from '@/components/ui/UiToastContainer.vue';

const { t } = useI18n();
const localePath = useLocalePath();
</script>
