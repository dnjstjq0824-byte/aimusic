<!-- Facet landing body (WI-S4): shared by mood/genre/use pages. -->
<template>
  <div class="space-y-6">
    <div>
      <h1 class="font-display text-2xl font-medium text-white sm:text-3xl">{{ heading }}</h1>
      <p class="mt-2 max-w-2xl text-sm leading-relaxed text-text-secondary">{{ description }}</p>
      <p class="mt-1 text-xs text-text-muted">{{ t('seo.landingCount', { count: fmt.number(count) }) }}</p>
    </div>

    <div v-if="loading" class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
      <UiSkeleton v-for="i in 6" :key="i" height="h-20" width="w-full" rounded="xl" />
    </div>

    <ol v-else class="space-y-1">
      <li v-for="tr in tracks" :key="tr.id">
        <TrackRow :track="tr" @play="playOne" @enqueue="enqueueOne" />
      </li>
    </ol>

    <div class="text-center">
      <RouterLink :to="localePath({ name: 'explore', query: exploreQuery })">
        <UiButton variant="secondary" size="sm">{{ t('seo.landingMore') }}</UiButton>
      </RouterLink>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { usePlayerStore } from '@/stores/playerStore';
import { useToast } from '@/composables/useToast';
import { useFormat } from '@/composables/useFormat';
import type { Track } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import TrackRow from '@/components/TrackRow.vue';
import type { LandingKind } from '@/utils/seo';

const props = defineProps<{
  kind: LandingKind;
  tagKey: string;
  heading: string;
  description: string;
  count: number;
  tracks: Track[];
  loading: boolean;
}>();

const localePath = useLocalePath();
const { t } = useI18n();
const player = usePlayerStore();
const toast = useToast();
const fmt = useFormat();

const exploreQuery = computed(() => {
  if (props.kind === 'mood') return { mood: props.tagKey };
  if (props.kind === 'genre') return { genre: props.tagKey };
  return { mood: props.tagKey };
});

function playOne(tr: Track): void {
  player.setQueue([tr], tr.id);
}

function enqueueOne(tr: Track): void {
  player.enqueue(tr);
  toast.info(t('track.toastQueued'));
}
</script>
