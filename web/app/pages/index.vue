<template>
  <div class="space-y-24 pb-8 sm:space-y-32">
    <!-- Hero (v3 flow): light-weight display type over a soft violet/blue
         bloom, a frosted prompt box and persona chips. Same for guests and
         members so SSR HTML always matches the first client render. -->
    <section class="relative pt-8 text-center sm:pt-20">
      <div
        class="hero-glow pointer-events-none absolute left-1/2 top-[-8rem] h-[40rem] w-screen -translate-x-1/2"
        aria-hidden="true"
      />
      <div class="relative mx-auto max-w-3xl">
        <h1 class="type-display text-5xl text-white sm:text-7xl">
          {{ t('home.heroTitle') }}
        </h1>
        <p class="mx-auto mt-5 max-w-xl text-base leading-relaxed text-text-secondary [text-wrap:pretty] sm:text-lg">
          {{ t('home.heroSub') }}
        </p>

        <form
          class="prompt-box mx-auto mt-10 rounded-2xl text-left transition-colors"
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
            class="w-full bg-transparent px-5 pb-3 pt-5 text-base text-white placeholder:text-text-muted focus:outline-none"
          >
          <div class="flex items-center justify-between gap-2 px-3 pb-3">
            <div class="flex items-center gap-1">
              <RouterLink
                :to="localePath('/explore')"
                class="flex h-10 w-10 items-center justify-center rounded-full text-text-secondary transition-colors hover:bg-white/10 hover:text-white"
                :aria-label="t('home.heroFilters')"
                :title="t('home.heroFilters')"
              >
                <svg class="h-[18px] w-[18px]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" aria-hidden="true">
                  <path d="M4 7h10M18 7h2M4 17h4M12 17h8" />
                  <circle cx="16" cy="7" r="2" />
                  <circle cx="10" cy="17" r="2" />
                </svg>
              </RouterLink>
              <span
                class="inline-flex cursor-not-allowed items-center gap-1.5 rounded-full px-3 py-2 text-xs text-text-muted"
                :title="t('home.heroCreateNote')"
              >
                <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" aria-hidden="true">
                  <path d="M4 10v4M8 7v10M12 4v16M16 8v8M20 11v2" />
                </svg>
                {{ t('home.heroCreateMode') }}
              </span>
            </div>
            <button
              type="submit"
              class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-brand-500 text-on-brand transition-colors hover:bg-brand-400 active:scale-95"
              :aria-label="t('home.heroSearchGo')"
            >
              <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                <path d="M12 19V5M6 11l6-6 6 6" />
              </svg>
            </button>
          </div>
        </form>

        <nav class="mt-6 flex flex-wrap justify-center gap-2" :aria-label="t('home.personasLabel')">
          <RouterLink
            v-for="p in personas"
            :key="p.key"
            :to="localePath(p.to)"
            class="inline-flex min-h-[40px] items-center gap-2 rounded-full border border-white/15 bg-white/[0.03] px-4 text-sm text-white transition-colors hover:border-white/30 hover:bg-white/[0.08]"
          >
            <svg class="h-4 w-4 text-brand-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <path :d="p.icon" />
            </svg>
            {{ t(p.key) }}
          </RouterLink>
        </nav>
      </div>
    </section>

    <!-- Members: pick up where you left off (client-only branch). -->
    <section v-if="isAuthed && (continueTrack || recent.length > 0 || playlists.playlists.length > 0)" class="space-y-10">
      <div v-if="continueTrack" class="flex flex-col gap-4 rounded-3xl border border-white/10 bg-white/[0.03] p-4 sm:flex-row sm:items-center sm:p-5">
        <UiCoverArt :seed="continueTrack.id" :title="continueTrack.title" size="lg" rounded="xl" :verified="continueTrack.status === 'READY'" />
        <div class="min-w-0 flex-1">
          <p class="text-xs text-text-muted">{{ t('home.continueTitle') }}</p>
          <p class="mt-1 truncate text-lg font-medium text-white">{{ continueTrack.title }}</p>
          <p v-if="continueTrack.creatorName" class="truncate text-sm text-text-secondary">{{ continueTrack.creatorName }}</p>
        </div>
        <button
          type="button"
          class="shrink-0 rounded-full bg-brand-500 px-5 py-2.5 text-sm font-semibold text-on-brand transition-colors hover:bg-brand-400"
          @click="resumeContinue"
        >
          {{ t('home.continueResume') }}
        </button>
      </div>

      <section v-if="recent.length > 0" :aria-label="t('home.recentTitle')">
        <h2 class="mb-4 text-xl font-medium text-white">{{ t('home.recentTitle') }}</h2>
        <div class="scrollbar-none -mx-4 flex snap-x gap-4 overflow-x-auto px-4 pb-2 sm:mx-0 sm:px-0">
          <button
            v-for="tr in recent"
            :key="tr.id"
            type="button"
            class="group w-40 shrink-0 snap-start text-left"
            @click="playOne(tr)"
          >
            <UiCoverArt :seed="tr.id" :title="tr.title" size="fill" rounded="xl" :verified="tr.status === 'READY'" />
            <span class="mt-2 block truncate text-sm text-white group-hover:text-brand-300">{{ tr.title }}</span>
          </button>
        </div>
      </section>

      <section v-if="playlists.playlists.length > 0" :aria-label="t('home.myPlTitle')">
        <h2 class="mb-4 text-xl font-medium text-white">{{ t('home.myPlTitle') }}</h2>
        <div class="scrollbar-none -mx-4 flex snap-x gap-4 overflow-x-auto px-4 pb-2 sm:mx-0 sm:px-0">
          <RouterLink
            v-for="pl in playlists.playlists.slice(0, 8)"
            :key="pl.id"
            :to="localePath({ name: 'playlist-detail', params: { id: detailParam(pl) } })"
            class="group w-40 shrink-0 snap-start"
          >
            <UiCoverArt :seed="pl.id" :title="pl.title" size="fill" rounded="xl" />
            <span class="mt-2 block truncate text-sm text-white group-hover:text-brand-300">{{ pl.title }}</span>
          </RouterLink>
        </div>
      </section>
    </section>

    <!-- Get started: photo-led use-case cards (N1: hydrate on visible). -->
    <LazyLandingStarterRail class="!mt-16 sm:!mt-20" hydrate-on-visible />

    <!-- Feature 1: Listen (live tracks). -->
    <section id="listen-now" class="grid scroll-mt-24 items-center gap-10 lg:grid-cols-[minmax(0,5fr)_minmax(0,7fr)] lg:gap-16" aria-labelledby="feat-listen-h">
      <div>
        <p class="text-sm font-medium text-brand-400">{{ t('home.featListenEyebrow') }}</p>
        <h2 id="feat-listen-h" class="type-display mt-3 text-3xl text-white sm:text-5xl">{{ t('home.featListenTitle') }}</h2>
        <p class="mt-5 max-w-md text-base leading-relaxed text-text-secondary">{{ t('home.featListenBody') }}</p>
        <RouterLink :to="localePath('/explore')" class="mt-8 inline-flex min-h-[44px] items-center gap-2 rounded-full border border-white/15 px-5 text-sm font-medium text-white transition-colors hover:bg-white/[0.08]">
          {{ t('home.listenMore') }}
          <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 12h14M13 6l6 6-6 6" /></svg>
        </RouterLink>
      </div>
      <div class="grid grid-cols-2 gap-3 sm:grid-cols-3 sm:gap-4">
        <template v-if="fresh.length > 0">
          <button
            v-for="(tr, i) in fresh.slice(0, 3)"
            :key="tr.id"
            type="button"
            class="group text-left"
            :class="i === 2 ? 'hidden sm:block' : ''"
            @click="playOne(tr)"
          >
            <div class="relative">
              <UiCoverArt :seed="tr.id" :title="tr.title" size="fill" rounded="xl" :verified="tr.status === 'READY'" />
              <span class="absolute left-3 top-3 flex h-10 w-10 items-center justify-center rounded-full bg-black/45 text-white opacity-90 backdrop-blur transition-opacity group-hover:opacity-100" aria-hidden="true">
                <svg class="ml-0.5 h-4 w-4" viewBox="0 0 24 24" fill="currentColor"><path d="M8 5v14l11-7z" /></svg>
              </span>
            </div>
            <span class="mt-3 block truncate text-sm font-medium text-white group-hover:text-brand-300">{{ tr.title }}</span>
            <span class="block truncate text-xs text-text-muted">{{ tr.genre || tr.creatorName }}</span>
          </button>
        </template>
        <template v-else>
          <div v-for="n in 3" :key="n" class="aspect-square animate-pulse rounded-xl bg-surface-2" :class="n === 3 ? 'hidden sm:block' : ''" aria-hidden="true" />
        </template>
      </div>
    </section>

    <!-- Feature 2: Verify (certificate card). -->
    <section class="grid items-center gap-10 lg:grid-cols-[minmax(0,7fr)_minmax(0,5fr)] lg:gap-16" aria-labelledby="feat-verify-h">
      <div class="order-2 lg:order-1">
        <div class="relative mx-auto max-w-lg overflow-hidden rounded-3xl border border-white/10 bg-gradient-to-br from-white/[0.08] to-white/[0.02] p-6 sm:p-8">
          <div class="hero-glow pointer-events-none absolute -right-24 -top-24 h-72 w-72 opacity-70" aria-hidden="true" />
          <div class="relative">
            <div class="flex items-center gap-2 text-sm text-verified">
              <VerifiedSeal :size="20" :label="t('home.featVerifyStatus')" :full-label="t('home.featVerifyStatus')" :show-label="false" />
              {{ t('home.featVerifyStatus') }}
            </div>
            <p class="mt-6 truncate text-2xl font-medium text-white">{{ certTrack?.title ?? 'AVIBE' }}</p>
            <p class="mt-1 truncate text-sm text-text-secondary">{{ certTrack?.creatorName ?? '' }}&nbsp;</p>
            <dl class="mt-8 grid grid-cols-2 gap-4 border-t border-white/10 pt-5 text-sm">
              <div>
                <dt class="text-xs text-text-muted">{{ t('home.featVerifyCertLabel') }}</dt>
                <dd class="mt-1 font-mono text-white">AVB-••••-••••</dd>
              </div>
              <div>
                <dt class="text-xs text-text-muted">{{ t('home.featVerifyIssued') }}</dt>
                <dd class="mt-1 font-mono text-white">{{ certIssued }}</dd>
              </div>
            </dl>
            <ul class="mt-6 space-y-2.5 text-sm text-text-primary">
              <li v-for="k in ['home.strip1', 'home.strip2', 'home.strip3']" :key="k" class="flex items-center gap-2.5">
                <svg class="h-4 w-4 shrink-0 text-verified" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M20 6 9 17l-5-5" /></svg>
                {{ t(k) }}
              </li>
            </ul>
          </div>
        </div>
      </div>
      <div class="order-1 lg:order-2">
        <p class="text-sm font-medium text-brand-400">{{ t('home.featVerifyEyebrow') }}</p>
        <h2 id="feat-verify-h" class="type-display mt-3 text-3xl text-white sm:text-5xl">{{ t('home.featVerifyTitle') }}</h2>
        <p class="mt-5 max-w-md text-base leading-relaxed text-text-secondary">{{ t('home.featVerifyBody') }}</p>
        <RouterLink :to="localePath('/trust')" class="mt-8 inline-flex min-h-[44px] items-center gap-2 rounded-full border border-white/15 px-5 text-sm font-medium text-white transition-colors hover:bg-white/[0.08]">
          {{ t('home.stripHow') }}
          <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 12h14M13 6l6 6-6 6" /></svg>
        </RouterLink>
      </div>
    </section>

    <!-- Feature 3: Business (photo). -->
    <section class="grid items-center gap-10 lg:grid-cols-[minmax(0,5fr)_minmax(0,7fr)] lg:gap-16" aria-labelledby="feat-biz-h">
      <div>
        <p class="text-sm font-medium text-brand-400">{{ t('home.featBizEyebrow') }}</p>
        <h2 id="feat-biz-h" class="type-display mt-3 text-3xl text-white sm:text-5xl">{{ t('home.featBizTitle') }}</h2>
        <p class="mt-5 max-w-md text-base leading-relaxed text-text-secondary">{{ t('home.featBizBody') }}</p>
        <RouterLink :to="localePath('/business')" class="mt-8 inline-flex min-h-[44px] items-center gap-2 rounded-full border border-white/15 px-5 text-sm font-medium text-white transition-colors hover:bg-white/[0.08]">
          {{ t('home.heroBiz') }}
          <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 12h14M13 6l6 6-6 6" /></svg>
        </RouterLink>
      </div>
      <div class="relative overflow-hidden rounded-3xl bg-surface-2">
        <img src="/images/stock/feature-listen.webp" alt="" loading="lazy" decoding="async" class="aspect-[4/3] w-full object-cover">
        <div class="absolute inset-0 bg-gradient-to-t from-black/70 via-black/10 to-transparent" aria-hidden="true" />
        <div class="absolute inset-x-4 bottom-4 flex flex-wrap gap-2 sm:inset-x-6 sm:bottom-6">
          <span
            v-for="k in ['home.personaStore', 'home.personaVideo', 'home.startFocus']"
            :key="k"
            class="rounded-full border border-white/20 bg-black/35 px-3.5 py-1.5 text-xs text-white backdrop-blur-md sm:text-sm"
          >
            {{ t(k) }}
          </span>
        </div>
      </div>
    </section>

    <!-- Feature 4: Creators (capability list). -->
    <section class="grid items-center gap-10 lg:grid-cols-[minmax(0,7fr)_minmax(0,5fr)] lg:gap-16" aria-labelledby="feat-creator-h">
      <div class="order-2 lg:order-1">
        <ul class="mx-auto max-w-lg divide-y divide-white/[0.08] overflow-hidden rounded-3xl border border-white/10 bg-white/[0.03]">
          <li v-for="item in creatorItems" :key="item.key" class="flex items-center gap-4 px-6 py-5">
            <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-white/[0.06] text-brand-400" aria-hidden="true">
              <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path :d="item.icon" /></svg>
            </span>
            <span class="text-base text-white">{{ t(item.key) }}</span>
          </li>
          <li class="px-6 py-4 text-sm text-text-muted">{{ t('home.featCreatorMore') }}</li>
        </ul>
      </div>
      <div class="order-1 lg:order-2">
        <p class="text-sm font-medium text-brand-400">{{ t('home.featCreatorEyebrow') }}</p>
        <h2 id="feat-creator-h" class="type-display mt-3 text-3xl text-white sm:text-5xl">{{ t('home.featCreatorTitle') }}</h2>
        <p class="mt-5 max-w-md text-base leading-relaxed text-text-secondary">{{ t('home.featCreatorBody') }}</p>
        <RouterLink :to="localePath('/for-creators')" class="mt-8 inline-flex min-h-[44px] items-center gap-2 rounded-full border border-white/15 px-5 text-sm font-medium text-white transition-colors hover:bg-white/[0.08]">
          {{ t('home.custCreatorC') }}
          <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 12h14M13 6l6 6-6 6" /></svg>
        </RouterLink>
      </div>
    </section>

    <!-- Top chart -->
    <section v-if="chartTop.length > 0" aria-labelledby="home-chart-h">
      <div class="mb-5 flex items-end justify-between">
        <h2 id="home-chart-h" class="text-xl font-medium text-white sm:text-2xl">{{ t('home.chartTitle') }}</h2>
        <RouterLink :to="localePath('/charts')" class="text-sm text-text-secondary hover:text-white">
          {{ t('home.chartMore') }} →
        </RouterLink>
      </div>
      <ol class="grid gap-x-8 gap-y-1 md:grid-cols-2">
        <li v-for="(tr, i) in chartTop" :key="tr.id">
          <LazyTrackRow
            :track="tr"
            :rank="i + 1"
            hydrate-on-visible
            @play="playOne"
            @enqueue="enqueue"
            @add-to-playlist="openAddToPlaylist"
            @report="openReport"
          />
        </li>
      </ol>
    </section>

    <!-- Recommended playlists (only when there are some). -->
    <section v-if="recPlaylists.length > 0" aria-labelledby="home-rec-h">
      <div class="mb-5 flex items-end justify-between">
        <h2 id="home-rec-h" class="text-xl font-medium text-white sm:text-2xl">{{ t('home.recTitle') }}</h2>
        <RouterLink :to="localePath('/playlists')" class="text-sm text-text-secondary hover:text-white">
          {{ t('home.recMore') }} →
        </RouterLink>
      </div>
      <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <RouterLink
          v-for="pl in recPlaylists"
          :key="pl.id"
          :to="localePath({ name: 'playlist-detail', params: { id: detailParam(pl) } })"
          class="group"
        >
          <UiCoverArt :seed="plSeeds[pl.id]?.[0] ?? pl.id" :title="pl.title" size="fill" rounded="xl" />
          <span class="mt-2 block truncate text-sm font-medium text-white group-hover:text-brand-300">{{ pl.title }}</span>
        </RouterLink>
      </div>
    </section>

    <!-- Closing CTA -->
    <section class="relative py-10 text-center sm:py-16">
      <div class="hero-glow pointer-events-none absolute left-1/2 top-1/2 h-[28rem] w-screen -translate-x-1/2 -translate-y-1/2 opacity-80" aria-hidden="true" />
      <div class="relative">
        <h2 class="type-display text-4xl text-white sm:text-6xl">{{ t('home.ctaTitle') }}</h2>
        <RouterLink
          :to="localePath(isAuthed ? '/explore' : '/signup')"
          class="mt-8 inline-flex min-h-[48px] items-center rounded-full bg-brand-500 px-7 text-base font-semibold text-on-brand transition-colors hover:bg-brand-400"
        >
          {{ isAuthed ? t('home.listenMore') : t('home.custListenerC') }}
        </RouterLink>
        <p class="mt-4 text-sm text-text-muted">{{ t('home.ctaNote') }}</p>
      </div>
    </section>

    <DisputeReportModal
      v-if="reportTarget"
      :track-id="reportTarget.id"
      :track-title="reportTarget.title"
      @close="reportTarget = null"
      @reported="onReported"
    />

    <UiPlaylistPickerAsync
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
import { computed, defineAsyncComponent, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { trackApi, playlistApi, playbackApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { useAuthStore } from '@/stores/authStore';
import { usePlayerStore } from '@/stores/playerStore';
import { usePlaylistStore } from '@/stores/playlistStore';
import { useToast } from '@/composables/useToast';
import { useApiError } from '@/composables/useApiError';
import { useFormat } from '@/composables/useFormat';
import type { Track, Playlist } from '@/api/types';
import UiCoverArt from '@/components/ui/UiCoverArt.vue';
import VerifiedSeal from '@/components/ui/VerifiedSeal.vue';
// N1: below-fold + modals load on demand (Lazy* auto-imports).
const DisputeReportModal = defineAsyncComponent(() => import('@/components/legal/DisputeReportModal.vue'));
const UiPlaylistPickerAsync = defineAsyncComponent(() => import('@/components/ui/UiPlaylistPicker.vue'));
import { detailParam } from '@/utils/seo';

const { t } = useI18n();
const localePath = useLocalePath();
const player = usePlayerStore();
const playlists = usePlaylistStore();
const toast = useToast();
const apiError = useApiError();
const auth = useAuthStore();
const router = useRouter();
const { dateISO } = useFormat();

// Persona chips (hero). Icon paths are 24px stroke glyphs.
const personas = [
  { key: 'home.personaListener', to: '/explore', icon: 'M3 14v-2a9 9 0 0 1 18 0v2M3 14h3v6H4a1 1 0 0 1-1-1zM21 14h-3v6h2a1 1 0 0 0 1-1z' },
  { key: 'home.personaVideo', to: { path: '/explore', query: { mood: 'cinematic' } }, icon: 'M3 7h12v10H3zM15 10l6-3v10l-6-3' },
  { key: 'home.personaStore', to: '/business', icon: 'M4 9l1.5-5h13L20 9M4 9v11h16V9M4 9h16M9 20v-6h6v6' },
  { key: 'home.personaArtist', to: '/for-creators', icon: 'M12 3a3 3 0 0 1 3 3v5a3 3 0 0 1-6 0V6a3 3 0 0 1 3-3zM5 11a7 7 0 0 0 14 0M12 18v3' },
];

const creatorItems = [
  { key: 'home.featCreatorItem1', icon: 'M12 3l7 3v6c0 4.5-3 7.5-7 9-4-1.5-7-4.5-7-9V6zM9 12l2 2 4-4' },
  { key: 'home.featCreatorItem2', icon: 'M6 3h9l4 4v14H6zM14 3v5h5M9 13h7M9 17h5' },
  { key: 'home.featCreatorItem3', icon: 'M4 20V10M10 20V4M16 20v-7M22 20H2' },
  { key: 'home.featCreatorItem4', icon: 'M12 9v4M12 17h.01M10.3 3.9 2 18a2 2 0 0 0 1.7 3h16.6a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0z' },
];

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
// Certificate preview shows a real verified track; the number stays masked.
const certTrack = computed(() => fresh.value.find((x) => x.status === 'READY') ?? fresh.value[0] ?? null);
const certIssued = computed(() => {
  const at = certTrack.value?.createdAt;
  return at ? dateISO(at) : '—';
});

function submitHero(): void {
  const q = heroQuery.value.trim();
  void router.push(q ? localePath({ path: '/explore', query: { q } }) : localePath('/explore'));
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
