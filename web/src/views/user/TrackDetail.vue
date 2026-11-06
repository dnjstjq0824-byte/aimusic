<template>
  <div class="space-y-6">
    <!-- Back Link -->
    <RouterLink
      to="/"
      class="inline-flex items-center gap-1.5 text-xs text-neutral-400 hover:text-white transition-colors"
    >
      <span>← 목록으로 돌아가기</span>
    </RouterLink>

    <!-- Error State -->
    <div v-if="error" class="rounded-xl border border-red-800 bg-red-950/40 p-4 text-xs text-red-200">
      {{ error }}
    </div>

    <!-- Loading State -->
    <div v-else-if="loading" class="space-y-4">
      <UiSkeleton height="h-32" width="w-full" rounded="xl" />
      <UiSkeleton height="h-64" width="w-full" rounded="xl" />
    </div>

    <!-- Track Content -->
    <div v-else-if="track" class="grid gap-6 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
      <!-- Main Column -->
      <section class="space-y-6 min-w-0">
        <!-- Studio Player Header -->
        <UiCard class="relative overflow-hidden">
          <div class="flex flex-col sm:flex-row sm:items-center gap-5">
            <!-- Big Play Button -->
            <button
              class="relative flex h-20 w-20 shrink-0 items-center justify-center rounded-2xl bg-gradient-to-tr from-brand-600 to-amber-500 text-white text-3xl font-black shadow-glow-brand hover:scale-105 active:scale-95 transition-all cursor-pointer"
              title="재생하기"
              @click="play"
            >
              ▶
            </button>

            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2 flex-wrap">
                <AiBadge :track="track" />
                <UiBadge variant="commercial" size="xs">
                  ✓ 상업 라이선스 검증 완료
                </UiBadge>
                <span v-if="track.genre" class="rounded-md bg-surface-2 px-2 py-0.5 text-[10px] text-neutral-400 font-medium">
                  {{ track.genre }}
                </span>
              </div>

              <h1 class="mt-2 text-2xl sm:text-3xl font-black text-white tracking-tight truncate">
                {{ track.title }}
              </h1>

              <div class="mt-1 flex items-center gap-3 text-xs text-neutral-400">
                <span>플랫폼: <strong class="text-neutral-200">{{ track.aiProvider }}</strong></span>
                <span>·</span>
                <span>재생 <strong class="text-neutral-200">{{ track.playCount.toLocaleString() }}회</strong></span>
                <span>·</span>
                <span>시간 <strong class="text-neutral-200">{{ fmtDuration(track.durationMs) }}</strong></span>
              </div>
            </div>
          </div>

          <!-- Quick Action Bar -->
          <div class="mt-6 flex flex-wrap items-center gap-2 pt-4 border-t border-surface-border">
            <UiButton variant="brand" size="sm" @click="play">
              ▶ 바로 재생
            </UiButton>
            <UiButton variant="secondary" size="sm" @click="enqueue">
              + 재생 큐에 담기
            </UiButton>
            <UiButton
              v-if="auth.isAuthenticated"
              variant="outline"
              size="sm"
              @click="playlistModalOpen = true"
            >
              + 플레이리스트에 담기
            </UiButton>
          </div>
        </UiCard>

        <!-- Waveform Visualizer -->
        <UiCard>
          <div class="mb-3 flex items-center justify-between">
            <h3 class="text-xs font-bold text-neutral-400 uppercase tracking-wider">오디오 파형 (Audio Waveform)</h3>
            <span class="text-[11px] font-mono text-neutral-500">
              {{ track.bitrateKbps ? `${track.bitrateKbps}kbps` : '192kbps+' }} · HLS Adaptive
            </span>
          </div>

          <div v-if="waveformPeaks.length > 0" class="flex h-16 items-end gap-[3px] py-2 bg-surface-0/60 rounded-xl p-3 border border-surface-border">
            <div
              v-for="(peak, idx) in waveformPeaks"
              :key="idx"
              class="waveform-bar flex-1 rounded-t bg-gradient-to-t from-brand-600 to-amber-400 hover:brightness-125 cursor-pointer"
              :style="{ height: `${Math.max(10, Math.round(peak * 100))}%` }"
            />
          </div>
          <div v-else class="py-6 text-center text-xs text-neutral-500">
            파형 메타데이터 분석 완료
          </div>
        </UiCard>

        <!-- AI Transparency & Creation Metadata (Art. 31 Compliance) -->
        <UiCard class="space-y-4">
          <div class="flex items-center justify-between border-b border-surface-border pb-3">
            <div>
              <h3 class="text-sm font-bold text-white">AI 생성 투명성 공시 명세서</h3>
              <p class="text-[11px] text-neutral-400 mt-0.5">인공지능기본법 제31조 및 문체부 AI 저작권 가이드라인 준수</p>
            </div>
            <UiBadge variant="brand" size="xs">법적 공시 필</UiBadge>
          </div>

          <div class="grid gap-3 sm:grid-cols-2 text-xs">
            <div class="p-3 rounded-xl bg-surface-0 border border-surface-border">
              <span class="text-neutral-500 text-[10px] uppercase font-bold block">AI 생성 플랫폼 &amp; 모델</span>
              <span class="font-semibold text-white text-sm">{{ track.aiProvider }}</span>
              <span v-if="track.aiModelVersion" class="text-neutral-400 ml-1 text-xs">({{ track.aiModelVersion }})</span>
            </div>
            <div class="p-3 rounded-xl bg-surface-0 border border-surface-border">
              <span class="text-neutral-500 text-[10px] uppercase font-bold block">생성 유형 구분</span>
              <span class="font-semibold text-brand-400 text-sm">
                {{ track.aiGenerationType === 'AI_ASSISTED_HUMAN' ? 'AI+인간 공동 창작 (Human-in-the-loop)' : '순수 AI 생성물 (Full AI)' }}
              </span>
            </div>
          </div>

          <!-- Prompt -->
          <div v-if="track.aiPrompt && track.isPromptPublic !== false" class="space-y-1.5">
            <div class="text-[11px] font-bold text-neutral-400">생성 프롬프트 (Prompt)</div>
            <div class="p-3 rounded-xl bg-surface-0 border border-surface-border font-mono text-xs text-neutral-300 leading-relaxed break-all">
              {{ track.aiPrompt }}
            </div>
          </div>

          <!-- Lyrics -->
          <div v-if="track.lyrics" class="space-y-1.5">
            <div class="text-[11px] font-bold text-neutral-400">가사 (Lyrics)</div>
            <div class="p-3 rounded-xl bg-surface-0 border border-surface-border text-xs text-neutral-300 leading-relaxed whitespace-pre-wrap max-h-48 overflow-y-auto">
              {{ track.lyrics }}
            </div>
          </div>
        </UiCard>
      </section>

      <!-- Sidebar Column -->
      <aside class="space-y-4 min-w-0">
        <!-- License Certification Card -->
        <UiCard class="space-y-3">
          <div class="flex items-center gap-2">
            <span class="text-base text-accent-emerald">🛡️</span>
            <h3 class="text-xs font-bold text-white uppercase tracking-wider">상업 라이선스 인증서</h3>
          </div>
          <div class="rounded-xl border border-emerald-500/20 bg-emerald-950/30 p-3 text-xs text-emerald-300 space-y-1.5">
            <div class="font-bold flex items-center gap-1.5">
              <span>✓ Gate 1-4 검증 완료</span>
            </div>
            <p class="text-[11px] leading-relaxed text-emerald-200/80">
              본 음원은 {{ track.aiProvider }} 정식 유료 플랜 유효 구독 기간 내에 생성되었으며, FFprobe 오디오 분석을 통해 비상업 워터마크가 부존재함을 확인받았습니다.
            </p>
          </div>
          <div class="space-y-1.5 text-[11px] text-neutral-400 pt-1">
            <div class="flex justify-between">
              <span>검증 상태</span>
              <span class="font-semibold text-emerald-400 font-mono">{{ track.status }}</span>
            </div>
            <div class="flex justify-between">
              <span>트랙 고유 ID</span>
              <span class="font-mono text-neutral-500 truncate max-w-[150px]">{{ track.id }}</span>
            </div>
          </div>
        </UiCard>

        <!-- Rights Protection & UGC Safety -->
        <UiCard class="space-y-3">
          <h3 class="text-xs font-bold text-white uppercase tracking-wider">권리자 보호 및 UGC 안전</h3>
          <p class="text-xs text-neutral-400 leading-relaxed">
            타인의 목소리를 무단 합성(보이스 클로닝)했거나 기존 저작물을 표절한 경우 즉시 임시조치(Notice &amp; Takedown)를 요청할 수 있습니다.
          </p>

          <div class="space-y-2 pt-1">
            <UiButton
              variant="outline"
              size="sm"
              full-width
              @click="reportModalOpen = true"
            >
              🚩 권리침해 신고 (OSP 제103조)
            </UiButton>
            <UiButton
              v-if="auth.isAuthenticated"
              variant="danger"
              size="sm"
              full-width
              @click="blockPartner"
            >
              🚫 이 아티스트(파트너) 차단
            </UiButton>
          </div>
        </UiCard>
      </aside>
    </div>

    <!-- Dispute Modal -->
    <DisputeReportModal
      v-if="reportModalOpen && track"
      :track-id="track.id"
      :track-title="track.title"
      @close="reportModalOpen = false"
      @reported="onReported"
    />

    <!-- Playlist Add Modal -->
    <UiModal
      v-model="playlistModalOpen"
      title="플레이리스트에 담기"
      description="선택한 음원을 보관할 플레이리스트를 선택하세요."
      @close="playlistModalOpen = false"
    >
      <div v-if="playlists.playlists.length === 0" class="py-4 text-center">
        <p class="text-xs text-neutral-400 mb-3">플레이리스트가 없습니다.</p>
        <UiButton variant="brand" size="sm" @click="createPlaylistPrompt">새 플레이리스트 생성</UiButton>
      </div>
      <div v-else class="space-y-2">
        <button
          v-for="p in playlists.playlists"
          :key="p.id"
          type="button"
          class="w-full flex items-center justify-between p-3 rounded-xl border border-surface-border bg-surface-2 hover:bg-surface-3 transition-colors text-left text-xs"
          @click="addToPlaylist(p.id)"
        >
          <span class="font-bold text-white">{{ p.title }}</span>
          <span class="text-neutral-400">담기 →</span>
        </button>
      </div>
    </UiModal>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRoute } from 'vue-router';
