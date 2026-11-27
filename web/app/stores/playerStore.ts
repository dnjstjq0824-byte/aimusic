import { defineStore } from 'pinia';
import type Hls from 'hls.js';
import { computed, ref, watch } from 'vue';
import { playbackApi, trackApi } from '@/api/endpoints';
import { errCodeOf, tKey } from '@/api/client';
import { trackEvent } from '@/utils/analytics';
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

// Queue/volume survive a reload (client only). Bounded so a huge queue never
// blows the storage quota.
const PERSIST_KEY = 'avibe.player.v1';
const PERSIST_MAX_TRACKS = 300;
// MediaError codes that mean "this file is broken", not "the network blipped".
const MEDIA_ERR_DECODE = 3;
const MEDIA_ERR_SRC_NOT_SUPPORTED = 4;

interface PersistedPlayer {
  queue: Track[];
  order: number[];
  orderPos: number;
  shuffle: boolean;
  repeat: RepeatMode;
  volume: number;
  muted: boolean;
}

function readPersisted(): PersistedPlayer | null {
  if (typeof window === 'undefined') return null;
  try {
    const raw = window.localStorage.getItem(PERSIST_KEY);
    if (!raw) return null;
    const v = JSON.parse(raw) as Partial<PersistedPlayer>;
    if (!Array.isArray(v.queue) || !Array.isArray(v.order)) return null;
    const n = v.queue.length;
    const orderOk = v.order.length === n && v.order.every((i) => Number.isInteger(i) && i >= 0 && i < n);
    return {
      queue: orderOk ? v.queue : [],
      order: orderOk ? v.order : [],
      orderPos: orderOk && Number.isInteger(v.orderPos) && (v.orderPos as number) < n ? (v.orderPos as number) : 0,
      shuffle: v.shuffle === true,
      repeat: v.repeat === 'all' || v.repeat === 'one' ? v.repeat : 'off',
      volume: typeof v.volume === 'number' && v.volume >= 0 && v.volume <= 1 ? v.volume : 1,
      muted: v.muted === true,
    };
  } catch {
    return null;
  }
}

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
  const volume = ref(1);
  const muted = ref(false);

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
  /** S5: per-track analytics flags (play_started once, play_30s at 30s). */
  let analyticTrackId: string | null = null;
  let analyticSent30 = false;

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

  /** Tracks in play order (what the queue panel shows). */
  const orderedQueue = computed<Track[]>(() =>
    order.value.map((qi) => queue.value[qi]).filter((t): t is Track => t !== undefined),
  );

  /**
   * Replaces the queue with `tracks` as the new play order, keeping the
   * current track selected. Used by play-next / reorder so the visible order
   * is exactly what plays next, shuffle or not.
   */
  function commitPlayOrder(tracks: Track[], currentId: string | null): void {
    queue.value = tracks;
    order.value = tracks.map((_, i) => i);
    const pos = currentId ? tracks.findIndex((t) => t.id === currentId) : -1;
    orderPos.value = pos >= 0 ? pos : 0;
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

  /** Appends a track; false when it is already queued (no duplicates). */
  function enqueue(track: Track): boolean {
    if (queue.value.some((t) => t.id === track.id)) return false;
    queue.value.push(track);
    order.value = [...order.value, queue.value.length - 1];
    return true;
  }

  /** Puts a track right after the current one (moving it if already queued). */
  function queueNext(track: Track): void {
    const cur = currentTrack.value;
    if (!cur) {
      setQueue([track], track.id);
      return;
    }
    if (cur.id === track.id) return;
    const next = orderedQueue.value.filter((t) => t.id !== track.id);
    const at = next.findIndex((t) => t.id === cur.id);
    next.splice(at + 1, 0, track);
    commitPlayOrder(next, cur.id);
  }

  /** Moves an entry within the play order (indices into orderedQueue). */
  function moveInQueue(from: number, to: number): void {
    const list = [...orderedQueue.value];
    if (from < 0 || from >= list.length || to < 0 || to >= list.length || from === to) return;
    const [moved] = list.splice(from, 1);
    if (!moved) return;
    list.splice(to, 0, moved);
    commitPlayOrder(list, currentTrack.value?.id ?? null);
  }

  function applyVolume(): void {
    if (typeof window === 'undefined') return;
    const el = audio();
    el.volume = Math.min(1, Math.max(0, volume.value));
    el.muted = muted.value;
  }

  function setVolume(v: number): void {
    volume.value = Math.min(1, Math.max(0, v));
    if (volume.value > 0 && muted.value) muted.value = false;
    applyVolume();
  }

  function toggleMute(): void {
    muted.value = !muted.value;
    applyVolume();
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

  /**
   * Skips a track whose media is broken (decode / unsupported source) and
   * tells the listener. Stops instead when every queued track has failed,
   * so an all-broken queue never loops forever.
   */
  async function skipBroken(seq: number, track: Track, message: string): Promise<void> {
    if (seq !== loadSeq) return;
    if (!skippedIds.value.includes(track.id)) skippedIds.value.push(track.id);
    if (skippedIds.value.length >= queue.value.length) {
      failPlayback(seq, message);
      return;
    }
    stopHeartbeat();
    destroyHls();
    loadedTrackId = null;
    error.value = message;
    errorCode.value = null;
    await advance(true);
  }

  function bindAudioEvents(): void {
    if (bound) return;
    bound = true;
    const el = audio();
    applyVolume();
    el.addEventListener('timeupdate', () => {
      positionMs.value = el.currentTime * 1000;
      if (Number.isFinite(el.duration)) durationMs.value = el.duration * 1000;
      if (!analyticSent30 && loadedTrackId && el.currentTime >= 30) {
        analyticSent30 = true;
        trackEvent({ name: 'play_30s' });
      }
    });
    el.addEventListener('loadedmetadata', () => {
      if (Number.isFinite(el.duration)) durationMs.value = el.duration * 1000;
    });
    el.addEventListener('play', () => {
      isPlaying.value = true;
      if (loadedTrackId && loadedTrackId !== analyticTrackId) {
        analyticTrackId = loadedTrackId;
        analyticSent30 = false;
        trackEvent({ name: 'play_started' });
      }
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
      const code = el.error?.code;
      const track = currentTrack.value;
      if (track && (code === MEDIA_ERR_DECODE || code === MEDIA_ERR_SRC_NOT_SUPPORTED)) {
        void skipBroken(
          loadSeq,
          track,
          tKey('player.errSkipCorrupt', 'Skipped a damaged audio file: {title}', { title: track.title }),
        );
        return;
      }
      failPlayback(
        loadSeq,
        tKey(
          'player.errPlayFailed',
          "Can't play this track. Check your connection and try again.",
        ),
      );
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
          // A failed manifest (CDN 403 / expired signature / CORS / offline)
          // cannot be resumed with startLoad(): it issues no request and no
          // further error fires, leaving a spinner forever. Surface it; the
          // next play press fetches a fresh grant.
          const manifestFailed =
            data.details === HlsImpl.ErrorDetails.MANIFEST_LOAD_ERROR ||
            data.details === HlsImpl.ErrorDetails.MANIFEST_LOAD_TIMEOUT ||
            data.details === HlsImpl.ErrorDetails.MANIFEST_PARSING_ERROR;
          if (data.type === HlsImpl.ErrorTypes.NETWORK_ERROR && !manifestFailed && networkRetries < MAX_NETWORK_RETRIES) {
            networkRetries += 1;
            instance.startLoad();
            return;
          }
          if (data.type === HlsImpl.ErrorTypes.MEDIA_ERROR && !mediaRecovered) {
            mediaRecovered = true;
            instance.recoverMediaError();
            return;
          }
          if (data.type === HlsImpl.ErrorTypes.MEDIA_ERROR) {
            // Recovery already failed once: the media itself is broken.
            void skipBroken(
              seq,
              track,
              tKey('player.errSkipCorrupt', 'Skipped a damaged audio file: {title}', { title: track.title }),
            );
            return;
          }
          failPlayback(
            seq,
            tKey('player.errStreamLost', 'Stream interrupted: {title}', {
              title: track.title,
            }),
          );
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
        await skipBroken(
          seq,
          track,
          tKey('player.errSkipUnplayable', 'Skipping unplayable track: {title}', { title: track.title }),
        );
        return;
      }
      if (code === 'ERR_UNAUTHORIZED') {
        error.value = tKey('player.errSkipLogin', 'Log in to play.');
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

  // Restore the last session's queue (paused: autoplay needs a gesture).
  // Runs after hydration (app layout onMounted): restoring at store creation
  // would be overwritten by the SSR payload's empty queue, and saving before
  // the restore would clobber the stored session with that empty state.
  let restored = false;
  function restore(): void {
    if (restored || typeof window === 'undefined') return;
    restored = true;
    const persisted = readPersisted();
    if (!persisted) return;
    volume.value = persisted.volume;
    muted.value = persisted.muted;
    if (queue.value.length === 0 && persisted.queue.length > 0) {
      queue.value = persisted.queue;
      order.value = persisted.order;
      orderPos.value = persisted.orderPos;
      shuffle.value = persisted.shuffle;
      repeat.value = persisted.repeat;
      durationMs.value = currentTrack.value?.durationMs ?? 0;
    }
  }
  function saveNow(): void {
    try {
      // Keep the window around the current track when trimming.
      let q = queue.value;
      let o = order.value;
      let pos = orderPos.value;
      if (q.length > PERSIST_MAX_TRACKS) {
        const playOrder = orderedQueue.value;
        const start = Math.max(0, Math.min(pos, playOrder.length - PERSIST_MAX_TRACKS));
        q = playOrder.slice(start, start + PERSIST_MAX_TRACKS);
        o = q.map((_, i) => i);
        pos -= start;
      }
      const snap: PersistedPlayer = {
        queue: q,
        order: o,
        orderPos: pos,
        shuffle: shuffle.value,
        repeat: repeat.value,
        volume: volume.value,
        muted: muted.value,
      };
      window.localStorage.setItem(PERSIST_KEY, JSON.stringify(snap));
    } catch {
      // Storage full / disabled: persistence is best-effort.
    }
  }
  if (typeof window !== 'undefined') {
    let saveTimer: number | null = null;
    watch(
      [queue, order, orderPos, shuffle, repeat, volume, muted],
      () => {
        if (!restored) return;
        if (saveTimer !== null) window.clearTimeout(saveTimer);
        saveTimer = window.setTimeout(() => {
          saveTimer = null;
          saveNow();
        }, 250);
      },
      { deep: true },
    );
    // A reload / tab close inside the debounce window must not drop the
    // latest track change: flush synchronously on the way out.
    window.addEventListener('pagehide', () => {
      if (saveTimer === null) return;
      window.clearTimeout(saveTimer);
      saveTimer = null;
      saveNow();
    });
  }

  return {
    queue,
    orderedQueue,
    volume,
    muted,
    queueNext,
    moveInQueue,
    restore,
    setVolume,
    toggleMute,
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
