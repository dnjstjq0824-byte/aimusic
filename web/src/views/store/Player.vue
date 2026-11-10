<template>
  <div class="flex min-h-screen flex-col bg-surface-0 font-sans text-neutral-100 selection:bg-brand-500">
    <header class="border-b border-surface-border bg-surface-1/90 px-4 py-3 sm:px-6">
      <div class="mx-auto flex max-w-4xl items-center justify-between gap-3">
        <div class="flex items-center gap-3">
          <BrandLogo :size="30" tagline="Store Player" />
          <span v-if="player.queue" class="hidden text-xs text-neutral-400 sm:inline">
            {{ player.queue.businessName }}
          </span>
        </div>
        <div class="flex items-center gap-2 text-xs">
          <UiBadge :variant="player.isOnline ? 'success' : 'danger'" size="xs">
            {{ player.isOnline ? '온라인' : '오프라인' }}
          </UiBadge>
          <UiBadge v-if="player.queue" :variant="player.queue.open ? 'success' : 'warning'" size="xs">
            {{ player.queue.open ? '영업 중' : queueReasonKo }}
          </UiBadge>
          <button
            v-if="player.claimed"
            type="button"
            class="text-neutral-500 hover:text-neutral-300"
            @click="unclaimConfirmOpen = true"
          >
            연결 해제
          </button>
        </div>
      </div>
    </header>

    <main class="mx-auto flex w-full max-w-4xl flex-1 flex-col px-4 py-6 sm:px-6">
      <div v-if="!player.claimed || player.needsClaim" class="mx-auto w-full max-w-md flex-1 content-center">
        <UiCard padding="md">
          <h1 class="text-lg font-bold text-white">플레이어 연결</h1>
          <p class="mt-1 text-xs text-neutral-400">
            파트너 콘솔의 매장 관리에서 발급한 8자리 페어링 코드를 입력하세요.
          </p>
          <form class="mt-4 space-y-3" @submit.prevent="doClaim">
            <UiInput
              v-model="pairingCode"
              label="페어링 코드"
              placeholder="예: AB3DEF9H"
              :maxlength="8"
              autocapitalize="characters"
              required
            />
            <UiErrorBanner :message="claimError" title="연결 실패" />
            <UiButton variant="brand" size="md" type="submit" :disabled="claiming" class="w-full">
              {{ claiming ? '연결 중…' : '연결하기' }}
            </UiButton>
          </form>
        </UiCard>
      </div>

      <div v-else class="flex flex-1 flex-col gap-5">
        <UiErrorBanner
          v-if="player.error && player.error !== 'OFFLINE_SNAPSHOT'"
          :message="player.error"
          title="재생 오류"
          retry-label="큐 새로고침"
          @retry="player.refreshQueue()"
        />
        <div v-if="player.error === 'OFFLINE_SNAPSHOT'" class="rounded-lg border border-amber-500/40 bg-amber-500/10 px-4 py-2 text-xs text-amber-200">
          오프라인 상태입니다. 마지막으로 저장된 편성을 표시합니다. 서명 URL이 만료되면(5분) 온라인 연결이 필요합니다.
        </div>

        <section class="rounded-2xl border border-surface-border bg-surface-1 p-5 sm:p-6">
          <div class="flex flex-col gap-4 sm:flex-row sm:items-center">
            <UiCoverArt :seed="nowTitle" :title="nowTitle" size="lg" />
            <div class="min-w-0 flex-1">
              <p class="text-[11px] font-semibold uppercase tracking-wider text-neutral-500">
                {{ player.queue?.playlistName || '편성 없음' }}
              </p>
              <h1 class="mt-1 truncate text-xl font-bold text-white sm:text-2xl">{{ nowTitle }}</h1>
              <div class="mt-2 flex flex-wrap items-center gap-2 text-[11px]">
                <UiBadge variant="info" size="xs">AI 생성음원</UiBadge>
                <span v-if="player.currentDetail?.aiProvider" class="text-neutral-400">
                  {{ player.currentDetail.aiProvider }}
                </span>
                <span v-if="player.currentDetail?.aiDisclosureText" class="text-neutral-500">
                  · {{ player.currentDetail.aiDisclosureText }}
                </span>
              </div>
            </div>
            <div class="flex shrink-0 items-center gap-2">
              <UiButton variant="secondary" size="md" :disabled="!canPlay" @click="togglePlay">
                {{ player.isPlaying ? '일시정지' : '재생' }}
              </UiButton>
              <UiButton variant="ghost" size="md" :disabled="!canPlay" @click="player.advance()">다음 곡</UiButton>
            </div>
          </div>
          <div class="mt-4 flex items-center gap-3 text-[11px] text-neutral-400">
            <span>매장 볼륨 {{ player.queue?.volume ?? 70 }}</span>
            <div class="h-1.5 flex-1 overflow-hidden rounded-full bg-surface-2">
              <div class="h-full rounded-full bg-brand-500" :style="{ width: `${player.queue?.volume ?? 70}%` }" />
            </div>
            <span class="shrink-0">원격 제어 (파트너 콘솔)</span>
          </div>
        </section>

        <section>
          <h2 class="mb-2 text-xs font-bold uppercase tracking-wider text-neutral-400">
            오늘의 편성 {{ queueCount }}곡
          </h2>
          <div v-if="!player.queue || player.queue.tracks.length === 0" class="rounded-lg border border-dashed border-surface-border px-4 py-6 text-center text-xs text-neutral-500">
            {{ queueEmptyText }}
          </div>
          <ol v-else class="divide-y divide-surface-border overflow-hidden rounded-lg border border-surface-border">
            <li
              v-for="(t, i) in player.queue.tracks"
              :key="t.trackId"
              class="flex items-center gap-3 px-4 py-2.5 text-xs"
              :class="i === player.trackIndex ? 'bg-brand-950/60 font-bold text-white' : 'text-neutral-300 hover:bg-surface-2/50'"
            >
              <span class="w-6 shrink-0 font-mono text-neutral-500">{{ String(i + 1).padStart(2, '0') }}</span>
              <button type="button" class="min-w-0 flex-1 truncate text-left hover:text-white" @click="player.playIndex(i)">
                {{ t.title }}
              </button>
              <span v-if="i === player.trackIndex" class="shrink-0 text-brand-300">
                {{ player.isPlaying ? '재생 중' : '선택됨' }}
              </span>
            </li>
          </ol>
        </section>
      </div>
    </main>

    <UiConfirmDialog
      v-model="unclaimConfirmOpen"
      title="매장 연결 해제"
      description="이 태블릿의 매장 페어링을 해제하고 재생을 중지합니다."
      body="다시 연결하려면 파트너 콘솔에서 새 페어링 코드를 발급받아야 합니다."
      confirm-label="연결 해제"
      danger
      @confirm="player.unclaim()"
    />
  </div>
