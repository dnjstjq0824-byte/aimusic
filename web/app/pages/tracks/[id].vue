<template>
  <div class="space-y-6">
    <RouterLink
      to="/"
      class="inline-flex items-center gap-1.5 text-xs text-text-secondary hover:text-white transition-colors"
    >
      <span>← {{ t('track.back') }}</span>
    </RouterLink>

    <h1 v-if="error" class="sr-only">{{ t('track.errTitle') }}</h1>
    <UiErrorBanner
      :message="error"
      :title="t('track.errTitle')"
      :retry-label="t('track.retry')"
      @retry="loadTrack"
    />

    <div v-if="!error && loading" class="space-y-4">
      <UiSkeleton height="h-32" width="w-full" rounded="xl" />
      <UiSkeleton height="h-64" width="w-full" rounded="xl" />
    </div>

    <div v-else-if="!error && track" class="grid gap-6 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
      <!-- Main Column -->
      <section class="space-y-6 min-w-0">
        <UiCard class="relative overflow-hidden">
          <div class="flex flex-col sm:flex-row sm:items-center gap-5">
            <div class="relative mx-auto shrink-0 sm:mx-0">
              <UiCoverArt :seed="track.id" :title="track.title" size="xl" rounded="xl" :verified="track.status === 'READY'" />
              <button
                type="button"
                class="absolute inset-0 flex items-center justify-center rounded-2xl bg-black/40 text-white opacity-0 transition-opacity hover:opacity-100 focus:opacity-100"
                :aria-label="t('track.playNow')"
                :disabled="track.status !== 'READY'"
                @click="play"
              >
                <svg class="h-10 w-10" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                  <path d="M8 5v14l11-7z" />
                </svg>
              </button>
            </div>

            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2 flex-wrap">
                <AiBadge :track="track" />
                <UiBadge v-if="track.genre" variant="neutral" size="xs">
                  {{ tagName(track.genre) }}
                </UiBadge>
                <UiBadge v-for="mood in (track.moodTags ?? []).slice(0, 3)" :key="mood" variant="neutral" size="xs">
                  {{ tagName(mood) }}
                </UiBadge>
              </div>

              <h1 class="font-display mt-2 text-2xl sm:text-3xl font-black text-white tracking-tight break-words">
                {{ track.title }}
              </h1>
              <p v-if="track.creatorName" class="mt-1 flex items-center gap-1.5 text-sm text-text-primary">
                <RouterLink
                  v-if="track.creatorHandle"
                  :to="localePath({ name: 'creators-handle', params: { handle: track.creatorHandle } })"
                  class="hover:text-brand-300 hover:underline"
                >
                  {{ track.creatorName }}
                </RouterLink>
                <span v-else>{{ track.creatorName }}</span>
                <VerifiedSeal
                  v-if="cert"
                  :size="16"
                  :label="t('player.verified')"
                  :full-label="t('player.verified')"
                  :show-label="false"
                />
              </p>

              <div class="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-text-secondary">
                <span>{{ t('track.plays', { count: fmt.number(track.playCount) }) }}</span>
                <span aria-hidden="true">·</span>
                <span>{{ fmtDuration(track.durationMs) }}</span>
                <template v-if="track.bpm">
                  <span aria-hidden="true">·</span>
                  <span>BPM {{ fmtBpm(track.bpm) }}</span>
                </template>
                <template v-if="track.musicalKey">
                  <span aria-hidden="true">·</span>
                  <span>{{ track.musicalKey }}</span>
                </template>
              </div>
            </div>
          </div>

          <div class="mt-6 flex flex-wrap items-center gap-2 pt-4 border-t border-surface-border">
            <UiButton variant="brand" size="sm" :disabled="track.status !== 'READY'" @click="play">
              {{ t('track.playNow') }}
            </UiButton>
            <UiButton
              variant="outline"
              size="sm"
              :aria-pressed="liked"
              :aria-label="liked ? t('library.unlikeAria') : t('library.likeAria')"
              :class="liked ? 'border-accent-crimson/60 text-accent-crimson' : ''"
              @click="toggleLike"
            >
              <span aria-hidden="true">{{ liked ? '♥' : '♡' }}</span>
            </UiButton>
            <UiButton variant="secondary" size="sm" @click="openAddToPlaylist()">
              {{ t('track.addToPlaylist') }}
            </UiButton>
            <UiButton variant="outline" size="sm" @click="shareTrack">
              {{ t('common.actions.share') }}
            </UiButton>
            <UiMenu :items="moreItems" :label="t('track.moreActions')" @select="onMoreSelect" />
          </div>
          <div class="mt-4 border-t border-surface-border pt-4">
            <ShareButtons :url="shareUrl" :title="track.title" />
          </div>
        </UiCard>

        <UiCard>
          <div class="mb-3 flex items-center justify-between">
            <h3 class="text-xs font-bold text-text-secondary uppercase tracking-wider">{{ t('track.waveTitle') }}</h3>
            <span class="text-xs font-mono text-text-secondary">{{ t('track.waveQuality') }}</span>
          </div>
          <div
            v-if="waveformPeaks.length > 0"
            class="flex h-16 items-end gap-0.5 sm:gap-[3px] py-2 bg-surface-0/60 rounded-xl p-3 border border-surface-border"
            role="img"
            :aria-label="t('track.waveTitle')"
          >
            <div
              v-for="(peak, idx) in waveformPeaks"
              :key="idx"
              class="waveform-bar min-w-[2px] flex-1 rounded-t bg-gradient-to-t from-brand-600 to-amber-400"
              :style="{ height: `${Math.max(10, Math.round(peak * 100))}%` }"
            />
          </div>
          <div v-else class="py-6 text-center text-xs text-text-muted">
            {{ t('track.waveEmpty') }}
          </div>
        </UiCard>

        <UiCard v-if="track.lyrics" class="space-y-1.5">
          <h3 class="text-caption font-bold text-text-secondary">{{ t('track.lyrics') }}</h3>
          <div class="text-xs text-text-primary leading-relaxed whitespace-pre-wrap max-h-48 overflow-y-auto">
            {{ track.lyrics }}
          </div>
        </UiCard>

        <UiCard class="space-y-4">
          <div class="flex items-start justify-between gap-3 border-b border-surface-border pb-3">
            <div class="min-w-0">
              <h3 class="text-sm font-bold text-white">{{ t('track.aiTitle') }}</h3>
              <p class="text-caption text-text-secondary mt-0.5">{{ t('track.aiSub') }}</p>
            </div>
          </div>

          <div class="grid gap-3 sm:grid-cols-2 text-xs">
            <div class="p-3 rounded-xl bg-surface-0 border border-surface-border">
              <span class="text-text-muted text-caption uppercase font-bold block">{{ t('track.aiPlatformModel') }}</span>
              <span class="font-semibold text-white text-sm">{{ providerDisplayName(track.aiProvider) }}</span>
              <span v-if="track.aiModelVersion" class="text-text-secondary ml-1 text-xs">({{ track.aiModelVersion }})</span>
            </div>
            <div class="p-3 rounded-xl bg-surface-0 border border-surface-border">
              <span class="text-text-muted text-caption uppercase font-bold block">{{ t('track.aiGenType') }}</span>
              <span class="font-semibold text-brand-400 text-sm">
                {{ track.aiGenerationType === 'AI_ASSISTED_HUMAN' ? t('track.aiAssisted') : t('track.aiFull') }}
              </span>
            </div>
            <div v-if="aiCreditText" class="p-3 rounded-xl bg-surface-0 border border-surface-border sm:col-span-2">
              <span class="text-text-muted text-caption uppercase font-bold block">{{ t('track.aiCredits') }}</span>
              <span class="font-semibold text-white text-sm">{{ aiCreditText }}</span>
            </div>
          </div>

          <div v-if="track.aiPrompt && track.isPromptPublic !== false" class="space-y-1.5">
            <div class="text-caption font-bold text-text-secondary">{{ t('track.aiPrompt') }}</div>
            <div class="p-3 rounded-xl bg-surface-0 border border-surface-border font-mono text-xs text-text-primary leading-relaxed break-all">
              {{ track.aiPrompt }}
            </div>
          </div>
        </UiCard>

        <section v-if="inPlaylists.length > 0" :aria-label="t('track.inPlaylists')">
          <h3 class="mb-2 text-sm font-bold text-white">{{ t('track.inPlaylists') }}</h3>
          <div class="grid gap-2 sm:grid-cols-2">
            <RouterLink
              v-for="pl in inPlaylists"
              :key="pl.id"
              :to="localePath({ name: 'playlist-detail', params: { id: detailParam(pl) } })"
              class="truncate rounded-xl border border-surface-border bg-surface-1 px-3 py-2 text-xs font-semibold text-text-primary transition-colors hover:border-brand-500/50 hover:text-brand-300"
            >
              {{ pl.title }}
            </RouterLink>
          </div>
        </section>

        <section v-if="similar.length > 0" :aria-label="t('track.similar')">
          <h3 class="mb-2 text-sm font-bold text-white">{{ t('track.similar') }}</h3>
          <div class="space-y-1">
            <TrackRow
              v-for="tr in similar"
              :key="tr.id"
              :track="tr"
              @play="playOther"
              @enqueue="enqueueOther"
              @add-to-playlist="openAddToPlaylist"
              @report="openReport"
            />
          </div>
        </section>

        <section v-if="sameCreator.length > 0" :aria-label="t('track.sameCreator')">
          <h3 class="mb-2 text-sm font-bold text-white">{{ t('track.sameCreator') }}</h3>
          <div class="space-y-1">
            <TrackRow
              v-for="tr in sameCreator"
              :key="tr.id"
              :track="tr"
              @play="playOther"
              @enqueue="enqueueOther"
              @add-to-playlist="openAddToPlaylist"
              @report="openReport"
            />
          </div>
        </section>

        <div class="pt-2 text-center space-x-4">
          <button
            type="button"
            class="text-xs text-text-muted hover:text-accent-crimson hover:underline"
            @click="openReportSelf"
          >
            {{ t('track.reportAction') }}
          </button>
          <ClientOnly>
            <button
              v-if="auth.isAuthenticated"
              type="button"
              class="text-xs text-text-muted hover:text-accent-crimson hover:underline"
              @click="openBlock"
            >
              {{ t('track.blockAction') }}
            </button>
          </ClientOnly>
        </div>
      </section>

      <!-- Sidebar: license panel -->
      <aside class="min-w-0">
        <div class="space-y-4 lg:sticky lg:top-24">
          <UiCard class="space-y-3">
            <h3 class="text-sm font-bold text-white">{{ t('track.licTitle') }}</h3>
            <ul class="space-y-1.5 text-xs text-text-primary">
              <li class="flex items-center gap-2">
                <span class="h-1.5 w-1.5 rounded-full bg-verified" aria-hidden="true" />
                {{ t('track.licBgm') }}
              </li>
              <li class="flex items-center gap-2">
                <span class="h-1.5 w-1.5 rounded-full bg-verified" aria-hidden="true" />
                {{ t('track.licVideo') }}
              </li>
            </ul>
            <RouterLink :to="localePath('/signup/business')">
              <UiButton variant="brand" size="sm" full-width>
                {{ t('track.licCta') }}
              </UiButton>
            </RouterLink>
          </UiCard>

          <UiCard class="space-y-3">
            <div class="flex items-center gap-2">
              <svg class="h-4 w-4 text-verified" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                <path d="M12 3 5 6v5c0 4.4 3 8.4 7 10 4-1.6 7-5.6 7-10V6l-7-3z" />
                <path d="m9 12 2 2 4-4" />
              </svg>
              <h3 class="text-xs font-bold text-white uppercase tracking-wider">
                {{ cert ? t('track.licCertified') : t('track.licPending') }}
              </h3>
            </div>
            <div v-if="!cert" class="rounded-xl border border-surface-border bg-surface-2 p-3 text-xs leading-relaxed text-text-primary">
              {{ t('track.licPendingBody') }}
            </div>
            <div v-else class="rounded-xl border border-verified/20 bg-verified/10 p-3 text-xs space-y-1.5">
              <UiBadge variant="success" size="xs">{{ t('track.licPassed') }}</UiBadge>
              <div class="flex items-center justify-between gap-2 pt-1">
                <span class="text-text-secondary">{{ t('track.licCertNumber') }}</span>
                <RouterLink
                  :to="`/certificate/${cert.certificate.certNumber}`"
                  class="font-mono font-semibold text-brand-400 hover:underline"
                >
                  {{ cert.certificate.certNumber }}
                </RouterLink>
              </div>
            </div>
            <div v-if="!cert && canIssueCert" class="pt-1">
              <UiButton variant="secondary" size="xs" :loading="issuingCert" @click="issueCert">
                {{ t('track.licIssue') }}
              </UiButton>
            </div>
            <div class="flex items-center justify-between gap-2 border-t border-surface-border pt-3">
              <span class="text-caption text-text-secondary">{{ t('track.licTrackId') }}</span>
              <button
                type="button"
                class="shrink-0 rounded-lg border border-surface-border px-2.5 py-1 text-caption font-semibold text-brand-400 hover:border-brand-500/50"
                @click="copyTrackId"
              >
                {{ t('track.licCopyId') }}
              </button>
            </div>
          </UiCard>
        </div>
      </aside>
    </div>

    <DisputeReportModal
      v-if="reportModalOpen && reportTarget"
      :track-id="reportTarget.id"
      :track-title="reportTarget.title"
      @close="reportModalOpen = false"
      @reported="onReported"
    />

    <UiPlaylistPicker
      v-model="playlistModalOpen"
      :track-title="playlistTarget?.title ?? ''"
      :playlists="playlists.playlists"
      :create-loading="creatingPlaylist"
      @select="addToPlaylist"
      @create="createPlaylist"
    />

    <UiConfirmDialog
      v-model="blockConfirmOpen"
      :title="t('track.blockTitle')"
      :body="t('track.blockBody')"
      :confirm-label="t('track.blockConfirm')"
      danger
      :loading="blocking"
      @confirm="confirmBlockPartner"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { trackApi, playlistApi, userBlocksApi, adminApi, engagementApi } from '@/api/endpoints';
