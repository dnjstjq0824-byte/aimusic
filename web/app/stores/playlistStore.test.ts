import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { tokenStore } from '@/api/client';
import { usePlaylistStore } from '@/stores/playlistStore';

const listMine = vi.fn();
const listTracks = vi.fn();
const trackGet = vi.fn();

vi.mock('@/api/endpoints', () => ({
  playlistApi: {
    listMine: (...args: unknown[]) => listMine(...args),
    listTracks: (...args: unknown[]) => listTracks(...args),
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

describe('playlistStore fetchTracks', () => {
  it('resolves entries in order and skips missing tracks', async () => {
    listTracks.mockResolvedValue([{ trackId: 't1' }, { trackId: 't2' }, { trackId: 't3' }]);
    trackGet.mockImplementation(async (id: string) => {
      if (id === 't2') throw new Error('taken down');
      return { id, title: `T-${id}` };
    });
    const s = usePlaylistStore();
    await s.fetchTracks('p1');
    expect(s.entries['p1']).toHaveLength(3);
    expect(s.entryTracks['p1']?.map((t) => t.id)).toEqual(['t1', 't3']);
  });

  it('fetches track details in parallel, not one by one', async () => {
    listTracks.mockResolvedValue([{ trackId: 't1' }, { trackId: 't2' }, { trackId: 't3' }]);
    const seen: string[] = [];
    let release!: () => void;
    const gate = new Promise<void>((r) => {
      release = r;
    });
    trackGet.mockImplementation(async (id: string) => {
      seen.push(id);
      await gate;
      return { id, title: 'T' };
    });
    const s = usePlaylistStore();
    const pending = s.fetchTracks('p1');
    await new Promise((r) => setTimeout(r, 0));
    expect(seen).toEqual(['t1', 't2', 't3']);
    release();
    await pending;
    expect(s.entryTracks['p1']).toHaveLength(3);
  });
});