</template>

<script setup lang="ts">
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

const player = useStorePlayerStore();
const listenerPlayer = usePlayerStore();
const pairingCode = ref('');
const claiming = ref(false);
const claimError = ref<string | null>(null);
const unclaimConfirmOpen = ref(false);

const nowTitle = computed(() => player.currentTrack?.title ?? '재생할 곡이 없습니다');
const canPlay = computed(() => (player.queue?.tracks.length ?? 0) > 0);
const queueCount = computed(() => player.queue?.tracks.length ?? 0);

const reasonKo: Record<string, string> = {
  closed_day: '휴무일',
  off_hours: '영업시간 외',
  no_schedule_no_curation: '편성 없음',
  no_playlist_no_curation: '선곡 없음',
  scheduled: '예약 편성',
  fallback_curation: '공식 큐레이션',
};

const queueReasonKo = computed(() => reasonKo[player.queue?.reason ?? ''] ?? player.queue?.reason ?? '');

const queueEmptyText = computed(() => {
  if (!player.queue) return '편성을 불러오는 중입니다.';
  if (!player.queue.open) return `지금은 재생 시간이 아닙니다. (${queueReasonKo.value}) 영업시간·편성은 파트너 콘솔에서 변경하세요.`;
  return '편성된 곡이 없습니다. 파트너 콘솔에서 선곡을 등록하세요.';
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

onMounted(() => {
  // Never play listener audio and store audio at once on one device.
  listenerPlayer.pause();
  if (player.claimed) {
    void player.refreshQueue().then(() => player.startQueueRefresh());
  }
});

onUnmounted(() => {
  player.stopAll();
});
</script>
