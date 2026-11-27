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
            {{ playlist.isPublic || playlist.isOfficialCuration ? t('playlist.detailPublic') : t('playlist.detailPrivate') }}
          </p>
          <h1 class="font-display mt-1 break-words text-2xl font-medium tracking-[-0.03em] text-white sm:text-3xl">
            {{ playlist.title }}
          </h1>
          <p v-if="playlist.description" class="mt-2 whitespace-pre-line break-words text-sm text-text-secondary">
            {{ playlist.description }}
          </p>
          <p class="mt-1 text-xs text-text-secondary" role="status">
            <span v-if="playlist.ownerName">{{ playlist.ownerName }} · </span>{{ metaLine }}
          </p>
          <div class="mt-3 flex flex-wrap items-center gap-2">
            <UiButton variant="brand" size="sm" :disabled="playable.length === 0" @click="playAll(false)">
              {{ t('playlist.playAll') }}
            </UiButton>
            <UiButton variant="secondary" size="sm" :disabled="playable.length === 0" @click="playAll(true)">
              {{ t('playlist.shuffle') }}
            </UiButton>
            <UiButton v-if="ownerView" variant="outline" size="sm" @click="openEdit">
              {{ t('playlist.editInfo') }}
            </UiButton>
            <UiButton v-else variant="outline" size="sm" :aria-pressed="saved" @click="toggleSave">
              {{ saved ? t('playlist.unsave') : t('playlist.save') }}
            </UiButton>
            <UiButton v-if="isPublicView" variant="outline" size="sm" @click="sharePlaylist">
              {{ t('playlist.share') }}
            </UiButton>
            <UiMenu
              v-if="ownerView"
              :items="ownerMenu"
              :label="t('track.moreActions')"
              @select="onOwnerMenu"
            />
            <UiMenu
              v-else
              :items="followMenu"
              :label="t('track.moreActions')"
              @select="toggleFollow"
            />
          </div>
          <div v-if="isPublicView" class="mt-4 border-t border-surface-border pt-4">
            <ShareButtons :url="shareUrl" :title="playlist?.title ?? ''" />
          </div>
        </div>
      </section>

      <!-- Tracks -->
      <section :aria-label="t('playlist.tracksTitle')">
        <div v-if="rows.length === 0" class="py-10">
          <UiEmptyState :title="t('playlist.trackEmptyT')" :description="t('playlist.trackEmptyD')">
            <template v-if="ownerView" #action>
              <RouterLink :to="localePath('/explore')">
                <UiButton variant="secondary" size="sm">{{ t('playlist.trackEmptyCta') }}</UiButton>
              </RouterLink>
            </template>
          </UiEmptyState>
        </div>
        <ol v-else class="space-y-1">
          <li
            v-for="(row, i) in rows"
            :key="row.trackId"
            class="flex items-center gap-2 rounded-xl border px-2 py-1 transition-colors"
            :class="[
              dragOver === i && dragFrom !== null && dragFrom !== i ? 'border-brand-500/60 bg-brand-950/30' : 'border-transparent hover:border-surface-border hover:bg-surface-1',
              dragFrom === i ? 'opacity-50' : '',
            ]"
            @dragover.prevent="ownerView && (dragOver = i)"
            @dragleave="dragOver = dragOver === i ? null : dragOver"
            @drop.prevent="ownerView && onDrop(i)"
          >
            <span
              v-if="ownerView"
              class="flex cursor-grab touch-none select-none items-center text-text-muted active:cursor-grabbing"
              draggable="true"
              :aria-label="t('playlist.dragHandle', { title: row.track?.title ?? '' })"
              @dragstart="onDragStart(i, $event)"
              @dragend="dragFrom = null; dragOver = null"
            >
              <svg class="h-4 w-4" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                <circle cx="9" cy="6" r="1.4" /><circle cx="15" cy="6" r="1.4" />
                <circle cx="9" cy="12" r="1.4" /><circle cx="15" cy="12" r="1.4" />
                <circle cx="9" cy="18" r="1.4" /><circle cx="15" cy="18" r="1.4" />
              </svg>
            </span>
            <span class="w-6 shrink-0 text-center font-mono text-xs text-text-muted">{{ i + 1 }}</span>
            <div class="min-w-0 flex-1" :class="row.playable ? '' : 'opacity-50'">
              <TrackRow v-if="row.track" :track="row.track" @play="playFrom" />
            </div>
            <UiBadge v-if="!row.playable" variant="warning" size="xs">{{ t('playlist.unavailable') }}</UiBadge>
            <div v-if="ownerView" class="hidden shrink-0 items-center gap-0.5 sm:flex">
              <button
                type="button"
                class="rounded p-1 text-text-muted hover:text-white disabled:opacity-30"
                :disabled="i === 0 || busyOrder"
                :aria-label="t('playlist.trackUpAria', { title: row.track?.title ?? '' })"
                @click="moveTrack(i, -1)"
              >
                <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="m18 15-6-6-6 6" /></svg>
              </button>
              <button
                type="button"
                class="rounded p-1 text-text-muted hover:text-white disabled:opacity-30"
                :disabled="i === rows.length - 1 || busyOrder"
                :aria-label="t('playlist.trackDownAria', { title: row.track?.title ?? '' })"
                @click="moveTrack(i, 1)"
              >
                <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="m6 9 6 6 6-6" /></svg>
              </button>
            </div>
            <UiMenu
              v-if="ownerView"
              :items="rowMenu(i)"
              :label="t('track.moreActions')"
              @select="(idx) => onRowMenu(row.trackId, i, idx)"
            />
          </li>
        </ol>
      </section>
    </template>

    <UiModal v-if="ownerView" v-model="editOpen" :title="t('playlist.editInfo')" max-width="sm">
      <form class="space-y-4 p-5" @submit.prevent="submitEdit">
        <UiInput v-model="editTitle" :label="t('playlist.editTitle')" :maxlength="100" required />
        <UiTextarea
          v-model="editDescription"
          :label="t('playlist.editDescription')"
          :placeholder="t('playlist.editDescriptionPh')"
          :maxlength="500"
          :rows="3"
        />
        <UiCheckbox v-model="editPublic">{{ t('playlist.editVisibility') }}</UiCheckbox>
        <div class="flex gap-2">
          <UiButton variant="secondary" size="md" class="flex-1" type="button" @click="editOpen = false">
            {{ t('playlist.editCancel') }}
          </UiButton>
          <UiButton variant="brand" size="md" class="flex-1" type="submit" :loading="busy" :disabled="!editTitle.trim()">
            {{ t('playlist.editSave') }}
          </UiButton>
        </div>
      </form>
    </UiModal>

    <UiConfirmDialog
      v-if="ownerView"
      v-model="deleteOpen"
      :title="t('playlist.delTitle')"
      :body="t('playlist.delBody')"
      :confirm-label="t('playlist.delConfirm')"
      danger
      :loading="busy"
      @confirm="submitDelete"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { playlistApi, engagementApi } from '@/api/endpoints';
