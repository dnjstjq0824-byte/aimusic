import { defineStore } from 'pinia';
import type Hls from 'hls.js';
import { computed, ref } from 'vue';
import { playbackApi, trackApi } from '@/api/endpoints';
import { errCodeOf } from '@/api/client';
import type { Track } from '@/api/types';

export type RepeatMode = 'off' | 'all' | 'one';

// Resume positions below this are treated as "from the start".
const RESUME_MIN_MS = 5000;
// Resume positions this close to the end are treated as "finished": resuming
// a few seconds before the end makes the next song start almost immediately,
// which listeners perceive as "a different song suddenly plays".
const RESUME_TAIL_MS = 30000;
// Server rejects positionMs=0 (binding:"required"), so 1ms marks "reset".
const RESET_POSITION_MS = 1;
// Transient network failures are retried before giving up on a track.
const MAX_NETWORK_RETRIES = 3;

// Singleton audio element shared across the whole app (gapless-ish queue).
let audioEl: HTMLAudioElement | null = null;
function audio(): HTMLAudioElement {
  if (!audioEl) {
    audioEl = new Audio();
    audioEl.preload = 'auto';
  }
  return audioEl;
}

function shuffled<T>(items: T[]): T[] {
  const arr = [...items];
  for (let i = arr.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [arr[i], arr[j]] = [arr[j] as T, arr[i] as T];
  }
  return arr;
}

/** Picks a safe resume offset (ms) or 0 when the saved position should be ignored. */
export function resumeOffsetMs(savedMs: number, durationMs: number | null | undefined): number {
  if (!Number.isFinite(savedMs) || savedMs < RESUME_MIN_MS) return 0;
  if (durationMs && durationMs > 0 && savedMs >= durationMs - RESUME_TAIL_MS) return 0;
  return savedMs;
}

