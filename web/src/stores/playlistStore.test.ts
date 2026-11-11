import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const listMine = vi.fn();

vi.mock('@/api/endpoints', () => ({
  playlistApi: {
    listMine: (...args: unknown[]) => listMine(...args),
  },
  trackApi: {
    get: (id: string) => Promise.resolve({ id, title: 'T' }),
  },
}));

import { tokenStore } from '@/api/client';
import { usePlaylistStore } from '@/stores/playlistStore';

beforeEach(() => {
  setActivePinia(createPinia());
  localStorage.clear();
  listMine.mockReset();
  listMine.mockResolvedValue([]);
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
