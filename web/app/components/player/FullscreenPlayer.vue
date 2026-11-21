<template>
  <div class="fixed inset-0 z-50" role="dialog" aria-modal="true" :aria-label="t('player.expand')">
    <div class="absolute inset-0 bg-black/70" @click="$emit('close')" />
    <section class="absolute inset-0 overflow-y-auto bg-surface-0 sm:left-auto sm:right-0 sm:w-[420px] sm:border-l sm:border-surface-border">
      <div class="mx-auto flex min-h-full w-full max-w-md flex-col px-5 py-4 sm:max-w-none">
        <div class="flex items-center justify-between">
          <button
            type="button"
            class="flex h-10 w-10 items-center justify-center rounded-lg text-text-secondary hover:bg-surface-2 hover:text-white"
            :aria-label="t('player.close')"
            @click="$emit('close')"
          >
            <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <path d="m6 9 6 6 6-6" />
            </svg>
          </button>
          <span class="text-caption font-semibold uppercase tracking-wider text-text-muted">{{ t('player.nowPlaying') }}</span>
          <span class="w-10" aria-hidden="true" />
        </div>

        <template v-if="player.currentTrack">
          <div class="mt-2 flex justify-center">
            <UiCoverArt
              :seed="player.currentTrack.id"
              :title="player.currentTrack.title"
              size="xl"
              rounded="xl"
              :verified="player.currentTrack.status === 'READY'"
            />
          </div>

          <div class="mt-4 flex items-start justify-between gap-3">
            <div class="min-w-0">
              <h2 class="truncate text-lg font-semibold text-white">{{ player.currentTrack.title }}</h2>
              <p class="mt-0.5 truncate text-xs text-text-secondary">{{ creatorLabel }}</p>
            </div>
            <VerifiedSeal
              v-if="player.currentTrack.status === 'READY'"
              :size="20"
              :label="t('player.verified')"
              :full-label="t('player.verified')"
              :show-label="false"
              :cert-number="player.currentTrack.certNumber ?? null"
            />
          </div>

          <div class="mt-4">
            <WaveformSeekbar
              :position-ms="player.positionMs"
              :duration-ms="player.durationMs || player.currentTrack.durationMs || 0"
              :seed="player.currentTrack.id"
              @seek="player.seek($event)"
            />
            <div class="mt-1 flex justify-between font-mono text-caption text-text-secondary">
              <span>{{ fmtPos(player.positionMs) }}</span>
              <span>{{ fmtDur(player.durationMs || player.currentTrack.durationMs || 0) }}</span>
            </div>
          </div>

          <div class="mt-2 flex items-center justify-center gap-5">
            <button type="button" class="rounded p-1 transition-colors" :class="player.shuffle ? 'text-brand-400' : 'text-text-secondary hover:text-white'" :aria-pressed="player.shuffle" :aria-label="t('player.shuffle')" @click="player.toggleShuffle()">
              <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                <path d="M16 3h5v5M4 20 21 3M21 16v5h-5M15 15l6 6M4 4l5 5" />
              </svg>
            </button>
            <button type="button" class="rounded p-1 text-text-primary hover:text-white transition-colors" :aria-label="t('player.previous')" @click="player.playPrev()">
              <svg class="h-6 w-6" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                <path d="M6 5h2v14H6zM20 5v14L9 12z" />
              </svg>
            </button>
            <button
              type="button"
              class="flex h-14 w-14 items-center justify-center rounded-full bg-brand-500 text-on-brand shadow-glow-brand hover:bg-brand-400 active:scale-95 transition-all disabled:opacity-50"
              :disabled="player.isLoading || !player.currentTrack"
              :aria-label="player.isPlaying ? t('player.pause') : t('player.play')"
              @click="player.toggle()"
            >
              <svg v-if="player.isLoading" class="h-5 w-5 animate-spin" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z" />
              </svg>
              <svg v-else-if="player.isPlaying" class="h-5 w-5" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                <path d="M7 5h4v14H7zM13 5h4v14h-4z" />
              </svg>
              <svg v-else class="ml-0.5 h-5 w-5" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                <path d="M8 5v14l11-7z" />
              </svg>
            </button>
            <button type="button" class="rounded p-1 text-text-primary hover:text-white transition-colors" :aria-label="t('player.next')" @click="player.playNext(true)">
              <svg class="h-6 w-6" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                <path d="M16 5h2v14h-2zM4 5v14l11-7z" />
              </svg>
            </button>
            <button type="button" class="relative rounded p-1 transition-colors" :class="player.repeat !== 'off' ? 'text-brand-400' : 'text-text-secondary hover:text-white'" :aria-pressed="player.repeat !== 'off'" :aria-label="t('player.repeat', { mode: repeatLabel })" @click="player.cycleRepeat()">
              <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                <path d="m17 2 4 4-4 4" />
                <path d="M3 11v-1a4 4 0 0 1 4-4h14M7 22l-4-4 4-4" />
                <path d="M21 13v1a4 4 0 0 1-4 4H3" />
              </svg>
              <span v-if="player.repeat === 'one'" class="absolute -top-1 -right-1 rounded-full bg-brand-500 px-1 text-caption font-bold text-on-brand">1</span>
            </button>
          </div>

          <div class="mt-5 flex gap-2 border-b border-surface-border" role="tablist">
            <button
              v-for="tab in tabs"
              :key="tab.id"
              type="button"
              role="tab"
              :aria-selected="activeTab === tab.id"
              class="flex-1 border-b-2 pb-2 text-xs font-bold transition-colors"
              :class="activeTab === tab.id ? 'border-brand-500 text-white' : 'border-transparent text-text-muted hover:text-text-secondary'"
              @click="activeTab = tab.id"
            >
              {{ t(tab.labelKey) }}
            </button>
          </div>

          <div class="mt-3 flex-1 pb-6">
            <ol v-if="activeTab === 'tabQueue'" class="space-y-1">
              <li
                v-for="(entry, pos) in orderedQueue"
                :key="entry.track.id"
                class="flex items-center gap-2 rounded-lg px-2 py-1.5 text-xs"
                :class="pos === player.orderPos ? 'bg-surface-2 text-white' : 'text-text-secondary'"
              >
                <button type="button" class="min-w-0 flex-1 truncate text-left hover:text-white" @click="player.playAt(pos)">
                  <span class="mr-2 font-mono text-caption text-text-muted">{{ pos + 1 }}</span>{{ entry.track.title }}
                </button>
                <button
                  type="button"
                  class="rounded p-1 text-text-muted hover:text-white"
                  :aria-label="t('player.remove')"
                  @click="player.removeFromQueue(entry.track.id)"
                >
                  <svg class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
                    <path d="M18 6 6 18M6 6l12 12" />
                  </svg>
                </button>
              </li>
            </ol>
            <p v-else-if="activeTab === 'tabLyrics'" class="whitespace-pre-line text-xs leading-relaxed text-text-secondary">
              {{ player.currentTrack.lyrics || t('player.noLyrics') }}
            </p>
            <div v-else class="space-y-2 text-xs text-text-secondary">
              <p><span class="font-bold text-brand-300">{{ player.currentTrack.aiProvider }}</span></p>
              <p v-if="player.currentTrack.aiDisclosureText" class="leading-relaxed">{{ player.currentTrack.aiDisclosureText }}</p>
              <RouterLink :to="localePath({ name: 'track-detail', params: { id: player.currentTrack ? detailParam(player.currentTrack) : '' } })" class="inline-block font-medium text-brand-400 hover:underline" @click="$emit('close')">
                {{ t('player.viewTrack') }}
              </RouterLink>
            </div>
          </div>
        </template>

        <div v-else class="flex flex-1 flex-col items-center justify-center gap-2 py-16 text-center">
          <p class="text-sm font-semibold text-text-secondary">{{ t('player.idle') }}</p>
          <p class="text-xs text-text-muted">{{ t('player.idleHint') }}</p>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue';
