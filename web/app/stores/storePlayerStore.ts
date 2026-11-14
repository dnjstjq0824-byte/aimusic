import { defineStore } from 'pinia';
import type Hls from 'hls.js';
import { computed, ref } from 'vue';
import { storeDeviceApi, trackApi } from '@/api/endpoints';
import { deviceTokenStore, errCodeOf } from '@/api/client';
import type { DeviceQueue, DeviceQueueTrack, StoreDevice, Track } from '@/api/types';

const HEARTBEAT_MS = 10000;
const QUEUE_REFRESH_MS = 5 * 60 * 1000;
const SNAPSHOT_KEY = 'avibe.storeQueueSnapshot';

interface QueueSnapshot {
  queue: DeviceQueue;
  trackIndex: number;
  positionSecs: number;
  savedAt: number;
}

let storeAudioEl: HTMLAudioElement | null = null;
function storeAudio(): HTMLAudioElement {
  if (!storeAudioEl) {
    storeAudioEl = new Audio();
    storeAudioEl.preload = 'auto';
  }
  return storeAudioEl;
}

export function loadQueueSnapshot(): QueueSnapshot | null {
  try {
    const raw = localStorage.getItem(SNAPSHOT_KEY);
    if (!raw) return null;
    return JSON.parse(raw) as QueueSnapshot;
  } catch {
    return null;
  }
}

export function saveQueueSnapshot(snap: QueueSnapshot): void {
  try {
    localStorage.setItem(SNAPSHOT_KEY, JSON.stringify(snap));
  } catch {
    // Storage full/private mode: offline resume degrades gracefully.
  }
}

