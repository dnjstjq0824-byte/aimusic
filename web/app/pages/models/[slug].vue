<!-- AI model card (/models/:slug): conditions, rejections, checks + public tracks made with it. -->
<template>
  <div v-if="meta && item" class="space-y-12">
    <NuxtLink :to="localePath('/models')" class="inline-flex items-center gap-1 text-xs text-text-secondary hover:text-white">
      <span aria-hidden="true">←</span> {{ c.models.back }}
    </NuxtLink>

    <UiPageHeader :title="meta.name" :description="item.summary">
      <template #eyebrow>
        <span
          class="rounded-full px-2.5 py-0.5 text-xs font-medium"
          :class="meta.status === 'blocked' ? 'bg-accent-crimson/10 text-accent-crimson' : 'bg-verified/10 text-verified'"
        >
          {{ statusLabel }}
        </span>
        <span class="text-xs text-text-muted">{{ item.vendor }}</span>
      </template>
    </UiPageHeader>

    <section class="grid gap-4 md:grid-cols-3">
      <div v-if="item.accepted.length > 0" class="rounded-2xl border border-white/[0.08] bg-white/[0.03] p-6">
        <h2 class="text-sm font-medium text-verified">{{ c.models.acceptedLabel }}</h2>
        <ul class="mt-3 space-y-2.5">
          <li v-for="line in item.accepted" :key="line" class="flex gap-2 text-sm text-text-primary">
            <svg class="mt-0.5 h-4 w-4 shrink-0 text-verified" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m5 12 5 5L20 7" /></svg>
            {{ line }}
          </li>
        </ul>
      </div>
      <div class="rounded-2xl border border-white/[0.08] bg-white/[0.03] p-6">
        <h2 class="text-sm font-medium text-accent-crimson">{{ c.models.rejectedLabel }}</h2>
        <ul class="mt-3 space-y-2.5">
          <li v-for="line in item.rejected" :key="line" class="flex gap-2 text-sm text-text-primary">
            <svg class="mt-0.5 h-4 w-4 shrink-0 text-accent-crimson" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" aria-hidden="true"><path d="M6 6l12 12M18 6 6 18" /></svg>
            {{ line }}
          </li>
        </ul>
      </div>
      <div class="rounded-2xl border border-white/[0.08] bg-white/[0.03] p-6">
        <h2 class="text-sm font-medium text-text-muted">{{ c.models.notesLabel }}</h2>
        <ul class="mt-3 space-y-2.5">
          <li v-for="line in item.notes" :key="line" class="flex gap-2 text-sm text-text-secondary">
            <span class="mt-2 h-1 w-1 shrink-0 rounded-full bg-text-muted" aria-hidden="true" />{{ line }}
          </li>
        </ul>
      </div>
    </section>

    <section v-if="meta.status !== 'blocked'" :aria-label="c.models.tracksLabel">
      <div class="mb-4 flex items-end justify-between gap-2">
        <h2 class="type-display text-2xl text-white">{{ c.models.tracksLabel }}</h2>
        <NuxtLink
          v-if="tracks.length > 0"
          :to="localePath({ path: '/explore', query: { provider: meta.code } })"
          class="text-sm text-text-secondary hover:text-white"
        >
          {{ t('common.seeAll') }} →
        </NuxtLink>
      </div>
      <div v-if="tracks.length > 0" class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <TrackCard v-for="tr in tracks" :key="tr.id" :track="tr" @play="playFrom(tr)" />
      </div>
      <p v-else class="rounded-2xl border border-dashed border-white/10 px-6 py-10 text-center text-sm text-text-secondary">{{ c.models.tracksEmpty }}</p>
    </section>

    <p class="text-xs text-text-muted">{{ c.models.basis }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useSiteContent } from '@/data/site';
import { modelBySlug } from '@/data/models';
import { usePublicSeo } from '@/composables/usePublicSeo';
import { fetchPublic } from '@/utils/publicApi';
import { breadcrumbLd, canonicalFor } from '@/utils/seo';
import { usePlayerStore } from '@/stores/playerStore';
import type { Track, TrackPage } from '@/api/types';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import TrackCard from '@/components/TrackCard.vue';

const localePath = useLocalePath();
const { t, locale } = useI18n();
const route = useRoute();
const player = usePlayerStore();

const slug = computed(() => String(route.params.slug ?? ''));
const meta = computed(() => modelBySlug(slug.value));
if (!meta.value) {
  throw showError({ statusCode: 404, statusMessage: 'Model not found' });
}
const c = await useSiteContent();
type ModelItems = typeof c.value.models.items;
const item = computed(() => (c.value.models.items as Record<string, ModelItems[keyof ModelItems]>)[slug.value]);

const { data: trackData } = await useAsyncData(
  () => `model-tracks-${slug.value}`,
  () => fetchPublic<TrackPage>('/tracks', { limit: 12, provider: meta.value?.code ?? '', sort: 'popular' }).catch(() => null),
);
const tracks = computed<Track[]>(() => trackData.value?.items ?? []);

const statusLabel = computed(() => {
  const s = meta.value?.status;
  if (s === 'own') return c.value.models.statusOwn;
  if (s === 'blocked') return c.value.models.statusBlocked;
  return c.value.models.statusAccepted;
});

function playFrom(tr: Track): void {
  player.setQueue(tracks.value, tr.id);
}

usePublicSeo({
  title: computed(() => `${meta.value?.name ?? ''} · ${c.value.models.eyebrow} · AVIBE`),
  description: computed(() => item.value?.summary ?? c.value.models.lead),
  path: computed(() => `/models/${slug.value}`),
  locale,
});
useSchemaOrg([
  breadcrumbLd([
    { name: 'AVIBE', url: canonicalFor('/', locale.value) },
    { name: c.value.models.eyebrow, url: canonicalFor('/models', locale.value) },
    { name: meta.value?.name ?? '', url: canonicalFor(`/models/${slug.value}`, locale.value) },
  ]),
]);
</script>
