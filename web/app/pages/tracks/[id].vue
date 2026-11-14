<template>
  <div class="space-y-6">
    <!-- Back Link -->
    <RouterLink
      to="/"
      class="inline-flex items-center gap-1.5 text-xs text-neutral-400 hover:text-white transition-colors"
    >
      <span>← 목록으로 돌아가기</span>
    </RouterLink>

    <h1 v-if="error" class="sr-only">음원을 불러오지 못했습니다</h1>
    <UiErrorBanner
      :message="error"
      title="음원을 불러오지 못했습니다"
      retry-label="다시 시도"
      @retry="loadTrack"
    />

    <!-- Loading State -->
    <div v-if="!error && loading" class="space-y-4">
      <UiSkeleton height="h-32" width="w-full" rounded="xl" />
      <UiSkeleton height="h-64" width="w-full" rounded="xl" />
    </div>

    <!-- Track Content -->
    <div v-else-if="!error && track" class="grid gap-6 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
      <!-- Main Column -->
      <section class="space-y-6 min-w-0">
        <!-- Studio Player Header -->
        <UiCard class="relative overflow-hidden">
          <div class="flex flex-col sm:flex-row sm:items-center gap-5">
            <div class="relative shrink-0">
              <UiCoverArt :seed="track.id" :title="track.title" size="xl" rounded="xl" :verified="track.status === 'READY'" />
              <button
                type="button"
                class="absolute inset-0 flex items-center justify-center rounded-2xl bg-black/40 text-white opacity-0 transition-opacity hover:opacity-100 focus:opacity-100"
                :aria-label="`「${track.title}」 재생`"
                :disabled="track.status !== 'READY'"
                @click="play"
              >
                <svg class="h-10 w-10" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                  <path d="M8 5v14l11-7z" />
                </svg>
              </button>
            </div>

            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2 flex-wrap">
                <AiBadge :track="track" />
                <UiBadge v-if="track.genre" variant="neutral" size="xs">
                  {{ track.genre }}
                </UiBadge>
              </div>

              <h1 class="font-display mt-2 text-2xl sm:text-3xl font-black text-white tracking-tight break-words">
                {{ track.title }}
              </h1>

              <div class="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-neutral-400">
                <span>플랫폼: <strong class="text-neutral-200">{{ track.aiProvider }}</strong></span>
                <span>·</span>
                <span>재생 <strong class="text-neutral-200">{{ track.playCount.toLocaleString() }}회</strong></span>
                <span>·</span>
                <span>시간 <strong class="text-neutral-200">{{ fmtDuration(track.durationMs) }}</strong></span>
                <template v-if="track.bpm">
                  <span>·</span>
                  <span>BPM <strong class="text-neutral-200">{{ fmtBpm(track.bpm) }}</strong></span>
                </template>
                <template v-if="track.musicalKey">
                  <span>·</span>
                  <span>키 <strong class="text-neutral-200">{{ track.musicalKey }}</strong></span>
                </template>
                <template v-if="track.isrc">
                  <span>·</span>
                  <span>ISRC <strong class="font-mono text-neutral-200">{{ track.isrc }}</strong></span>
                </template>
              </div>
            </div>
          </div>

          <!-- Quick Action Bar -->
          <div class="mt-6 flex flex-wrap items-center gap-2 pt-4 border-t border-surface-border">
            <UiButton variant="brand" size="sm" :disabled="track.status !== 'READY'" @click="play">
              바로 재생
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
            <UiButton
              v-if="auth.isAuthenticated"
              variant="secondary"
              size="sm"
              :disabled="track.status !== 'READY'"
              :loading="downloading"
              @click="downloadTrack"
            >
              다운로드 (유료)
            </UiButton>
          </div>
        </UiCard>

        <!-- Waveform Visualizer -->
        <UiCard>
          <div class="mb-3 flex items-center justify-between">
            <h3 class="text-xs font-bold text-neutral-400 uppercase tracking-wider">오디오 파형 (Audio Waveform)</h3>
            <span class="text-xs font-mono text-neutral-400">스트리밍 SQ 128k · HQ 256k(이용권)</span>
          </div>

          <div
            v-if="waveformPeaks.length > 0"
            class="flex h-16 items-end gap-0.5 sm:gap-[3px] py-2 bg-surface-0/60 rounded-xl p-3 border border-surface-border"
            role="img"
            aria-label="오디오 파형"
          >
            <div
              v-for="(peak, idx) in waveformPeaks"
              :key="idx"
              class="waveform-bar min-w-[2px] flex-1 rounded-t bg-gradient-to-t from-brand-600 to-amber-400"
              :style="{ height: `${Math.max(10, Math.round(peak * 100))}%` }"
            />
          </div>
          <div v-else class="py-6 text-center text-xs text-neutral-500">
            이 음원의 파형 데이터가 아직 없습니다.
          </div>
        </UiCard>

        <!-- AI Transparency & Creation Metadata (Art. 31 Compliance) -->
        <UiCard class="space-y-4">
          <div class="flex items-start justify-between gap-3 border-b border-surface-border pb-3">
            <div class="min-w-0">
              <h3 class="text-sm font-bold text-white">AI 생성 투명성 공시 명세서</h3>
              <p class="text-[11px] text-neutral-400 mt-0.5">인공지능기본법 제31조 취지에 따른 AI 생성 표시</p>
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
            <div v-if="aiCreditKo" class="p-3 rounded-xl bg-surface-0 border border-surface-border sm:col-span-2">
              <span class="text-neutral-500 text-[10px] uppercase font-bold block">AI 생성 범위 (AI 크레딧)</span>
              <span class="font-semibold text-white text-sm">{{ aiCreditKo }}</span>
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
            <svg class="h-4 w-4 text-accent-emerald" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <path d="M12 3 5 6v5c0 4.4 3 8.4 7 10 4-1.6 7-5.6 7-10V6l-7-3z" />
              <path d="m9 12 2 2 4-4" />
            </svg>
            <h3 class="text-xs font-bold text-white uppercase tracking-wider">
              {{ cert ? '상업적 이용권 보유 인증' : '권리 확인 상태' }}
            </h3>
          </div>
          <div v-if="!cert" class="rounded-xl border border-surface-border bg-surface-2 p-3 text-xs leading-relaxed text-neutral-300">
            곡별 인증서 발급 전입니다. 이 음원은 플랫폼 확인 절차(Gate 1-4)를 거쳐 재생 가능 상태로 등록되었습니다.
            이 표시는 파트너 신고와 자동 검사에 근거하며 권리 관계를 보증하지 않습니다.
          </div>
          <div v-else class="rounded-xl border border-emerald-500/20 bg-emerald-950/30 p-3 text-xs text-emerald-300 space-y-1.5">
            <div class="font-bold flex items-center gap-1.5">
              <UiBadge variant="success" size="xs">Gate 1-4 확인 절차 통과</UiBadge>
            </div>
            <p class="text-[11px] leading-relaxed text-emerald-200/80">
              본 음원은 {{ track.aiProvider }} 유료 플랜 구독 증빙 대조와 FFprobe 자동 검사를 거쳤으며,
              자동 검사에서 비상업 표지가 발견되지 않았습니다.
              본 표시는 파트너 신고와 자동 검사에 근거하며 권리 관계를 보증하지 않습니다.
            </p>
          </div>
          <div class="space-y-1.5 text-[11px] text-neutral-400 pt-1">
            <div class="flex justify-between">
              <span>검증 상태</span>
              <span class="font-semibold text-emerald-400">{{ statusKo }}</span>
            </div>
            <div v-if="cert" class="flex items-center justify-between gap-2">
              <span>인증서 번호</span>
              <RouterLink
                :to="`/certificate/${cert.certificate.certNumber}`"
                class="font-mono font-semibold text-brand-400 hover:underline"
              >
                {{ cert.certificate.certNumber }}
              </RouterLink>
            </div>
            <div v-else-if="canIssueCert" class="pt-1">
              <UiButton variant="secondary" size="xs" :loading="issuingCert" @click="issueCert">
                곡별 인증서 발급
              </UiButton>
            </div>
            <div class="space-y-1">
              <div class="flex items-center justify-between gap-2">
                <span>트랙 고유 ID</span>
                <button
                  type="button"
                  class="shrink-0 font-semibold text-brand-400 hover:underline"
                  @click="copyTrackId"
                >
                  복사
                </button>
              </div>
              <div class="break-all font-mono text-neutral-300">{{ track.id }}</div>
            </div>
          </div>
        </UiCard>

        <!-- Rights Protection & UGC Safety -->
        <UiCard class="space-y-3">
          <h3 class="text-xs font-bold text-white uppercase tracking-wider">권리자 보호 및 UGC 안전</h3>
          <p class="text-xs text-neutral-400 leading-relaxed">
            타인의 목소리를 무단 합성(보이스 클로닝)했거나 기존 저작물을 표절한 경우 임시조치(Notice &amp; Takedown)를 요청할 수 있습니다. 신고는 검토 후 처리됩니다.
          </p>

          <div class="space-y-2 pt-1">
            <UiButton
              variant="outline"
              size="sm"
              full-width
              @click="openReport"
            >
              권리침해 신고 (OSP 제103조)
            </UiButton>
            <UiButton
              v-if="auth.isAuthenticated"
              variant="danger"
              size="sm"
              full-width
              @click="blockConfirmOpen = true"
            >
              이 아티스트(파트너) 차단
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
    <UiPlaylistPicker
      v-model="playlistModalOpen"
      :track-title="track?.title ?? ''"
      :playlists="playlists.playlists"
      :create-loading="creatingPlaylist"
      @select="addToPlaylist"
      @create="createPlaylist"
    />

    <!-- Artist Block Confirm -->
    <UiConfirmDialog
      v-model="blockConfirmOpen"
      title="아티스트 차단"
      description="차단한 아티스트의 모든 음원이 재생 큐에서 제외됩니다."
      body="차단은 권리보호센터에서 언제든 해제할 수 있습니다. 정말 차단하시겠습니까?"
      confirm-label="차단하기"
      danger
      :loading="blocking"
      @confirm="confirmBlockPartner"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { trackApi, playlistApi, userBlocksApi, adminApi } from '@/api/endpoints';
