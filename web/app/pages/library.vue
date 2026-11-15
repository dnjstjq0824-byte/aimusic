<template>
  <div class="space-y-6">
    <UiPageHeader :title="t('library.title')">
      <template #actions>
        <UiButton
          v-if="canPlayAll"
          variant="brand"
          size="sm"
          :disabled="loading"
          @click="playAll"
        >
          {{ t('library.playAll') }}
        </UiButton>
      </template>
    </UiPageHeader>

    <UiErrorBanner :message="error" :title="t('track.errTitle')" :retry-label="t('track.retry')" @retry="load" />

    <div class="flex gap-1 overflow-x-auto rounded-xl border border-surface-border bg-surface-1 p-1 scrollbar-none" role="tablist" :aria-label="t('library.title')">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        type="button"
        role="tab"
        :aria-selected="activeTab === tab.id"
        class="flex-1 whitespace-nowrap rounded-lg px-3 py-2 text-xs font-bold transition-colors"
        :class="activeTab === tab.id ? 'bg-brand-500 text-on-brand' : 'text-text-secondary hover:text-white'"
        @click="activeTab = tab.id"
      >
        {{ t(tab.labelKey) }}
      </button>
    </div>

    <div v-if="loading" class="space-y-2">
      <UiSkeleton v-for="i in 6" :key="i" height="h-12" width="w-full" rounded="lg" />
    </div>

    <template v-else-if="!error">
      <!-- Liked tracks -->
      <div v-if="activeTab === 'liked'" class="space-y-1">
        <div v-if="liked.length === 0" class="py-10">
          <UiEmptyState :title="t('library.likedEmptyT')" :description="t('library.likedEmptyD')">
            <template #action>
              <RouterLink :to="localePath('/explore')">
                <UiButton variant="secondary" size="sm">{{ t('library.likedEmptyCta') }}</UiButton>
              </RouterLink>
            </template>
          </UiEmptyState>
        </div>
        <TrackRow
          v-for="tr in liked"
          :key="tr.id"
          :track="tr"
          @play="playOne"
          @enqueue="enqueue"
          @add-to-playlist="openAddToPlaylist"
          @report="openReport"
        />
      </div>

      <!-- Recent plays -->
      <div v-if="activeTab === 'recent'" class="space-y-1">
        <div v-if="recent.length === 0" class="py-10">
          <UiEmptyState :title="t('library.recentEmptyT')" :description="t('library.recentEmptyD')" />
        </div>
        <TrackRow
          v-for="tr in recent"
          :key="tr.id"
          :track="tr"
          @play="playOne"
          @enqueue="enqueue"
          @add-to-playlist="openAddToPlaylist"
          @report="openReport"
        />
      </div>

      <!-- Saved playlists -->
      <div v-if="activeTab === 'saved'" class="space-y-3">
        <div v-if="saved.length === 0" class="py-10">
          <UiEmptyState :title="t('library.savedEmptyT')" :description="t('library.savedEmptyD')" />
        </div>
        <div v-else class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          <UiCard v-for="pl in saved" :key="pl.id" class="group">
            <RouterLink :to="localePath({ name: 'playlist-detail', params: { id: pl.id } })" class="block">
              <CoverMosaic :seeds="[pl.id]" :title="pl.title" size="md" />
              <span class="mt-2 block truncate text-sm font-bold text-white group-hover:text-brand-400">{{ pl.title }}</span>
            </RouterLink>
            <button
              type="button"
              class="mt-1 text-caption text-text-muted hover:text-accent-crimson hover:underline"
              @click="unsavePlaylist(pl.id)"
            >
              {{ t('library.unsave') }}
            </button>
          </UiCard>
        </div>
      </div>

      <!-- Follows -->
      <div v-if="activeTab === 'follows'" class="space-y-6">
        <div v-if="follows.length === 0" class="py-10">
          <UiEmptyState :title="t('library.followsEmptyT')" :description="t('library.followsEmptyD')" />
        </div>
        <section v-if="followCreators.length > 0">
          <h3 class="mb-2 text-sm font-bold text-white">{{ t('library.followsCreatorsT') }}</h3>
          <ul class="space-y-1">
            <li
              v-for="f in followCreators"
              :key="f.targetId"
              class="flex items-center justify-between gap-2 rounded-xl border border-surface-border bg-surface-1 px-3 py-2"
            >
              <span class="truncate text-xs font-bold text-white">{{ f.displayName }}</span>
              <button
                type="button"
                class="shrink-0 text-caption text-text-muted hover:text-accent-crimson hover:underline"
                @click="unfollow('creator', f.targetId)"
              >
                {{ t('library.unfollow') }}
              </button>
            </li>
          </ul>
        </section>
        <section v-if="followPlaylists.length > 0">
          <h3 class="mb-2 text-sm font-bold text-white">{{ t('library.followsPlaylistsT') }}</h3>
          <ul class="space-y-1">
            <li
              v-for="f in followPlaylists"
              :key="f.targetId"
              class="flex items-center justify-between gap-2 rounded-xl border border-surface-border bg-surface-1 px-3 py-2"
            >
              <RouterLink
                :to="localePath({ name: 'playlist-detail', params: { id: f.targetId } })"
                class="truncate text-xs font-bold text-white hover:text-brand-400"
              >
                {{ f.displayName }}
              </RouterLink>
              <button
                type="button"
                class="shrink-0 text-caption text-text-muted hover:text-accent-crimson hover:underline"
                @click="unfollow('playlist', f.targetId)"
              >
                {{ t('library.unfollow') }}
              </button>
            </li>
          </ul>
        </section>
      </div>

      <!-- My playlists link card -->
      <RouterLink
        :to="localePath('/me/playlists')"
        class="flex items-center justify-between gap-2 rounded-xl border border-surface-border bg-surface-1 px-4 py-3 transition-colors hover:border-brand-500/50"
      >
        <span class="text-sm font-bold text-white">{{ t('library.myPlCard', { count: myCount }) }}</span>
        <span class="text-brand-400" aria-hidden="true">→</span>
      </RouterLink>
    </template>

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
defineI18nRoute(false);
import { computed, onMounted, ref } from 'vue';
import { engagementApi, playlistApi, playbackApi } from '@/api/endpoints';
import { usePlayerStore } from '@/stores/playerStore';
import { usePlaylistStore } from '@/stores/playlistStore';
import { useToast } from '@/composables/useToast';
import { useApiError } from '@/composables/useApiError';
import type { Track, Playlist, Follow } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';
import UiPlaylistPicker from '@/components/ui/UiPlaylistPicker.vue';
import CoverMosaic from '@/components/ui/CoverMosaic.vue';
import TrackRow from '@/components/TrackRow.vue';
import DisputeReportModal from '@/components/legal/DisputeReportModal.vue';

