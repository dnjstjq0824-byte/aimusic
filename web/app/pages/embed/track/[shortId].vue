<!-- Track embed player (WI-S4): lightweight SSR, noindex, crawlable. -->
<template>
  <div class="mx-auto flex h-screen max-h-[180px] min-w-[280px] max-w-[640px] items-center gap-3 p-3">
    <template v-if="track">
      <UiCoverArt :seed="track.id" :title="track.title" size="md" rounded="lg" :verified="false" />
      <div class="min-w-0 flex-1">
        <p class="truncate text-sm font-bold text-white">{{ track.title }}</p>
        <p class="truncate text-xs text-text-secondary">{{ track.creatorName }}</p>
        <div class="mt-2 flex items-center gap-2">
          <button
            type="button"
            class="inline-flex h-9 min-w-9 items-center justify-center rounded-full bg-brand-500 px-3 text-sm font-bold text-white"
            :aria-label="t('seo.embedPlay')"
            @click="openFull"
          >
            ▶
          </button>
          <a
            :href="fullUrl"
            target="_blank"
            rel="noopener"
            class="text-xs font-semibold text-brand-400 hover:underline"
          >
            {{ t('seo.embedOpen') }}
          </a>
        </div>
      </div>
      <span class="text-caption font-semibold tracking-tight text-text-muted">AVIBE</span>
    </template>
    <p v-else class="w-full text-center text-xs text-text-muted">{{ t('seo.embedMissing') }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue';
import { useRoute } from 'vue-router';
import { fetchPublic } from '@/utils/publicApi';
import { usePublicSeo } from '@/composables/usePublicSeo';
import { canonicalTrackPath, normalizeSeoLocale } from '@/utils/seo';
import { trackEvent } from '@/utils/analytics';
import type { Track } from '@/api/types';
import UiCoverArt from '@/components/ui/UiCoverArt.vue';

defineI18nRoute(false);

const { t, locale, setLocale } = useI18n();
const route = useRoute();

const shortId = computed(() => String(route.params.shortId ?? ''));
const wantLang = normalizeSeoLocale(typeof route.query.lang === 'string' ? route.query.lang : '');
if (route.query.lang && wantLang !== locale.value) {
  await setLocale(wantLang);
}

const { data: track } = await useAsyncData(() => `embed-track-${shortId.value}`, () =>
  fetchPublic<Track>(`/tracks/${encodeURIComponent(shortId.value)}`).catch((): null => null),
);

const fullUrl = computed(() =>
  track.value?.slug && track.value?.shortId
    ? `https://avibemusic.com${canonicalTrackPath(track.value.slug, track.value.shortId)}`
    : 'https://avibemusic.com/',
);

usePublicSeo({
  title: track.value ? `${track.value.title} – AVIBE` : 'AVIBE',
  description: t('seo.embedDescription'),
  path: `/embed/track/${shortId.value}`,
  locale: locale.value,
  noindex: true,
});

function openFull(): void {
  window.open(fullUrl.value, '_blank', 'noopener');
}

onMounted(() => {
  trackEvent({ name: 'embed_loaded', props: { kind: 'track' } });
});
</script>
