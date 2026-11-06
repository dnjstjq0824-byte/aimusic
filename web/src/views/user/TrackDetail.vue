<template>
  <div>
    <RouterLink to="/" class="text-sm text-neutral-400 underline">← 목록으로</RouterLink>
    <p v-if="error" class="mt-3 text-sm text-red-400">{{ error }}</p>
    <div v-else-if="track" class="mt-3 grid gap-6 md:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
      <section class="min-w-0">
        <div class="flex items-center gap-4">
          <button
            class="flex h-16 w-16 shrink-0 items-center justify-center rounded-xl bg-brand-600 text-2xl font-bold"
            title="재생"
            @click="play"
          >
            ▶
          </button>
          <div class="min-w-0">
            <h1 class="truncate text-2xl font-bold">{{ track.title }}</h1>
            <div class="mt-1"><AiBadge :track="track" /></div>
          </div>
        </div>
        <dl class="mt-4 grid grid-cols-2 gap-2 text-sm sm:grid-cols-4">
          <div class="rounded bg-neutral-900 p-2">
            <dt class="text-xs text-neutral-400">AI 플랫폼</dt>
            <dd class="font-semibold">{{ track.aiProvider }}</dd>
          </div>
          <div class="rounded bg-neutral-900 p-2">
            <dt class="text-xs text-neutral-400">재생 수</dt>
            <dd class="font-semibold">{{ track.playCount }}</dd>
          </div>
          <div class="rounded bg-neutral-900 p-2">
            <dt class="text-xs text-neutral-400">길이</dt>
            <dd class="font-semibold">{{ fmtDuration(track.durationMs) }}</dd>
          </div>
          <div class="rounded bg-neutral-900 p-2">
            <dt class="text-xs text-neutral-400">비트레이트</dt>
            <dd class="font-semibold">{{ track.bitrateKbps ? `${track.bitrateKbps}kbps` : '–' }}</dd>
          </div>
        </dl>
        <div v-if="track.aiDisclosureText" class="mt-4 rounded bg-neutral-900 p-3 text-sm">
          <h2 class="mb-1 text-xs font-bold text-neutral-400">AI 생성 정보 (AI 기본법 제31조 준수)</h2>
          <p class="whitespace-pre-wrap">{{ track.aiDisclosureText }}</p>
          <p v-if="track.aiModelVersion" class="mt-1 text-xs text-neutral-400">모델 버전: {{ track.aiModelVersion }}</p>
        </div>
        <div v-if="waveformPeaks.length > 0" class="mt-4 rounded bg-neutral-900 p-3">
          <h2 class="mb-2 text-xs font-bold text-neutral-400">오디오 파형 (Waveform)</h2>
          <div class="flex h-12 items-end gap-[2px]">
            <div
              v-for="(peak, idx) in waveformPeaks"
              :key="idx"
              class="flex-1 rounded-t bg-brand-500/80"
              :style="{ height: `${Math.max(12, Math.round(peak * 100))}%` }"
            />
          </div>
        </div>
        <div v-if="track.aiPrompt && track.isPromptPublic !== false" class="mt-4 rounded bg-neutral-900 p-3 text-sm">
          <h2 class="mb-1 text-xs font-bold text-neutral-400">생성 프롬프트</h2>
          <p class="whitespace-pre-wrap font-mono text-xs text-neutral-200">{{ track.aiPrompt }}</p>
        </div>
        <div v-if="track.lyrics" class="mt-4 rounded bg-neutral-900 p-3 text-sm">
          <h2 class="mb-1 text-xs font-bold text-neutral-400">가사 (Lyrics)</h2>
          <p class="whitespace-pre-wrap text-neutral-200">{{ track.lyrics }}</p>
        </div>
        <div class="mt-4 flex flex-wrap gap-2">
          <button class="rounded bg-brand-600 px-4 py-1.5 text-sm font-bold" @click="play">재생</button>
          <button class="rounded bg-neutral-800 px-4 py-1.5 text-sm" @click="enqueue">+ 재생목록에 추가</button>
        </div>
      </section>
      <aside class="min-w-0 space-y-4">
        <div class="rounded bg-neutral-900 p-3 text-sm">
          <h2 class="mb-1 text-xs font-bold text-neutral-400">상업 라이선스 증명 요약</h2>
          <p v-if="track.status === 'READY'" class="text-emerald-400">
            ✓ 상업 라이선스 검증 완료 ({{ track.aiProvider }} 유료 상업 구독 기간 내 생성 · Gate 1-4 통과)
          </p>
          <p v-else class="text-neutral-400">상태: {{ track.status }}</p>
        </div>
        <div v-if="auth.isAuthenticated" class="rounded bg-neutral-900 p-3 text-sm">
          <h2 class="mb-2 text-xs font-bold text-neutral-400">플레이리스트에 담기</h2>
          <div class="flex gap-2">
            <select v-model="playlistPick" class="min-w-0 flex-1 rounded bg-neutral-800 p-1.5">
              <option value="">선택…</option>
              <option v-for="p in playlists.playlists" :key="p.id" :value="p.id">{{ p.title }}</option>
            </select>
            <button class="rounded bg-neutral-700 px-3 py-1.5 disabled:opacity-50" :disabled="!playlistPick" @click="addToPlaylist">
              담기
            </button>
          </div>
          <p v-if="added" class="mt-1 text-xs text-emerald-400">담았습니다.</p>
        </div>
        <div class="space-y-2 rounded bg-neutral-900 p-3 text-sm">
          <h2 class="mb-1 text-xs font-bold text-neutral-400">권리 보호 및 UGC 안전</h2>
          <div class="flex flex-wrap gap-2">
            <button class="rounded bg-neutral-800 px-3 py-1.5" @click="reportTarget = track">권리침해 신고</button>
            <button
              v-if="auth.isAuthenticated"
              class="rounded bg-red-900/50 px-3 py-1.5 text-red-200"
              @click="blockPartner"
            >
              이 아티스트(파트너) 차단
            </button>
          </div>
          <p v-if="artistBlocked" class="text-xs text-amber-400">해당 파트너가 차단되어 재생 큐에서 제외되었습니다.</p>
        </div>
      </aside>
    </div>
    <p v-else class="mt-3 text-sm text-neutral-400">불러오는 중…</p>
    <DisputeReportModal
      v-if="reportTarget"
      :track-id="reportTarget.id"
      :track-title="reportTarget.title"
      @close="reportTarget = null"
      @reported="onReported"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute } from 'vue-router';
