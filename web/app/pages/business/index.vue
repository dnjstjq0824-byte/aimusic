<template>
  <div class="space-y-12">
    <section class="mx-auto max-w-3xl px-4 pt-4 text-center sm:pt-8">
      <h1 class="font-display text-4xl font-medium leading-tight tracking-[-0.03em] text-white sm:text-5xl">
        {{ t('business.heroTitle') }}
      </h1>
      <p class="mx-auto mt-4 max-w-xl text-sm leading-relaxed text-text-secondary sm:text-base">
        {{ t('business.heroSub') }}
      </p>
      <div class="mt-6 flex flex-col items-center justify-center gap-2 sm:flex-row">
        <UiButton variant="brand" size="md" @click="scrollToForm">
          {{ t('business.heroCta1') }}
        </UiButton>
        <UiButton variant="secondary" size="md" @click="scrollToSamples">
          {{ t('business.heroCta2') }}
        </UiButton>
      </div>
    </section>

    <section class="grid gap-4 md:grid-cols-3">
      <UiCard v-for="v in values" :key="v.titleKey" class="p-6">
        <h2 class="text-base font-bold text-white">{{ t(v.titleKey) }}</h2>
        <p class="mt-1 text-xs leading-relaxed text-text-secondary">{{ t(v.descKey) }}</p>
      </UiCard>
    </section>

    <section id="samples" :aria-label="t('business.useTitle')" class="scroll-mt-24 space-y-4">
      <h2 class="font-display text-xl font-medium tracking-[-0.03em] text-white">{{ t('business.useTitle') }}</h2>
      <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        <UiCard v-for="pl in samples" :key="pl.id" class="group">
          <CoverMosaic :seeds="sampleSeeds[pl.id] ?? [pl.id]" :title="pl.title" size="md" />
          <p class="mt-2 truncate text-xs font-bold text-white">{{ pl.title }}</p>
          <UiButton variant="secondary" size="sm" class="mt-2" full-width @click="playSample(pl.id)">
            {{ t('business.usePlay') }}
          </UiButton>
        </UiCard>
      </div>
    </section>

    <section :aria-label="t('business.stepsTitle')">
      <h2 class="font-display mb-4 text-xl font-medium tracking-[-0.03em] text-white">{{ t('business.stepsTitle') }}</h2>
      <ol class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <li v-for="(s, i) in steps" :key="s.titleKey" class="rounded-2xl border border-surface-border bg-surface-1 p-5">
          <span class="font-mono text-2xl font-semibold text-brand-500">{{ i + 1 }}</span>
          <h3 class="mt-2 text-sm font-bold text-white">{{ t(s.titleKey) }}</h3>
          <p class="mt-1 text-xs text-text-secondary">{{ t(s.descKey) }}</p>
        </li>
      </ol>
    </section>

    <section :aria-label="t('business.faqTitle')">
      <h2 class="font-display mb-3 text-xl font-medium tracking-[-0.03em] text-white">{{ t('business.faqTitle') }}</h2>
      <div class="space-y-2">
        <details v-for="item in faqItems" :key="item.qKey" class="rounded-xl border border-surface-border bg-surface-1 px-4 py-3">
          <summary class="cursor-pointer text-sm font-bold text-white">{{ t(item.qKey) }}</summary>
          <p class="mt-2 text-xs leading-relaxed text-text-secondary">{{ t(item.aKey) }}</p>
        </details>
      </div>
    </section>

    <section id="inquiry" aria-label="t('business.inqTitle')" class="scroll-mt-24">
      <UiCard class="mx-auto max-w-2xl p-6">
        <h2 class="text-xl font-semibold text-white">{{ t('business.inqTitle') }}</h2>
        <p class="mt-1 text-xs text-text-secondary">{{ t('business.inqSub') }}</p>
        <div class="mt-4">
          <InquiryForm />
        </div>
      </UiCard>
    </section>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { playlistApi, trackApi } from '@/api/endpoints';
import { usePlayerStore } from '@/stores/playerStore';
import { useApiError } from '@/composables/useApiError';
import type { Playlist } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import CoverMosaic from '@/components/ui/CoverMosaic.vue';
import InquiryForm from '@/components/business/InquiryForm.vue';

const { t } = useI18n();
const player = usePlayerStore();
const apiError = useApiError();

// NOTE: labelKeys resolve dynamically (invisible to check-i18n usage
// scan — they are used, do not delete).
const values = [
  { titleKey: 'business.val1T', descKey: 'business.val1D' },
  { titleKey: 'business.val2T', descKey: 'business.val2D' },
  { titleKey: 'business.val3T', descKey: 'business.val3D' },
];
const steps = [
  { titleKey: 'business.step1T', descKey: 'business.step1D' },
  { titleKey: 'business.step2T', descKey: 'business.step2D' },
  { titleKey: 'business.step3T', descKey: 'business.step3D' },
  { titleKey: 'business.step4T', descKey: 'business.step4D' },
];
const faqItems = [
  { qKey: 'business.faq1Q', aKey: 'business.faq1A' },
  { qKey: 'business.faq2Q', aKey: 'business.faq2A' },
  { qKey: 'business.faq3Q', aKey: 'business.faq3A' },
];

const samples = ref<Playlist[]>([]);
const sampleSeeds = ref<Record<string, string[]>>({});
const sampleError = ref<string | null>(null);

function scrollToForm(): void {
  document.getElementById('inquiry')?.scrollIntoView({ behavior: 'smooth', block: 'start' });
}

function scrollToSamples(): void {
  document.getElementById('samples')?.scrollIntoView({ behavior: 'smooth', block: 'start' });
}

async function playSample(playlistId: string): Promise<void> {
  try {
    const entries = (await playlistApi.listTracks(playlistId)) ?? [];
    const resolved = [];
    for (const entry of entries.slice(0, 20)) {
      try {
        resolved.push(await trackApi.get(entry.trackId));
      } catch {
        // Skip vanished tracks.
      }
    }
    if (resolved.length > 0) player.setQueue(resolved, resolved[0]?.id);
  } catch (err: unknown) {
    sampleError.value = apiError(err);
  }
}

onMounted(() => {
  void playlistApi.listPublic(6).then(
    (items) => {
      samples.value = items ?? [];
      for (const pl of samples.value) {
        void playlistApi.listTracks(pl.id).then(
          (trs) => { sampleSeeds.value[pl.id] = trs.slice(0, 4).map((x) => x.trackId); },
          () => { sampleSeeds.value[pl.id] = [pl.id]; },
        );
      }
    },
    (err: unknown) => { sampleError.value = apiError(err); },
  );
});
</script>
