<template>
  <div class="flex items-center gap-3 rounded-xl border border-transparent px-2 py-2 transition-colors hover:border-surface-border hover:bg-surface-1">
    <span v-if="rank != null" class="w-6 shrink-0 text-center font-mono text-sm font-black text-text-muted">
      {{ rank }}
    </span>
    <button
      type="button"
      class="group relative shrink-0"
      :aria-label="t('track.playNow')"
      :disabled="track.status !== 'READY'"
      @click="$emit('play', track)"
    >
      <UiCoverArt :seed="track.id" :title="track.title" size="sm" :verified="false" />
      <span class="absolute inset-0 flex items-center justify-center rounded-lg bg-black/45 text-white opacity-0 transition-opacity group-hover:opacity-100 group-focus:opacity-100 [@media(hover:none)]:opacity-0" aria-hidden="true">
        <svg class="h-4 w-4" viewBox="0 0 24 24" fill="currentColor">
          <path d="M8 5v14l11-7z" />
        </svg>
      </span>
    </button>
    <div class="min-w-0 flex-1">
      <RouterLink
        :to="localePath({ name: 'track-detail', params: { id: detailParam(track) } })"
        class="block truncate text-sm font-bold text-white transition-colors hover:text-brand-400"
      >
        {{ track.title }}
      </RouterLink>
      <p class="mt-0.5 truncate text-caption text-text-secondary">
        <span v-if="track.creatorName">{{ track.creatorName }}</span>
        <span v-if="track.creatorName && moodLabel"> · </span>
        <span v-if="moodLabel">{{ moodLabel }}</span>
      </p>
    </div>
    <VerifiedSeal
      v-if="track.status === 'READY'"
      :size="16"
      :label="t('player.verified')"
      :full-label="t('player.verified')"
      :show-label="false"
    />
    <span v-if="track.durationMs" class="hidden shrink-0 font-mono text-caption text-text-secondary sm:inline">
      {{ fmtDuration(track.durationMs) }}
    </span>
    <UiMenu :items="menuItems" :label="t('track.moreActions')" @select="onMenuSelect" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import UiCoverArt from '@/components/ui/UiCoverArt.vue';
import UiMenu, { type MenuItem } from '@/components/ui/UiMenu.vue';
import VerifiedSeal from '@/components/ui/VerifiedSeal.vue';
import { tagDisplayName } from '@/utils/tags';
import { detailParam } from '@/utils/seo';
import type { Track } from '@/api/types';
const localePath = useLocalePath();
const { t, te } = useI18n();

const props = defineProps<{ track: Track; rank?: number | null }>();

const emit = defineEmits<{
  (e: 'play', track: Track): void;
  (e: 'enqueue', track: Track): void;
  (e: 'add-to-playlist', track: Track): void;
  (e: 'report', track: Track): void;
}>();

const moodLabel = computed(() => {
  const first = props.track.moodTags?.[0];
  return first ? tagDisplayName(first, t, te) : '';
});

const menuItems = computed<MenuItem[]>(() => [
  { label: t('track.enqueue') },
  { label: t('track.addToPlaylist') },
  { label: t('track.reportAction'), danger: true },
]);

function onMenuSelect(index: number): void {
  if (index === 0) emit('enqueue', props.track);
  else if (index === 1) emit('add-to-playlist', props.track);
  else emit('report', props.track);
}

function fmtDuration(ms?: number | null): string {
  if (!ms) return '0:00';
  const totalSec = Math.round(ms / 1000);
  return `${Math.floor(totalSec / 60)}:${String(totalSec % 60).padStart(2, '0')}`;
}
</script>