import { errCodeOf } from '@/api/client';
import { usePlayerStore } from '@/stores/playerStore';
import { usePlaylistStore } from '@/stores/playlistStore';
import { useAuthStore } from '@/stores/authStore';
import { useToast } from '@/composables/useToast';
import { useApiError } from '@/composables/useApiError';
import { useFormat } from '@/composables/useFormat';
import { usePublicSeo } from '@/composables/usePublicSeo';
import { fetchPublic } from '@/utils/publicApi';
import { tagDisplayName } from '@/utils/tags';
import { providerDisplayName } from '@/utils/providerDisplay';
import {
  breadcrumbLd,
  canonicalFor,
  canonicalTrackPath,
  describeTrack,
  detailParam,
  musicRecordingLd,
  needsCjkThaiFont,
  oembedDiscoveryUrl,
  trackTitle,
} from '@/utils/seo';
import { trackEvent } from '@/utils/analytics';
import type { Track, TrackPage, CertificateView, Playlist } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiCoverArt from '@/components/ui/UiCoverArt.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiPlaylistPicker from '@/components/ui/UiPlaylistPicker.vue';
import UiConfirmDialog from '@/components/ui/UiConfirmDialog.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import UiMenu, { type MenuItem } from '@/components/ui/UiMenu.vue';
import VerifiedSeal from '@/components/ui/VerifiedSeal.vue';
import AiBadge from '@/components/legal/AiBadge.vue';
import DisputeReportModal from '@/components/legal/DisputeReportModal.vue';
import TrackRow from '@/components/TrackRow.vue';
import ShareButtons from '@/components/seo/ShareButtons.vue';
const localePath = useLocalePath();

