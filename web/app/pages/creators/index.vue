<!-- Creator directory (/creators): public creators with READY tracks, most-played first. -->
<template>
  <div class="space-y-12">
    <UiPageHeader :eyebrow="c.creators.eyebrow" :title="c.creators.title" :description="c.creators.lead" />

    <section v-if="creators.length > 0" class="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-5">
      <NuxtLink
        v-for="cr in creators"
        :key="cr.handle"
        :to="localePath(`/creators/${cr.handle}`)"
        class="group flex flex-col items-center rounded-2xl border border-white/[0.06] bg-white/[0.02] px-4 py-7 text-center transition-colors hover:border-white/20 hover:bg-white/[0.05]"
      >
        <div class="relative">
          <div class="h-24 w-24 overflow-hidden rounded-full ring-1 ring-white/10">
            <UiCoverArt :seed="cr.handle" :title="cr.displayName" size="fill" />
          </div>
          <span class="absolute bottom-0 right-0 flex h-6 w-6 items-center justify-center rounded-full border-2 border-surface-0 bg-verified" aria-hidden="true">
            <svg class="h-3 w-3 text-surface-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3.5" stroke-linecap="round" stroke-linejoin="round"><path d="m5 12 5 5L20 7" /></svg>
          </span>
        </div>
        <h2 class="mt-4 w-full truncate text-sm font-medium text-white group-hover:text-brand-300">{{ cr.displayName || cr.handle }}</h2>
        <p class="mt-0.5 w-full truncate text-xs text-text-muted">@{{ cr.handle }}</p>
        <p class="mt-3 text-xs text-text-secondary">
          {{ c.creators.tracks.replace('{n}', fmtNum(cr.trackCount)) }} · {{ c.creators.plays.replace('{n}', fmtNum(cr.totalPlays)) }}
        </p>
      </NuxtLink>
    </section>
    <UiEmptyState v-else :title="c.creators.empty" />

    <UiClosingCta
      :title="c.creators.ctaTitle"
      :sub="c.creators.ctaSub"
      :primary-to="localePath('/for-creators')"
      :primary-label="c.creators.ctaBtn"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useSiteContent } from '@/data/site';
import { usePublicSeo } from '@/composables/usePublicSeo';
import { fetchCreatorDirectory } from '@/utils/creatorDirectory';
import { breadcrumbLd, canonicalFor } from '@/utils/seo';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiCoverArt from '@/components/ui/UiCoverArt.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';
import UiClosingCta from '@/components/ui/UiClosingCta.vue';

const localePath = useLocalePath();
const { locale } = useI18n();
const c = await useSiteContent();
const nf = new Intl.NumberFormat();
const fmtNum = (n: number) => nf.format(n);

const { data } = await useAsyncData('creators-directory', () => fetchCreatorDirectory(60));
const creators = computed(() => data.value ?? []);

usePublicSeo({
  title: computed(() => `${c.value.creators.title} · AVIBE`),
  description: computed(() => c.value.creators.lead),
  path: '/creators',
  locale,
});
useSchemaOrg([
  breadcrumbLd([
    { name: 'AVIBE', url: canonicalFor('/', locale.value) },
    { name: c.value.creators.eyebrow, url: canonicalFor('/creators', locale.value) },
  ]),
]);
</script>
