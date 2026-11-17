<template>
  <UiCard hoverable class="group flex min-w-0 flex-col justify-between">
    <div>
      <div class="flex items-start gap-3">
        <div class="relative shrink-0">
          <UiCoverArt :seed="track.id" :title="track.title" size="md" :verified="track.status === 'READY'" />
          <button
            type="button"
            class="absolute inset-0 flex items-center justify-center rounded-xl bg-black/45 text-white opacity-0 transition-opacity group-hover:opacity-100 focus:opacity-100 [@media(hover:none)]:bg-black/25 [@media(hover:none)]:opacity-100"
            :aria-label="t('track.playNow')"
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
            :to="localePath({ name: 'track-detail', params: { id: detailParam(track) } })"
            class="block truncate text-sm font-medium text-white transition-colors hover:text-brand-300"
          >
            {{ track.title }}
          </RouterLink>
          <p v-if="track.creatorName" class="mt-0.5 truncate text-caption text-text-secondary">
            {{ track.creatorName }}
          </p>
          <div class="mt-1.5 flex flex-wrap items-center gap-1.5">
            <AiBadge :track="track" />
            <span class="text-caption font-medium text-text-muted">{{ providerDisplayName(track.aiProvider) }}</span>
          </div>
        </div>
        <UiMenu :items="menuItems" :label="t('track.moreActions')" @select="onMenuSelect" />
      </div>

      <div class="mt-3 flex items-center justify-between border-t border-surface-border pt-2 text-caption text-text-secondary">
        <span>{{ t('track.plays', { count: fmt.number(track.playCount) }) }}</span>
        <span v-if="track.durationMs">{{ fmtDuration(track.durationMs) }}</span>
      </div>
    </div>
  </UiCard>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiCoverArt from '@/components/ui/UiCoverArt.vue';
import UiMenu, { type MenuItem } from '@/components/ui/UiMenu.vue';
import AiBadge from '@/components/legal/AiBadge.vue';
import { useFormat } from '@/composables/useFormat';
import { providerDisplayName } from '@/utils/providerDisplay';
import { detailParam } from '@/utils/seo';
import type { Track } from '@/api/types';
const localePath = useLocalePath();
const { t } = useI18n();
const fmt = useFormat();

const props = defineProps<{ track: Track }>();

const emit = defineEmits<{
  (e: 'play', track: Track): void;
  (e: 'enqueue', track: Track): void;
  (e: 'add-to-playlist', track: Track): void;
  (e: 'report', track: Track): void;
}>();

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
