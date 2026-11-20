<template>
  <div class="mx-auto flex min-h-[60vh] max-w-lg flex-col items-center justify-center text-center px-4">
    <BrandLogo :size="40" />
    <div class="relative my-6" aria-hidden="true">
      <div class="text-8xl font-semibold text-surface-2 tracking-tighter select-none">500</div>
      <div class="absolute inset-0 flex items-center justify-center text-2xl font-semibold text-accent-crimson">
        ERROR
      </div>
    </div>
    <h1 class="text-xl font-bold text-white mb-2">{{ t('common.httpError.t500') }}</h1>
    <p class="text-xs text-text-secondary leading-relaxed mb-6 max-w-sm">
      {{ t('common.httpError.d500') }}
    </p>
    <div class="flex items-center gap-3">
      <UiButton variant="brand" size="md" @click="reload">{{ t('common.actions.reload') }}</UiButton>
      <RouterLink to="/">
        <UiButton variant="secondary" size="md">{{ t('common.httpError.home') }}</UiButton>
      </RouterLink>
    </div>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { useI18n } from 'vue-i18n';
import UiButton from '@/components/ui/UiButton.vue';
import BrandLogo from '@/components/BrandLogo.vue';

const { t } = useI18n();

function reload() {
  window.location.reload();
}
</script>
