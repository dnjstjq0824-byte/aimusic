<!-- Company page (/about): mission, principles, the 4-step path, audiences, live stats. -->
<template>
  <div class="space-y-20 sm:space-y-28">
    <section class="relative mx-auto max-w-3xl pt-6 text-center sm:pt-12">
      <div class="hero-glow pointer-events-none absolute left-1/2 top-0 h-80 w-[48rem] max-w-[100vw] -translate-x-1/2 opacity-70" aria-hidden="true" />
      <div class="relative">
        <p class="text-sm font-medium text-brand-400">{{ c.about.eyebrow }}</p>
        <h1 class="type-display mt-3 text-4xl text-white sm:text-6xl">{{ c.about.title }}</h1>
        <p class="mx-auto mt-5 max-w-2xl text-sm leading-relaxed text-text-secondary sm:text-base">{{ c.about.lead }}</p>
      </div>
    </section>

    <section class="grid gap-10 lg:grid-cols-[1fr_1.3fr] lg:gap-16">
      <h2 class="type-display text-3xl text-white sm:text-4xl">{{ c.about.missionTitle }}</h2>
      <div class="space-y-4 text-sm leading-relaxed text-text-secondary sm:text-base">
        <p v-for="(p, i) in c.about.mission" :key="i">{{ p }}</p>
      </div>
    </section>

    <section :aria-label="c.about.statsTitle">
      <h2 class="mb-6 text-sm font-medium text-brand-400">{{ c.about.statsTitle }}</h2>
      <dl class="grid grid-cols-1 gap-px overflow-hidden rounded-2xl border border-white/[0.06] bg-white/[0.06] sm:grid-cols-3">
        <div v-for="s in stats" :key="s.label" class="bg-surface-0 px-6 py-7">
          <dt class="text-xs text-text-muted">{{ s.label }}</dt>
          <dd class="type-display mt-2 text-4xl text-white">{{ s.value }}</dd>
        </div>
      </dl>
    </section>

    <section>
      <h2 class="type-display mb-8 text-3xl text-white sm:text-4xl">{{ c.about.principlesTitle }}</h2>
      <div class="grid gap-4 md:grid-cols-3">
        <article v-for="(p, i) in c.about.principles" :key="i" class="rounded-2xl border border-white/[0.08] bg-white/[0.03] p-6">
          <span class="font-mono text-xs text-text-muted">0{{ i + 1 }}</span>
          <h3 class="mt-3 text-lg font-medium text-white">{{ p.t }}</h3>
          <p class="mt-2 text-sm leading-relaxed text-text-secondary">{{ p.d }}</p>
        </article>
      </div>
    </section>

    <section>
      <div class="mb-8 flex flex-wrap items-end justify-between gap-4">
        <h2 class="type-display text-3xl text-white sm:text-4xl">{{ c.about.stepsTitle }}</h2>
        <NuxtLink :to="localePath('/trust')" class="text-sm text-text-secondary transition-colors hover:text-white">
          {{ c.cert.trustLink }} →
        </NuxtLink>
      </div>
      <ol class="relative grid gap-6 md:grid-cols-4">
        <li v-for="(s, i) in c.about.steps" :key="i" class="relative border-t border-white/10 pt-5">
          <span class="absolute -top-[5px] left-0 h-2.5 w-2.5 rounded-full bg-verified" aria-hidden="true" />
          <h3 class="text-base font-medium text-white">{{ s.t }}</h3>
          <p class="mt-2 text-sm leading-relaxed text-text-secondary">{{ s.d }}</p>
        </li>
      </ol>
    </section>

    <section>
      <h2 class="type-display mb-8 text-3xl text-white sm:text-4xl">{{ c.about.audiencesTitle }}</h2>
      <div class="grid gap-4 md:grid-cols-3">
        <NuxtLink
          v-for="(a, i) in c.about.audiences"
          :key="i"
          :to="localePath(a.to)"
          class="group relative flex min-h-56 flex-col justify-end overflow-hidden rounded-2xl border border-white/[0.08] p-6"
        >
          <img :src="audienceImages[i]" alt="" loading="lazy" class="absolute inset-0 h-full w-full object-cover opacity-50 transition-transform duration-500 group-hover:scale-105">
          <div class="absolute inset-0 bg-gradient-to-t from-surface-0 via-surface-0/70 to-transparent" aria-hidden="true" />
          <div class="relative">
            <h3 class="text-xl font-medium text-white">{{ a.t }}</h3>
            <p class="mt-1 text-sm text-text-secondary">{{ a.d }}</p>
            <span class="mt-4 inline-flex items-center gap-1 text-sm font-medium text-white">{{ a.cta }} <span aria-hidden="true">→</span></span>
          </div>
        </NuxtLink>
      </div>
    </section>

    <UiClosingCta
      :title="c.about.ctaTitle"
      :sub="c.about.ctaSub"
      :primary-to="localePath('/explore')"
      :primary-label="c.about.ctaBtn"
      :secondary-to="localePath('/contact')"
      :secondary-label="c.about.ctaBtn2"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useSiteContent } from '@/data/site';
import { usePublicSeo } from '@/composables/usePublicSeo';
import { fetchPublic } from '@/utils/publicApi';
import { fetchCreatorDirectory } from '@/utils/creatorDirectory';
import { breadcrumbLd, canonicalFor } from '@/utils/seo';
import UiClosingCta from '@/components/ui/UiClosingCta.vue';
import type { TrackPage } from '@/api/types';

const localePath = useLocalePath();
const { locale } = useI18n();
const c = await useSiteContent();
const fmt = new Intl.NumberFormat();

const { data: live } = await useAsyncData('about-stats', async () => {
  const [page, creators] = await Promise.all([
    fetchPublic<TrackPage>('/tracks', { limit: 1 }).catch(() => null),
    fetchCreatorDirectory(100).catch(() => null),
  ]);
  return { tracks: page?.total ?? null, creators: creators?.length ?? null };
});

const stats = computed(() => [
  { label: c.value.about.statTracks, value: live.value?.tracks != null ? fmt.format(live.value.tracks) : '—' },
  { label: c.value.about.statCreators, value: live.value?.creators != null ? fmt.format(live.value.creators) : '—' },
  { label: c.value.about.statLocales, value: '5' },
]);

const audienceImages = ['/images/stock/feature-listen.webp', '/images/stock/start-focus.webp', '/images/stock/start-store.webp'];

usePublicSeo({
  title: computed(() => `${c.value.about.eyebrow} · AVIBE`),
  description: computed(() => c.value.about.lead),
  path: '/about',
  locale,
});
useSchemaOrg([
  breadcrumbLd([
    { name: 'AVIBE', url: canonicalFor('/', locale.value) },
    { name: c.value.about.eyebrow, url: canonicalFor('/about', locale.value) },
  ]),
]);
</script>
