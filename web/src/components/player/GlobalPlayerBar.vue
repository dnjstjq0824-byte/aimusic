<template>
  <footer class="fixed inset-x-0 bottom-0 border-t border-neutral-800 bg-neutral-900/95 px-4 py-2 backdrop-blur">
    <div v-if="player.skippedIds.length > 0" class="mx-auto max-w-5xl pb-1 text-xs text-amber-400">
      차단된 음원 {{ player.skippedIds.length }}곡을 건너뛰었습니다.
    </div>
    <div v-if="player.error && player.currentTrack" class="mx-auto max-w-5xl pb-1 text-xs text-red-400">
      {{ player.error }}
    </div>
    <div class="mx-auto flex max-w-5xl items-center gap-3">
      <div class="min-w-0 flex-1">
        <div class="truncate text-sm font-semibold">
          {{ player.currentTrack?.title ?? '재생 중인 음원이 없습니다' }}
        </div>
        <div class="truncate text-xs text-neutral-400">
          {{ player.currentTrack ? `${player.currentTrack.aiProvider} · ${fmt(player.positionMs)} / ${fmt(player.durationMs)}` : '큐에 음원을 추가해 주세요' }}
        </div>
        <WaveformSeekbar
          :position-ms="player.positionMs"
          :duration-ms="player.durationMs || player.currentTrack?.durationMs || 0"
          :seed="player.currentTrack?.id ?? 'idle'"
          @seek="player.seek($event)"
        />
      </div>
      <div class="flex shrink-0 items-center gap-1 sm:gap-2">
        <button
          class="rounded px-1.5 py-1 text-xs sm:px-2 sm:text-sm"
          :class="player.shuffle ? 'bg-brand-600' : 'bg-neutral-800'"
          title="Shuffle"
          @click="player.toggleShuffle()"
        >
          SF
        </button>
        <button class="rounded bg-neutral-800 px-2 py-1 sm:px-3" title="Previous" @click="player.playPrev()">|◀</button>
        <button
          class="rounded bg-brand-600 px-3 py-1 font-bold disabled:opacity-50 sm:px-4"
          :disabled="player.isLoading || !player.currentTrack"
          @click="player.toggle()"
        >
          {{ player.isLoading ? '…' : player.isPlaying ? '❚❚' : '▶' }}
        </button>
        <button class="rounded bg-neutral-800 px-2 py-1 sm:px-3" title="Next" @click="player.playNext(true)">▶|</button>
        <button class="rounded px-1.5 py-1 text-xs sm:px-2 sm:text-sm" :class="player.repeat !== 'off' ? 'bg-brand-600' : 'bg-neutral-800'" title="Repeat" @click="player.cycleRepeat()">
          {{ repeatLabel }}
        </button>
      </div>
    </div>
  </footer>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { usePlayerStore } from '@/stores/playerStore';
import WaveformSeekbar from './WaveformSeekbar.vue';

const player = usePlayerStore();
const repeatLabel = computed(() => (player.repeat === 'one' ? 'R1' : player.repeat === 'all' ? 'RA' : 'R–'));

function fmt(ms: number): string {
  const s = Math.floor(ms / 1000);
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`;
}
</script>