const { t, te, locale } = useI18n();
const route = useRoute();
const router = useRouter();
const player = usePlayerStore();
const playlists = usePlaylistStore();
const auth = useAuthStore();
const toast = useToast();
const apiError = useApiError();
const fmt = useFormat();

const tagName = (key: string | null | undefined) => tagDisplayName(key, t, te);

const error = ref<string | null>(null);
const cert = ref<CertificateView | null>(null);
const issuingCert = ref(false);
const downloading = ref(false);
const similar = ref<Track[]>([]);
const sameCreator = ref<Track[]>([]);
const inPlaylists = ref<Playlist[]>([]);

// WI-S2: track + cert + internal-link blocks render server-side (guest view,
// identical HTML regardless of login). Related fetches are best-effort.
const param = computed(() => String(route.params.id ?? ''));
const { data: pageData, pending, error: pageError, refresh } = await useAsyncData(
  () => `track-${param.value}`,
  async () => {
    const enc = encodeURIComponent(param.value);
    const [track, cert] = await Promise.all([
      fetchPublic<Track>(`/tracks/${enc}`),
      fetchPublic<CertificateView | null>(`/tracks/${enc}/certificate`).catch((): null => null),
    ]);
    let similar: Track[] = [];
    let sameCreator: Track[] = [];
    let inPlaylists: Playlist[] = [];
    try {
      const mood = track.moodTags?.[0];
      const [byMood, byCreator, pls] = await Promise.all([
        fetchPublic<TrackPage>('/tracks', mood ? { limit: 7, mood } : { limit: 7 }),
        fetchPublic<TrackPage>('/tracks', { limit: 7, partnerId: track.partnerId }),
        fetchPublic<Playlist[]>(`/tracks/${encodeURIComponent(track.id)}/playlists`).catch((): Playlist[] => []),
      ]);
      similar = (byMood.items ?? []).filter((x) => x.id !== track.id).slice(0, 6);
      sameCreator = (byCreator.items ?? []).filter((x) => x.id !== track.id).slice(0, 6);
      inPlaylists = pls ?? [];
    } catch {
      // Related rails stay empty; the core page still renders.
    }
    return { track, cert, similar, sameCreator, inPlaylists };
  },
);
const track = computed<Track | null>(() => pageData.value?.track ?? null);
const loading = computed(() => pending.value);
watch(
  pageData,
  (v) => {
    if (!v) return;
    cert.value = v.cert;
    similar.value = v.similar;
    sameCreator.value = v.sameCreator;
    inPlaylists.value = v.inPlaylists;
  },
  { immediate: true },
);
if (pageError.value) {
  const st = (pageError.value as { statusCode?: number }).statusCode ?? 0;
  if (st === 404 || st === 403) {
    throw showError({ statusCode: 404, statusMessage: 'Track not found' });
  }
  error.value = apiError(pageError.value);
}
// WI-S1: legacy uuid / bare short ID / stale slug → canonical 301.
if (track.value?.slug && track.value?.shortId) {
  const canon = `${track.value.slug}-${track.value.shortId}`;
  if (param.value !== canon) {
    await navigateTo(localePath({ name: 'track-detail', params: { id: canon } }), { redirectCode: 301 });
  }
}
// WI-S2 head + JSON-LD + OG image (Thai/CJK titles use the textless fallback).
const pagePath = computed(() =>
  track.value?.slug && track.value?.shortId
    ? canonicalTrackPath(track.value.slug, track.value.shortId)
    : `/tracks/${param.value}`,
);
const creatorPageUrl = computed(() =>
  track.value?.creatorHandle ? canonicalFor(`/creators/${track.value.creatorHandle}`, locale.value) : null,
);
const shareUrl = computed(() => canonicalFor(pagePath.value, locale.value));
if (track.value && !pageError.value) {
  const tr = track.value;
  usePublicSeo({
    title: trackTitle(t, tr.title, tr.creatorName ?? ''),
    description: describeTrack(
      t,
      {
        title: tr.title,
        creatorName: tr.creatorName,
        genre: tr.genre,
        moodTags: tr.moodTags,
        bpm: tr.bpm,
        musicalKey: tr.musicalKey,
        durationMs: tr.durationMs,
      },
      (k) => tagDisplayName(k, t, te),
    ),
    path: pagePath.value,
    locale: locale.value,
    ogType: 'music.song',
    image: null,
    musicDuration: tr.durationMs ? Math.round(tr.durationMs / 1000) : null,
    musicMusician: creatorPageUrl.value,
    oembedUrl: oembedDiscoveryUrl(shareUrl.value),
  });
  useSchemaOrg([
    musicRecordingLd({
      track: tr,
      pageUrl: shareUrl.value,
      creatorUrl: creatorPageUrl.value,
      certNumber: cert.value?.certificate?.certNumber ?? null,
      datePublished: tr.createdAt.slice(0, 10),
    }),
    breadcrumbLd([
      { name: 'AVIBE', url: canonicalFor('/', locale.value) },
      { name: tr.title, url: shareUrl.value },
    ]),
  ]);
  defineOgImageComponent('AvibeOg', {
    title: tr.title,
    creator: tr.creatorName ?? '',
    verified: cert.value !== null,
    plain: needsCjkThaiFont(`${tr.title} ${tr.creatorName ?? ''}`),
  });
}