import { useAuthStore } from '@/stores/authStore';
import { usePlayerStore } from '@/stores/playerStore';
import { usePlaylistStore } from '@/stores/playlistStore';
import { useToast } from '@/composables/useToast';
import { useApiError } from '@/composables/useApiError';
import { useFormat } from '@/composables/useFormat';
import { usePublicSeo } from '@/composables/usePublicSeo';
import { fetchPublic } from '@/utils/publicApi';
import {
  breadcrumbLd,
  canonicalFor,
  canonicalPlaylistPath,
  describePlaylist,
  musicPlaylistLd,
  needsCjkThaiFont,
  oembedDiscoveryUrl,
  playlistTitle,
} from '@/utils/seo';
import { trackEvent } from '@/utils/analytics';
import type { Track, Playlist, PlaylistTrack } from '@/api/types';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiButton from '@/components/ui/UiButton.vue';
import UiCheckbox from '@/components/ui/UiCheckbox.vue';
import UiConfirmDialog from '@/components/ui/UiConfirmDialog.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiInput from '@/components/ui/UiInput.vue';
import UiMenu, { type MenuItem } from '@/components/ui/UiMenu.vue';
import UiModal from '@/components/ui/UiModal.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import UiTextarea from '@/components/ui/UiTextarea.vue';
import CoverMosaic from '@/components/ui/CoverMosaic.vue';
import TrackRow from '@/components/TrackRow.vue';
import ShareButtons from '@/components/seo/ShareButtons.vue';

