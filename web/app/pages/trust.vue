<template>
  <div class="space-y-12">
    <section class="mx-auto max-w-3xl px-4 pt-4 text-center sm:pt-8">
      <h1 class="font-display text-4xl font-medium leading-tight tracking-[-0.03em] text-white sm:text-5xl">
        {{ t('business.tHeroTitle') }}
      </h1>
      <p class="mx-auto mt-4 max-w-xl text-sm leading-relaxed text-text-secondary sm:text-base">
        {{ t('business.tHeroSub') }}
      </p>
    </section>

    <section>
      <ol class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <li v-for="(g, i) in gates" :key="g.titleKey" class="rounded-2xl border border-surface-border bg-surface-1 p-5">
          <span class="font-mono text-2xl font-semibold text-verified">{{ i + 1 }}</span>
          <h2 class="mt-2 text-sm font-bold text-white">{{ t(g.titleKey) }}</h2>
          <p class="mt-1 text-xs leading-relaxed text-text-secondary">{{ t(g.descKey) }}</p>
        </li>
      </ol>
    </section>

    <section>
      <h2 class="font-display mb-3 text-xl font-medium tracking-[-0.03em] text-white">{{ t('business.techTitle') }}</h2>
      <ul class="grid gap-2 sm:grid-cols-2">
        <li v-for="key in techKeys" :key="key" class="flex items-center gap-2 rounded-xl border border-surface-border bg-surface-1 px-4 py-3 text-xs font-semibold text-text-primary">
          <span class="h-1.5 w-1.5 shrink-0 rounded-full bg-brand-500" aria-hidden="true" />
          {{ t(key) }}
        </li>
      </ul>
    </section>

    <section class="text-center">
      <RouterLink :to="localePath('/explore')">
        <UiButton variant="brand" size="md">{{ t('business.tCta') }}</UiButton>
      </RouterLink>
    </section>
  </div>
</template>

<script setup lang="ts">
import UiButton from '@/components/ui/UiButton.vue';

const { t } = useI18n();
const localePath = useLocalePath();

// NOTE: labelKeys resolve dynamically (invisible to check-i18n usage
// scan — they are used, do not delete).
const gates = [
  { titleKey: 'business.g1T', descKey: 'business.g1D' },
  { titleKey: 'business.g2T', descKey: 'business.g2D' },
  { titleKey: 'business.g3T', descKey: 'business.g3D' },
  { titleKey: 'business.g4T', descKey: 'business.g4D' },
];
const techKeys = ['business.techHash', 'business.techProbe', 'business.techCert', 'business.techHls'];
</script>