const { t } = useI18n();
const localePath = useLocalePath();
const player = usePlayerStore();
const playlists = usePlaylistStore();
const toast = useToast();
const apiError = useApiError();

// NOTE: tab labelKeys resolve dynamically (invisible to check-i18n usage
// scan — they are used, do not delete).
const tabs = [
  { id: 'liked', labelKey: 'library.tabLiked' },
  { id: 'recent', labelKey: 'library.tabRecent' },
  { id: 'saved', labelKey: 'library.tabSaved' },
  { id: 'follows', labelKey: 'library.tabFollows' },
] as const;
const activeTab = ref<(typeof tabs)[number]['id']>('liked');

const liked = ref<Track[]>([]);
const recent = ref<Track[]>([]);
const saved = ref<Playlist[]>([]);
const follows = ref<Follow[]>([]);
const myCount = ref(0);
const loading = ref(true);
const error = ref<string | null>(null);

const reportTarget = ref<Track | null>(null);
const playlistTargetTrack = ref<Track | null>(null);
const creatingPlaylist = ref(false);

const followCreators = computed(() => follows.value.filter((f) => f.targetType === 'creator'));
const followPlaylists = computed(() => follows.value.filter((f) => f.targetType === 'playlist'));

const canPlayAll = computed(
  () => (activeTab.value === 'liked' && liked.value.length > 0) ||
    (activeTab.value === 'recent' && recent.value.length > 0),
);

async function load(): Promise<void> {
  loading.value = true;
  error.value = null;
  try {
    const [likedRes, recentRes, savedRes, followsRes, mineRes] = await Promise.all([
      engagementApi.likedTracks(50),
      playbackApi.recent(20),
      engagementApi.savedPlaylists(50),
      engagementApi.follows(undefined, 100),
      playlistApi.listMine(),
    ]);
    liked.value = likedRes ?? [];
    recent.value = recentRes ?? [];
    saved.value = savedRes ?? [];
    follows.value = followsRes ?? [];
    myCount.value = (mineRes ?? []).length;
  } catch (err: unknown) {
    error.value = apiError(err);
  } finally {
    loading.value = false;
  }
}

function playOne(tr: Track): void {
  player.setQueue([tr], tr.id);
}

function playAll(): void {
  const list = activeTab.value === 'liked' ? liked.value : recent.value;
  if (list.length === 0) return;
  player.setQueue(list, list[0]?.id);
}

function enqueue(tr: Track): void {
  player.enqueue(tr);
  toast.info(t('track.toastQueued'));
}

function openAddToPlaylist(tr: Track): void {
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
    myCount.value += 1;
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
  reportTarget.value = tr;
}

function onReported(): void {
  reportTarget.value = null;
  toast.success(t('track.toastReported'));
}

async function unsavePlaylist(id: string): Promise<void> {
  try {
    await playlistApi.unsave(id);
    saved.value = saved.value.filter((p) => p.id !== id);
    toast.info(t('library.toastUnfollowed'));
  } catch (err: unknown) {
    toast.error(apiError(err));
  }
}

async function unfollow(targetType: 'creator' | 'playlist', targetId: string): Promise<void> {
  try {
    await engagementApi.unfollow(targetType, targetId);
    follows.value = follows.value.filter((f) => !(f.targetType === targetType && f.targetId === targetId));
    toast.info(t('library.toastUnfollowed'));
  } catch (err: unknown) {
    toast.error(apiError(err));
  }
}

onMounted(() => {
  void load();
  void playlists.fetchMine();
});
</script>