const { t, locale } = useI18n();
const localePath = useLocalePath();
const route = useRoute();
const router = useRouter();
const auth = useAuthStore();
const player = usePlayerStore();
const store = usePlaylistStore();
const toast = useToast();
const apiError = useApiError();
const fmt = useFormat();

const playlist = ref<Playlist | null>(null);
const clientLoading = ref(false);
const busy = ref(false);
const busyOrder = ref(false);
const error = ref<string | null>(null);
const saved = ref(false);
const following = ref(false);

const editOpen = ref(false);
const editTitle = ref('');
const editDescription = ref('');
const editPublic = ref(false);
const deleteOpen = ref(false);
const dragFrom = ref<number | null>(null);
const dragOver = ref<number | null>(null);

const isOwner = computed(
  () => auth.isAuthenticated && !!playlist.value && playlist.value.ownerUserId === auth.userId,
);
// SSR-safe: owner controls resolve after mount (SSR always renders guest view).
const ready = ref(false);
const ownerView = computed(() => ready.value && isOwner.value);

const param = computed(() => String(route.params.id ?? ''));

// WI-S2: public playlists render server-side (guest view): the playlist and
// its ordered entries (each with the joined track) arrive in two requests,
// whatever the playlist length. Private lists 403 for guests and load
// client-side for the owner.
const { data: pageData, pending, error: pageError } = await useAsyncData(
  () => `playlist-${param.value}`,
  async () => {
    const enc = encodeURIComponent(param.value);
    const [pl, entries] = await Promise.all([
      fetchPublic<Playlist>(`/playlists/${enc}`),
      fetchPublic<PlaylistTrack[]>(`/playlists/${enc}/tracks`),
    ]);
    return { playlist: pl, entries: entries ?? [] };
  },
);
const loading = computed(() => pending.value || clientLoading.value);

/** Entries shown: the store's live copy (owner, after mount) or the page payload. */
const entries = computed<PlaylistTrack[]>(() => {
  const id = playlist.value?.id;
  const live = id ? store.entries[id] : undefined;
  return live ?? pageData.value?.entries ?? [];
});

interface Row {
  trackId: string;
  track: Track | undefined;
  playable: boolean;
}
const rows = computed<Row[]>(() =>
  entries.value
    .filter((e) => e.track)
    .map((e) => ({ trackId: e.trackId, track: e.track, playable: e.track?.status === 'READY' })),
);
const playable = computed<Track[]>(() =>
  rows.value.filter((r) => r.playable && r.track).map((r) => r.track as Track),
);
const seeds = computed(() => playable.value.slice(0, 4).map((x) => x.id));

const metaLine = computed(() => {
  const count = fmt.number(playable.value.length);
  const totalMs = playable.value.reduce((a, tr) => a + (tr.durationMs ?? 0), 0);
  const mins = Math.floor(totalMs / 60000);
  if (mins >= 60) {
    return t('playlist.metaHM', { count, h: fmt.number(Math.floor(mins / 60)), m: fmt.number(mins % 60) });
  }
  return t('playlist.metaM', { count, m: fmt.number(mins) });
});

function rowMenu(index: number): MenuItem[] {
  return [
    { label: t('playlist.trackUp'), disabled: index === 0 },
    { label: t('playlist.trackDown'), disabled: index === rows.value.length - 1 },
    { label: t('playlist.trackRemove'), danger: true },
  ];
}

const ownerMenu = computed<MenuItem[]>(() => [
  { label: t('playlist.meDelete'), danger: true },
]);

const followMenu = computed<MenuItem[]>(() => [
  { label: following.value ? t('library.unfollow') : t('library.follow') },
]);

function onRowMenu(trackId: string, index: number, menuIndex: number): void {
  if (menuIndex === 0) void moveTrack(index, -1);
  else if (menuIndex === 1) void moveTrack(index, 1);
  else void removeTrack(trackId);
}

function onOwnerMenu(): void {
  deleteOpen.value = true;
}

