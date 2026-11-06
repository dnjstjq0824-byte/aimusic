<template>
  <div>
    <div class="mb-4 flex items-center justify-between">
      <h1 class="text-xl font-bold">Ready Tracks</h1>
      <button class="rounded bg-brand-600 px-3 py-1.5 text-sm font-bold disabled:opacity-50" :disabled="tracks.length === 0" @click="playAll">
        전체 재생
      </button>
    </div>
    <p v-if="error" class="mb-2 text-sm text-red-400">{{ error }}</p>
    <p v-if="reported" class="mb-2 text-sm text-emerald-400">분쟁 신고가 접수되었습니다. ({{ reported }})</p>
    <ul class="grid gap-2 md:grid-cols-2">
      <li v-for="t in tracks" :key="t.id" class="rounded bg-neutral-900 p-3">
        <div class="flex items-center gap-2">
          <button class="rounded bg-neutral-800 px-2 py-1 text-sm" @click="playOne(t)">▶</button>
          <RouterLink :to="{ name: 'track-detail', params: { id: t.id } }" class="min-w-0 flex-1 truncate font-semibold hover:underline">
            {{ t.title }}
          </RouterLink>
          <AiBadge :track="t" />
        </div>
        <div class="mt-1 flex items-center gap-2 text-xs text-neutral-400">
          <span>{{ t.aiProvider }}</span>
          <span>plays {{ t.playCount }}</span>
          <button class="underline" @click="enqueue(t)">+ Queue</button>
          <button class="underline" @click="openReport(t)">신고</button>
          <select v-model="playlistPick" class="rounded bg-neutral-800 p-1" title="Add to playlist">
            <option value="">+ Playlist…</option>
            <option v-for="p in playlists.playlists" :key="p.id" :value="p.id">{{ p.title }}</option>
          </select>
          <button v-if="playlistPick" class="underline" @click="addToPlaylist(t.id)">추가</button>
        </div>
      </li>
    </ul>
    <div class="mt-4 flex gap-2">
      <button class="rounded bg-neutral-800 px-3 py-1.5 text-sm disabled:opacity-50" :disabled="offset === 0" @click="page(-1)">이전</button>
      <button class="rounded bg-neutral-800 px-3 py-1.5 text-sm" @click="page(1)">다음</button>
    </div>
    <DisputeReportModal
      v-if="reportTarget"
      :track-id="reportTarget.id"
      :track-title="reportTarget.title"
      @close="reportTarget = null"
      @reported="reported = $event"
    />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { trackApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { usePlayerStore } from '@/stores/playerStore';
import { usePlaylistStore } from '@/stores/playlistStore';
import { useAuthStore } from '@/stores/authStore';
import type { Track } from '@/api/types';
import AiBadge from '@/components/legal/AiBadge.vue';
import DisputeReportModal from '@/components/legal/DisputeReportModal.vue';

const player = usePlayerStore();
const playlists = usePlaylistStore();
const auth = useAuthStore();

const tracks = ref<Track[]>([]);
const error = ref<string | null>(null);
const offset = ref(0);
const limit = 20;
const reportTarget = ref<Track | null>(null);
const reported = ref<string | null>(null);
const playlistPick = ref('');

async function load(): Promise<void> {
  error.value = null;
  try {
    tracks.value = await trackApi.list(limit, offset.value);
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  }
}

function page(delta: number): void {
  offset.value = Math.max(0, offset.value + delta * limit);
  void load();
}

function playAll(): void {
  player.setQueue(tracks.value);
}

function playOne(t: Track): void {
  player.setQueue(tracks.value, t.id);
}

function enqueue(t: Track): void {
  player.enqueue(t);
}

function openReport(t: Track): void {
  reportTarget.value = t;
  reported.value = null;
}

async function addToPlaylist(trackId: string): Promise<void> {
  if (!playlistPick.value) return;
  try {
    await playlists.addTrack(playlistPick.value, trackId);
    playlistPick.value = '';
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  }
}

onMounted(() => {
  void load();
  if (auth.isAuthenticated) void playlists.fetchMine();
});
</script>
