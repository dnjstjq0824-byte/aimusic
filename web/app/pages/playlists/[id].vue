<template>
  <div class="space-y-6">
    <UiErrorBanner
      :message="error"
      :title="t('playlist.errTitle')"
      :retry-label="t('track.retry')"
      @retry="load"
    />

    <div v-if="loading" class="space-y-4">
      <UiSkeleton height="h-44" width="w-full" rounded="xl" />
      <UiSkeleton height="h-64" width="w-full" rounded="xl" />
    </div>

    <template v-else-if="!error && playlist">
      <!-- Header -->
      <section class="flex flex-col gap-4 sm:flex-row sm:items-end">
        <CoverMosaic :seeds="seeds" :title="playlist.title" size="lg" />
        <div class="min-w-0 flex-1">
          <p class="text-caption font-bold uppercase tracking-wider text-text-muted">
            {{ playlist.isPublic ? t('playlist.detailPublic') : t('playlist.detailPrivate') }}
          </p>
          <h1 class="font-display mt-1 break-words text-2xl font-black tracking-tight text-white sm:text-3xl">
            {{ playlist.title }}
          </h1>
          <p class="mt-1 text-xs text-text-secondary">
            <span v-if="playlist.ownerName">{{ playlist.ownerName }} · </span>{{ metaLine }}
          </p>
          <div class="mt-3 flex flex-wrap items-center gap-2">
            <UiButton variant="brand" size="sm" :disabled="tracks.length === 0" @click="playAll(false)">
              {{ t('playlist.playAll') }}
            </UiButton>
            <UiButton variant="secondary" size="sm" :disabled="tracks.length === 0" @click="playAll(true)">
              {{ t('playlist.shuffle') }}
            </UiButton>
            <UiButton v-if="ownerView" variant="outline" size="sm" @click="editOpen = true">
              {{ t('playlist.editInfo') }}
            </UiButton>
            <UiButton v-else variant="outline" size="sm" @click="toggleSave">
              {{ saved ? t('playlist.unsave') : t('playlist.save') }}
            </UiButton>
            <UiButton variant="outline" size="sm" @click="sharePlaylist">
              {{ t('playlist.share') }}
            </UiButton>
            <UiMenu
              v-if="!ownerView"
              :items="followMenu"
              :label="t('track.moreActions')"
              @select="toggleFollow"
            />
          </div>
        </div>
      </section>

      <!-- Tracks -->
      <section :aria-label="t('playlist.tracksTitle')">
        <div v-if="tracks.length === 0" class="py-10">
          <UiEmptyState :title="t('playlist.trackEmptyT')" :description="t('playlist.trackEmptyD')">
            <template v-if="isOwner" #action>
              <RouterLink :to="localePath('/explore')">
                <UiButton variant="secondary" size="sm">{{ t('playlist.trackEmptyCta') }}</UiButton>
              </RouterLink>
            </template>
          </UiEmptyState>
        </div>
        <ol v-else class="space-y-1">
          <li
            v-for="(tr, i) in tracks"
            :key="tr.id"
            class="flex items-center gap-2 rounded-xl border border-transparent px-2 py-1 transition-colors hover:border-surface-border hover:bg-surface-1"
            @dragover.prevent="isOwner && onDragOver()"
            @drop="isOwner && onDrop(i)"
          >
            <span
              v-if="ownerView"
              class="cursor-grab touch-none select-none text-text-muted"
              draggable="true"
              :aria-label="t('playlist.tracksTitle')"
              @dragstart="onDragStart(i)"
            >
              <svg class="h-4 w-4" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                <circle cx="9" cy="6" r="1.4" /><circle cx="15" cy="6" r="1.4" />
                <circle cx="9" cy="12" r="1.4" /><circle cx="15" cy="12" r="1.4" />
                <circle cx="9" cy="18" r="1.4" /><circle cx="15" cy="18" r="1.4" />
              </svg>
            </span>
            <span v-else class="w-6 shrink-0 text-center font-mono text-xs text-text-muted">{{ i + 1 }}</span>
            <div class="min-w-0 flex-1">
              <TrackRow
                :track="tr"
                @play="playFrom"
                @enqueue="enqueue"
                @add-to-playlist="openAddToPlaylist"
                @report="openReport"
              />
            </div>
            <UiMenu
              v-if="ownerView"
              :items="rowMenu"
              :label="t('track.moreActions')"
              @select="(idx) => onRowMenu(tr, i, idx)"
            />
          </li>
        </ol>
      </section>
    </template>

    <UiModal v-if="isOwner" v-model="editOpen" :title="t('playlist.editInfo')" max-width="sm">
      <div class="space-y-4 p-5">
        <UiInput v-model="editTitle" :label="t('playlist.editTitle')" required />
        <UiCheckbox v-model="editPublic">{{ t('playlist.editVisibility') }}</UiCheckbox>
        <div class="flex gap-2">
          <UiButton variant="secondary" size="md" class="flex-1" @click="editOpen = false">
            {{ t('playlist.editCancel') }}
          </UiButton>
          <UiButton variant="brand" size="md" class="flex-1" :loading="busy" :disabled="!editTitle.trim()" @click="submitEdit">
            {{ t('playlist.editSave') }}
          </UiButton>
        </div>
      </div>
    </UiModal>

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
import { computed, onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { playlistApi, trackApi, engagementApi } from '@/api/endpoints';
import { useAuthStore } from '@/stores/authStore';
import { usePlayerStore } from '@/stores/playerStore';
import { usePlaylistStore } from '@/stores/playlistStore';
import { useToast } from '@/composables/useToast';
import { useApiError } from '@/composables/useApiError';
import { useFormat } from '@/composables/useFormat';
import type { Track, Playlist } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiCheckbox from '@/components/ui/UiCheckbox.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiInput from '@/components/ui/UiInput.vue';
import UiMenu, { type MenuItem } from '@/components/ui/UiMenu.vue';
import UiModal from '@/components/ui/UiModal.vue';
import UiPlaylistPicker from '@/components/ui/UiPlaylistPicker.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import CoverMosaic from '@/components/ui/CoverMosaic.vue';
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
const fmt = useFormat();

const playlist = ref<Playlist | null>(null);
const tracks = ref<Track[]>([]);
const seeds = ref<string[]>([]);
const loading = ref(true);
const busy = ref(false);
const error = ref<string | null>(null);
const saved = ref(false);
const following = ref(false);

const editOpen = ref(false);
const editTitle = ref('');
const editPublic = ref(false);
const reportTarget = ref<Track | null>(null);
const playlistTargetTrack = ref<Track | null>(null);
const creatingPlaylist = ref(false);
const dragFrom = ref<number | null>(null);

const isOwner = computed(
  () => auth.isAuthenticated && !!playlist.value && playlist.value.ownerUserId === auth.userId,
);
// SSR-safe: owner controls resolve after mount (SSR always renders guest view).
const ready = ref(false);
const ownerView = computed(() => ready.value && isOwner.value);

const metaLine = computed(() => {
  const count = fmt.number(tracks.value.length);
  const totalMs = tracks.value.reduce((a, tr) => a + (tr.durationMs ?? 0), 0);
  const mins = Math.floor(totalMs / 60000);
  if (mins >= 60) {
    return t('playlist.metaHM', { count, h: fmt.number(Math.floor(mins / 60)), m: fmt.number(mins % 60) });
  }
  return t('playlist.metaM', { count, m: fmt.number(mins) });
});

const rowMenu = computed<MenuItem[]>(() => [
  { label: t('playlist.trackUp') },
  { label: t('playlist.trackDown') },
  { label: t('playlist.trackRemove'), danger: true },
]);

const followMenu = computed<MenuItem[]>(() => [
  { label: following.value ? t('library.unfollow') : t('library.follow') },
]);

function onRowMenu(tr: Track, index: number, menuIndex: number): void {
  if (menuIndex === 0) void moveTrack(index, -1);
  else if (menuIndex === 1) void moveTrack(index, 1);
  else void removeTrack(tr.id);
}

async function load(): Promise<void> {
  const id = route.params.id as string;
  if (!id) return;
  loading.value = true;
  error.value = null;
  try {
    const [pl, entries] = await Promise.all([
      playlistApi.get(id),
      playlistApi.listTracks(id),
    ]);
    playlist.value = pl;
    editTitle.value = pl.title;
    editPublic.value = pl.isPublic;
    const resolved: Track[] = [];
    for (const entry of entries ?? []) {
      try {
        resolved.push(await trackApi.get(entry.trackId));
      } catch {
        // Skip entries whose track vanished.
      }
    }
    tracks.value = resolved;
    seeds.value = resolved.slice(0, 4).map((x) => x.id);
    if (auth.isAuthenticated && !isOwner.value) {
      try {
        saved.value = (await playlistApi.saveState(id)).saved;
        const mine = await engagementApi.follows('playlist', 100);
        following.value = (mine ?? []).some((f) => f.targetId === id);
      } catch {
        saved.value = false;
        following.value = false;
      }
    }
  } catch (err: unknown) {
    error.value = apiError(err);
  } finally {
    loading.value = false;
  }
}

function playAll(shuffle: boolean): void {
  if (tracks.value.length === 0) return;
  const list = [...tracks.value];
  if (shuffle) {
    for (let i = list.length - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1));
      [list[i], list[j]] = [list[j]!, list[i]!];
    }
  }
  player.setQueue(list, list[0]?.id);
}