import { errCodeOf, errMessageOf } from '@/api/client';
import { usePlayerStore } from '@/stores/playerStore';
import { usePlaylistStore } from '@/stores/playlistStore';
import { useAuthStore } from '@/stores/authStore';
import { useToast } from '@/composables/useToast';
import type { Track, CertificateView } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiCoverArt from '@/components/ui/UiCoverArt.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiPlaylistPicker from '@/components/ui/UiPlaylistPicker.vue';
import UiConfirmDialog from '@/components/ui/UiConfirmDialog.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import AiBadge from '@/components/legal/AiBadge.vue';
import DisputeReportModal from '@/components/legal/DisputeReportModal.vue';

const route = useRoute();
const router = useRouter();
const player = usePlayerStore();
const playlists = usePlaylistStore();
const auth = useAuthStore();
const toast = useToast();

const track = ref<Track | null>(null);
const loading = ref(true);
const error = ref<string | null>(null);
const cert = ref<CertificateView | null>(null);
const issuingCert = ref(false);
const downloading = ref(false);

const canIssueCert = computed(
  () => auth.isAdmin && track.value?.status === 'READY' && cert.value === null,
);

const reportModalOpen = ref(false);
const playlistModalOpen = ref(false);
const creatingPlaylist = ref(false);
const blockConfirmOpen = ref(false);
const blocking = ref(false);

