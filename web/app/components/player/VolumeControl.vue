<template>
  <div class="flex items-center gap-1.5">
    <button
      type="button"
      class="rounded p-1 text-text-secondary transition-colors hover:text-white"
      :aria-pressed="silent"
      :aria-label="silent ? t('player.unmute') : t('player.mute')"
      @click="player.toggleMute()"
    >
      <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <path d="M11 5 6 9H2v6h4l5 4V5z" fill="currentColor" />
        <template v-if="silent">
          <path d="m22 9-6 6M16 9l6 6" />
        </template>
        <template v-else>
          <path d="M15.5 8.5a5 5 0 0 1 0 7" />
          <path v-if="player.volume > 0.5" d="M19 5a10 10 0 0 1 0 14" />
        </template>
      </svg>
    </button>
    <input
      type="range"
      min="0"
      max="100"
      step="1"
      class="h-1 w-20 cursor-pointer accent-brand-500 lg:w-24"
      :value="silent ? 0 : Math.round(player.volume * 100)"
      :aria-label="t('player.volume')"
      :aria-valuetext="t('player.volumePct', { pct: silent ? 0 : Math.round(player.volume * 100) })"
      @input="onInput"
    >
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { usePlayerStore } from '@/stores/playerStore';

const player = usePlayerStore();
const { t } = useI18n();

const silent = computed(() => player.muted || player.volume === 0);

function onInput(e: Event): void {
  const v = Number((e.target as HTMLInputElement).value);
  if (Number.isFinite(v)) player.setVolume(v / 100);
}
</script>
