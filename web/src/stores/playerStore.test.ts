import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { Track } from '@/api/types';

// Minimal stand-in for HTMLAudioElement: records the source the store attaches.
class FakeAudio extends EventTarget {
  src = '';
  preload = '';
  currentTime = 0;
  duration = NaN;
  paused = true;
  getAttribute(name: string): string | null {
    return name === 'src' && this.src ? this.src : null;
  }
  play(): Promise<void> {
    this.paused = false;
    this.dispatchEvent(new Event('play'));
    return Promise.resolve();
  }
  pause(): void {
    this.paused = true;
    this.dispatchEvent(new Event('pause'));
  }
}

let lastAudio: FakeAudio;
vi.stubGlobal(
  'Audio',
  class extends FakeAudio {
    constructor() {
      super();
      lastAudio = this;
    }
  },
);

// Native playback path (no MSE) keeps the test focused on queue logic.
vi.mock('hls.js', () => ({ default: { isSupported: () => false } }));

const stream = vi.fn();
const getState = vi.fn();
const setState = vi.fn();
vi.mock('@/api/endpoints', () => ({
  trackApi: { stream: (id: string) => stream(id) },
  playbackApi: {
    getState: (id: string) => getState(id),
    setState: (id: string, pos: number) => setState(id, pos),
    heartbeat: vi.fn(() => Promise.resolve()),
  },
}));

function track(id: string, durationMs = 300_000): Track {
  return { id, title: `title-${id}`, aiProvider: 'SUNO', durationMs } as unknown as Track;
}

function deferred<T>() {
  let resolve!: (v: T) => void;
  const promise = new Promise<T>((r) => (resolve = r));
  return { promise, resolve };
}

const flush = () => new Promise((r) => setTimeout(r, 0));

async function freshStore() {
  vi.resetModules();
  setActivePinia(createPinia());
  const mod = await import('./playerStore');
  return { store: mod.usePlayerStore(), resumeOffsetMs: mod.resumeOffsetMs };
}

describe('usePlayerStore', () => {
  beforeEach(() => {
    stream.mockReset();
    getState.mockReset().mockResolvedValue({ positionMs: 0 });
    setState.mockReset().mockResolvedValue({});
  });

  it('a slow grant for an earlier track never overrides the track picked later', async () => {
    const { store } = await freshStore();
    const slowA = deferred<{ url: string; jti: string }>();
    stream.mockImplementation((id: string) =>
      id === 'A' ? slowA.promise : Promise.resolve({ url: 'https://cdn/B.mp3', jti: 'jb' }),
    );

    store.setQueue([track('A'), track('B')], 'A'); // A starts loading (slow)
    await store.playAt(1); // user picks B; B loads instantly
    expect(lastAudio.src).toBe('https://cdn/B.mp3');

    slowA.resolve({ url: 'https://cdn/A.mp3', jti: 'ja' }); // A finally answers
    await flush();

    expect(store.currentTrack?.id).toBe('B');
    expect(lastAudio.src).toBe('https://cdn/B.mp3');
  });

  it('saves the leaving position under the track being left, not the next one', async () => {
    const { store } = await freshStore();
    stream.mockImplementation((id: string) => Promise.resolve({ url: `https://cdn/${id}.mp3`, jti: id }));

    store.setQueue([track('A'), track('B')], 'A');
    await flush();
    store.positionMs = 120_000;

    await store.playNext(true);

    expect(setState).toHaveBeenCalledWith('A', 120_000);
    expect(setState).not.toHaveBeenCalledWith('B', expect.anything());
    expect(store.currentTrack?.id).toBe('B');
  });

  it('finishing a track resets its resume point and advances', async () => {
    const { store } = await freshStore();
    stream.mockImplementation((id: string) => Promise.resolve({ url: `https://cdn/${id}.mp3`, jti: id }));

    store.setQueue([track('A'), track('B')], 'A');
    await flush();
    lastAudio.dispatchEvent(new Event('ended'));
    await flush();

    expect(setState).toHaveBeenCalledWith('A', 1);
    expect(store.currentTrack?.id).toBe('B');
    expect(lastAudio.src).toBe('https://cdn/B.mp3');
  });

  it('a native media error stops playback instead of jumping to another song', async () => {
    const { store } = await freshStore();
    stream.mockImplementation((id: string) => Promise.resolve({ url: `https://cdn/${id}.mp3`, jti: id }));

    store.setQueue([track('A'), track('B')], 'A');
    await flush();
    lastAudio.dispatchEvent(new Event('error'));
    await flush();

    expect(store.currentTrack?.id).toBe('A');
    expect(stream).toHaveBeenCalledTimes(1);
    expect(store.isPlaying).toBe(false);
    expect(store.error).toBeTruthy();
  });

  it('resumeOffsetMs ignores tiny and near-end positions', async () => {
    const { resumeOffsetMs } = await freshStore();
    expect(resumeOffsetMs(1, 300_000)).toBe(0);
    expect(resumeOffsetMs(4_000, 300_000)).toBe(0);
    expect(resumeOffsetMs(60_000, 300_000)).toBe(60_000);
    expect(resumeOffsetMs(295_000, 300_000)).toBe(0); // within the last 10s
    expect(resumeOffsetMs(299_000, 289_000)).toBe(0); // beyond this track's length
  });
});