const STATUS_KO: Record<string, string> = {
  UPLOADED: '업로드됨',
  TRANSCODING: '변환 중',
  LICENSE_GATE_PENDING: '심사 대기',
  LICENSE_GATE_REJECTED: '게이트 반려',
  READY: '재생 가능',
  TAKEN_DOWN: '임시조치 중',
};

const statusKo = computed(() => {
  if (!track.value) return '';
  return STATUS_KO[track.value.status] ?? '처리 중';
});

const AI_CREDIT_KO: Record<string, string> = { LYRICS: '가사', VOCALS: '보컬', INSTRUMENTAL: '반주' };

const aiCreditKo = computed(() => {
  const parts = track.value?.aiCreditParts ?? [];
  if (parts.length === 0) return '';
  const named = parts.map((p) => AI_CREDIT_KO[p] ?? p);
  const scope = parts.length >= 3 ? '전체 AI 생성' : '부분 AI 생성';
  return `${scope} (${named.join('·')})`;
});

function toPeaks(raw: unknown): number[] {
  if (Array.isArray(raw) && raw.length > 0 && raw.every((n) => typeof n === 'number')) {
    return raw as number[];
  }
  if (typeof raw === 'string' && raw.length > 0) {
    try {
      return toPeaks(JSON.parse(raw) as unknown);
    } catch {
      return [];
    }
  }
  return [];
}

