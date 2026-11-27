import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { tokenStore } from '@/api/client';
import { usePlaylistStore } from '@/stores/playlistStore';
import type { Track } from '@/api/types';

const listMine = vi.fn();
const listTracks = vi.fn();
const trackGet = vi.fn();
const addTrack = vi.fn();
const removeTrack = vi.fn();
const reorder = vi.fn();

vi.mock('@/api/endpoints', () => ({
  playlistApi: {
    listMine: (...args: unknown[]) => listMine(...args),
    listTracks: (...args: unknown[]) => listTracks(...args),
    addTrack: (...args: unknown[]) => addTrack(...args),
    removeTrack: (...args: unknown[]) => removeTrack(...args),
    reorder: (...args: unknown[]) => reorder(...args),
  },
  trackApi: {
    get: (...args: unknown[]) => trackGet(...args),
  },
}));

beforeEach(() => {
  setActivePinia(createPinia());
  localStorage.clear();
  listMine.mockReset();
  listMine.mockResolvedValue([]);
  listTracks.mockReset();
  listTracks.mockResolvedValue([]);
  trackGet.mockReset();
  addTrack.mockReset();
  removeTrack.mockReset();
  reorder.mockReset();
  tokenStore.clear();
  trackGet.mockImplementation(async (id: string) => ({ id, title: 'T' }));
});

describe('playlistStore fetchMine auth guard', () => {
  it('skips the API call for guests (no access token)', async () => {
    expect(tokenStore.getAccess()).toBeNull();
    const s = usePlaylistStore();
    await s.fetchMine();
    expect(listMine).not.toHaveBeenCalled();
    expect(s.playlists).toEqual([]);
    expect(s.error).toBeNull();
    expect(s.loading).toBe(false);
  });

  it('clears stale playlists when the session is gone', async () => {
    tokenStore.set('tok', 'ref');
    listMine.mockResolvedValue([{ id: 'p1', title: 'Mix' }]);
    const s = usePlaylistStore();
    await s.fetchMine();
    expect(s.playlists).toHaveLength(1);
    tokenStore.clear();
    await s.fetchMine();
    expect(listMine).toHaveBeenCalledTimes(1);
    expect(s.playlists).toEqual([]);
  });

  it('fetches normally when authenticated', async () => {
    tokenStore.set('tok', 'ref');
    listMine.mockResolvedValue([{ id: 'p1', title: 'Mix' }]);
    const s = usePlaylistStore();
    await s.fetchMine();
    expect(listMine).toHaveBeenCalledTimes(1);
    expect(s.playlists).toHaveLength(1);
    expect(s.error).toBeNull();
  });
});

describe('playlistStore entries', () => {
  const tr = (id: string, durationMs = 60_000, status = 'READY') =>
    ({ id, title: `T-${id}`, durationMs, status }) as unknown as Track;
  const entry = (id: string, position: number, status = 'READY') => ({
    id: `e-${id}`, playlistId: 'p1', trackId: id, position, addedAt: '', track: tr(id, 60_000, status),
  });

  it('loads every entry with its joined track in one request (no per-track fetch)', async () => {
    listTracks.mockResolvedValue([entry('t1', 0), entry('t2', 1, 'TAKEN_DOWN'), entry('t3', 2)]);
    const s = usePlaylistStore();
    await s.fetchTracks('p1');
    expect(listTracks).toHaveBeenCalledTimes(1);
    expect(trackGet).not.toHaveBeenCalled();
    expect(s.entries['p1']?.map((e) => e.trackId)).toEqual(['t1', 't2', 't3']);
  });

  it('addTrack reports duplicates and only counts real additions', async () => {
    tokenStore.set('tok', 'ref');
    listMine.mockResolvedValue([{ id: 'p1', title: 'Mix', trackCount: 0, totalDurationMs: 0, coverSeeds: [] }]);
    const s = usePlaylistStore();
    await s.fetchMine();
    addTrack.mockResolvedValueOnce({ added: true, duplicate: false });
    expect(await s.addTrack('p1', tr('t1', 90_000))).toEqual({ added: true, duplicate: false });
    addTrack.mockResolvedValueOnce({ added: false, duplicate: true });
    expect(await s.addTrack('p1', tr('t1', 90_000))).toEqual({ added: false, duplicate: true });
    const p = s.playlists[0]!;
    expect(p.trackCount).toBe(1);
    expect(p.totalDurationMs).toBe(90_000);
    expect(p.coverSeeds).toEqual(['t1']);
    expect(p.containsTrack).toBe(true);
    expect(s.revision['p1']).toBe(1);
  });

  it('reorder applies optimistically and restores the old order on failure', async () => {
    listTracks.mockResolvedValue([entry('a', 0), entry('b', 1), entry('c', 2)]);
    const s = usePlaylistStore();
    await s.fetchTracks('p1');
    reorder.mockResolvedValueOnce({ reordered: true });
    await s.reorder('p1', ['c', 'a', 'b']);
    expect(s.entries['p1']?.map((e) => e.trackId)).toEqual(['c', 'a', 'b']);
    expect(s.entries['p1']?.map((e) => e.position)).toEqual([0, 1, 2]);

    reorder.mockRejectedValueOnce(new Error('ERR_INVALID_INPUT'));
    await expect(s.reorder('p1', ['b', 'c', 'a'])).rejects.toThrow();
    expect(s.entries['p1']?.map((e) => e.trackId)).toEqual(['c', 'a', 'b']);
  });

  it('removeTrack drops the entry and recomputes READY-only stats', async () => {
    tokenStore.set('tok', 'ref');
    listMine.mockResolvedValue([{ id: 'p1', title: 'Mix', trackCount: 2 }]);
    listTracks.mockResolvedValue([entry('a', 0), entry('b', 1), entry('x', 2, 'TAKEN_DOWN')]);
    const s = usePlaylistStore();
    await s.fetchMine();
    await s.fetchTracks('p1');
    removeTrack.mockResolvedValueOnce({ removed: true });
    await s.removeTrack('p1', 'a');
    expect(s.entries['p1']?.map((e) => e.trackId)).toEqual(['b', 'x']);
    expect(s.playlists[0]?.trackCount).toBe(1);
    expect(s.playlists[0]?.coverSeeds).toEqual(['b']);
  });
});
