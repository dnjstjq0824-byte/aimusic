<template>
  <div class="space-y-10">
    <!-- Hero Banner -->
    <div class="relative overflow-hidden rounded-3xl border border-surface-border bg-gradient-to-br from-brandteal via-surface-1 to-surface-0 p-6 sm:p-10 shadow-2xl">
      <div class="pointer-events-none absolute -right-20 -top-20 h-64 w-64 rounded-full bg-brand-500/15 blur-3xl" />
      <div class="pointer-events-none absolute -bottom-24 -left-16 h-64 w-64 rounded-full bg-gold/10 blur-3xl" />

      <div class="relative z-10 max-w-2xl space-y-4">
        <UiBadge variant="brand" size="sm" dot>
          상업 라이선스 100% 검증 스트리밍
        </UiBadge>

        <h1 class="text-3xl sm:text-5xl font-black text-white tracking-tight leading-tight">
          저작권 걱정 없이 듣는<br />
          <span class="underline decoration-brand-500 decoration-4 underline-offset-8">검증된 AI 음악</span>
        </h1>

        <p class="text-xs sm:text-sm text-neutral-300 leading-relaxed">
          Suno Pro, AIVA Pro 등 유료 상업 플랜에서 생성된 음원만 4단계 게이트키퍼(Gate 1~4)로 엄격히 검증하여 서비스합니다. 무료 플랜 및 비상업용 음원은 원천 차단됩니다.
        </p>

        <div class="flex flex-wrap items-center gap-3 pt-2">
          <UiButton
            variant="brand"
            size="lg"
            :disabled="tracks.length === 0 || loading"
            @click="playAll"
          >
            전체 음원 스트리밍
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
      <dl class="mt-8 grid grid-cols-2 gap-3 sm:grid-cols-4 pt-6 border-t border-surface-border/60 text-xs">
        <div class="space-y-0.5">
          <dt class="text-[10px] uppercase font-bold text-neutral-500">라이선스 기준</dt>
          <dd class="font-bold text-emerald-400">100% 상업용 전용</dd>
        </div>
        <div class="space-y-0.5">
          <dt class="text-[10px] uppercase font-bold text-neutral-500">법적 컴플라이언스</dt>
          <dd class="font-bold text-white">인공지능기본법 제31조</dd>
        </div>
        <div class="space-y-0.5">
          <dt class="text-[10px] uppercase font-bold text-neutral-500">음향 포렌식</dt>
          <dd class="font-bold text-white">FFprobe 192k+ 검증</dd>
        </div>
        <div class="space-y-0.5">
          <dt class="text-[10px] uppercase font-bold text-neutral-500">백그라운드 스트리밍</dt>
          <dd class="font-bold text-brand-400">HLS 무중단 Gapless</dd>
        </div>
      </dl>
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
      <TrackCard
        v-for="t in tracks"
        :key="t.id"
        :track="t"
        @play="playOne"
        @enqueue="enqueue"
        @add-to-playlist="openAddToPlaylist"
        @report="openReport"
      />
    </div>

    <!-- Pagination -->
    <UiPagination
      v-if="tracks.length > 0"
      :page="Math.floor(offset / limit) + 1"
      :has-more="offset + tracks.length < total"
      :total="total"
      :limit="limit"
      :disabled="loading"
      @prev="page(-1)"
      @next="page(1)"
      @page="goPage"
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
    <UiPlaylistPicker
      :model-value="Boolean(playlistTargetTrack)"
      :track-title="playlistTargetTrack?.title ?? ''"
      :playlists="playlists.playlists"
      :create-loading="creatingPlaylist"
      @update:model-value="playlistTargetTrack = $event ? playlistTargetTrack : null"
      @select="addTrackToPlaylist"
      @create="createNewPlaylist"
    />

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
import UiBadge from '@/components/ui/UiBadge.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiPlaylistPicker from '@/components/ui/UiPlaylistPicker.vue';
import UiPagination from '@/components/ui/UiPagination.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';
import TrackCard from '@/components/TrackCard.vue';
import DisputeReportModal from '@/components/legal/DisputeReportModal.vue';
import OnboardingModal from '@/components/legal/OnboardingModal.vue';

const player = usePlayerStore();
const playlists = usePlaylistStore();
const toast = useToast();

const tracks = ref<Track[]>([]);
const loading = ref(false);
const offset = ref(0);
const limit = 18;
const total = ref(0);
const filterProvider = ref('');
const filterMood = ref('');

const reportTarget = ref<Track | null>(null);
const playlistTargetTrack = ref<Track | null>(null);
const creatingPlaylist = ref(false);
const onboardingOpen = ref(false);

async function load(): Promise<void> {
  loading.value = true;
  try {
    const page = await trackApi.list(limit, offset.value, {
      provider: filterProvider.value || undefined,
      mood: filterMood.value || undefined,
    });
    tracks.value = page.items;
    total.value = page.total;
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

function goPage(p: number): void {
  offset.value = Math.max(0, (p - 1) * limit);
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

async function createNewPlaylist(title: string) {
  creatingPlaylist.value = true;
  try {
    const created = await playlistApi.create(title, false);
    playlists.playlists.unshift(created);
    if (playlistTargetTrack.value) {
      await addTrackToPlaylist(created.id);
    }
  } catch (err) {
    toast.error(errMessageOf(err));
  } finally {
    creatingPlaylist.value = false;
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
  try {
    if (localStorage.getItem('aivibe.onboarded') !== '1') {
      onboardingOpen.value = true;
      localStorage.setItem('aivibe.onboarded', '1');
    }
  } catch {
    // Private mode: onboarding stays manual via the hero link.
  }
});
</script>
