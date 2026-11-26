<template>
  <div class="space-y-6">
    <UiPageHeader :title="t('track.chartsTitle')">
      <template #actions>
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
          @enqueue="enqueue"
          @add-to-playlist="openAddToPlaylist"
          @report="openReport"
        />
      </li>
    </ol>

    <DisputeReportModal
      v-if="reportTarget"
      :track-id="reportTarget.id"
      :track-title="reportTarget.title"
      @close="reportTarget = null"
      @reported="onReported"
    />

    <UiPlaylistPicker
      :model-value="Boolean(playlistTargetTrack)"
      :track-title="playlistTargetTrack?.title ?? ''"
      :playlists="playlists.playlists"
      :create-loading="creatingPlaylist"
      @update:model-value="playlistTargetTrack = $event ? playlistTargetTrack : null"
      @select="addTrackToPlaylist"
      @create="createNewPlaylist"
    />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { trackApi, playlistApi } from '@/api/endpoints';
import { useAuthStore } from '@/stores/authStore';
import { usePlayerStore } from '@/stores/playerStore';
import { usePlaylistStore } from '@/stores/playlistStore';
import { useToast } from '@/composables/useToast';
import { useApiError } from '@/composables/useApiError';
import type { Track } from '@/api/types';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';
import UiPlaylistPicker from '@/components/ui/UiPlaylistPicker.vue';
import TrackRow from '@/components/TrackRow.vue';
import DisputeReportModal from '@/components/legal/DisputeReportModal.vue';

const { t } = useI18n();
const localePath = useLocalePath();
const route = useRoute();
const router = useRouter();
const auth = useAuthStore();
const player = usePlayerStore();
const playlists = usePlaylistStore();
const toast = useToast();
const apiError = useApiError();

type Window = 'week' | 'month' | 'all';
// NOTE: labelKeys resolve dynamically (invisible to check-i18n usage
// scan — they are used, do not delete).
const windows: { id: Window; labelKey: string }[] = [
  { id: 'week', labelKey: 'track.chartWeek' },
  { id: 'month', labelKey: 'track.chartMonth' },
  { id: 'all', labelKey: 'track.chartAll' },
];
const window = ref<Window>(
  route.query.window === 'month' || route.query.window === 'all' ? route.query.window : 'week',
);

const tracks = ref<Track[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);
const offset = 0;
const reportTarget = ref<Track | null>(null);
const playlistTargetTrack = ref<Track | null>(null);
const creatingPlaylist = ref(false);

async function load(): Promise<void> {
  loading.value = true;
  error.value = null;
  try {
    tracks.value = (await trackApi.charts(window.value, 50)) ?? [];
  } catch (err: unknown) {
    error.value = apiError(err);
  } finally {
    loading.value = false;
  }
}

function selectWindow(w: Window): void {
  window.value = w;
  void load();
}

function playOne(tr: Track): void {
  player.setQueue([tr], tr.id);
}

function enqueue(tr: Track): void {
  player.enqueue(tr);
  toast.info(t('track.toastQueued'));
}

function requireLogin(): boolean {
  if (auth.isAuthenticated) return true;
  toast.info(t('track.toastNeedLogin'));
  void router.push(localePath({ name: 'login', query: { redirect: route.fullPath } }));
  return false;
}

function openAddToPlaylist(tr: Track): void {
  if (!requireLogin()) return;
  playlistTargetTrack.value = tr;
}

async function addTrackToPlaylist(playlistId: string): Promise<void> {
  if (!playlistTargetTrack.value) return;
  try {
    await playlistApi.addTrack(playlistId, playlistTargetTrack.value.id);
    toast.success(t('track.toastAdded'));
    playlistTargetTrack.value = null;
  } catch (err: unknown) {
    toast.error(apiError(err));
  }
}

async function createNewPlaylist(title: string): Promise<void> {
  creatingPlaylist.value = true;
  try {
    const created = await playlistApi.create(title, false);
    playlists.playlists.unshift(created);
    if (playlistTargetTrack.value) {
      await addTrackToPlaylist(created.id);
    }
  } catch (err: unknown) {
    toast.error(apiError(err));
  } finally {
    creatingPlaylist.value = false;
  }
}

function openReport(tr: Track): void {
  if (!requireLogin()) return;
  reportTarget.value = tr;
}

function onReported(): void {
  reportTarget.value = null;
  toast.success(t('track.toastReported'));
}

onMounted(() => {
  void load();
  void playlists.fetchMine();
});
</script>
