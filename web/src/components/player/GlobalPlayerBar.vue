<template>
  <div class="fixed inset-x-0 bottom-0 z-40 border-t border-surface-border bg-surface-1/95 px-4 py-2.5 backdrop-blur-2xl shadow-2xl transition-all">
    <!-- Notice strip if tracks were skipped -->
    <div v-if="player.skippedIds.length > 0" class="mx-auto max-w-7xl pb-1.5 text-center text-[11px] text-accent-amber font-medium" role="status">
      비상업 라이선스 사유로 차단된 음원 {{ player.skippedIds.length }}곡을 재생 큐에서 건너뛰었습니다.
    </div>

    <div v-if="player.error && player.currentTrack" class="mx-auto max-w-7xl pb-1.5 text-center text-[11px] text-accent-crimson font-medium" role="alert">
      {{ player.error }}
    </div>

    <div class="mx-auto flex max-w-7xl items-center justify-between gap-4">
      <!-- Left: Track info -->
      <div class="flex items-center gap-3 min-w-0 max-w-[280px] sm:max-w-xs">
        <UiCoverArt
          :seed="player.currentTrack?.id ?? 'idle'"
          :title="player.currentTrack?.title ?? '재생 대기 중'"
          size="sm"
          :verified="Boolean(player.currentTrack)"
        />

        <div class="min-w-0 flex-1">
          <div v-if="player.currentTrack">
            <RouterLink
              :to="{ name: 'track-detail', params: { id: player.currentTrack.id } }"
              class="block truncate text-xs font-bold text-white hover:text-brand-400 transition-colors"
            >
              {{ player.currentTrack.title }}
            </RouterLink>
            <div class="flex items-center gap-1.5 text-[10px] text-neutral-400 mt-0.5 truncate">
              <span class="font-medium text-brand-300">{{ player.currentTrack.aiProvider }}</span>
              <span aria-hidden="true">·</span>
              <span class="text-accent-emerald font-semibold">상업 검증됨</span>
            </div>
          </div>
          <div v-else>
            <div class="text-xs font-semibold text-neutral-400">재생 대기 중</div>
            <div class="text-[10px] text-neutral-500">원하는 음원을 선택하세요</div>
          </div>
        </div>
      </div>

      <!-- Center: Controls & Waveform Seekbar -->
      <div class="flex flex-1 max-w-xl flex-col items-center gap-1 min-w-0">
        <!-- Control buttons -->
        <div class="flex items-center gap-3 sm:gap-4">
          <!-- Shuffle -->
          <button
            type="button"
            class="rounded p-1 transition-colors"
            :class="player.shuffle ? 'text-brand-400' : 'text-neutral-400 hover:text-white'"
            :aria-pressed="player.shuffle"
            aria-label="셔플 재생"
            @click="player.toggleShuffle()"
          >
            <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <path d="M16 3h5v5M4 20 21 3M21 16v5h-5M15 15l6 6M4 4l5 5" />
            </svg>
          </button>

          <!-- Previous -->
          <button
            type="button"
            class="rounded p-1 text-neutral-300 hover:text-white transition-colors"
            aria-label="이전 곡"
            @click="player.playPrev()"
          >
            <svg class="h-5 w-5" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
              <path d="M6 5h2v14H6zM20 5v14L9 12z" />
            </svg>
          </button>

          <!-- Play / Pause -->
          <button
            type="button"
            class="flex h-9 w-9 items-center justify-center rounded-full bg-brand-500 text-[#04121a] shadow-glow-brand hover:bg-brand-400 active:scale-95 transition-all disabled:opacity-50 cursor-pointer"
            :disabled="player.isLoading || !player.currentTrack"
            :aria-label="player.isPlaying ? '일시정지' : '재생'"
            @click="player.toggle()"
          >
            <svg v-if="player.isLoading" class="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none" aria-hidden="true">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z" />
            </svg>
            <svg v-else-if="player.isPlaying" class="h-4 w-4" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
              <path d="M7 5h4v14H7zM13 5h4v14h-4z" />
            </svg>
            <svg v-else class="ml-0.5 h-4 w-4" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
              <path d="M8 5v14l11-7z" />
            </svg>
          </button>

          <!-- Next -->
          <button
            type="button"
            class="rounded p-1 text-neutral-300 hover:text-white transition-colors"
            aria-label="다음 곡"
            @click="player.playNext(true)"
          >
            <svg class="h-5 w-5" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
              <path d="M16 5h2v14h-2zM4 5v14l11-7z" />
            </svg>
          </button>

          <!-- Repeat -->
          <button
            type="button"
            class="relative rounded p-1 transition-colors"
            :class="player.repeat !== 'off' ? 'text-brand-400' : 'text-neutral-400 hover:text-white'"
            :aria-pressed="player.repeat !== 'off'"
            :aria-label="`반복 재생: ${repeatLabel}`"
            @click="player.cycleRepeat()"
          >
            <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <path d="m17 2 4 4-4 4" />
              <path d="M3 11v-1a4 4 0 0 1 4-4h14M7 22l-4-4 4-4" />
              <path d="M21 13v1a4 4 0 0 1-4 4H3" />
            </svg>
            <span v-if="player.repeat === 'one'" class="absolute -top-1 -right-1 rounded-full bg-brand-500 px-1 text-[8px] font-bold text-[#04121a]">1</span>
          </button>
        </div>

        <!-- Waveform Seekbar & time -->
        <div class="w-full flex items-center gap-2">
          <span class="text-[10px] font-mono text-neutral-400 w-8 text-right">{{ fmt(player.positionMs) }}</span>
          <div class="flex-1 min-w-0">
            <WaveformSeekbar
              :position-ms="player.positionMs"
              :duration-ms="player.durationMs || player.currentTrack?.durationMs || 0"
              :seed="player.currentTrack?.id ?? 'idle'"
              @seek="player.seek($event)"
            />
          </div>
          <span class="text-[10px] font-mono text-neutral-400 w-8">{{ fmt(player.durationMs || player.currentTrack?.durationMs || 0) }}</span>
        </div>
      </div>

      <!-- Right: Extras (Queue toggle) -->
      <div class="flex items-center gap-2.5 shrink-0">
        <button
          type="button"
          class="flex items-center gap-1.5 rounded-lg border border-surface-border bg-surface-2 px-2.5 py-1.5 text-xs text-neutral-300 hover:text-white hover:border-surface-border-hover transition-colors"
          aria-label="재생 대기열 열기"
          @click="$emit('toggle-queue')"
        >
          <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
            <path d="M4 6h16M4 12h16M4 18h10" />
          </svg>
          <span>큐</span>
          <span class="rounded-full bg-brand-500/20 text-brand-400 px-1.5 py-0.5 text-[10px] font-mono font-bold">
            {{ player.queue.length }}
          </span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { usePlayerStore } from '@/stores/playerStore';
import WaveformSeekbar from './WaveformSeekbar.vue';
import UiCoverArt from '@/components/ui/UiCoverArt.vue';

const player = usePlayerStore();

defineEmits<{
  (e: 'toggle-queue'): void;
}>();

const repeatLabel = computed(() => {
  if (player.repeat === 'one') return '한 곡 반복';
  if (player.repeat === 'all') return '전체 반복';
  return '끔';
});

function fmt(ms: number): string {
  const s = Math.floor(ms / 1000);
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`;
}
</script>