const canIssueCert = computed(
  () => auth.isAdmin && track.value?.status === 'READY' && cert.value === null,
);

const liked = ref(false);

const reportModalOpen = ref(false);
const reportTarget = ref<Track | null>(null);
const playlistModalOpen = ref(false);
const playlistTarget = ref<Track | null>(null);
const creatingPlaylist = ref(false);
const blockConfirmOpen = ref(false);
const blocking = ref(false);

const moreItems = computed<MenuItem[]>(() => [
  { label: t('track.enqueue') },
  { label: t('track.downloadPaid') },
  { label: t('track.reportAction'), danger: true },
]);

function onMoreSelect(index: number): void {
  if (index === 0) enqueue();
  else if (index === 1) void downloadTrack();
  else openReportSelf();
}

const aiCreditText = computed(() => {
  const parts = track.value?.aiCreditParts ?? [];
  if (parts.length === 0) return '';
  const named = parts.map((p) => {
    if (p === 'LYRICS') return t('track.aiCreditLYRICS');
    if (p === 'VOCALS') return t('track.aiCreditVOCALS');
    if (p === 'INSTRUMENTAL') return t('track.aiCreditINSTRUMENTAL');
    return p;
  });
  const scope = parts.length >= 3 ? t('track.aiCreditAll') : t('track.aiCreditPart');
  return `${scope} (${named.join('·')})`;
});

