<template>
  <div class="flex items-center gap-3 rounded-2xl px-2 py-2 transition-colors hover:bg-white/[0.04]">
    <span v-if="rank != null" class="w-6 shrink-0 text-center font-mono text-sm text-text-muted">
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
        class="block truncate text-sm font-medium text-white transition-colors hover:text-brand-300"
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
      :cert-number="track.certNumber ?? null"
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
import { useTrackActions } from '@/composables/useTrackActions';
import type { Track } from '@/api/types';
const localePath = useLocalePath();
const { t, te } = useI18n();

const props = defineProps<{ track: Track; rank?: number | null }>();

defineEmits<{
  (e: 'play', track: Track): void;
}>();

// Queue / play-next / add-to-playlist / report are app-wide flows: every
// list gets them without the page wiring handlers (no dead menu items).
const actions = useTrackActions();

const moodLabel = computed(() => {
  const first = props.track.moodTags?.[0];
  return first ? tagDisplayName(first, t, te) : '';
});

const playable = computed(() => props.track.status === 'READY');
const menuItems = computed<MenuItem[]>(() => [
  { label: t('track.playNext'), disabled: !playable.value },
  { label: t('track.enqueue'), disabled: !playable.value },
  { label: t('track.addToPlaylist'), disabled: !playable.value },
  { label: t('track.reportAction'), danger: true },
]);

function onMenuSelect(index: number): void {
  if (index === 0) actions.playNext(props.track);
  else if (index === 1) actions.enqueue(props.track);
  else if (index === 2) actions.addToPlaylist(props.track);
  else actions.report(props.track);
}

function fmtDuration(ms?: number | null): string {
  if (!ms) return '0:00';
  const totalSec = Math.round(ms / 1000);
  return `${Math.floor(totalSec / 60)}:${String(totalSec % 60).padStart(2, '0')}`;
}
</script>