function playFrom(tr: Track): void {
  const idx = tracks.value.findIndex((x) => x.id === tr.id);
  player.setQueue(tracks.value, tr.id);
  if (idx < 0) player.setQueue([tr], tr.id);
}

function enqueue(tr: Track): void {
  player.enqueue(tr);
  toast.info(t('track.toastQueued'));
}

function requireLogin(): boolean {
  if (auth.isAuthenticated) return true;
  toast.info(t('playlist.toastNeedLogin'));
  void router.push(localePath({ name: 'login', query: { redirect: route.fullPath } }));
  return false;
}

async function toggleSave(): Promise<void> {
  if (!requireLogin() || !playlist.value) return;
  try {
    if (saved.value) {
      await playlistApi.unsave(playlist.value.id);
      saved.value = false;
      toast.info(t('playlist.toastUnsaved'));
    } else {
      await playlistApi.save(playlist.value.id);
      saved.value = true;
      toast.success(t('playlist.toastSaved'));
    }
  } catch (err: unknown) {
    toast.error(apiError(err));
  }
}

async function toggleFollow(): Promise<void> {
  if (!requireLogin() || !playlist.value) return;
  try {
    if (following.value) {
      await engagementApi.unfollow('playlist', playlist.value.id);
      following.value = false;
      toast.info(t('library.toastUnfollowed'));
    } else {
      await engagementApi.follow('playlist', playlist.value.id);
      following.value = true;
      toast.success(t('library.toastFollowed'));
    }
  } catch (err: unknown) {
    toast.error(apiError(err));
  }
}