function toPeaks(raw: unknown): number[] {
  if (Array.isArray(raw) && raw.length > 0 && raw.every((n) => typeof n === 'number')) {
    return raw as number[];
  }
  if (typeof raw === 'string' && raw.length > 0) {
    try {
      return toPeaks(JSON.parse(raw) as unknown);
    } catch {
      return [];
    }
  }
  return [];
}

const WAVEFORM_BARS = 48;
function downsample(peaks: number[], n: number): number[] {
  if (peaks.length <= n) return peaks;
  const out: number[] = [];
  for (let i = 0; i < n; i++) {
    const from = Math.floor((i * peaks.length) / n);
    const to = Math.max(from + 1, Math.floor(((i + 1) * peaks.length) / n));
    out.push(Math.max(...peaks.slice(from, to)));
  }
  return out;
}
const waveformPeaks = computed<number[]>(() => downsample(toPeaks(track.value?.waveformJson), WAVEFORM_BARS));

function fmtBpm(bpm?: number | null): string {
  if (bpm === null || bpm === undefined) return '';
  return String(Math.round(bpm * 10) / 10);
}

async function copyTrackId(): Promise<void> {
  if (!track.value) return;
  try {
    await navigator.clipboard.writeText(track.value.id);
    toast.success(t('track.toastCopied'));
  } catch {
    toast.error(t('track.toastCopyFailed'));
  }
}

