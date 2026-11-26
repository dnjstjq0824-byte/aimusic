<template>
  <div class="space-y-20 sm:space-y-24">
    <section class="relative mx-auto max-w-3xl pt-6 text-center sm:pt-12">
      <div class="hero-glow pointer-events-none absolute left-1/2 top-0 h-80 w-[48rem] max-w-[100vw] -translate-x-1/2 opacity-70" aria-hidden="true" />
      <div class="relative">
        <span class="mx-auto flex h-14 w-14 items-center justify-center rounded-full border border-verified/40 bg-verified/10 text-verified" aria-hidden="true">
          <svg class="h-7 w-7" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M12 3 4 6v6c0 5 3.5 8 8 9 4.5-1 8-4 8-9V6l-8-3z" /><path d="m9 12 2 2 4-4" /></svg>
        </span>
        <h1 class="type-display mt-6 text-4xl text-white sm:text-6xl">{{ t('business.tHeroTitle') }}</h1>
        <p class="mx-auto mt-5 max-w-xl text-sm leading-relaxed text-text-secondary sm:text-base">{{ t('business.tHeroSub') }}</p>
      </div>
    </section>

    <section class="grid items-start gap-12 lg:grid-cols-[1.1fr_1fr] lg:gap-16">
      <ol class="relative ml-4 space-y-8 border-l border-white/10 pl-8">
        <li v-for="(g, i) in gates" :key="g.titleKey" class="relative">
          <span class="absolute -left-[45px] flex h-7 w-7 items-center justify-center rounded-full border border-verified/50 bg-surface-0 font-mono text-xs text-verified" aria-hidden="true">{{ i + 1 }}</span>
          <h2 class="text-lg font-medium text-white sm:text-xl">{{ t(g.titleKey) }}</h2>
          <p class="mt-1.5 text-sm leading-relaxed text-text-secondary">{{ t(g.descKey) }}</p>
        </li>
      </ol>

      <!-- Sample certificate (illustrative; real ones live at /certificate/:number). -->
      <div class="relative">
        <div class="hero-glow pointer-events-none absolute -inset-10 opacity-50" aria-hidden="true" />
        <div class="relative rounded-3xl border border-white/10 bg-white/[0.04] p-6 shadow-2xl backdrop-blur sm:p-8">
          <div class="flex items-center gap-2 text-xs font-medium text-verified">
            <span class="h-1.5 w-1.5 rounded-full bg-verified" aria-hidden="true" />
            {{ t('legal.cert.verified') }}
          </div>
          <p class="mt-5 font-mono text-lg tracking-wider text-white">AVB-••••••••••</p>
          <ul class="mt-6 space-y-3 border-t border-white/[0.08] pt-5">
            <li v-for="key in techKeys" :key="key" class="flex items-center gap-3 text-sm text-text-primary">
              <svg class="h-4 w-4 shrink-0 text-verified" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m5 12 5 5L20 7" /></svg>
              {{ t(key) }}
            </li>
          </ul>
          <p class="mt-6 text-xs text-text-muted">{{ t('business.techTitle') }}</p>
        </div>
      </div>
    </section>

    <section class="grid gap-4 sm:grid-cols-2">
      <NuxtLink
        v-for="l in links"
        :key="l.to"
        :to="localePath(l.to)"
        class="group flex items-center justify-between rounded-2xl border border-white/[0.08] bg-white/[0.02] px-6 py-5 transition-colors hover:border-white/20 hover:bg-white/[0.05]"
      >
        <span class="text-base font-medium text-white">{{ t(l.labelKey) }}</span>
        <span class="text-text-muted transition-transform group-hover:translate-x-0.5 group-hover:text-white" aria-hidden="true">→</span>
      </NuxtLink>
    </section>

    <UiClosingCta
      :title="t('business.tHeroTitle')"
      :primary-to="localePath('/explore')"
      :primary-label="t('business.tCta')"
      :secondary-to="localePath('/business')"
      :secondary-label="t('common.footer.lBusiness')"
    />
  </div>
</template>

<script setup lang="ts">
import UiClosingCta from '@/components/ui/UiClosingCta.vue';

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
const links = [
  { to: '/models', labelKey: 'common.footer.lModels' },
  { to: '/certificate', labelKey: 'common.footer.lCert' },
];
</script>
