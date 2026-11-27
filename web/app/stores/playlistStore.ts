import { defineStore } from 'pinia';
import { ref } from 'vue';
import { tokenStore } from '@/api/client';
import { playlistApi } from '@/api/endpoints';
import { trackEvent } from '@/utils/analytics';
import type { Playlist, PlaylistTrack, Track } from '@/api/types';

export interface PlaylistPatch {
  title: string;
  isPublic: boolean;
  /** undefined = unchanged, '' = clear. */
  description?: string;
}

export interface AddResult {
  added: boolean;
  duplicate: boolean;
}

/**
 * The listener's own playlists plus per-playlist entry caches.
 *
 * Every mutation updates local state in place (count, runtime, cover seeds,
 * entry order) so lists and detail pages reflect it without a reload, and
 * bumps `revision[id]` so any open view of that playlist can refetch.
 */
export const usePlaylistStore = defineStore('playlist', () => {
  const playlists = ref<Playlist[]>([]);
  const entries = ref<Record<string, PlaylistTrack[]>>({});
  const revision = ref<Record<string, number>>({});
  const loading = ref(false);
  const error = ref<string | null>(null);

  function bump(id: string): void {
    revision.value = { ...revision.value, [id]: (revision.value[id] ?? 0) + 1 };
  }

  function upsert(p: Playlist): void {
    const idx = playlists.value.findIndex((x) => x.id === p.id);
    if (idx >= 0) playlists.value[idx] = { ...playlists.value[idx], ...p };
    else playlists.value.unshift(p);
  }

  function patchStats(id: string, fn: (p: Playlist) => Playlist): void {
    const idx = playlists.value.findIndex((x) => x.id === id);
    const cur = playlists.value[idx];
    if (idx >= 0 && cur) playlists.value[idx] = fn(cur);
  }

  /** Recomputes list stats from a fully loaded entry list (READY only). */
  function syncStatsFromEntries(id: string): void {
    const list = entries.value[id];
    if (!list) return;
    const ready = list.filter((e) => e.track?.status === 'READY');
    patchStats(id, (p) => ({
      ...p,
      trackCount: ready.length,
      totalDurationMs: ready.reduce((a, e) => a + (e.track?.durationMs ?? 0), 0),
      coverSeeds: ready.slice(0, 4).map((e) => e.trackId),
    }));
  }

  async function fetchMine(opts?: { containsTrack?: string }): Promise<Playlist[]> {
    // Public pages mount this store for guests: never fire the authed call
    // without a session (it only produces a 401 + console noise).
    if (!tokenStore.getAccess()) {
      playlists.value = [];
      error.value = null;
      loading.value = false;
      return [];
    }
    loading.value = true;
    error.value = null;
    try {
      // The API serializes an empty list as `null`; normalize so views can iterate.
      playlists.value = (await playlistApi.listMine(opts)) ?? [];
    } catch (err: unknown) {
      error.value = err instanceof Error ? err.message : 'failed';
    } finally {
      loading.value = false;
    }
    return playlists.value;
  }

  async function create(title: string, isPublic: boolean, description?: string): Promise<Playlist> {
    const p = await playlistApi.create(title, isPublic, description);
    upsert({ trackCount: 0, totalDurationMs: 0, coverSeeds: [], ...p });
    trackEvent({ name: 'playlist_created' });
    return p;
  }

  async function update(id: string, patch: PlaylistPatch): Promise<Playlist> {
    const p = await playlistApi.update(id, patch.title, patch.isPublic, patch.description);
    upsert(p);
    bump(id);
    return p;
  }

  async function remove(id: string): Promise<void> {
    await playlistApi.remove(id);
    playlists.value = playlists.value.filter((p) => p.id !== id);
    const { [id]: _dropped, ...rest } = entries.value;
    entries.value = rest;
    bump(id);
  }

  /** Loads ordered entries (each with its joined track) in one request. */
  async function fetchTracks(playlistId: string): Promise<PlaylistTrack[]> {
    const list = (await playlistApi.listTracks(playlistId)) ?? [];
    entries.value = { ...entries.value, [playlistId]: list };
    return list;
  }

  async function addTrack(playlistId: string, track: Track): Promise<AddResult> {
    const res = await playlistApi.addTrack(playlistId, track.id);
    const added = res.added !== false;
    if (added) {
      patchStats(playlistId, (p) => ({
        ...p,
        trackCount: (p.trackCount ?? 0) + 1,
        totalDurationMs: (p.totalDurationMs ?? 0) + (track.durationMs ?? 0),
        coverSeeds: (p.coverSeeds ?? []).length < 4 ? [...(p.coverSeeds ?? []), track.id] : p.coverSeeds,
        containsTrack: true,
      }));
      const cached = entries.value[playlistId];
      if (cached) {
        const position = cached.length ? Math.max(...cached.map((e) => e.position)) + 1 : 0;
        entries.value = {
          ...entries.value,
          [playlistId]: [
            ...cached,
            { id: `local-${track.id}`, playlistId, trackId: track.id, position, addedAt: new Date().toISOString(), track },
          ],
        };
      }
      bump(playlistId);
    } else {
      patchStats(playlistId, (p) => ({ ...p, containsTrack: true }));
    }
    return { added, duplicate: !added };
  }

  async function removeTrack(playlistId: string, trackId: string): Promise<void> {
    await playlistApi.removeTrack(playlistId, trackId);
    const cached = entries.value[playlistId];
    if (cached) {
      entries.value = { ...entries.value, [playlistId]: cached.filter((e) => e.trackId !== trackId) };
      syncStatsFromEntries(playlistId);
    } else {
      patchStats(playlistId, (p) => ({ ...p, trackCount: Math.max(0, (p.trackCount ?? 1) - 1) }));
    }
    bump(playlistId);
  }

  /**
   * Optimistic reorder: the cached order changes immediately and is restored
   * if the server rejects it. trackIds must name every entry.
   */
  async function reorder(playlistId: string, trackIds: string[]): Promise<void> {
    const before = entries.value[playlistId];
    if (before) {
      const byId = new Map(before.map((e) => [e.trackId, e]));
      const next = trackIds
        .map((id, i) => {
          const e = byId.get(id);
          return e ? { ...e, position: i } : null;
        })
        .filter((e): e is PlaylistTrack => e !== null);
      entries.value = { ...entries.value, [playlistId]: next };
      syncStatsFromEntries(playlistId);
    }
    try {
      await playlistApi.reorder(playlistId, trackIds);
    } catch (err) {
      if (before) {
        entries.value = { ...entries.value, [playlistId]: before };
        syncStatsFromEntries(playlistId);
      }
      throw err;
    }
    bump(playlistId);
  }

  return {
    playlists,
    entries,
    revision,
    loading,
    error,
    fetchMine,
    create,
    update,
    remove,
    fetchTracks,
    addTrack,
    removeTrack,
    reorder,
    upsert,
  };
});