async function sharePlaylist(): Promise<void> {
  try {
    await navigator.clipboard.writeText(window.location.href);
    toast.success(t('playlist.toastShared'));
  } catch {
    toast.error(t('track.toastCopyFailed'));
  }
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

async function submitEdit(): Promise<void> {
  if (!playlist.value) return;
  const title = editTitle.value.trim();
  if (!title) return;
  busy.value = true;
  try {
    playlist.value = await playlistApi.update(playlist.value.id, title, editPublic.value);
    toast.success(t('playlist.toastUpdated'));
    editOpen.value = false;
  } catch (err: unknown) {
    toast.error(apiError(err));
  } finally {
    busy.value = false;
  }
}

async function removeTrack(trackId: string): Promise<void> {
  if (!playlist.value) return;
  try {
    await playlistApi.removeTrack(playlist.value.id, trackId);
    tracks.value = tracks.value.filter((x) => x.id !== trackId);
    toast.success(t('playlist.toastRemoved'));
  } catch (err: unknown) {
    toast.error(apiError(err));
  }
}

async function moveTrack(index: number, delta: -1 | 1): Promise<void> {
  if (!playlist.value) return;
  const to = index + delta;
  if (to < 0 || to >= tracks.value.length) return;
  const next = [...tracks.value];
  const [moved] = next.splice(index, 1);
  if (!moved) return;
  next.splice(to, 0, moved);
  tracks.value = next;
  await persistOrder();
}

function onDragStart(index: number): void {
  dragFrom.value = index;
}

function onDragOver(): void {
  // Required to allow drop; reorder commits on drop.
}

async function onDrop(index: number): Promise<void> {
  const from = dragFrom.value;
  dragFrom.value = null;
  if (from == null || from === index || !playlist.value) return;
  const next = [...tracks.value];
  const [moved] = next.splice(from, 1);
  if (!moved) return;
  next.splice(index, 0, moved);
  tracks.value = next;
  await persistOrder();
}

async function persistOrder(): Promise<void> {
  if (!playlist.value) return;
  try {
    await playlistApi.reorder(playlist.value.id, tracks.value.map((x) => x.id));
    toast.success(t('playlist.toastReordered'));
  } catch (err: unknown) {
    toast.error(apiError(err));
  }
}

onMounted(() => {
  ready.value = true;
  void load();
  if (auth.isAuthenticated) {
    void playlists.fetchMine();
  }
});
</script>