import { usePlayerStore } from '@/stores/playerStore';
import WaveformSeekbar from './WaveformSeekbar.vue';
import UiCoverArt from '@/components/ui/UiCoverArt.vue';
import VerifiedSeal from '@/components/ui/VerifiedSeal.vue';
import { detailParam } from '@/utils/seo';

const emit = defineEmits<{
  (e: 'close'): void;
}>();

const player = usePlayerStore();
const { t } = useI18n();
const localePath = useLocalePath();

// NOTE: tab labelKeys resolve dynamically (invisible to check-i18n usage
// scan — they are used, do not delete).
const tabs = [
  { id: 'tabQueue', labelKey: 'player.tabQueue' },
  { id: 'tabLyrics', labelKey: 'player.tabLyrics' },
  { id: 'tabLicense', labelKey: 'player.tabLicense' },
] as const;
const activeTab = ref<(typeof tabs)[number]['id']>('tabQueue');

// Track carries no creator display name (partnerId only) — show the provider.
const creatorLabel = computed(() => player.currentTrack?.aiProvider ?? '');

const orderedQueue = computed(() =>
  player.order.map((qi) => ({ track: player.queue[qi]! })).filter((e) => e.track),
);

const repeatLabel = computed(() => {
  if (player.repeat === 'one') return t('player.repeatOne');
  if (player.repeat === 'all') return t('player.repeatAll');
  return t('player.repeatOff');
});

function fmtPos(ms: number): string {
  const s = Math.floor(ms / 1000);
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`;
}

function fmtDur(ms: number): string {
  const s = Math.round(ms / 1000);
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`;
}

function onKeydown(e: KeyboardEvent): void {
  if (e.key === 'Escape') emit('close');
}

onMounted(() => window.addEventListener('keydown', onKeydown));
onUnmounted(() => window.removeEventListener('keydown', onKeydown));
</script>