let needsClientLoad = false;
function syncPageData(): void {
  if (!pageData.value) return;
  playlist.value = pageData.value.playlist;
}
syncPageData();
if (pageError.value) {
  const st = (pageError.value as { statusCode?: number }).statusCode ?? 0;
  if (st === 404) {
    throw showError({ statusCode: 404, statusMessage: 'Playlist not found' });
  }
  // 403 (private) or transport failure: retry client-side with auth.
  needsClientLoad = true;
}
// WI-S1: legacy uuid / bare short ID / stale slug → canonical 301.
if (playlist.value?.slug && playlist.value?.shortId) {
  const canon = `${playlist.value.slug}-${playlist.value.shortId}`;
  if (param.value !== canon) {
    await navigateTo(localePath({ name: 'playlist-detail', params: { id: canon } }), { redirectCode: 301 });
  }
}
const pagePath = computed(() =>
  playlist.value?.slug && playlist.value?.shortId
    ? canonicalPlaylistPath(playlist.value.slug, playlist.value.shortId)
    : `/playlists/${param.value}`,
);
const shareUrl = computed(() => canonicalFor(pagePath.value, locale.value));
const isPublicView = computed(() => !!playlist.value && (playlist.value.isPublic || playlist.value.isOfficialCuration));
if (playlist.value && !pageError.value) {
  const pl = playlist.value;
  const pub = pl.isPublic || pl.isOfficialCuration;
  usePublicSeo({
    title: playlistTitle(t, pl.title, pl.ownerName ?? ''),
    description: describePlaylist(t, pl, playable.value.length),
    path: pagePath.value,
    locale: locale.value,
    ogType: 'music.playlist',
    image: null,
    noindex: !pub,
    oembedUrl: pub ? oembedDiscoveryUrl(shareUrl.value) : null,
  });
  if (pub) {
    useSchemaOrg([
      musicPlaylistLd({
        playlist: pl,
        pageUrl: shareUrl.value,
        tracks: playable.value.map((x) => ({
          title: x.title,
          url: canonicalFor(`/tracks/${x.slug && x.shortId ? `${x.slug}-${x.shortId}` : x.id}`, locale.value),
          durationMs: x.durationMs,
        })),
        trackCount: playable.value.length,
      }),
      breadcrumbLd([
        { name: 'AVIBE', url: canonicalFor('/', locale.value) },
        { name: pl.title, url: shareUrl.value },
      ]),
    ]);
    // Server only: OG tags are for crawlers; keeps nuxt-og-image out of client JS.
    if (import.meta.server) defineOgImageComponent('AvibeOg', {
      title: pl.title,
      creator: pl.ownerName ?? '',
      verified: false,
      plain: needsCjkThaiFont(`${pl.title} ${pl.ownerName ?? ''}`),
    });
  }
}

async function loadViewerState(pl: Playlist): Promise<void> {
  if (!auth.isAuthenticated || isOwner.value) return;
  try {
    const [s, mine] = await Promise.all([playlistApi.saveState(pl.id), engagementApi.follows('playlist', 100)]);
    saved.value = s.saved;
    following.value = (mine ?? []).some((f) => f.targetId === pl.id);
  } catch {
    saved.value = false;
    following.value = false;
  }
}

/** Authed reload: playlist + entries (owner also gets non-READY entries). */
async function load(): Promise<void> {
  const id = param.value;
  if (!id) return;
  clientLoading.value = true;
  error.value = null;
  try {
    const pl = await playlistApi.get(id);
    playlist.value = pl;
    const list = await store.fetchTracks(pl.id);
    pageData.value = { playlist: pl, entries: list };
    void loadViewerState(pl);
  } catch (err: unknown) {
    error.value = apiError(err);
  } finally {
    clientLoading.value = false;
  }
}

function playAll(shuffle: boolean): void {
  const list = [...playable.value];
  if (list.length === 0) return;
  if (shuffle) {
    for (let i = list.length - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1));
      [list[i], list[j]] = [list[j]!, list[i]!];
    }
  }
  if (player.shuffle) player.toggleShuffle();
  player.setQueue(list, list[0]?.id);
}