export const usePlayerStore = defineStore('player', () => {
  const queue = ref<Track[]>([]);
  const order = ref<number[]>([]); // playback order (shuffled when enabled)
  const orderPos = ref(0);
  const isPlaying = ref(false);
  const isLoading = ref(false);
  const shuffle = ref(false);
  const repeat = ref<RepeatMode>('off');
  const positionMs = ref(0);
  const durationMs = ref(0);
  const error = ref<string | null>(null);
  const errorCode = ref<string | null>(null);
  const skippedIds = ref<string[]>([]);
  const currentJti = ref<string | null>(null);
  const currentQuality = ref<'HQ' | 'SQ'>('SQ');

  let hls: Hls | null = null;
  let heartbeatTimer: number | null = null;
  let lastHeartbeatAt = 0;
  let bound = false;
  /**
   * Monotonic load token. Every playCurrent() call takes a new value; any async
   * continuation (stream grant, resume lookup, hls.js callbacks) that finishes
   * after a newer load started is discarded. Without this, a slow response for
   * an earlier track could overwrite the audio source of the track the user
   * picked later, so the UI shows one song while another one plays.
   */
  let loadSeq = 0;
  /** Track id whose media is currently attached to the audio element. */
  let loadedTrackId: string | null = null;

  const currentTrack = computed<Track | null>(() => {
    if (queue.value.length === 0 || order.value.length === 0) return null;
    const qi = order.value[orderPos.value] ?? 0;
    return queue.value[qi] ?? null;
  });

  function rebuildOrder(keepCurrentId?: string): void {
    const idx = queue.value.map((_, i) => i);
    order.value = shuffle.value ? shuffled(idx) : idx;
    if (keepCurrentId) {
      const qi = queue.value.findIndex((t) => t.id === keepCurrentId);
      const pos = order.value.indexOf(qi);
      orderPos.value = pos >= 0 ? pos : 0;
    } else {
      orderPos.value = 0;
    }
  }

  /** Saves where the listener left the *currently loaded* track. */
  function saveLeavingPosition(): void {
    const id = loadedTrackId;
    const pos = Math.floor(positionMs.value);
    if (!id || pos <= 0) return;
    void playbackApi.setState(id, pos).catch(() => undefined);
  }

  function setQueue(tracks: Track[], startId?: string): void {
    saveLeavingPosition();
    queue.value = [...tracks];
    skippedIds.value = [];
    rebuildOrder();
    if (startId) {
      const qi = queue.value.findIndex((t) => t.id === startId);
      const pos = order.value.indexOf(qi);
      if (pos >= 0) orderPos.value = pos;
    }
    void playCurrent();
  }

  function enqueue(track: Track): void {
    queue.value.push(track);
    rebuildOrder(currentTrack.value?.id);
  }

  function removeFromQueue(trackId: string): void {
    const currentId = currentTrack.value?.id;
    queue.value = queue.value.filter((t) => t.id !== trackId);
    rebuildOrder(currentId && currentId !== trackId ? currentId : undefined);
    if (currentId === trackId) void playCurrent();
  }

  function clearQueue(): void {
    saveLeavingPosition();
    loadSeq += 1; // cancel any in-flight load
    stopHeartbeat();
    destroyHls();
    audio().pause();
    loadedTrackId = null;
    queue.value = [];
    order.value = [];
    orderPos.value = 0;
    isPlaying.value = false;
    isLoading.value = false;
    positionMs.value = 0;
    durationMs.value = 0;
    error.value = null;
    errorCode.value = null;
  }

  function toggleShuffle(): void {
    shuffle.value = !shuffle.value;
    rebuildOrder(currentTrack.value?.id);
  }

  function cycleRepeat(): void {
    repeat.value = repeat.value === 'off' ? 'all' : repeat.value === 'all' ? 'one' : 'off';
  }

  function destroyHls(): void {
    if (hls) {
      hls.destroy();
      hls = null;
    }
  }

  function stopHeartbeat(): void {
    if (heartbeatTimer !== null) {
      window.clearInterval(heartbeatTimer);
      heartbeatTimer = null;
    }
  }

  function startHeartbeat(trackId: string): void {
    stopHeartbeat();
    lastHeartbeatAt = Date.now();
    heartbeatTimer = window.setInterval(() => {
      if (loadedTrackId !== trackId || !isPlaying.value) return;
      const now = Date.now();
      const playMs = now - lastHeartbeatAt;
      lastHeartbeatAt = now;
      void playbackApi
        .heartbeat({ trackId, positionMs: Math.floor(positionMs.value), playMs, jti: currentJti.value ?? undefined })
        .catch(() => undefined);
    }, 10000);
  }

  function updateMediaSession(track: Track): void {
    if (!('mediaSession' in navigator)) return;
    try {
      navigator.mediaSession.metadata = new MediaMetadata({
        title: track.title,
        artist: track.aiProvider,
        album: track.aiDisclosureText ?? 'AVIBE',
      });
      navigator.mediaSession.setActionHandler('play', () => void resume());
      navigator.mediaSession.setActionHandler('pause', () => pause());
      navigator.mediaSession.setActionHandler('previoustrack', () => void playPrev());
      navigator.mediaSession.setActionHandler('nexttrack', () => void playNext(true));
      navigator.mediaSession.setActionHandler('seekto', (d) => {
        if (typeof d.seekTime === 'number') seek(d.seekTime * 1000);
      });
      navigator.mediaSession.playbackState = 'playing';
    } catch {
      // Media Session is best-effort.
    }
  }

  function failPlayback(seq: number, message: string): void {
    if (seq !== loadSeq) return;
    stopHeartbeat();
    destroyHls();
    audio().pause();
    loadedTrackId = null; // next play press reloads with a fresh grant
    isPlaying.value = false;
    isLoading.value = false;
    error.value = message;
    errorCode.value = null;
  }

  function bindAudioEvents(): void {
    if (bound) return;
    bound = true;
    const el = audio();
    el.addEventListener('timeupdate', () => {
      positionMs.value = el.currentTime * 1000;
      if (Number.isFinite(el.duration)) durationMs.value = el.duration * 1000;
    });
    el.addEventListener('loadedmetadata', () => {
      if (Number.isFinite(el.duration)) durationMs.value = el.duration * 1000;
    });
    el.addEventListener('play', () => {
      isPlaying.value = true;
    });
    el.addEventListener('pause', () => {
      isPlaying.value = false;
    });
    el.addEventListener('ended', () => {
      if (repeat.value === 'one') {
        el.currentTime = 0;
        void el.play().catch(() => undefined);
        return;
      }
      // Finished: reset resume so the next visit starts from the top
      // instead of jumping straight to the (already finished) tail.
      if (loadedTrackId) {
        void playbackApi.setState(loadedTrackId, RESET_POSITION_MS).catch(() => undefined);
      }
      positionMs.value = 0;
      void advance(false);
    });
    el.addEventListener('error', () => {
      // hls.js owns error handling while attached (MSE); only native playback
      // reports here. Never silently jump to a different song on a transport
      // error: surface it so the listener knows what happened.
      if (hls || !el.getAttribute('src')) return;
      failPlayback(loadSeq, '음원을 재생할 수 없습니다. 네트워크 상태를 확인한 뒤 다시 시도해 주세요.');
    });
  }

  async function playCurrent(): Promise<void> {
    bindAudioEvents();
    const seq = ++loadSeq;
    const track = currentTrack.value;
    const el = audio();
    destroyHls();
    stopHeartbeat();
    el.pause();
    error.value = null;
    errorCode.value = null;
    positionMs.value = 0;
    durationMs.value = track?.durationMs ?? 0;
    loadedTrackId = null;
    if (!track) {
      isPlaying.value = false;
      isLoading.value = false;
      return;
    }
    isLoading.value = true;
    try {
      // Grant + resume lookup in parallel; resume is best-effort.
      const [grant, savedMs] = await Promise.all([
        trackApi.stream(track.id),
        playbackApi
          .getState(track.id)
          .then((s) => s.positionMs)
          .catch(() => 0),
      ]);
      if (seq !== loadSeq) return; // a newer track was picked meanwhile

      currentJti.value = grant.jti;
      currentQuality.value = grant.quality;
      loadedTrackId = track.id;
      const startMs = resumeOffsetMs(savedMs, track.durationMs);
      const url = grant.url;

      // hls.js loads lazily: keeps it out of SSR and the initial bundle.
      const { default: HlsImpl } = url.includes('.m3u8') ? await import('hls.js') : { default: null };
      if (HlsImpl && HlsImpl.isSupported()) {
        let networkRetries = 0;
        let mediaRecovered = false;
        const instance = new HlsImpl({ enableWorker: true, startPosition: startMs / 1000 });
        hls = instance;
        instance.on(HlsImpl.Events.ERROR, (_evt, data) => {
          if (seq !== loadSeq || hls !== instance || !data.fatal) return;
          if (data.type === HlsImpl.ErrorTypes.NETWORK_ERROR && networkRetries < MAX_NETWORK_RETRIES) {
            networkRetries += 1;
            instance.startLoad();
            return;
          }
          if (data.type === HlsImpl.ErrorTypes.MEDIA_ERROR && !mediaRecovered) {
            mediaRecovered = true;
            instance.recoverMediaError();
            return;
          }
          failPlayback(seq, `스트리밍 연결이 끊겼습니다: ${track.title}`);
        });
        instance.loadSource(url);
        instance.attachMedia(el);
      } else {
        el.src = url;
        if (startMs > 0) {
          el.addEventListener(
            'loadedmetadata',
            () => {
              if (seq === loadSeq) el.currentTime = startMs / 1000;
            },
            { once: true },
          );
        }
      }
      positionMs.value = startMs;
      await el.play().catch(() => undefined);
      if (seq !== loadSeq) return;

      updateMediaSession(track);
      startHeartbeat(track.id);
    } catch (err: unknown) {
      if (seq !== loadSeq) return;
      const code = errCodeOf(err);
      errorCode.value = code;
      if (code === 'ERR_NOT_STREAMABLE' || code === 'ERR_FORBIDDEN') {
        // Takedown / blocked track: skip it explicitly and say so.
        skippedIds.value.push(track.id);
        error.value = `재생할 수 없는 음원을 건너뜁니다: ${track.title}`;
        await advance(true);
        return;
      }
      if (code === 'ERR_UNAUTHORIZED') {
        error.value = '재생하려면 로그인이 필요합니다.';
        isPlaying.value = false;
        return;
      }
      error.value = code;
      isPlaying.value = false;
    } finally {
      if (seq === loadSeq) isLoading.value = false;
    }
  }

  /** Moves to the next entry in play order without persisting (caller decides). */
  async function advance(force: boolean): Promise<void> {
    if (order.value.length === 0) return;
    const atEnd = orderPos.value >= order.value.length - 1;
    if (atEnd) {
      if (repeat.value === 'all' || force) {
        orderPos.value = 0;
      } else {
        isPlaying.value = false;
        audio().pause();
        return;
      }
    } else {
      orderPos.value += 1;
    }
    await playCurrent();
  }

  async function playNext(force = false): Promise<void> {
    // Persist BEFORE moving: the position belongs to the track being left.
    saveLeavingPosition();
    await advance(force);
  }

  async function playPrev(): Promise<void> {
    if (positionMs.value > 3000) {
      seek(0);
      return;
    }
    saveLeavingPosition();
    if (orderPos.value > 0) orderPos.value -= 1;
    await playCurrent();
  }

  async function playAt(orderIndex: number): Promise<void> {
    if (orderIndex < 0 || orderIndex >= order.value.length) return;
    saveLeavingPosition();
    orderPos.value = orderIndex;
    await playCurrent();
  }

  async function resume(): Promise<void> {
    bindAudioEvents();
    error.value = null;
    errorCode.value = null;
    const track = currentTrack.value;
    if (track && loadedTrackId !== track.id) {
      await playCurrent();
      return;
    }
    await audio().play().catch(() => undefined);
  }

  function pause(): void {
    audio().pause();
    if ('mediaSession' in navigator) {
      try {
        navigator.mediaSession.playbackState = 'paused';
      } catch {
        // ignore
      }
    }
    saveLeavingPosition();
  }

  function toggle(): void {
    if (isPlaying.value) pause();
    else void resume();
  }

  function seek(ms: number): void {
    const el = audio();
    el.currentTime = Math.max(0, ms / 1000);
    positionMs.value = Math.max(0, ms);
  }

  return {
    queue,
    orderPos,
    order,
    isPlaying,
    isLoading,
    shuffle,
    repeat,
    positionMs,
    durationMs,
    error,
    errorCode,
    skippedIds,
    currentQuality,
    currentTrack,
    setQueue,
    enqueue,
    removeFromQueue,
    clearQueue,
    toggleShuffle,
    cycleRepeat,
    playNext,
    playPrev,
    playAt,
    playCurrent,
    resume,
    pause,
    toggle,
    seek,
  };
});