import { trackApi, userBlocksApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { usePlayerStore } from '@/stores/playerStore';
import { usePlaylistStore } from '@/stores/playlistStore';
import { useAuthStore } from '@/stores/authStore';
import type { Track } from '@/api/types';
import AiBadge from '@/components/legal/AiBadge.vue';
import DisputeReportModal from '@/components/legal/DisputeReportModal.vue';

const route = useRoute();
const player = usePlayerStore();
const playlists = usePlaylistStore();
const auth = useAuthStore();

const track = ref<Track | null>(null);
const error = ref<string | null>(null);
const playlistPick = ref('');
const added = ref(false);
const artistBlocked = ref(false);
const reportTarget = ref<Track | null>(null);

const waveformPeaks = computed<number[]>(() => {
  if (!track.value?.waveformJson) return [];
  try {
    const parsed = JSON.parse(track.value.waveformJson);
    return Array.isArray(parsed) ? parsed.map((n) => Number(n) || 0.3) : [];
  } catch {
    return [];
  }
});

async function blockPartner(): Promise<void> {
  if (!track.value) return;
  try {
    await userBlocksApi.blockArtist(track.value.partnerId, '사용자 직접 아티스트 차단');
    artistBlocked.value = true;
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  }
}

async function load(id: string): Promise<void> {
  error.value = null;
  track.value = null;
  try {
    track.value = await trackApi.get(id);
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  }
}

function fmtDuration(ms?: number | null): string {
  if (!ms || ms <= 0) return '–';
  const s = Math.floor(ms / 1000);
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`;
}

function play(): void {
  if (track.value) player.setQueue([track.value], track.value.id);
}

function enqueue(): void {
  if (track.value) player.enqueue(track.value);
}

async function addToPlaylist(): Promise<void> {
  if (!track.value || !playlistPick.value) return;
  try {
    await playlists.addTrack(playlistPick.value, track.value.id);
    playlistPick.value = '';
    added.value = true;
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  }
}

function onReported(): void {
  reportTarget.value = null;
}

onMounted(() => {
  void load(route.params.id as string);
  if (auth.isAuthenticated) void playlists.fetchMine();
});

watch(
  () => route.params.id,
  (id) => {
    if (typeof id === 'string') void load(id);
  },
);
</script>
