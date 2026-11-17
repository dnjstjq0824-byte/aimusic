<template>
  <div class="mx-auto flex min-h-[60vh] max-w-lg flex-col items-center justify-center text-center px-4">
    <BrandLogo :size="40" />
    <div class="relative my-6" aria-hidden="true">
      <div class="text-8xl font-black text-surface-2 tracking-tighter select-none">404</div>
      <div class="absolute inset-0 flex items-center justify-center text-2xl font-black text-brand-500">
        NOT FOUND
      </div>
    </div>
    <h1 class="text-xl font-bold text-white mb-2">{{ t('common.httpError.t404') }}</h1>
    <p class="text-xs text-text-secondary leading-relaxed mb-2 max-w-sm">
      {{ t('common.httpError.d404') }}
    </p>
    <p class="text-caption font-mono text-text-muted mb-6 max-w-sm break-all">{{ attemptedPath }}</p>
    <div class="flex items-center gap-3">
      <RouterLink to="/">
        <UiButton variant="brand" size="md">{{ t('common.httpError.backHome') }}</UiButton>
      </RouterLink>
      <RouterLink to="/explore">
        <UiButton variant="secondary" size="md">{{ t('common.httpError.exploreCta') }}</UiButton>
      </RouterLink>
    </div>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute } from 'vue-router';
import UiButton from '@/components/ui/UiButton.vue';
import BrandLogo from '@/components/BrandLogo.vue';

const { t } = useI18n();

const route = useRoute();
const attemptedPath = computed(() => route.fullPath);

// Real 404 status on SSR (no soft-404 for crawlers or monitors).
if (import.meta.server) {
  const event = useRequestEvent();
  if (event) setResponseStatus(event, 404);
}
</script>
