<template>
  <div class="space-y-12">
    <section class="mx-auto max-w-3xl px-4 pt-4 text-center sm:pt-8">
      <h1 class="font-display text-4xl font-medium leading-tight tracking-[-0.03em] text-white sm:text-5xl">
        {{ t('business.cHeroTitle') }}
      </h1>
      <p class="mx-auto mt-4 max-w-xl text-sm leading-relaxed text-text-secondary sm:text-base">
        {{ t('business.cHeroSub') }}
      </p>
      <div class="mt-6 flex flex-col items-center justify-center gap-2 sm:flex-row">
        <RouterLink :to="localePath('/signup/creator')">
          <UiButton variant="brand" size="md">{{ t('business.cCta1') }}</UiButton>
        </RouterLink>
        <RouterLink :to="localePath('/trust')">
          <UiButton variant="secondary" size="md">{{ t('business.cCta2') }}</UiButton>
        </RouterLink>
      </div>
    </section>

    <section>
      <ol class="grid gap-4 sm:grid-cols-3">
        <li v-for="(s, i) in steps" :key="s.titleKey" class="rounded-2xl border border-surface-border bg-surface-1 p-5">
          <span class="font-mono text-2xl font-semibold text-brand-500">{{ i + 1 }}</span>
          <h2 class="mt-2 text-sm font-bold text-white">{{ t(s.titleKey) }}</h2>
          <p class="mt-1 text-xs text-text-secondary">{{ t(s.descKey) }}</p>
        </li>
      </ol>
    </section>

    <section class="grid gap-4 md:grid-cols-2">
      <UiCard class="p-6">
        <h2 class="text-lg font-bold text-white">{{ t('business.cPayT') }}</h2>
        <p class="mt-1 text-xs leading-relaxed text-text-secondary">{{ t('business.cPayD') }}</p>
        <RouterLink :to="localePath('/signup/creator')" class="mt-3 inline-block text-xs font-bold text-brand-400 hover:underline">
          {{ t('business.cCta1') }} →
        </RouterLink>
      </UiCard>
      <UiCard class="p-6">
        <h2 class="text-lg font-bold text-white">{{ t('business.cTrustT') }}</h2>
        <p class="mt-1 text-xs leading-relaxed text-text-secondary">{{ t('business.cTrustD') }}</p>
        <RouterLink :to="localePath('/trust')" class="mt-3 inline-block text-xs font-bold text-brand-400 hover:underline">
          {{ t('business.cTrustCta') }} →
        </RouterLink>
      </UiCard>
    </section>
  </div>
</template>

<script setup lang="ts">
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';

const { t } = useI18n();
const localePath = useLocalePath();

// NOTE: labelKeys resolve dynamically (invisible to check-i18n usage
// scan — they are used, do not delete).
const steps = [
  { titleKey: 'business.cStep1T', descKey: 'business.cStep1D' },
  { titleKey: 'business.cStep2T', descKey: 'business.cStep2D' },
  { titleKey: 'business.cStep3T', descKey: 'business.cStep3D' },
];
</script>