// At most WAVEFORM_BARS bars (bucket max) so every bar keeps a visible
// width on a 320-390px phone (U-01).
const WAVEFORM_BARS = 48;
function downsample(peaks: number[], n: number): number[] {
  if (peaks.length <= n) return peaks;
  const out: number[] = [];
  for (let i = 0; i < n; i++) {
    const from = Math.floor((i * peaks.length) / n);
    const to = Math.max(from + 1, Math.floor(((i + 1) * peaks.length) / n));
    out.push(Math.max(...peaks.slice(from, to)));
  }
  return out;
}
const waveformPeaks = computed<number[]>(() => downsample(toPeaks(track.value?.waveformJson), WAVEFORM_BARS));

// EV-P03: BPM to integer or 1 decimal (never float32 artifacts).
function fmtBpm(bpm?: number | null): string {
  if (bpm === null || bpm === undefined) return '';
  return String(Math.round(bpm * 10) / 10);
}

async function copyTrackId(): Promise<void> {
  if (!track.value) return;
  try {
    await navigator.clipboard.writeText(track.value.id);
    toast.success('트랙 ID를 복사했습니다.');
  } catch {
    toast.error('클립보드 복사에 실패했습니다.');
  }
}

function fmtDuration(ms?: number | null) {
  if (!ms) return '0:00';
  const totalSec = Math.round(ms / 1000);
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
    cert.value = await trackApi.getCertificate(id).catch(() => null);
  } catch (err) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
}

async function issueCert(): Promise<void> {
  if (!track.value) return;
  issuingCert.value = true;
  try {
    await adminApi.issueTrackCertificate(track.value.id);
    cert.value = await trackApi.getCertificate(track.value.id).catch(() => null);
    toast.success('곡별 인증서가 발급되었습니다.');
  } catch (err) {
    toast.error(errMessageOf(err), '발급 실패');
  } finally {
    issuingCert.value = false;
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

async function downloadTrack(): Promise<void> {
  if (!track.value) return;
  downloading.value = true;
  try {
    const { blob, filename } = await trackApi.download(track.value.id);
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    a.remove();
    URL.revokeObjectURL(url);
    toast.success('다운로드가 시작되었습니다.');
  } catch (err) {
    const code = errCodeOf(err);
    if (code === 'ERR_DOWNLOAD_REQUIRES_PASS') {
      toast.error('다운로드는 유료 이용권이 필요합니다.', '유료 전용');
    } else if (code === 'ERR_DOWNLOAD_QUOTA') {
      toast.error('일일 다운로드 한도(50회)를 초과했습니다.', '한도 초과');
    } else {
      toast.error(errMessageOf(err), '다운로드 실패');
    }
  } finally {
    downloading.value = false;
  }
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

async function createPlaylist(title: string) {
  creatingPlaylist.value = true;
  try {
    const created = await playlistApi.create(title, false);
    playlists.playlists.unshift(created);
    await addToPlaylist(created.id);
  } catch (err) {
    toast.error(errMessageOf(err));
  } finally {
    creatingPlaylist.value = false;
  }
}

async function confirmBlockPartner() {
  if (!track.value?.partnerId) return;
  blocking.value = true;
  try {
    await userBlocksApi.blockArtist(track.value.partnerId, 'User requested block');
    toast.warning('아티스트가 차단되었습니다.');
    blockConfirmOpen.value = false;
  } catch (err) {
    toast.error(errMessageOf(err), '차단 실패');
  } finally {
    blocking.value = false;
  }
}

function openReport() {
  if (!auth.isAuthenticated) {
    toast.info('신고하려면 먼저 로그인하세요.');
    void router.push({ name: 'login', query: { redirect: route.fullPath } });
    return;
  }
  reportModalOpen.value = true;
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
