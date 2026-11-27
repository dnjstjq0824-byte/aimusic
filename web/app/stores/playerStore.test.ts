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
      // Capturing the constructed fake is the point (src assertions + event dispatch below).
      // eslint-disable-next-line @typescript-eslint/no-this-alias
      lastAudio = this;
    }
  },
);

// Native playback path (no MSE) by default keeps tests focused on queue
// logic; hlsState.supported switches on a scriptable fake for HLS paths.
const hlsState = vi.hoisted(() => ({ supported: false, last: null as null | { emit: (ev: string, data: unknown) => void; startLoad: () => void; starts: number } }));
vi.mock('hls.js', () => {
  class FakeHls {
    static isSupported = () => hlsState.supported;
    static Events = { ERROR: 'hlsError' };
    static ErrorTypes = { NETWORK_ERROR: 'networkError', MEDIA_ERROR: 'mediaError' };
    static ErrorDetails = {
      MANIFEST_LOAD_ERROR: 'manifestLoadError',
      MANIFEST_LOAD_TIMEOUT: 'manifestLoadTimeOut',
      MANIFEST_PARSING_ERROR: 'manifestParsingError',
    };
    handlers: Record<string, (ev: string, data: unknown) => void> = {};
    starts = 0;
    constructor() {
      hlsState.last = this;
    }
    on(ev: string, fn: (ev: string, data: unknown) => void) {
      this.handlers[ev] = fn;
    }
    emit(ev: string, data: unknown) {
      this.handlers[ev]?.(ev, data);
    }
    startLoad() {
      this.starts += 1;
    }
    loadSource() {}
    attachMedia() {}
    destroy() {}
    recoverMediaError() {}
  }
  return { default: FakeHls };
});

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

  it('a rejected grant exposes its code so the UI can offer login', async () => {
    const { store } = await freshStore();
    stream.mockRejectedValue({
      isAxiosError: true,
      response: { data: { error: { code: 'ERR_UNAUTHORIZED', message: 'login required' } } },
    });

    store.setQueue([track('A')], 'A');
    await flush();

    expect(store.errorCode).toBe('ERR_UNAUTHORIZED');
    expect(store.error).toBe('Log in to play.');
    expect(store.isPlaying).toBe(false);
  });

  it('a rejected grant resolves the app locale when a translator is installed', async () => {
    const { store } = await freshStore();
    const client = await import('@/api/client');
    client.setKeyTranslator((key) => (key === 'player.errSkipLogin' ? 'KO-LOGIN-REQ' : null));
    stream.mockRejectedValue({
      isAxiosError: true,
      response: { data: { error: { code: 'ERR_UNAUTHORIZED', message: 'login required' } } },
    });

    store.setQueue([track('A')], 'A');
    await flush();

    expect(store.error).toBe('KO-LOGIN-REQ');
    client.setKeyTranslator(null);
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
    expect(resumeOffsetMs(295_000, 300_000)).toBe(0); // within the last 30s
    expect(resumeOffsetMs(289_000, 300_000)).toBe(0); // seen live: resumed at 4:49 of 5:00
    expect(resumeOffsetMs(260_000, 300_000)).toBe(260_000);
    expect(resumeOffsetMs(299_000, 289_000)).toBe(0); // beyond this track's length
  });

  it('queueNext puts a track right after the current one (moving it if queued)', async () => {
    const { store } = await freshStore();
    stream.mockImplementation((id: string) => Promise.resolve({ url: `https://cdn/${id}.mp3`, jti: id }));
    store.setQueue([track('A'), track('B'), track('C')], 'A');
    await flush();
    store.queueNext(track('C'));
    expect(store.orderedQueue.map((t) => t.id)).toEqual(['A', 'C', 'B']);
    expect(store.currentTrack?.id).toBe('A');
    store.queueNext(track('D'));
    expect(store.orderedQueue.map((t) => t.id)).toEqual(['A', 'D', 'C', 'B']);
  });

  it('enqueue refuses duplicates and moveInQueue keeps the current track', async () => {
    const { store } = await freshStore();
    stream.mockImplementation((id: string) => Promise.resolve({ url: `https://cdn/${id}.mp3`, jti: id }));
    store.setQueue([track('A'), track('B')], 'B');
    await flush();
    expect(store.enqueue(track('A'))).toBe(false);
    expect(store.enqueue(track('C'))).toBe(true);
    store.moveInQueue(2, 0);
    expect(store.orderedQueue.map((t) => t.id)).toEqual(['C', 'A', 'B']);
    expect(store.currentTrack?.id).toBe('B');
  });

  it('a decode error skips to the next track and says why', async () => {
    const { store } = await freshStore();
    stream.mockImplementation((id: string) => Promise.resolve({ url: `https://cdn/${id}.mp3`, jti: id }));
    store.setQueue([track('A'), track('B')], 'A');
    await flush();
    Object.defineProperty(lastAudio, 'error', { value: { code: 3 }, configurable: true });
    lastAudio.dispatchEvent(new Event('error'));
    await flush();
    await flush();
    expect(store.currentTrack?.id).toBe('B');
    // The player bar's "skipped N" strip reads skippedIds (the per-track
    // error text is replaced once the next track starts).
    expect(store.skippedIds).toEqual(['A']);
    expect(stream).toHaveBeenCalledTimes(2);
  });

  it('stops instead of looping when every queued track is broken', async () => {
    const { store } = await freshStore();
    stream.mockImplementation((id: string) => Promise.resolve({ url: `https://cdn/${id}.mp3`, jti: id }));
    store.setQueue([track('A')], 'A');
    await flush();
    Object.defineProperty(lastAudio, 'error', { value: { code: 4 }, configurable: true });
    lastAudio.dispatchEvent(new Event('error'));
    await flush();
    expect(stream).toHaveBeenCalledTimes(1);
    expect(store.isPlaying).toBe(false);
    expect(store.error).toBeTruthy();
  });

  it('restores the saved queue and volume after hydration', async () => {
    localStorage.setItem(
      'avibe.player.v1',
      JSON.stringify({ queue: [track('A'), track('B')], order: [0, 1], orderPos: 1, shuffle: false, repeat: 'all', volume: 0.4, muted: false }),
    );
    const { store } = await freshStore();
    expect(store.currentTrack).toBeNull();
    store.restore();
    expect(store.currentTrack?.id).toBe('B');
    expect(store.repeat).toBe('all');
    expect(store.volume).toBe(0.4);
    expect(store.isPlaying).toBe(false);
    localStorage.clear();
  });

  it('a failed HLS manifest surfaces an error instead of spinning forever', async () => {
    hlsState.supported = true;
    try {
      const { store } = await freshStore();
      stream.mockImplementation((id: string) => Promise.resolve({ url: `https://cdn/${id}/stream_0.m3u8`, jti: id }));
      store.setQueue([track('A'), track('B')], 'A');
      await flush();
      await flush();
      hlsState.last?.emit('hlsError', { fatal: true, type: 'networkError', details: 'manifestLoadError' });
      expect(hlsState.last?.starts).toBe(0);
      expect(store.isLoading).toBe(false);
      expect(store.isPlaying).toBe(false);
      expect(store.error).toBeTruthy();
      expect(store.currentTrack?.id).toBe('A');
    } finally {
      hlsState.supported = false;
    }
  });

  it('a fatal segment network error retries the load before failing', async () => {
    hlsState.supported = true;
    try {
      const { store } = await freshStore();
      stream.mockImplementation((id: string) => Promise.resolve({ url: `https://cdn/${id}/stream_0.m3u8`, jti: id }));
      store.setQueue([track('A')], 'A');
      await flush();
      await flush();
      hlsState.last?.emit('hlsError', { fatal: true, type: 'networkError', details: 'fragLoadError' });
      expect(hlsState.last?.starts).toBe(1);
      expect(store.error).toBeNull();
    } finally {
      hlsState.supported = false;
    }
  });

  it('flushes a pending queue save when the page is hidden (reload inside the debounce)', async () => {
    localStorage.clear();
    const { store } = await freshStore();
    stream.mockImplementation((id: string) => Promise.resolve({ url: `https://cdn/${id}.mp3`, jti: id }));
    store.restore();
    store.setQueue([track('A'), track('B')], 'A');
    await flush();
    await store.playNext(true);
    // No timer has fired yet; pagehide must persist the new position now.
    window.dispatchEvent(new Event('pagehide'));
    const saved = JSON.parse(localStorage.getItem('avibe.player.v1') ?? 'null');
    expect(saved?.orderPos).toBe(1);
    localStorage.clear();
  });
});