export const useStorePlayerStore = defineStore('storePlayer', () => {
  const device = ref<StoreDevice | null>(null);
  const claimed = computed(() => deviceTokenStore.get() !== null);
  const queue = ref<DeviceQueue | null>(null);
  const trackIndex = ref(0);
  const currentDetail = ref<Track | null>(null);
  const isPlaying = ref(false);
  const isLoading = ref(false);
  const isOnline = ref(typeof navigator === 'undefined' ? true : navigator.onLine);
  const error = ref<string | null>(null);
  const needsClaim = ref(false);
  const positionSecs = ref(0);
  const playedSecs = ref(0);

  const currentTrack = computed<DeviceQueueTrack | null>(() => {
    const q = queue.value;
    if (!q || q.tracks.length === 0) return null;
    return q.tracks[trackIndex.value % q.tracks.length] ?? null;
  });

  let hls: Hls | null = null;
  let heartbeatTimer: number | null = null;
  let queueTimer: number | null = null;
  let loadSeq = 0;
  let bound = false;

  function destroyHls(): void {
    if (hls) {
      hls.destroy();
      hls = null;
    }
  }

  function stopTimers(): void {
    if (heartbeatTimer !== null) {
      window.clearInterval(heartbeatTimer);
      heartbeatTimer = null;
    }
  }

  function snapshot(): void {
    const q = queue.value;
    if (!q) return;
    saveQueueSnapshot({ queue: q, trackIndex: trackIndex.value, positionSecs: positionSecs.value, savedAt: Date.now() });
  }

  function applySnapshot(snap: QueueSnapshot): void {
    queue.value = snap.queue;
    trackIndex.value = snap.trackIndex;
    positionSecs.value = snap.positionSecs;
  }

  function fail(seq: number, message: string): void {
    if (seq !== loadSeq) return;
    stopTimers();
    destroyHls();
    storeAudio().pause();
    isPlaying.value = false;
    isLoading.value = false;
    error.value = message;
  }

  function bindEvents(): void {
    if (bound) return;
    bound = true;
    const el = storeAudio();
    el.addEventListener('timeupdate', () => {
      positionSecs.value = el.currentTime;
    });
    el.addEventListener('play', () => {
      isPlaying.value = true;
    });
    el.addEventListener('pause', () => {
      isPlaying.value = false;
    });
    el.addEventListener('ended', () => {
      void advance();
    });
    el.addEventListener('error', () => {
      if (hls || !el.getAttribute('src')) return;
      fail(loadSeq, '음원을 재생할 수 없습니다. 네트워크 상태를 확인해 주세요.');
    });
    window.addEventListener('online', () => {
      isOnline.value = true;
      void refreshQueue();
    });
    window.addEventListener('offline', () => {
      isOnline.value = false;
    });
  }

  function startHeartbeat(trackId: string): void {
    stopTimers();
    playedSecs.value = 0;
    heartbeatTimer = window.setInterval(() => {
      const t = currentTrack.value;
      if (!t || t.trackId !== trackId || !isPlaying.value) return;
      playedSecs.value += HEARTBEAT_MS / 1000;
      void storeDeviceApi.heartbeat(trackId, Math.floor(playedSecs.value)).catch(() => undefined);
    }, HEARTBEAT_MS);
  }

  async function claim(pairingCode: string): Promise<void> {
    error.value = null;
    isLoading.value = true;
    try {
      const res = await storeDeviceApi.claim(pairingCode.trim().toUpperCase());
      deviceTokenStore.set(res.token);
      device.value = res.device;
      needsClaim.value = false;
      await refreshQueue();
    } catch (err: unknown) {
      error.value = errCodeOf(err);
      throw err;
    } finally {
      isLoading.value = false;
    }
  }

  function unclaim(): void {
    stopAll();
    deviceTokenStore.clear();
    device.value = null;
    queue.value = null;
    needsClaim.value = true;
  }

  async function refreshQueue(): Promise<void> {
    if (!claimed.value) {
      needsClaim.value = true;
      return;
    }
    try {
      queue.value = await storeDeviceApi.queue();
      needsClaim.value = false;
      error.value = null;
      const el = storeAudio();
      el.volume = Math.min(1, Math.max(0, (queue.value.volume ?? 70) / 100));
      if (trackIndex.value >= queue.value.tracks.length) trackIndex.value = 0;
      snapshot();
    } catch (err: unknown) {
      if (errCodeOf(err) === 'ERR_UNAUTHORIZED') {
        needsClaim.value = true;
        return;
      }
      // Offline or transient: fall back to the last snapshot.
      const snap = loadQueueSnapshot();
      if (snap) {
        applySnapshot(snap);
        error.value = 'OFFLINE_SNAPSHOT';
      } else {
        error.value = errCodeOf(err);
      }
    }
  }

  async function playIndex(index: number): Promise<void> {
    bindEvents();
    const seq = ++loadSeq;
    const q = queue.value;
    if (!q || q.tracks.length === 0) return;
    const track = q.tracks[index % q.tracks.length];
    if (!track) return;
    trackIndex.value = index % q.tracks.length;
    const el = storeAudio();
    destroyHls();
    stopTimers();
    el.pause();
    error.value = null;
    isLoading.value = true;
    positionSecs.value = 0;
    try {
      const [grant, detail] = await Promise.all([
        storeDeviceApi.stream(track.trackId),
        trackApi.get(track.trackId).catch(() => null),
      ]);
      if (seq !== loadSeq) return;
      currentDetail.value = detail;
      const url = grant.url;
      // hls.js loads lazily: keeps it out of SSR and the initial bundle.
      const { default: HlsImpl } = url.includes('.m3u8') ? await import('hls.js') : { default: null };
      if (HlsImpl && HlsImpl.isSupported()) {
        const instance = new HlsImpl({ enableWorker: true });
        hls = instance;
        instance.on(HlsImpl.Events.ERROR, (_evt, data) => {
          if (seq !== loadSeq || hls !== instance || !data.fatal) return;
          fail(seq, `스트리밍 연결이 끊겼습니다: ${track.title}`);
        });
        instance.loadSource(url);
        instance.attachMedia(el);
      } else {
        el.src = url;
      }
      el.volume = Math.min(1, Math.max(0, (q.volume ?? 70) / 100));
      await el.play().catch(() => undefined);
      if (seq !== loadSeq) return;
      startHeartbeat(track.trackId);
      snapshot();
    } catch (err: unknown) {
      if (seq !== loadSeq) return;
      const code = errCodeOf(err);
      if (code === 'ERR_UNAUTHORIZED') {
        needsClaim.value = true;
        error.value = code;
        isLoading.value = false;
        return;
      }
      fail(seq, code);
    } finally {
      if (seq === loadSeq) isLoading.value = false;
    }
  }

  async function advance(): Promise<void> {
    const q = queue.value;
    if (!q || q.tracks.length === 0) return;
    // Loop the scheduled queue all day (store auto-play).
    await playIndex((trackIndex.value + 1) % q.tracks.length);
  }

  function pause(): void {
    storeAudio().pause();
  }

  function resume(): void {
    const el = storeAudio();
    if (!el.getAttribute('src') && !hls) {
      void playIndex(trackIndex.value);
      return;
    }
    void el.play().catch(() => undefined);
  }

  function stopAll(): void {
    loadSeq += 1;
    stopTimers();
    if (queueTimer !== null) {
      window.clearInterval(queueTimer);
      queueTimer = null;
    }
    destroyHls();
    storeAudio().pause();
    isPlaying.value = false;
    isLoading.value = false;
  }

  function startQueueRefresh(): void {
    if (queueTimer !== null) return;
    queueTimer = window.setInterval(() => {
      void refreshQueue();
    }, QUEUE_REFRESH_MS);
  }

  return {
    device,
    claimed,
    queue,
    trackIndex,
    currentTrack,
    currentDetail,
    isPlaying,
    isLoading,
    isOnline,
    error,
    needsClaim,
    positionSecs,
    claim,
    unclaim,
    refreshQueue,
    playIndex,
    advance,
    pause,
    resume,
    stopAll,
    startQueueRefresh,
  };
});