async function shareTrack(): Promise<void> {
  try {
    await navigator.clipboard.writeText(shareUrl.value);
    trackEvent({ name: 'share_clicked', props: { channel: 'copy' } });
    toast.success(t('track.toastLinkCopied'));
  } catch {
    toast.error(t('track.toastCopyFailed'));
  }
}

function fmtDuration(ms?: number | null): string {
  if (!ms) return '0:00';
  const totalSec = Math.round(ms / 1000);
  return `${Math.floor(totalSec / 60)}:${String(totalSec % 60).padStart(2, '0')}`;
}

async function loadTrack(): Promise<void> {
  error.value = null;
  await refresh();
  if (pageError.value) {
    error.value = apiError(pageError.value);
  }
}

async function issueCert(): Promise<void> {
  if (!track.value) return;
  issuingCert.value = true;
  try {
    await adminApi.issueTrackCertificate(track.value.id);
    cert.value = await trackApi.getCertificate(track.value.id).catch(() => null);
    toast.success(t('track.certIssued'));
  } catch (err: unknown) {
    toast.error(apiError(err));
  } finally {
    issuingCert.value = false;
  }
}

function play(): void {
  if (!track.value) return;
  player.setQueue([track.value], track.value.id);
}

async function toggleLike(): Promise<void> {
  if (!track.value || !requireLogin()) return;
  try {
    if (liked.value) {
      await engagementApi.unlike(track.value.id);
      liked.value = false;
      toast.info(t('library.toastUnliked'));
    } else {
      await engagementApi.like(track.value.id);
      liked.value = true;
      trackEvent({ name: 'track_liked' });
      toast.success(t('library.toastLiked'));
    }
  } catch (err: unknown) {
    toast.error(apiError(err));
  }
}