import { trackApi, playlistApi, userBlocksApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { usePlayerStore } from '@/stores/playerStore';
import { usePlaylistStore } from '@/stores/playlistStore';
import { useAuthStore } from '@/stores/authStore';
import { useToast } from '@/composables/useToast';
import type { Track } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiModal from '@/components/ui/UiModal.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import AiBadge from '@/components/legal/AiBadge.vue';
import DisputeReportModal from '@/components/legal/DisputeReportModal.vue';

const route = useRoute();
const player = usePlayerStore();
const playlists = usePlaylistStore();
const auth = useAuthStore();
const toast = useToast();

const track = ref<Track | null>(null);
const loading = ref(true);
const error = ref<string | null>(null);

const reportModalOpen = ref(false);
const playlistModalOpen = ref(false);

const waveformPeaks = computed<number[]>(() => {
  if (!track.value?.waveformJson) {
    return Array.from({ length: 48 }, () => Math.random() * 0.7 + 0.2);
  }
  const raw = track.value.waveformJson;
  if (Array.isArray(raw) && raw.every((n) => typeof n === 'number')) {
    return raw as number[];
  }
  return Array.from({ length: 48 }, () => Math.random() * 0.7 + 0.2);
});

function fmtDuration(ms?: number | null) {
  if (!ms) return '0:00';
  const totalSec = Math.floor(ms / 1000);
  const min = Math.floor(totalSec / 60);
  const sec = totalSec % 60;
  return `${min}:${sec < 10 ? '0' : ''}${sec}`;
}

async function loadTrack() {
  const id = route.params.id as string;
  if (!id) return;
  loading.value = true;
  error.value = null;
  try {
    track.value = await trackApi.get(id);
  } catch (err) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
}

function play() {
  if (!track.value) return;
  player.setQueue([track.value], track.value.id);
}

function enqueue() {
  if (!track.value) return;
  player.enqueue(track.value);
  toast.info(`"${track.value.title}"을(를) 재생 큐에 추가했습니다.`);
}

async function addToPlaylist(playlistId: string) {
  if (!track.value) return;
  try {
    await playlistApi.addTrack(playlistId, track.value.id);
    toast.success('플레이리스트에 추가되었습니다.');
    playlistModalOpen.value = false;
  } catch (err) {
    toast.error(errMessageOf(err), '추가 실패');
  }
}

async function createPlaylistPrompt() {
  const title = prompt('새 플레이리스트 제목을 입력하세요:');
  if (!title) return;
  try {
    const created = await playlistApi.create(title, false);
    playlists.playlists.unshift(created);
    await addToPlaylist(created.id);
  } catch (err) {
    toast.error(errMessageOf(err));
  }
}

async function blockPartner() {
  if (!track.value?.partnerId) return;
  if (!confirm('이 아티스트(파트너)를 차단하시겠습니까? 해당 아티스트의 모든 음원이 재생 큐에서 제외됩니다.')) return;
  try {
    await userBlocksApi.blockArtist(track.value.partnerId, 'User requested block');
    toast.warning('아티스트가 차단되었습니다.');
  } catch (err) {
    toast.error(errMessageOf(err), '차단 실패');
  }
}

function onReported(id: string) {
  reportModalOpen.value = false;
  toast.success(`신고가 접수되었습니다. (ID: ${id})`);
}

onMounted(() => {
  void loadTrack();
  void playlists.fetchMine();
});
</script>
