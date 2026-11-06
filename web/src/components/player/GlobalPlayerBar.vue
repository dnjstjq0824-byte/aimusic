<template>
  <div class="fixed inset-x-0 bottom-0 z-40 border-t border-surface-border bg-surface-1/95 px-4 py-2.5 backdrop-blur-2xl shadow-2xl transition-all">
    <!-- Notice strip if tracks were skipped -->
    <div v-if="player.skippedIds.length > 0" class="mx-auto max-w-7xl pb-1.5 text-center text-[11px] text-accent-amber font-medium">
      ⚠️ 비상업 라이선스 사유로 차단된 음원 {{ player.skippedIds.length }}곡을 재생 큐에서 안전하게 건너뛰었습니다.
    </div>

    <div v-if="player.error && player.currentTrack" class="mx-auto max-w-7xl pb-1.5 text-center text-[11px] text-accent-crimson font-medium">
      {{ player.error }}
    </div>

    <div class="mx-auto flex max-w-7xl items-center justify-between gap-4">
      <!-- Left: Track info -->
      <div class="flex items-center gap-3 min-w-0 max-w-[280px] sm:max-w-xs">
        <div class="relative flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-gradient-to-tr from-surface-2 to-surface-3 border border-surface-border overflow-hidden">
          <span v-if="!player.isPlaying" class="text-lg">🎵</span>
          <div v-else class="flex items-end gap-0.5 h-4">
            <span class="w-1 bg-brand-500 rounded-full animate-bounce h-full" style="animation-delay: 0.1s" />
            <span class="w-1 bg-brand-400 rounded-full animate-bounce h-3/4" style="animation-delay: 0.3s" />
            <span class="w-1 bg-amber-400 rounded-full animate-bounce h-1/2" style="animation-delay: 0.2s" />
          </div>
        </div>

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
              <span>·</span>
              <span class="text-accent-emerald font-semibold">✓ 상업 검증</span>
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
            class="text-xs transition-colors p-1"
            :class="player.shuffle ? 'text-brand-400 font-bold' : 'text-neutral-400 hover:text-white'"
            title="셔플 재생 (Shuffle)"
            @click="player.toggleShuffle()"
          >
            🔀
          </button>

          <!-- Previous -->
          <button
            type="button"
            class="text-sm text-neutral-300 hover:text-white transition-colors p-1"
            title="이전 곡"
            @click="player.playPrev()"
          >
            ⏮
          </button>

          <!-- Play / Pause -->
          <button
            type="button"
            class="flex h-9 w-9 items-center justify-center rounded-full bg-brand-500 text-white font-bold shadow-glow-brand hover:scale-105 active:scale-95 transition-all disabled:opacity-50 cursor-pointer"
            :disabled="player.isLoading || !player.currentTrack"
            title="재생 / 일시정지"
            @click="player.toggle()"
          >
            <span v-if="player.isLoading" class="animate-spin text-xs">⏳</span>
            <span v-else-if="player.isPlaying" class="text-xs">❚❚</span>
            <span v-else class="text-xs ml-0.5">▶</span>
          </button>

          <!-- Next -->
          <button
            type="button"
            class="text-sm text-neutral-300 hover:text-white transition-colors p-1"
            title="다음 곡"
            @click="player.playNext(true)"
          >
            ⏭
          </button>

          <!-- Repeat -->
          <button
            type="button"
            class="text-xs transition-colors p-1 relative"
            :class="player.repeat !== 'off' ? 'text-brand-400 font-bold' : 'text-neutral-400 hover:text-white'"
            title="반복 재생 (Repeat)"
            @click="player.cycleRepeat()"
          >
            🔁
            <span v-if="player.repeat === 'one'" class="absolute -top-1 -right-1 text-[8px] bg-brand-500 text-white rounded-full px-1">1</span>
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
          title="재생 대기열 (Queue)"
          @click="$emit('toggle-queue')"
        >
          <span>큐</span>
          <span class="rounded-full bg-brand-500/20 text-brand-400 px-1.5 py-0.2 text-[10px] font-mono font-bold">
            {{ player.queue.length }}
          </span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { usePlayerStore } from '@/stores/playerStore';
import WaveformSeekbar from './WaveformSeekbar.vue';

const player = usePlayerStore();

defineEmits<{
  (e: 'toggle-queue'): void;
}>();

function fmt(ms: number): string {
  const s = Math.floor(ms / 1000);
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`;
}
</script>
