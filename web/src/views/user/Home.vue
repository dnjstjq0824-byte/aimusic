<template>
  <div class="space-y-10">
    <!-- Hero Banner (Linear / Apple aesthetic) -->
    <div class="relative overflow-hidden rounded-3xl border border-surface-border bg-gradient-to-br from-surface-1 via-surface-0 to-surface-1 p-6 sm:p-10 shadow-2xl">
      <!-- Glow background circles -->
      <div class="pointer-events-none absolute -right-20 -top-20 h-64 w-64 rounded-full bg-brand-500/15 blur-3xl" />
      <div class="pointer-events-none absolute -left-20 -bottom-20 h-64 w-64 rounded-full bg-accent-purple/15 blur-3xl" />

      <div class="relative z-10 max-w-2xl space-y-4">
        <div class="inline-flex items-center gap-2 rounded-full border border-brand-500/30 bg-brand-950/60 px-3 py-1 text-xs text-brand-300">
          <span class="h-2 w-2 rounded-full bg-brand-500 animate-pulse" />
          <span>차세대 상업용 AI 음원 스트리밍 플랫폼</span>
        </div>

        <h1 class="text-3xl sm:text-5xl font-black text-white tracking-tight leading-tight">
          저작권 분쟁 없는 <br class="hidden sm:inline" />
          <span class="bg-gradient-to-r from-brand-400 via-brand-500 to-amber-300 bg-clip-text text-transparent">
            검증된 AI 음악
          </span>을 듣다
        </h1>

        <p class="text-xs sm:text-sm text-neutral-300 leading-relaxed">
          Suno Pro, AIVA Pro 등 유료 상업 플랜에서 생성된 음원만 4단계 게이트키퍼(Gate 1~4)로 엄격히 검증하여 서비스합니다. 무료 플랜 및 비상업용 음원은 원천 차단됩니다.
        </p>

        <div class="flex flex-wrap items-center gap-3 pt-2">
          <UiButton
            variant="brand"
            size="lg"
            :disabled="tracks.length === 0"
            @click="playAll"
          >
            ▶ 전체 음원 스트리밍
          </UiButton>
          <RouterLink to="/explore">
            <UiButton variant="secondary" size="lg">
              플랫폼별 탐색
            </UiButton>
          </RouterLink>
          <button
            type="button"
            class="text-xs text-neutral-400 hover:text-white underline ml-2 transition-colors"
            @click="onboardingOpen = true"
          >
            플랫폼 안전 규정 안내 →
          </button>
        </div>
      </div>

      <!-- Trust Metrics Bar -->
      <div class="mt-8 grid grid-cols-2 gap-3 sm:grid-cols-4 pt-6 border-t border-surface-border/60 text-xs">
        <div class="space-y-0.5">
          <div class="text-[10px] uppercase font-bold text-neutral-500">라이선스 기준</div>
          <div class="font-bold text-emerald-400">100% 상업용 전용</div>
        </div>
        <div class="space-y-0.5">
          <div class="text-[10px] uppercase font-bold text-neutral-500">법적 컴플라이언스</div>
          <div class="font-bold text-white">인공지능기본법 제31조</div>
        </div>
        <div class="space-y-0.5">
          <div class="text-[10px] uppercase font-bold text-neutral-500">음향 포렌식</div>
          <div class="font-bold text-white">FFprobe 192k+ 검증</div>
        </div>
        <div class="space-y-0.5">
          <div class="text-[10px] uppercase font-bold text-neutral-500">백그라운드 스트리밍</div>
          <div class="font-bold text-brand-400">HLS 무중단 Gapless</div>
        </div>
      </div>
    </div>

    <!-- Quick Curation Chips -->
    <div class="flex items-center justify-between gap-4">
      <div class="flex flex-wrap items-center gap-2">
        <button
          type="button"
          class="rounded-xl px-3.5 py-1.5 text-xs font-semibold transition-all duration-150"
          :class="!filterProvider && !filterMood ? 'bg-white text-neutral-900 shadow font-bold' : 'bg-surface-1 text-neutral-300 hover:bg-surface-2 border border-surface-border'"
          @click="clearFilters"
        >
          전체 실시간
        </button>
        <button
          v-for="prov in ['SUNO', 'UDIO', 'AIVA', 'STABLE_AUDIO']"
          :key="prov"
          type="button"
          class="rounded-xl px-3.5 py-1.5 text-xs font-semibold transition-all duration-150"
          :class="filterProvider === prov ? 'bg-brand-500 text-white shadow-glow-brand' : 'bg-surface-1 text-neutral-300 hover:bg-surface-2 border border-surface-border'"
          @click="toggleProvider(prov)"
        >
          {{ prov }}
        </button>
        <button
          v-for="mood in ['Focus', 'Chill', 'Cyberpunk', 'Cinematic']"
          :key="mood"
          type="button"
          class="rounded-xl px-3.5 py-1.5 text-xs font-semibold transition-all duration-150"
          :class="filterMood === mood ? 'bg-brand-500 text-white shadow-glow-brand' : 'bg-surface-1 text-neutral-300 hover:bg-surface-2 border border-surface-border'"
          @click="toggleMood(mood)"
        >
          #{{ mood }}
        </button>
      </div>

      <RouterLink to="/explore" class="shrink-0 text-xs font-semibold text-brand-400 hover:underline">
        상세 필터 →
      </RouterLink>
    </div>

    <!-- Ready Tracks Section Header -->
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-2">
        <h2 class="text-xl font-bold text-white tracking-tight">검증 음원 목록</h2>
        <span class="rounded-full bg-surface-2 px-2.5 py-0.5 text-xs text-neutral-400 font-mono">
          {{ tracks.length }}
        </span>
      </div>
      <div class="text-xs text-neutral-400">
        최근 승인된 상업 라이선스 트랙
      </div>
    </div>

    <!-- Loading Skeleton -->
    <div v-if="loading" class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <UiCard v-for="i in 6" :key="i" class="space-y-3">
        <div class="flex items-center gap-3">
          <UiSkeleton height="h-12" width="w-12" rounded="lg" />
          <div class="flex-1 space-y-1.5">
            <UiSkeleton height="h-4" width="w-3/4" />
            <UiSkeleton height="h-3" width="w-1/2" />
          </div>
        </div>
      </UiCard>
    </div>

    <!-- Empty State -->
    <div v-else-if="tracks.length === 0" class="py-12">
      <UiEmptyState
        icon="🎵"
        title="등록된 검증 음원이 없습니다"
        description="조건에 맞는 음원이 아직 없거나 모든 필터가 해제되었습니다. 파트너 스튜디오에서 새로운 상업용 음원을 등록해 보세요."
      >
        <template #action>
          <div class="flex gap-2">
            <UiButton variant="secondary" size="sm" @click="clearFilters">필터 초기화</UiButton>
            <RouterLink to="/partner/upload">
              <UiButton variant="brand" size="sm">음원 업로드</UiButton>
            </RouterLink>
          </div>
        </template>
      </UiEmptyState>
    </div>

    <!-- Tracks Grid -->
    <div v-else class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
      <UiCard
        v-for="t in tracks"
        :key="t.id"
        hoverable
        class="group flex flex-col justify-between"
      >
        <div>
          <div class="flex items-start gap-3">
            <!-- Play trigger button -->
            <button
              class="relative flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-surface-2 group-hover:bg-brand-500 text-neutral-300 group-hover:text-white transition-all shadow-md active:scale-95 cursor-pointer"
              title="재생"
              @click="playOne(t)"
            >
              <span class="text-base font-black">▶</span>
            </button>
            <div class="min-w-0 flex-1">
              <RouterLink
                :to="{ name: 'track-detail', params: { id: t.id } }"
                class="block truncate font-bold text-white hover:text-brand-400 transition-colors text-sm"
              >
                {{ t.title }}
              </RouterLink>
              <div class="mt-1 flex flex-wrap items-center gap-1.5">
                <AiBadge :track="t" />
                <span class="text-[11px] font-medium text-neutral-400">{{ t.aiProvider }}</span>
              </div>
            </div>
          </div>

          <!-- Metadata row -->
          <div class="mt-3 flex items-center justify-between text-[11px] text-neutral-400 pt-2 border-t border-surface-border">
            <span>재생 {{ t.playCount.toLocaleString() }}회</span>
            <span v-if="t.durationMs">{{ fmtDuration(t.durationMs) }}</span>
            <span v-if="t.bitrateKbps">{{ t.bitrateKbps }}kbps</span>
          </div>
        </div>

        <!-- Action Strip -->
        <div class="mt-3 flex items-center justify-between gap-1 pt-2">
          <button
            class="text-[11px] font-medium text-neutral-400 hover:text-white transition-colors"
            @click="enqueue(t)"
          >
            + 큐에 추가
          </button>
          <button
            class="text-[11px] font-medium text-neutral-400 hover:text-brand-400 transition-colors"
            @click="openAddToPlaylist(t)"
          >
            + 플레이리스트
          </button>
          <button
            class="text-[11px] font-medium text-neutral-500 hover:text-accent-crimson transition-colors"
            @click="openReport(t)"
          >
            신고
          </button>
        </div>
      </UiCard>
    </div>

    <!-- Pagination -->
    <UiPagination
      v-if="tracks.length > 0"
      :page="Math.floor(offset / limit) + 1"
      :has-more="tracks.length === limit"
      :disabled="loading"
      @prev="page(-1)"
      @next="page(1)"
    />

    <!-- Dispute Modal -->
    <DisputeReportModal
      v-if="reportTarget"
      :track-id="reportTarget.id"
      :track-title="reportTarget.title"
      @close="reportTarget = null"
      @reported="onReported"
    />

    <!-- Playlist Add Modal -->
    <UiModal
      :model-value="Boolean(playlistTargetTrack)"
      title="내 플레이리스트에 담기"
      description="선택한 음원을 추가할 플레이리스트를 선택하세요."
      @close="playlistTargetTrack = null"
    >
      <div v-if="playlists.playlists.length === 0" class="py-4 text-center">
        <p class="text-xs text-neutral-400 mb-3">생성된 플레이리스트가 없습니다.</p>
        <UiButton variant="brand" size="sm" @click="createNewPlaylist">새 플레이리스트 생성</UiButton>
      </div>
      <div v-else class="space-y-2">
        <button
          v-for="p in playlists.playlists"
          :key="p.id"
          type="button"
          class="w-full flex items-center justify-between p-3 rounded-xl border border-surface-border bg-surface-2 hover:bg-surface-3 transition-colors text-left text-xs"
          @click="addTrackToPlaylist(p.id)"
        >
          <span class="font-bold text-white">{{ p.title }}</span>
          <span class="text-neutral-400">담기 →</span>
        </button>
      </div>
    </UiModal>

    <!-- Compliance Onboarding Modal -->
    <OnboardingModal v-model="onboardingOpen" />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { trackApi, playlistApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { usePlayerStore } from '@/stores/playerStore';
import { usePlaylistStore } from '@/stores/playlistStore';
import { useToast } from '@/composables/useToast';
import type { Track } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiModal from '@/components/ui/UiModal.vue';
import UiPagination from '@/components/ui/UiPagination.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';
import AiBadge from '@/components/legal/AiBadge.vue';
import DisputeReportModal from '@/components/legal/DisputeReportModal.vue';
import OnboardingModal from '@/components/legal/OnboardingModal.vue';

const player = usePlayerStore();
const playlists = usePlaylistStore();
const toast = useToast();

const tracks = ref<Track[]>([]);
const loading = ref(false);
const offset = ref(0);
const limit = 18;
const filterProvider = ref('');
const filterMood = ref('');

const reportTarget = ref<Track | null>(null);
const playlistTargetTrack = ref<Track | null>(null);
const onboardingOpen = ref(false);

function fmtDuration(ms?: number | null) {
  if (!ms) return '0:00';
  const totalSec = Math.floor(ms / 1000);
  const min = Math.floor(totalSec / 60);
  const sec = totalSec % 60;
  return `${min}:${sec < 10 ? '0' : ''}${sec}`;
}

async function load(): Promise<void> {
  loading.value = true;
  try {
    tracks.value = await trackApi.list(limit, offset.value, {
      provider: filterProvider.value || undefined,
      mood: filterMood.value || undefined,
    });
  } catch (err: unknown) {
    toast.error(errMessageOf(err), '음원 로드 실패');
  } finally {
    loading.value = false;
  }
}

function toggleProvider(prov: string) {
  filterProvider.value = filterProvider.value === prov ? '' : prov;
  offset.value = 0;
  void load();
}

function toggleMood(mood: string) {
  filterMood.value = filterMood.value === mood ? '' : mood;
  offset.value = 0;
  void load();
}

function clearFilters() {
  filterProvider.value = '';
  filterMood.value = '';
  offset.value = 0;
  void load();
}

function page(delta: number): void {
  offset.value = Math.max(0, offset.value + delta * limit);
  void load();
  window.scrollTo({ top: 0, behavior: 'smooth' });
}

function playOne(t: Track): void {
  player.setQueue([t], t.id);
}

function playAll(): void {
  if (tracks.value.length === 0) return;
  player.setQueue(tracks.value, tracks.value[0]?.id);
  toast.success(`${tracks.value.length}개 검증 음원을 스트리밍합니다.`);
}

function enqueue(t: Track): void {
  player.enqueue(t);
  toast.info(`"${t.title}"을(를) 재생 큐에 추가했습니다.`);
}

function openAddToPlaylist(t: Track) {
  playlistTargetTrack.value = t;
}

async function addTrackToPlaylist(playlistId: string) {
  if (!playlistTargetTrack.value) return;
  try {
    await playlistApi.addTrack(playlistId, playlistTargetTrack.value.id);
    toast.success('플레이리스트에 추가되었습니다.');
    playlistTargetTrack.value = null;
  } catch (err) {
    toast.error(errMessageOf(err), '추가 실패');
  }
}

async function createNewPlaylist() {
  const title = prompt('새 플레이리스트 제목을 입력하세요:');
  if (!title) return;
  try {
    const created = await playlistApi.create(title, false);
    playlists.playlists.unshift(created);
    if (playlistTargetTrack.value) {
      await addTrackToPlaylist(created.id);
    }
  } catch (err) {
    toast.error(errMessageOf(err));
  }
}

function openReport(t: Track): void {
  reportTarget.value = t;
}

function onReported(id: string): void {
  reportTarget.value = null;
  toast.success(`신고가 성공적으로 접수되었습니다. (ID: ${id})`);
}

onMounted(() => {
  void load();
  void playlists.fetchMine();
});
</script>
