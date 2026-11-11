import { defineStore } from 'pinia';
import { ref } from 'vue';
import { tokenStore } from '@/api/client';
import { playlistApi, trackApi } from '@/api/endpoints';
import type { Playlist, PlaylistTrack, Track } from '@/api/types';

export const usePlaylistStore = defineStore('playlist', () => {
  const playlists = ref<Playlist[]>([]);
  const entries = ref<Record<string, PlaylistTrack[]>>({});
  const entryTracks = ref<Record<string, Track[]>>({});
  const loading = ref(false);
  const error = ref<string | null>(null);

  async function fetchMine(): Promise<void> {
    // Public pages mount this store for guests: never fire the authed call
    // without a session (it only produces a 401 + console noise).
    if (!tokenStore.getAccess()) {
      playlists.value = [];
      error.value = null;
      loading.value = false;
      return;
    }
    loading.value = true;
    error.value = null;
    try {
      // The API serializes an empty list as `null`; normalize so views can iterate.
      playlists.value = (await playlistApi.listMine()) ?? [];
    } catch (err: unknown) {
      error.value = err instanceof Error ? err.message : 'failed';
    } finally {
      loading.value = false;
    }
  }

  async function create(title: string, isPublic: boolean): Promise<Playlist> {
    const p = await playlistApi.create(title, isPublic);
    playlists.value.unshift(p);
    return p;
  }

  async function remove(id: string): Promise<void> {
    await playlistApi.remove(id);
    playlists.value = playlists.value.filter((p) => p.id !== id);
    delete entries.value[id];
    delete entryTracks.value[id];
  }

  async function fetchTracks(playlistId: string): Promise<void> {
    const list = (await playlistApi.listTracks(playlistId)) ?? [];
    entries.value[playlistId] = list;
    const tracks: Track[] = [];
    for (const e of list) {
      try {
        tracks.push(await trackApi.get(e.trackId));
      } catch {
        // Skip missing/taken-down tracks in detail resolution.
      }
    }
    entryTracks.value[playlistId] = tracks;
  }

  async function addTrack(playlistId: string, trackId: string): Promise<void> {
    await playlistApi.addTrack(playlistId, trackId);
    await fetchTracks(playlistId);
  }

  async function removeTrack(playlistId: string, trackId: string): Promise<void> {
    await playlistApi.removeTrack(playlistId, trackId);
    await fetchTracks(playlistId);
  }

  async function reorder(playlistId: string, trackIds: string[]): Promise<void> {
    await playlistApi.reorder(playlistId, trackIds);
    await fetchTracks(playlistId);
  }

  return {
    playlists,
    entries,
    entryTracks,
    loading,
    error,
    fetchMine,
    create,
    remove,
    fetchTracks,
    addTrack,
    removeTrack,
    reorder,
  };
});
