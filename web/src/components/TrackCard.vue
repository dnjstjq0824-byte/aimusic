<template>
  <UiCard hoverable class="group flex min-w-0 flex-col justify-between">
    <div>
      <div class="flex items-start gap-3">
        <div class="relative shrink-0">
          <UiCoverArt :seed="track.id" :title="track.title" size="md" :verified="track.status === 'READY'" />
          <button
            type="button"
            class="absolute inset-0 flex items-center justify-center rounded-xl bg-black/45 text-white opacity-0 transition-opacity group-hover:opacity-100 focus:opacity-100 [@media(hover:none)]:bg-black/25 [@media(hover:none)]:opacity-100"
            :aria-label="`「${track.title}」 재생`"
            :disabled="track.status !== 'READY'"
            @click="$emit('play', track)"
          >
            <svg class="h-6 w-6" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
              <path d="M8 5v14l11-7z" />
            </svg>
          </button>
        </div>
        <div class="min-w-0 flex-1">
          <RouterLink
            :to="{ name: 'track-detail', params: { id: track.id } }"
            class="block truncate text-sm font-bold text-white transition-colors hover:text-brand-400"
          >
            {{ track.title }}
          </RouterLink>
          <div class="mt-1.5 flex flex-wrap items-center gap-1.5">
            <AiBadge :track="track" />
          </div>
        </div>
      </div>

      <div class="mt-3 flex items-center justify-between border-t border-surface-border pt-2 text-[11px] text-neutral-400">
        <span>재생 {{ track.playCount.toLocaleString() }}회</span>
        <span v-if="track.durationMs">{{ fmtDuration(track.durationMs) }}</span>
        <span v-if="track.bitrateKbps">{{ track.bitrateKbps }}kbps</span>
      </div>
    </div>

    <div class="mt-3 flex items-center justify-between gap-1 pt-1">
      <button
        type="button"
        class="-my-1.5 py-1.5 text-[11px] font-medium text-neutral-400 transition-colors hover:text-white"
        @click="$emit('enqueue', track)"
      >
        + 큐에 추가
      </button>
      <button
        type="button"
        class="text-[11px] font-medium text-neutral-400 transition-colors hover:text-brand-400"
        @click="$emit('add-to-playlist', track)"
      >
        + 플레이리스트
      </button>
      <button
        type="button"
        class="text-[11px] font-medium text-neutral-500 transition-colors hover:text-accent-crimson"
        @click="$emit('report', track)"
      >
        신고
      </button>
    </div>
  </UiCard>
</template>

<script setup lang="ts">
import UiCard from '@/components/ui/UiCard.vue';
import UiCoverArt from '@/components/ui/UiCoverArt.vue';
import AiBadge from '@/components/legal/AiBadge.vue';
import type { Track } from '@/api/types';

defineProps<{ track: Track }>();

defineEmits<{
  (e: 'play', track: Track): void;
  (e: 'enqueue', track: Track): void;
  (e: 'add-to-playlist', track: Track): void;
  (e: 'report', track: Track): void;
}>();

function fmtDuration(ms?: number | null): string {
  if (!ms) return '0:00';
  const totalSec = Math.floor(ms / 1000);
  return `${Math.floor(totalSec / 60)}:${String(totalSec % 60).padStart(2, '0')}`;
}
</script>
