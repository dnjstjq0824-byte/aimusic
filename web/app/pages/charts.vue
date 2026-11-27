<template>
  <div class="space-y-6">
    <UiPageHeader :title="t('track.chartsTitle')">
      <template #actions>
        <UiSelect v-model="genre" :label="t('explore.genreLabel')" class="w-36" @change="selectGenre">
          <option value="">{{ t('explore.genreAll') }}</option>
          <option v-for="g in GENRES" :key="g" :value="g">{{ tagDisplayName(g, t, te) }}</option>
        </UiSelect>
        <div class="flex gap-1 rounded-lg border border-surface-border bg-surface-1 p-1" role="tablist" :aria-label="t('track.chartsTitle')">
          <button
            v-for="w in windows"
            :key="w.id"
            type="button"
            role="tab"
            :aria-selected="window === w.id"
            class="whitespace-nowrap rounded-md px-3 py-1.5 text-xs font-bold transition-colors"
            :class="window === w.id ? 'bg-brand-500 text-on-brand' : 'text-text-secondary hover:text-white'"
            @click="selectWindow(w.id)"
          >
            {{ t(w.labelKey) }}
          </button>
        </div>
      </template>
    </UiPageHeader>

    <UiErrorBanner :message="error" :title="t('track.errTitle')" :retry-label="t('track.retry')" @retry="load" />

    <div v-if="loading" class="space-y-2">
      <UiSkeleton v-for="i in 8" :key="i" height="h-12" width="w-full" rounded="lg" />
    </div>

    <div v-else-if="!error && tracks.length === 0" class="py-12">
      <UiEmptyState :title="t('track.chartsEmpty')" />
    </div>

    <ol v-else class="space-y-1">
      <li v-for="(tr, i) in tracks" :key="tr.id">
        <TrackRow
          :track="tr"
          :rank="offset + i + 1"
          @play="playOne"
        />
      </li>
    </ol>

  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { trackApi } from '@/api/endpoints';
import { usePlayerStore } from '@/stores/playerStore';
import { useApiError } from '@/composables/useApiError';
import type { Track } from '@/api/types';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';
import UiSelect from '@/components/ui/UiSelect.vue';
import { GENRES, tagDisplayName } from '@/utils/tags';
import TrackRow from '@/components/TrackRow.vue';

const { t, te } = useI18n();
const route = useRoute();
const router = useRouter();
const player = usePlayerStore();
const apiError = useApiError();

type Window = 'day' | 'week' | 'month' | 'all';
// NOTE: labelKeys resolve dynamically (invisible to check-i18n usage
// scan — they are used, do not delete).
const windows: { id: Window; labelKey: string }[] = [
  { id: 'day', labelKey: 'track.chartDay' },
  { id: 'week', labelKey: 'track.chartWeek' },
  { id: 'month', labelKey: 'track.chartMonth' },
  { id: 'all', labelKey: 'track.chartAll' },
];
const window = ref<Window>(
  route.query.window === 'day' || route.query.window === 'month' || route.query.window === 'all' ? route.query.window : 'week',
);
// Genre charts (?genre=); only known genres are accepted from the URL.
const genre = ref<string>(
  typeof route.query.genre === 'string' && GENRES.includes(route.query.genre) ? route.query.genre : '',
);

const tracks = ref<Track[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);
const offset = 0;

async function load(): Promise<void> {
  loading.value = true;
  error.value = null;
  try {
    tracks.value = (await trackApi.charts(window.value, 50, genre.value)) ?? [];
  } catch (err: unknown) {
    error.value = apiError(err);
  } finally {
    loading.value = false;
  }
}

function syncQuery(): void {
  void router.replace({ query: { ...route.query, window: window.value, genre: genre.value || undefined } });
}

function selectWindow(w: Window): void {
  window.value = w;
  syncQuery();
  void load();
}

function selectGenre(): void {
  syncQuery();
  void load();
}

// The whole chart becomes the queue: the next rank plays when one ends.
function playOne(tr: Track): void {
  player.setQueue(tracks.value, tr.id);
}

onMounted(() => {
  void load();
});
</script>
