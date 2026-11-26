// Public creator directory (GET /creators). Older engines without that
// endpoint fall back to grouping the public track list by creator handle.
import { fetchPublic } from '@/utils/publicApi';
import type { TrackPage } from '@/api/types';

export interface CreatorCard {
  handle: string;
  displayName: string;
  trackCount: number;
  totalPlays: number;
}

export async function fetchCreatorDirectory(limit = 60): Promise<CreatorCard[]> {
  const res = await fetchPublic<{ items: CreatorCard[] }>('/creators', { limit }).catch(() => null);
  if (res?.items) return res.items;
  const page = await fetchPublic<TrackPage>('/tracks', { limit: 100 }).catch(() => null);
  const map = new Map<string, CreatorCard>();
  for (const tr of page?.items ?? []) {
    const handle = tr.creatorHandle;
    if (!handle) continue;
    const cur = map.get(handle) ?? { handle, displayName: tr.creatorName ?? handle, trackCount: 0, totalPlays: 0 };
    cur.trackCount += 1;
    cur.totalPlays += tr.playCount ?? 0;
    map.set(handle, cur);
  }
  return [...map.values()].sort((a, b) => b.totalPlays - a.totalPlays).slice(0, limit);
}
