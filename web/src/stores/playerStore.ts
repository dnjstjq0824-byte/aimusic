import { defineStore } from 'pinia';
import Hls from 'hls.js';
import { computed, ref } from 'vue';
import { playbackApi, trackApi } from '@/api/endpoints';
import { errCodeOf } from '@/api/client';
import type { Track } from '@/api/types';

export type RepeatMode = 'off' | 'all' | 'one';

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
  const skippedIds = ref<string[]>([]);
  const currentJti = ref<string | null>(null);

  let hls: Hls | null = null;
  let heartbeatTimer: number | null = null;
  let lastHeartbeatAt = 0;
  let bound = false;

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

  function setQueue(tracks: Track[], startId?: string): void {
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
    stopHeartbeat();
    destroyHls();
    audio().pause();
    queue.value = [];
    order.value = [];
    orderPos.value = 0;
    isPlaying.value = false;
    positionMs.value = 0;
    durationMs.value = 0;
    error.value = null;
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

  function startHeartbeat(): void {
    stopHeartbeat();
    lastHeartbeatAt = Date.now();
    heartbeatTimer = window.setInterval(() => {
      const t = currentTrack.value;
      if (!t || !isPlaying.value) return;
      const now = Date.now();
      const playMs = now - lastHeartbeatAt;
      lastHeartbeatAt = now;
      void playbackApi
        .heartbeat({ trackId: t.id, positionMs: Math.floor(positionMs.value), playMs, jti: currentJti.value ?? undefined })
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
    } catch {
      // Media Session is best-effort.
    }
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
      void playNext();
    });
    el.addEventListener('error', () => {
      void playNext(true);
    });
  }

  async function playCurrent(): Promise<void> {
    bindAudioEvents();
    const track = currentTrack.value;
    const el = audio();
    destroyHls();
    stopHeartbeat();
    error.value = null;
    if (!track) {
      el.pause();
      isPlaying.value = false;
      return;
    }
    // Record resume position of previous state is handled by heartbeat; load fresh grant.
    isLoading.value = true;
    try {
      const grant = await trackApi.stream(track.id);
      currentJti.value = grant.jti;
      const url = grant.url;
      if (url.includes('.m3u8') && Hls.isSupported()) {
        hls = new Hls({ enableWorker: true });
        hls.on(Hls.Events.ERROR, (_evt, data) => {
          if (data.fatal) void playNext(true);
        });
        hls.loadSource(url);
        hls.attachMedia(el);
        await el.play().catch(() => undefined);
      } else {
        el.src = url;
        await el.play().catch(() => undefined);
      }
      // Restore resume position (best-effort).
      try {
        const state = await playbackApi.getState(track.id);
        if (state.positionMs > 5000) {
          el.currentTime = state.positionMs / 1000;
          positionMs.value = state.positionMs;
        }
      } catch {
        // No resume state yet.
      }
      updateMediaSession(track);
      startHeartbeat();
      if ('mediaSession' in navigator) {
        try {
          navigator.mediaSession.playbackState = 'playing';
        } catch {
          // ignore
        }
      }
    } catch (err: unknown) {
      const code = errCodeOf(err);
      if (code === 'ERR_NOT_STREAMABLE' || code === 'ERR_FORBIDDEN') {
        // Takedown / blocked track: auto-skip to next.
        skippedIds.value.push(track.id);
        error.value = `차단된 음원을 건너뜁니다: ${track.title}`;
        await playNext(true);
        return;
      }
      error.value = code;
      isPlaying.value = false;
    } finally {
      isLoading.value = false;
    }
  }

  async function playNext(force = false): Promise<void> {
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
    persistPosition().catch(() => undefined);
    await playCurrent();
  }

  async function playPrev(): Promise<void> {
    if (positionMs.value > 3000) {
      seek(0);
      return;
    }
    if (orderPos.value > 0) orderPos.value -= 1;
    await playCurrent();
  }

  async function resume(): Promise<void> {
    bindAudioEvents();
    error.value = null;
    if (!audioEl?.src && currentTrack.value) {
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
    persistPosition().catch(() => undefined);
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

  async function persistPosition(): Promise<void> {
    const t = currentTrack.value;
    if (!t) return;
    const pos = Math.floor(positionMs.value);
    if (pos > 0) await playbackApi.setState(t.id, pos);
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
    skippedIds,
    currentTrack,
    setQueue,
    enqueue,
    removeFromQueue,
    clearQueue,
    toggleShuffle,
    cycleRepeat,
    playNext,
    playPrev,
    playCurrent,
    resume,
    pause,
    toggle,
    seek,
  };
});
