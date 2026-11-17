<template>
  <div class="min-h-screen bg-surface-0 text-text-primary flex flex-col font-sans selection:bg-brand-500 selection:text-on-brand">
    <header class="mx-auto w-full max-w-7xl flex items-center justify-between px-4 py-4 sm:px-6">
      <NuxtLink :to="localePath({ name: 'home' })" :aria-label="t('common.nav.home')">
        <BrandLogo :size="32" />
      </NuxtLink>
      <LocaleSwitcher />
    </header>

    <main class="w-full flex-1 px-4 py-6 sm:py-10">
      <slot />
    </main>

    <footer class="mx-auto w-full max-w-7xl px-4 pb-8 sm:px-6">
      <div class="flex flex-col items-center gap-2 border-t border-surface-border pt-4 text-xs text-text-muted sm:flex-row sm:justify-between">
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
