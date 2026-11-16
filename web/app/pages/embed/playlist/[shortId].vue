<!-- Playlist embed player (WI-S4): lightweight SSR, noindex, crawlable. -->
<template>
  <div class="mx-auto flex h-screen max-h-[360px] min-w-[280px] max-w-[640px] flex-col gap-2 overflow-hidden p-3">
    <template v-if="playlist">
      <div class="flex items-center gap-3">
        <CoverMosaic :seeds="seeds" :title="playlist.title" size="md" />
        <div class="min-w-0 flex-1">
          <p class="truncate text-sm font-bold text-white">{{ playlist.title }}</p>
          <p class="truncate text-xs text-text-secondary">
            {{ playlist.ownerName }} · {{ t('seo.embedTrackCount', { count: tracks.length }) }}
          </p>
        </div>
        <span class="text-caption font-black tracking-tight text-text-muted">AVIBE</span>
      </div>
      <ol class="min-h-0 flex-1 space-y-1 overflow-y-auto">
        <li
          v-for="(tr, i) in tracks"
          :key="tr.id"
          class="flex items-center gap-2 rounded-lg px-2 py-1.5 text-xs hover:bg-surface-1"
        >
          <span class="w-5 shrink-0 text-right font-mono text-text-muted">{{ i + 1 }}</span>
          <span class="min-w-0 flex-1 truncate font-semibold text-text-primary">{{ tr.title }}</span>
          <span class="shrink-0 text-text-muted">{{ fmtDuration(tr.durationMs) }}</span>
        </li>
      </ol>
      <a :href="fullUrl" target="_blank" rel="noopener" class="text-center text-xs font-semibold text-brand-400 hover:underline">
        {{ t('seo.embedOpen') }}
      </a>
    </template>
    <p v-else class="w-full py-8 text-center text-xs text-text-muted">{{ t('seo.embedMissing') }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue';
import { useRoute } from 'vue-router';
import { fetchPublic } from '@/utils/publicApi';
import { usePublicSeo } from '@/composables/usePublicSeo';
import { canonicalPlaylistPath, normalizeSeoLocale } from '@/utils/seo';
import { trackEvent } from '@/utils/analytics';
import type { Playlist, PlaylistTrack, Track } from '@/api/types';
import CoverMosaic from '@/components/ui/CoverMosaic.vue';

defineI18nRoute(false);

const { t, locale, setLocale } = useI18n();
const route = useRoute();

const shortId = computed(() => String(route.params.shortId ?? ''));
const wantLang = normalizeSeoLocale(typeof route.query.lang === 'string' ? route.query.lang : '');
if (route.query.lang && wantLang !== locale.value) {
  await setLocale(wantLang);
}

const { data: pageData } = await useAsyncData(
  () => `embed-playlist-${shortId.value}`,
  async () => {
    const enc = encodeURIComponent(shortId.value);
    try {
      const [pl, entries] = await Promise.all([
        fetchPublic<Playlist>(`/playlists/${enc}`),
        fetchPublic<PlaylistTrack[]>(`/playlists/${enc}/tracks`),
      ]);
      if (!pl.isPublic && !pl.isOfficialCuration) return null;
      const tracks = (
        await Promise.all(
          (entries ?? []).slice(0, 20).map((e) => fetchPublic<Track>(`/tracks/${encodeURIComponent(e.trackId)}`).catch((): null => null)),
        )
      ).filter((x): x is Track => x !== null);
      return { playlist: pl, tracks };
    } catch {
      return null;
    }
  },
);
const playlist = computed(() => pageData.value?.playlist ?? null);
const tracks = computed(() => pageData.value?.tracks ?? []);
const seeds = computed(() => tracks.value.slice(0, 4).map((x) => x.id));

const fullUrl = computed(() =>
  playlist.value?.slug && playlist.value?.shortId
    ? `https://avibemusic.com${canonicalPlaylistPath(playlist.value.slug, playlist.value.shortId)}`
    : 'https://avibemusic.com/',
);

usePublicSeo({
  title: playlist.value ? `${playlist.value.title} – AVIBE` : 'AVIBE',
  description: t('seo.embedDescription'),
  path: `/embed/playlist/${shortId.value}`,
  locale: locale.value,
  noindex: true,
});

function fmtDuration(ms?: number | null): string {
  if (!ms) return '--:--';
  const total = Math.round(ms / 1000);
  return `${Math.floor(total / 60)}:${String(total % 60).padStart(2, '0')}`;
}

onMounted(() => {
  trackEvent({ name: 'embed_loaded', props: { kind: 'playlist' } });
});
</script>
