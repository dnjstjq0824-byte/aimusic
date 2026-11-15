<template>
  <div class="space-y-12">
    <!-- Hero (guests) / Continue (signed in). Branch flips after mount so
         SSR HTML (guest) always matches first client render. -->
    <section v-if="!isAuthed" class="relative mx-auto max-w-3xl px-4 pt-4 text-center sm:pt-8">
      <div
        class="pointer-events-none absolute left-1/2 top-0 h-72 w-full max-w-[36rem] -translate-x-1/2 rounded-full bg-brand-500/10 blur-3xl"
        aria-hidden="true"
      />
      <div class="relative space-y-5">
        <UiBadge variant="brand" size="sm" dot>
          {{ t('home.heroBadge') }}
        </UiBadge>

        <h1 class="font-display text-4xl font-extrabold leading-tight tracking-tight text-white sm:text-6xl">
          {{ t('home.heroTitle') }}
        </h1>
        <p class="mx-auto max-w-xl text-sm leading-relaxed text-text-secondary sm:text-base">
          {{ t('home.heroSub') }}
        </p>

        <form
          class="prompt-box mx-auto max-w-2xl rounded-2xl p-2 pl-5 text-left transition-all"
          role="search"
          :aria-label="t('home.heroSearchGo')"
          @submit.prevent="submitHero"
        >
          <label for="hero-q" class="sr-only">{{ t('home.heroSearchPh') }}</label>
          <input
            id="hero-q"
            v-model="heroQuery"
            type="search"
            autocomplete="off"
            :placeholder="t('home.heroSearchPh')"
            class="w-full bg-transparent py-3 text-sm text-white placeholder:text-text-muted focus:outline-none sm:text-base"
          />
          <div class="flex items-center justify-between gap-2 pb-1 pl-0 pr-1">
            <span
              class="inline-flex cursor-not-allowed items-center gap-1 rounded-full border border-surface-border px-2.5 py-1 text-xs font-medium text-text-muted"
              :title="t('home.heroCreateNote')"
            >
              {{ t('home.heroCreateMode') }}
            </span>
            <button
              type="submit"
              class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-brand-500 text-on-brand shadow-glow-brand transition-all hover:bg-brand-400 active:scale-95"
              :aria-label="t('home.heroSearchGo')"
            >
              <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                <path d="M5 12h14M13 6l6 6-6 6" />
              </svg>
            </button>
          </div>
        </form>

        <div class="flex flex-col items-center justify-center gap-2 sm:flex-row">
          <UiButton variant="brand" size="md" @click="scrollToListen">
            {{ t('home.heroListen') }}
          </UiButton>
          <RouterLink :to="localePath('/business')">
            <UiButton variant="secondary" size="md">
              {{ t('home.heroBiz') }}
            </UiButton>
          </RouterLink>
        </div>
      </div>
    </section>

    <section v-else class="space-y-6">
      <UiCard v-if="continueTrack" class="flex flex-col gap-4 sm:flex-row sm:items-center">
        <UiCoverArt :seed="continueTrack.id" :title="continueTrack.title" size="md" :verified="continueTrack.status === 'READY'" />
        <div class="min-w-0 flex-1">
          <p class="text-caption font-bold uppercase tracking-wider text-text-muted">{{ t('home.continueTitle') }}</p>
          <p class="mt-1 truncate text-lg font-black text-white">{{ continueTrack.title }}</p>
          <p v-if="continueTrack.creatorName" class="truncate text-xs text-text-secondary">{{ continueTrack.creatorName }}</p>
        </div>
        <UiButton variant="brand" size="md" class="shrink-0" @click="resumeContinue">
          {{ t('home.continueResume') }}
        </UiButton>
      </UiCard>

      <section v-if="recent.length > 0" :aria-label="t('home.recentTitle')">
        <h2 class="font-display mb-3 text-xl font-bold tracking-tight text-white">{{ t('home.recentTitle') }}</h2>
        <div class="scrollbar-none -mx-4 flex snap-x gap-3 overflow-x-auto px-4 pb-2 sm:mx-0 sm:px-0">
          <button
            v-for="tr in recent"
            :key="tr.id"
            type="button"
            class="group w-40 shrink-0 snap-start text-left"
            @click="playOne(tr)"
          >
            <UiCoverArt :seed="tr.id" :title="tr.title" size="md" :verified="tr.status === 'READY'" />
            <span class="mt-1.5 block truncate text-xs font-bold text-white group-hover:text-brand-400">{{ tr.title }}</span>
          </button>
        </div>
      </section>

      <section v-if="playlists.playlists.length > 0" :aria-label="t('home.myPlTitle')">
        <h2 class="font-display mb-3 text-xl font-bold tracking-tight text-white">{{ t('home.myPlTitle') }}</h2>
        <div class="scrollbar-none -mx-4 flex snap-x gap-3 overflow-x-auto px-4 pb-2 sm:mx-0 sm:px-0">
          <RouterLink
            v-for="pl in playlists.playlists.slice(0, 8)"
            :key="pl.id"
            :to="localePath({ name: 'playlist-detail', params: { id: pl.id } })"
            class="group w-40 shrink-0 snap-start"
          >
            <CoverMosaic :seeds="[pl.id]" :title="pl.title" size="md" />
            <span class="mt-1.5 block truncate text-xs font-bold text-white group-hover:text-brand-400">{{ pl.title }}</span>
          </RouterLink>
        </div>
      </section>
    </section>

    <!-- Cert strip -->
    <section class="flex flex-col items-center justify-between gap-3 rounded-2xl border border-surface-border bg-surface-1 px-5 py-4 sm:flex-row" :aria-label="t('home.stripHow')">
      <div class="flex flex-wrap items-center justify-center gap-x-5 gap-y-1 text-xs font-semibold text-text-primary">
        <span class="inline-flex items-center gap-1.5">
          <VerifiedSeal :size="16" :label="t('player.verified')" :full-label="t('player.verified')" :show-label="false" />
          {{ t('home.strip1') }}
        </span>
        <span>{{ t('home.strip2') }}</span>
        <span>{{ t('home.strip3') }}</span>
      </div>
      <RouterLink :to="localePath('/trust')" class="shrink-0 text-xs font-semibold text-brand-400 hover:underline">
        {{ t('home.stripHow') }}
      </RouterLink>
    </section>

    <!-- Listen now -->
    <section id="listen-now" aria-labelledby="listen-now-h" class="scroll-mt-24">
      <div class="mb-4 flex items-end justify-between px-1">
        <h2 id="listen-now-h" class="font-display text-xl font-bold tracking-tight text-white">{{ t('home.listenNow') }}</h2>
        <RouterLink :to="localePath('/explore')" class="text-xs font-semibold text-brand-400 hover:underline">
          {{ t('home.listenMore') }} →
        </RouterLink>
      </div>
      <div class="scrollbar-none -mx-4 flex snap-x snap-mandatory gap-3 overflow-x-auto scroll-px-4 px-4 pb-2 sm:mx-0 sm:scroll-px-0 sm:px-0">
        <button
          v-for="tr in fresh"
          :key="tr.id"
          type="button"
          class="group w-40 shrink-0 snap-start text-left"
          @click="playOne(tr)"
        >
          <UiCoverArt :seed="tr.id" :title="tr.title" size="md" :verified="tr.status === 'READY'" />
          <span class="mt-1.5 block truncate text-xs font-bold text-white group-hover:text-brand-400">{{ tr.title }}</span>
          <span v-if="tr.creatorName" class="block truncate text-caption text-text-muted">{{ tr.creatorName }}</span>
        </button>
      </div>
    </section>

    <!-- Mood chips -->
    <section :aria-label="t('home.moodsLabel')">
      <div class="flex gap-2 overflow-x-auto pb-2 scrollbar-none">
        <RouterLink
          v-for="mood in MOODS"
          :key="mood"
          :to="localePath({ path: '/explore', query: { mood } })"
          class="shrink-0 rounded-full border border-surface-border bg-surface-1 px-4 py-1.5 text-xs font-semibold text-text-primary transition-colors hover:border-brand-500/50 hover:text-brand-300"
        >
          {{ tagName(mood) }}
        </RouterLink>
      </div>
    </section>

    <!-- Top chart -->
    <section aria-labelledby="home-chart-h">
      <div class="mb-4 flex items-end justify-between px-1">
        <h2 id="home-chart-h" class="font-display text-xl font-bold tracking-tight text-white">{{ t('home.chartTitle') }}</h2>
        <RouterLink :to="localePath('/charts')" class="text-xs font-semibold text-brand-400 hover:underline">
          {{ t('home.chartMore') }} →
        </RouterLink>
      </div>
      <ol class="grid gap-x-6 gap-y-1 md:grid-cols-2">
        <li v-for="(tr, i) in chartTop" :key="tr.id">
          <TrackRow
            :track="tr"
            :rank="i + 1"
            @play="playOne"
            @enqueue="enqueue"
            @add-to-playlist="openAddToPlaylist"
            @report="openReport"
          />
        </li>
      </ol>
    </section>

    <!-- Recommended playlists -->
    <section aria-labelledby="home-rec-h">
      <div class="mb-4 flex items-end justify-between px-1">
        <h2 id="home-rec-h" class="font-display text-xl font-bold tracking-tight text-white">{{ t('home.recTitle') }}</h2>
        <RouterLink :to="localePath('/playlists')" class="text-xs font-semibold text-brand-400 hover:underline">
          {{ t('home.recMore') }} →
        </RouterLink>
      </div>
      <p v-if="recPlaylists.length === 0" class="py-6 text-center text-xs text-text-muted">
        {{ t('home.recEmpty') }}
      </p>
      <div v-else class="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <RouterLink
          v-for="pl in recPlaylists"
          :key="pl.id"
          :to="localePath({ name: 'playlist-detail', params: { id: pl.id } })"
          class="group"
        >
          <CoverMosaic :seeds="plSeeds[pl.id] ?? [pl.id]" :title="pl.title" size="md" />
          <span class="mt-1.5 block truncate text-xs font-bold text-white group-hover:text-brand-400">{{ pl.title }}</span>
        </RouterLink>
      </div>
    </section>

    <!-- Customer cards -->
    <section class="grid gap-4 md:grid-cols-3">
      <UiCard
        v-for="card in customerCards"
        :key="card.to"
        class="flex flex-col items-start gap-2 p-6 shadow-2xl"
      >
        <h2 class="text-lg font-bold text-white">{{ t(card.titleKey) }}</h2>
        <p class="min-h-10 text-xs leading-relaxed text-text-secondary">{{ t(card.descKey) }}</p>
        <RouterLink :to="localePath(card.to)" class="mt-2 text-xs font-bold text-brand-400 hover:underline">
          {{ t(card.ctaKey) }} →
        </RouterLink>
      </UiCard>
    </section>

    <DisputeReportModal
      v-if="reportTarget"
      :track-id="reportTarget.id"
      :track-title="reportTarget.title"
      @close="reportTarget = null"
      @reported="onReported"
    />

    <UiPlaylistPicker
      :model-value="Boolean(playlistTargetTrack)"
      :track-title="playlistTargetTrack?.title ?? ''"
      :playlists="playlists.playlists"
      :create-loading="creatingPlaylist"
      @update:model-value="playlistTargetTrack = $event ? playlistTargetTrack : null"
      @select="addTrackToPlaylist"
      @create="createNewPlaylist"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { trackApi, playlistApi, playbackApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { useAuthStore } from '@/stores/authStore';
import { usePlayerStore } from '@/stores/playerStore';
import { usePlaylistStore } from '@/stores/playlistStore';
import { useToast } from '@/composables/useToast';
import { useApiError } from '@/composables/useApiError';
import { tagDisplayName } from '@/utils/tags';
import type { Track, Playlist } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiCoverArt from '@/components/ui/UiCoverArt.vue';
import UiPlaylistPicker from '@/components/ui/UiPlaylistPicker.vue';
import CoverMosaic from '@/components/ui/CoverMosaic.vue';
import VerifiedSeal from '@/components/ui/VerifiedSeal.vue';
import TrackRow from '@/components/TrackRow.vue';
import DisputeReportModal from '@/components/legal/DisputeReportModal.vue';

const { t, te } = useI18n();
const localePath = useLocalePath();
const player = usePlayerStore();
const playlists = usePlaylistStore();
const toast = useToast();
const apiError = useApiError();
const auth = useAuthStore();
const router = useRouter();

const MOODS = ['focus', 'chill', 'cafe', 'workout', 'drive', 'sleep', 'cinematic'];
const tagName = (key: string) => tagDisplayName(key, t, te);

const heroQuery = ref('');
// SSR-safe auth branch: false until mounted (middleware hydrates auth first).
const isAuthed = ref(false);
const fresh = ref<Track[]>([]);
const chartTop = ref<Track[]>([]);
const recPlaylists = ref<Playlist[]>([]);
const plSeeds = ref<Record<string, string[]>>({});
const recent = ref<Track[]>([]);

const reportTarget = ref<Track | null>(null);
const playlistTargetTrack = ref<Track | null>(null);
const creatingPlaylist = ref(false);

const continueTrack = computed(() => recent.value[0] ?? null);

// NOTE: card keys resolve via label objects (invisible to check-i18n usage
// scan — they are used, do not delete).
const customerCards = [
  { to: '/signup', titleKey: 'home.custListenerT', descKey: 'home.custListenerD', ctaKey: 'home.custListenerC' },
  { to: '/for-creators', titleKey: 'home.custCreatorT', descKey: 'home.custCreatorD', ctaKey: 'home.custCreatorC' },
  { to: '/business', titleKey: 'home.custBizT', descKey: 'home.custBizD', ctaKey: 'home.custBizC' },
];

function submitHero(): void {
  const q = heroQuery.value.trim();
  void router.push(q ? localePath({ path: '/explore', query: { q } }) : localePath('/explore'));
}

function scrollToListen(): void {
  document.getElementById('listen-now')?.scrollIntoView({ behavior: 'smooth', block: 'start' });
}

function playOne(tr: Track): void {
  player.setQueue([tr], tr.id);
}

function resumeContinue(): void {
  if (continueTrack.value) player.setQueue([continueTrack.value], continueTrack.value.id);
}

function enqueue(tr: Track): void {
  player.enqueue(tr);
  toast.info(t('track.toastQueued'));
}

function requireLogin(): boolean {
  if (auth.isAuthenticated) return true;
  toast.info(t('track.toastNeedLogin'));
  void router.push(localePath({ name: 'login', query: { redirect: '/' } }));
  return false;
}

function openAddToPlaylist(tr: Track): void {
  if (!requireLogin()) return;
  playlistTargetTrack.value = tr;
}

async function addTrackToPlaylist(playlistId: string): Promise<void> {
  if (!playlistTargetTrack.value) return;
  try {
    await playlistApi.addTrack(playlistId, playlistTargetTrack.value.id);
    toast.success(t('track.toastAdded'));
    playlistTargetTrack.value = null;
  } catch (err: unknown) {
    toast.error(apiError(err));
  }
}

async function createNewPlaylist(title: string): Promise<void> {
  creatingPlaylist.value = true;
  try {
    const created = await playlistApi.create(title, false);
    playlists.playlists.unshift(created);
    if (playlistTargetTrack.value) {
      await addTrackToPlaylist(created.id);
    }
  } catch (err: unknown) {
    toast.error(apiError(err));
  } finally {
    creatingPlaylist.value = false;
  }
}

function openReport(tr: Track): void {
  if (!requireLogin()) return;
  reportTarget.value = tr;
}

function onReported(): void {
  reportTarget.value = null;
  toast.success(t('track.toastReported'));
}

onMounted(() => {
  isAuthed.value = auth.isAuthenticated;
  void trackApi.list(12, 0, { sort: 'recent' }).then(
    (p) => { fresh.value = p.items ?? []; },
    (err: unknown) => toast.error(errMessageOf(err)),
  );
  void trackApi.charts('all', 10).then(
    (items) => { chartTop.value = items ?? []; },
    () => { chartTop.value = []; },
  );
  void playlistApi.listPublic(4).then(
    (items) => {
      recPlaylists.value = items ?? [];
      for (const pl of recPlaylists.value) {
        void playlistApi.listTracks(pl.id).then(
          (trs) => { plSeeds.value[pl.id] = trs.slice(0, 4).map((x) => x.trackId); },
          () => { plSeeds.value[pl.id] = [pl.id]; },
        );
      }
    },
    () => { recPlaylists.value = []; },
  );
  if (auth.isAuthenticated) {
    void playbackApi.recent(10).then(
      (items) => { recent.value = items ?? []; },
      () => { recent.value = []; },
    );
    void playlists.fetchMine();
  }
});
</script>
