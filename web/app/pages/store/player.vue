<template>
  <div class="flex min-h-screen flex-col bg-surface-0 font-sans text-text-primary selection:bg-brand-500">
    <header class="border-b border-surface-border bg-surface-1/90 px-4 py-3 sm:px-6">
      <div class="mx-auto flex max-w-4xl items-center justify-between gap-3">
        <div class="flex items-center gap-3">
          <BrandLogo :size="30" tagline="Store Player" />
          <span v-if="player.queue" class="hidden text-xs text-text-secondary sm:inline">
            {{ player.queue.businessName }}
          </span>
        </div>
        <div class="flex items-center gap-2 text-xs">
          <UiBadge :variant="player.isOnline ? 'success' : 'danger'" size="xs">
            {{ player.isOnline ? t('business.player.online') : t('business.player.offline') }}
          </UiBadge>
          <UiBadge v-if="player.queue" :variant="player.queue.open ? 'success' : 'warning'" size="xs">
            {{ player.queue.open ? t('business.player.open') : queueReason }}
          </UiBadge>
          <button
            v-if="player.claimed"
            type="button"
            class="text-text-muted hover:text-text-primary"
            @click="unclaimConfirmOpen = true"
          >
            {{ t('business.player.unclaim') }}
          </button>
        </div>
      </div>
    </header>

    <main class="mx-auto flex w-full max-w-4xl flex-1 flex-col px-4 py-6 sm:px-6">
      <div v-if="!player.claimed || player.needsClaim" class="mx-auto w-full max-w-md flex-1 content-center">
        <UiCard padding="md">
          <h1 class="text-lg font-bold text-white">{{ t('business.player.claimTitle') }}</h1>
          <p class="mt-1 text-xs text-text-secondary">
            {{ t('business.player.claimDesc') }}
          </p>
          <form class="mt-4 space-y-3" @submit.prevent="doClaim">
            <UiInput
              v-model="pairingCode"
              :label="t('business.player.claimCode')"
              :placeholder="t('business.player.claimCodePh')"
              :maxlength="8"
              autocapitalize="characters"
              required
            />
            <UiErrorBanner :message="claimError" :title="t('business.player.claimFail')" />
            <UiButton variant="brand" size="md" type="submit" :disabled="claiming" class="w-full">
              {{ claiming ? t('business.player.claiming') : t('business.player.claimGo') }}
            </UiButton>
          </form>
        </UiCard>
      </div>

      <div v-else class="flex flex-1 flex-col gap-5">
        <UiErrorBanner
          v-if="player.error && player.error !== 'OFFLINE_SNAPSHOT'"
          :message="player.error"
          :title="t('business.player.playError')"
          :retry-label="t('business.player.retryQueue')"
          @retry="player.refreshQueue()"
        />
        <div v-if="player.error === 'OFFLINE_SNAPSHOT'" class="rounded-lg border border-amber-500/40 bg-amber-500/10 px-4 py-2 text-xs text-amber-200">
          {{ t('business.player.offlineSnapshot') }}
        </div>

        <section class="rounded-2xl border border-surface-border bg-surface-1 p-5 sm:p-6">
          <div class="flex flex-col gap-4 sm:flex-row sm:items-center">
            <UiCoverArt :seed="nowTitle" :title="nowTitle" size="lg" />
            <div class="min-w-0 flex-1">
              <p class="text-caption font-semibold uppercase tracking-wider text-text-muted">
                {{ player.queue?.playlistName || t('business.player.noPlaylist') }}
              </p>
              <h1 class="mt-1 truncate text-xl font-bold text-white sm:text-2xl">{{ nowTitle }}</h1>
              <div class="mt-2 flex flex-wrap items-center gap-2 text-caption">
                <UiBadge variant="info" size="xs">{{ t('business.player.aiMade') }}</UiBadge>
                <span v-if="player.currentDetail?.aiProvider" class="text-text-secondary">
                  {{ player.currentDetail.aiProvider }}
                </span>
                <span v-if="player.currentDetail?.aiDisclosureText" class="text-text-muted">
                  · {{ player.currentDetail.aiDisclosureText }}
                </span>
              </div>
            </div>
            <div class="flex shrink-0 items-center gap-2">
              <UiButton variant="secondary" size="md" :disabled="!canPlay" @click="togglePlay">
                {{ player.isPlaying ? t('business.player.pause') : t('business.player.play') }}
              </UiButton>
              <UiButton variant="ghost" size="md" :disabled="!canPlay" @click="player.advance()">{{ t('business.player.next') }}</UiButton>
            </div>
          </div>
          <div class="mt-4 flex items-center gap-3 text-caption text-text-secondary">
            <span>{{ t('business.player.volume', { n: player.queue?.volume ?? 70 }) }}</span>
            <div class="h-1.5 flex-1 overflow-hidden rounded-full bg-surface-2">
              <div class="h-full rounded-full bg-brand-500" :style="{ width: `${player.queue?.volume ?? 70}%` }" />
            </div>
            <span class="shrink-0">{{ t('business.player.remote') }}</span>
          </div>
        </section>

        <section>
          <h2 class="mb-2 text-xs font-bold uppercase tracking-wider text-text-secondary">
            {{ t('business.player.queueTitle', { n: queueCount }) }}
          </h2>
          <div v-if="!player.queue || player.queue.tracks.length === 0" class="rounded-lg border border-dashed border-surface-border px-4 py-6 text-center text-xs text-text-muted">
            {{ queueEmptyText }}
          </div>
          <ol v-else class="divide-y divide-surface-border overflow-hidden rounded-lg border border-surface-border">
            <li
              v-for="(trk, i) in player.queue.tracks"
              :key="trk.trackId"
              class="flex items-center gap-3 px-4 py-2.5 text-xs"
              :class="i === player.trackIndex ? 'bg-brand-950/60 font-bold text-white' : 'text-text-primary hover:bg-surface-2/50'"
            >
              <span class="w-6 shrink-0 font-mono text-text-muted">{{ String(i + 1).padStart(2, '0') }}</span>
              <button type="button" class="min-w-0 flex-1 truncate text-left hover:text-white" @click="player.playIndex(i)">
                {{ trk.title }}
              </button>
              <span v-if="i === player.trackIndex" class="shrink-0 text-brand-300">
                {{ player.isPlaying ? t('business.player.playing') : t('business.player.selected') }}
              </span>
            </li>
          </ol>
        </section>
      </div>
    </main>

    <UiConfirmDialog
      v-model="unclaimConfirmOpen"
      :title="t('business.player.unclaimTitle')"
      :description="t('business.player.unclaimDesc')"
      :body="t('business.player.unclaimBody')"
      :confirm-label="t('business.player.unclaim')"
      danger
      @confirm="player.unclaim()"
    />
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { computed, onMounted, onUnmounted, ref } from 'vue';
import { usePlayerStore } from '@/stores/playerStore';
import { useStorePlayerStore } from '@/stores/storePlayerStore';
import { errMessageOf } from '@/api/client';
import BrandLogo from '@/components/BrandLogo.vue';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiConfirmDialog from '@/components/ui/UiConfirmDialog.vue';
import UiCoverArt from '@/components/ui/UiCoverArt.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiInput from '@/components/ui/UiInput.vue';