// Clicking row N loads the whole playlist as the queue starting at N, so
// N+1 plays when it ends.
function playFrom(tr: Track): void {
  if (!playable.value.some((x) => x.id === tr.id)) return;
  player.setQueue(playable.value, tr.id);
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
    await navigator.clipboard.writeText(shareUrl.value);
    trackEvent({ name: 'share_clicked', props: { channel: 'copy' } });
    toast.success(t('playlist.toastShared'));
  } catch {
    toast.error(t('track.toastCopyFailed'));
  }
}

function openEdit(): void {
  if (!playlist.value) return;
  editTitle.value = playlist.value.title;
  editDescription.value = playlist.value.description ?? '';
  editPublic.value = playlist.value.isPublic;
  editOpen.value = true;
}

async function submitEdit(): Promise<void> {
  if (!playlist.value) return;
  const title = editTitle.value.trim();
  if (!title) return;
  busy.value = true;
  try {
    playlist.value = await store.update(playlist.value.id, {
      title,
      isPublic: editPublic.value,
      description: editDescription.value.trim(),
    });
    if (pageData.value) pageData.value = { ...pageData.value, playlist: playlist.value };
    toast.success(t('playlist.toastUpdated'));
    editOpen.value = false;
  } catch (err: unknown) {
    toast.error(apiError(err));
  } finally {
    busy.value = false;
  }
}

async function submitDelete(): Promise<void> {
  if (!playlist.value) return;
  busy.value = true;
  try {
    await store.remove(playlist.value.id);
    toast.success(t('playlist.toastDeleted'));
    deleteOpen.value = false;
    await router.replace(localePath('/me/playlists'));
  } catch (err: unknown) {
    toast.error(apiError(err));
  } finally {
    busy.value = false;
  }
}

async function removeTrack(trackId: string): Promise<void> {
  if (!playlist.value) return;
  try {
    await store.removeTrack(playlist.value.id, trackId);
    toast.success(t('playlist.toastRemoved'));
  } catch (err: unknown) {
    toast.error(apiError(err));
  }
}

/** Persists a new order (every entry, playable or not) optimistically. */
async function commitOrder(next: string[]): Promise<void> {
  if (!playlist.value) return;
  busyOrder.value = true;
  try {
    await store.reorder(playlist.value.id, next);
    toast.success(t('playlist.toastReordered'));
  } catch (err: unknown) {
    toast.error(apiError(err));
  } finally {
    busyOrder.value = false;
  }
}

function moved(from: number, to: number): string[] | null {
  const ids = entries.value.map((e) => e.trackId);
  if (from < 0 || from >= ids.length || to < 0 || to >= ids.length || from === to) return null;
  const [m] = ids.splice(from, 1);
  if (!m) return null;
  ids.splice(to, 0, m);
  return ids;
}

async function moveTrack(index: number, delta: -1 | 1): Promise<void> {
  const next = moved(index, index + delta);
  if (next) await commitOrder(next);
}

function onDragStart(index: number, e: DragEvent): void {
  dragFrom.value = index;
  if (e.dataTransfer) {
    e.dataTransfer.effectAllowed = 'move';
    e.dataTransfer.setData('text/plain', String(index));
  }
}

async function onDrop(index: number): Promise<void> {
  const from = dragFrom.value;
  dragFrom.value = null;
  dragOver.value = null;
  if (from == null) return;
  const next = moved(from, index);
  if (next) await commitOrder(next);
}

// Another surface (global add-to-playlist picker) changed this playlist:
// the store already patched its entries; refresh header stats from it.
watch(
  () => (playlist.value ? store.revision[playlist.value.id] : undefined),
  () => {
    const id = playlist.value?.id;
    const fresh = id ? store.playlists.find((p) => p.id === id) : undefined;
    if (fresh && playlist.value) playlist.value = { ...playlist.value, ...fresh };
  },
);

onMounted(() => {
  ready.value = true;
  if (needsClientLoad) {
    // Private lists (403 guest) and SSR transport failures reload with auth.
    needsClientLoad = false;
    void load();
    return;
  }
  const pl = playlist.value;
  if (!pl || !auth.isAuthenticated) return;
  if (isOwner.value) {
    // Owner view: live store copy (includes in-review / taken-down entries
    // so a reorder always names every member).
    void store.fetchTracks(pl.id).catch((err: unknown) => {
      error.value = apiError(err);
    });
    void store.fetchMine();
  } else {
    void loadViewerState(pl);
  }
});
</script>