function playOther(tr: Track): void {
  player.setQueue([tr], tr.id);
}

function enqueue(): void {
  if (!track.value) return;
  player.enqueue(track.value);
  toast.info(t('track.toastQueued'));
}

function enqueueOther(tr: Track): void {
  player.enqueue(tr);
  toast.info(t('track.toastQueued'));
}

async function downloadTrack(): Promise<void> {
  if (!track.value) return;
  downloading.value = true;
  try {
    const { blob, filename } = await trackApi.download(track.value.id);
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    a.remove();
    URL.revokeObjectURL(url);
    toast.success(t('track.toastDlStarted'));
  } catch (err: unknown) {
    const code = errCodeOf(err);
    if (code === 'ERR_DOWNLOAD_REQUIRES_PASS') {
      toast.error(t('track.toastNeedPass'));
    } else if (code === 'ERR_DOWNLOAD_QUOTA') {
      toast.error(t('track.toastQuota'));
    } else {
      toast.error(apiError(err));
    }
  } finally {
    downloading.value = false;
  }
}

function requireLogin(): boolean {
  if (auth.isAuthenticated) return true;
  toast.info(t('track.toastNeedLogin'));
  void router.push(localePath({ name: 'login', query: { redirect: route.fullPath } }));
  return false;
}

function openAddToPlaylist(tr?: Track): void {
  if (!requireLogin()) return;
  playlistTarget.value = tr ?? track.value;
  playlistModalOpen.value = true;
}

async function addToPlaylist(playlistId: string): Promise<void> {
  if (!playlistTarget.value) return;
  try {
    await playlistApi.addTrack(playlistId, playlistTarget.value.id);
    toast.success(t('track.toastAdded'));
    playlistTarget.value = null;
    playlistModalOpen.value = false;
  } catch (err: unknown) {
    toast.error(apiError(err));
  }
}

async function createPlaylist(title: string): Promise<void> {
  creatingPlaylist.value = true;
  try {
    const created = await playlistApi.create(title, false);
    playlists.playlists.unshift(created);
    if (playlistTarget.value) {
      await addToPlaylist(created.id);
    }
  } catch (err: unknown) {
    toast.error(apiError(err));
  } finally {
    creatingPlaylist.value = false;
  }
}

function openReport(tr?: Track): void {
  if (!requireLogin()) return;
  reportTarget.value = tr ?? track.value;
  reportModalOpen.value = true;
}

function openReportSelf(): void {
  openReport();
}

function onReported(): void {
  reportTarget.value = null;
  reportModalOpen.value = false;
  toast.success(t('track.toastReported'));
}

function openBlock(): void {
  if (!requireLogin()) return;
  blockConfirmOpen.value = true;
}

async function confirmBlockPartner(): Promise<void> {
  if (!track.value) return;
  blocking.value = true;
  try {
    await userBlocksApi.blockArtist(track.value.partnerId);
    player.clearQueue();
    blockConfirmOpen.value = false;
    toast.success(t('track.toastReported'));
  } catch (err: unknown) {
    toast.error(apiError(err));
  } finally {
    blocking.value = false;
  }
}

onMounted(() => {
  // Client-only personalization (SSR HTML stays login-agnostic).
  if (auth.isAuthenticated) {
    void playlists.fetchMine();
    if (track.value) {
      void engagementApi.likeState(track.value.id).then(
        (s) => {
          liked.value = s.liked;
        },
        () => false,
      );
    }
  }
});
</script>