const { t, te, locale, setLocale } = useI18n();
const route = useRoute();
const player = useStorePlayerStore();
const listenerPlayer = usePlayerStore();
const pairingCode = ref('');
const claiming = ref(false);
const claimError = ref<string | null>(null);
const unclaimConfirmOpen = ref(false);

const nowTitle = computed(() => player.currentTrack?.title ?? t('business.player.noTrack'));
const canPlay = computed(() => (player.queue?.tracks.length ?? 0) > 0);
const queueCount = computed(() => player.queue?.tracks.length ?? 0);

function reasonLabel(reason: string): string {
  const key = `business.player.reason.${reason}`;
  return te(key) ? String(t(key)) : reason;
}

const queueReason = computed(() => reasonLabel(player.queue?.reason ?? ''));

const queueEmptyText = computed(() => {
  if (!player.queue) return t('business.player.queueLoading');
  if (!player.queue.open) return t('business.player.queueClosed', { reason: queueReason.value });
  return t('business.player.queueEmpty');
});

function togglePlay(): void {
  if (player.isPlaying) player.pause();
  else player.resume();
}

async function doClaim(): Promise<void> {
  claiming.value = true;
  claimError.value = null;
  try {
    await player.claim(pairingCode.value);
    pairingCode.value = '';
  } catch (err: unknown) {
    claimError.value = errMessageOf(err);
  } finally {
    claiming.value = false;
  }
}

// R-B: the store player follows the device language (?lang= override first),
// falling back to the saved locale. Supported: ko/en/ja/zh-cn/th.
type PlayerLocale = 'ko' | 'en' | 'ja' | 'zh-cn' | 'th';
function resolvePlayerLocale(): PlayerLocale | null {
  const q = route.query.lang;
  const raw = Array.isArray(q) ? q[0] : q;
  const pick = (code: string): PlayerLocale | null => {
    const c = code.toLowerCase();
    if (c.startsWith('ko')) return 'ko';
    if (c.startsWith('ja')) return 'ja';
    if (c.startsWith('zh')) return 'zh-cn';
    if (c.startsWith('th')) return 'th';
    if (c.startsWith('en')) return 'en';
    return null;
  };
  if (typeof raw === 'string' && raw !== '') {
    const forced = pick(raw);
    if (forced) return forced;
  }
  if (typeof navigator !== 'undefined') {
    for (const lang of navigator.languages ?? [navigator.language]) {
      const found = pick(lang ?? '');
      if (found) return found;
    }
  }
  return null;
}

onMounted(() => {
  // Never play listener audio and store audio at once on one device.
  listenerPlayer.pause();
  const resolved = resolvePlayerLocale();
  if (resolved && resolved !== locale.value) {
    void setLocale(resolved);
  }
  if (player.claimed) {
    void player.refreshQueue().then(() => player.startQueueRefresh());
  }
});

onUnmounted(() => {
  player.stopAll();
});
</script>
