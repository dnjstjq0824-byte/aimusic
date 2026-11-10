<template>
  <div class="space-y-14">
    <!-- Studio hero -->
    <section class="relative mx-auto max-w-3xl px-4 pt-4 text-center sm:pt-8">
      <div
        class="pointer-events-none absolute left-1/2 top-0 h-72 w-full max-w-[36rem] -translate-x-1/2 rounded-full bg-brand-500/10 blur-3xl"
        aria-hidden="true"
      />
      <div class="relative space-y-5">
        <UiBadge variant="brand" size="sm" dot>
          상업 이용 조건 확인 절차를 거친 AI 음원
        </UiBadge>

        <h1 class="font-display text-4xl font-extrabold leading-tight tracking-tight text-white sm:text-6xl">
          나만의 <span class="text-brand-400">스튜디오</span>
        </h1>
        <p
          class="font-display bg-gradient-to-r from-brand-200 via-brand-400 to-brand-500 bg-clip-text text-xl font-bold tracking-tight text-transparent sm:text-3xl"
        >
          {{ SLOGAN }}
        </p>
        <p class="mx-auto max-w-xl text-sm leading-relaxed text-neutral-400 sm:text-base">
          상상하는 모든 노래를 확인 절차를 거친 음원으로. 이용 조건을 확인한 음원으로 듣고,
          플레이리스트에 담고, 영상에 쓰세요.
        </p>

        <!-- Prompt box -->
        <form
          class="prompt-box mx-auto max-w-2xl rounded-2xl p-2 pl-5 text-left transition-all"
          role="search"
          aria-label="음원 검색"
          @submit.prevent="submitHero"
        >
          <label for="hero-q" class="sr-only">음원 검색어</label>
          <input
            id="hero-q"
            v-model="heroQuery"
            type="search"
            autocomplete="off"
            placeholder="어떤 음악을 찾으세요? (제목·장르·무드)"
            class="w-full bg-transparent py-3 text-sm text-white placeholder:text-neutral-500 focus:outline-none sm:text-base"
          />
          <div class="flex items-center justify-between gap-2 pb-1 pl-0 pr-1">
            <div class="flex items-center gap-1">
              <button
                v-for="hint in ['Focus', 'Chill', 'Cinematic']"
                :key="hint"
                type="button"
                class="rounded-full px-2.5 py-1 text-xs font-medium text-neutral-500 transition-colors hover:bg-surface-3 hover:text-brand-300"
                @click="applyHint(hint)"
              >
                #{{ hint }}
              </button>
            </div>
            <button
              type="submit"
              :disabled="loading"
              class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-brand-500 text-[#04121a] shadow-glow-brand transition-all hover:bg-brand-400 active:scale-95 disabled:opacity-50"
              aria-label="검색"
            >
              <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                <path d="M12 19V5M5 12l7-7 7 7" />
              </svg>
            </button>
          </div>
        </form>

        <!-- Shortcut pills -->
        <div class="flex flex-wrap items-center justify-center gap-2 pt-1">
          <RouterLink
            to="/explore"
            class="inline-flex items-center gap-1.5 rounded-full border border-surface-border bg-surface-1 px-4 py-1.5 text-xs font-semibold text-neutral-300 transition-colors hover:border-brand-500/50 hover:text-brand-300"
          >
            음원 탐색
          </RouterLink>
          <RouterLink
            to="/playlists"
            class="inline-flex items-center gap-1.5 rounded-full border border-surface-border bg-surface-1 px-4 py-1.5 text-xs font-semibold text-neutral-300 transition-colors hover:border-brand-500/50 hover:text-brand-300"
          >
            플레이리스트
          </RouterLink>
          <RouterLink
            to="/my-library"
            class="inline-flex items-center gap-1.5 rounded-full border border-surface-border bg-surface-1 px-4 py-1.5 text-xs font-semibold text-neutral-300 transition-colors hover:border-brand-500/50 hover:text-brand-300"
          >
            내 라이브러리
          </RouterLink>
          <RouterLink
            to="/pricing"
            class="inline-flex items-center gap-1.5 rounded-full border border-surface-border bg-surface-1 px-4 py-1.5 text-xs font-semibold text-neutral-300 transition-colors hover:border-brand-500/50 hover:text-brand-300"
          >
            요금제
          </RouterLink>
          <RouterLink
            to="/partner/upload"
            class="inline-flex items-center gap-1.5 rounded-full border border-brand-500/40 bg-brand-950/50 px-4 py-1.5 text-xs font-semibold text-brand-300 transition-colors hover:bg-brand-950"
          >
            파트너 스튜디오
          </RouterLink>
        </div>
      </div>
    </section>

    <!-- Starter rail -->
    <section aria-label="바로 듣기">
      <div class="mb-4 flex items-end justify-between px-1">
        <h2 class="font-display text-xl font-bold tracking-tight text-white">바로 듣기</h2>
        <RouterLink to="/explore" class="text-xs font-semibold text-brand-400 hover:underline">
          전체 탐색 →
        </RouterLink>
      </div>
      <div class="scrollbar-none -mx-4 flex snap-x snap-mandatory gap-3 overflow-x-auto scroll-px-4 px-4 pb-2 sm:mx-0 sm:scroll-px-0 sm:px-0">
        <button
          v-for="card in starterCards"
          :key="card.key"
          type="button"
          class="group relative h-52 w-40 shrink-0 snap-start overflow-hidden rounded-2xl border border-surface-border text-left transition-all hover:border-brand-500/50 sm:h-60 sm:w-44"
          :class="card.gradient"
          @click="card.action"
        >
          <span class="absolute inset-0 bg-gradient-to-t from-black/70 via-black/10 to-transparent" aria-hidden="true" />
          <svg
            class="absolute right-3 top-3 h-6 w-6 text-white/70 transition-transform group-hover:scale-110"
            viewBox="0 0 24 24"
            fill="currentColor"
            aria-hidden="true"
          >
            <path d="M9 18.5A3.5 3.5 0 1 1 5.5 15c.54 0 1.05.12 1.5.34V6.6L19 4v11.5a3.5 3.5 0 1 1-2-3.16V7.7l-8 1.5v9.3z" />
          </svg>
          <span class="absolute inset-x-0 bottom-0 p-3">
            <span class="block text-[10px] font-semibold uppercase tracking-wider text-white/60">{{ card.kicker }}</span>
            <span class="mt-0.5 block text-sm font-bold leading-snug text-white">{{ card.title }}</span>
          </span>
        </button>
      </div>
    </section>

    <!-- Track list -->
    <section id="track-list" aria-label="검증 음원 목록" class="scroll-mt-24 space-y-5">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="flex items-center gap-2">
          <h2 class="font-display text-xl font-bold tracking-tight text-white">검증 음원 목록</h2>
          <span class="rounded-full bg-surface-2 px-2.5 py-0.5 font-mono text-xs text-neutral-400">
            {{ total }}
          </span>
          <span v-if="activeFilterLabel" class="rounded-full bg-brand-950 px-2.5 py-0.5 text-xs font-semibold text-brand-300">
            {{ activeFilterLabel }}
          </span>
        </div>
        <div class="flex items-center gap-2">
          <UiButton variant="secondary" size="sm" :disabled="tracks.length === 0 || loading" @click="playAll">
            전체 재생
          </UiButton>
          <RouterLink to="/explore" class="shrink-0 text-xs font-semibold text-brand-400 hover:underline">
            상세 필터 →
          </RouterLink>
        </div>
      </div>

      <!-- Quick filter chips -->
      <div class="flex flex-wrap items-center gap-2">
        <button
          type="button"
          class="rounded-full px-3.5 py-1.5 text-xs font-semibold transition-all duration-150"
          :class="!filterProvider && !filterMood && !searchQuery ? 'bg-white text-neutral-900 shadow font-bold' : 'bg-surface-1 text-neutral-300 hover:bg-surface-2 border border-surface-border'"
          @click="clearFilters"
        >
          전체 실시간
        </button>
        <button
          v-for="prov in ['SUNO', 'UDIO', 'AIVA', 'STABLE_AUDIO']"
          :key="prov"
          type="button"
          class="rounded-full px-3.5 py-1.5 text-xs font-semibold transition-all duration-150"
          :class="filterProvider === prov ? 'bg-brand-500 text-[#04121a] shadow-glow-brand' : 'bg-surface-1 text-neutral-300 hover:bg-surface-2 border border-surface-border'"
          @click="toggleProvider(prov)"
        >
          {{ prov }}
        </button>
        <button
          v-for="mood in ['Focus', 'Chill', 'Cyberpunk', 'Cinematic']"
          :key="mood"
          type="button"
          class="rounded-full px-3.5 py-1.5 text-xs font-semibold transition-all duration-150"
          :class="filterMood === mood ? 'bg-brand-500 text-[#04121a] shadow-glow-brand' : 'bg-surface-1 text-neutral-300 hover:bg-surface-2 border border-surface-border'"
          @click="toggleMood(mood)"
        >
          #{{ mood }}
        </button>
        <button
          v-if="searchQuery"
          type="button"
          class="rounded-full border border-brand-500/50 bg-brand-950 px-3.5 py-1.5 text-xs font-semibold text-brand-300 transition-colors hover:bg-brand-900"
          :title="'검색 해제: ' + searchQuery"
          @click="clearSearch"
        >
          “{{ searchQuery }}” ✕
        </button>
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
    </section>

    <!-- Feature trio -->
    <section aria-label="플랫폼 특징" class="space-y-10">
      <div class="grid items-center gap-6 sm:grid-cols-2">
        <div class="space-y-3">
          <p class="text-xs font-semibold uppercase tracking-wider text-neutral-500">검증</p>
          <h3 class="font-display text-2xl font-extrabold tracking-tight text-white sm:text-3xl">
            4단계 게이트를 통과한 <span class="text-brand-400">음원만</span>
          </h3>
          <p class="text-sm leading-relaxed text-neutral-400">
            상업 플랜 증빙부터 생성일 윈도우, Sync-Only 판별, SHA-256 중복 차단까지.
            무료·비상업 음원은 목록에 오르지 않습니다.
          </p>
          <RouterLink to="/explore" class="inline-block text-sm font-semibold text-brand-400 hover:underline">
            검증 음원 탐색하기 →
          </RouterLink>
        </div>
        <div class="rounded-2xl border border-surface-border bg-surface-1 p-5">
          <dl class="grid grid-cols-2 gap-4 text-xs">
            <div class="space-y-1 rounded-xl bg-surface-2 p-3">
              <dt class="font-semibold text-neutral-500">라이선스 기준</dt>
              <dd class="font-bold text-emerald-400">유료 상업 플랜 생성분만 접수</dd>
            </div>
            <div class="space-y-1 rounded-xl bg-surface-2 p-3">
              <dt class="font-semibold text-neutral-500">법적 컴플라이언스</dt>
              <dd class="font-bold text-white">인공지능기본법 제31조</dd>
            </div>
            <div class="space-y-1 rounded-xl bg-surface-2 p-3">
              <dt class="font-semibold text-neutral-500">음향 포렌식</dt>
              <dd class="font-bold text-white">FFprobe 192k+ 검증</dd>
            </div>
            <div class="space-y-1 rounded-xl bg-surface-2 p-3">
              <dt class="font-semibold text-neutral-500">스트리밍</dt>
              <dd class="font-bold text-brand-400">HLS Gapless</dd>
            </div>
          </dl>
        </div>
      </div>

      <div class="grid items-center gap-6 sm:grid-cols-2">
        <div class="order-2 space-y-3 sm:order-1">
          <div class="flex items-center gap-3 rounded-2xl border border-surface-border bg-surface-1 p-4">
            <span class="flex h-11 w-11 items-center justify-center rounded-xl bg-brand-500 text-lg font-black text-[#04121a] shadow-glow-brand">▶</span>
            <div class="min-w-0 flex-1">
              <p class="truncate text-sm font-bold text-white">끊김 없는 감상</p>
              <p class="text-xs text-neutral-500">하트비트 집계 · 이어듣기 · 큐</p>
            </div>
            <RouterLink to="/my-library" class="shrink-0 text-xs font-semibold text-brand-400 hover:underline">
              라이브러리 →
            </RouterLink>
          </div>
        </div>
        <div class="order-1 space-y-3 sm:order-2">
          <p class="text-xs font-semibold uppercase tracking-wider text-neutral-500">감상</p>
          <h3 class="font-display text-2xl font-extrabold tracking-tight text-white sm:text-3xl">
            듣고, 담고, <span class="text-brand-400">영상으로</span>
          </h3>
          <p class="text-sm leading-relaxed text-neutral-400">
            30초 이상 감상은 정산에 집계되고, 플레이리스트와 권리보호센터가 창작자를
            끝까지 보호합니다.
          </p>
        </div>
      </div>

      <div class="grid items-center gap-6 sm:grid-cols-2">
        <div class="space-y-3">
          <p class="text-xs font-semibold uppercase tracking-wider text-neutral-500">창작</p>
          <h3 class="font-display text-2xl font-extrabold tracking-tight text-white sm:text-3xl">
            파트너 <span class="text-brand-400">스튜디오</span>
          </h3>
          <p class="text-sm leading-relaxed text-neutral-400">
            AI 구독 증빙 등록부터 프리체크·파이널라이즈까지. 승인된 음원은
            재생수 기반으로 투명하게 정산됩니다.
          </p>
          <RouterLink to="/partner/upload" class="inline-block text-sm font-semibold text-brand-400 hover:underline">
            음원 등록 시작하기 →
          </RouterLink>
        </div>
        <div class="rounded-2xl border border-brand-500/30 bg-brand-950/40 p-5 text-sm">
          <p class="font-bold text-brand-300">Gate 1 → 4 프리체크</p>
          <p class="mt-1 text-xs leading-relaxed text-neutral-400">
            상업 티어 · 생성일 윈도우 · Sync-Only · SHA-256 중복을 업로드 전에 즉시 판정합니다.
          </p>
        </div>
      </div>
    </section>

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
import { computed, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { trackApi, playlistApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { useAuthStore } from '@/stores/authStore';
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
import { GRADIENTS as COVER_GRADIENTS } from '@/components/ui/UiCoverArt.vue';
import { SLOGAN } from '@/components/BrandLogo.vue';

const player = usePlayerStore();
const playlists = usePlaylistStore();
const toast = useToast();
const auth = useAuthStore();
const router = useRouter();

const tracks = ref<Track[]>([]);
const loading = ref(true);
const offset = ref(0);
const limit = 18;
const total = ref(0);
const filterProvider = ref('');
const filterMood = ref('');
const heroQuery = ref('');
const searchQuery = ref('');

const reportTarget = ref<Track | null>(null);
const playlistTargetTrack = ref<Track | null>(null);
const creatingPlaylist = ref(false);
const onboardingOpen = ref(false);

const activeFilterLabel = computed(() => {
  if (searchQuery.value) return `“${searchQuery.value}” 검색`;
  if (filterProvider.value) return `${filterProvider.value} 필터`;
  if (filterMood.value) return `#${filterMood.value} 필터`;
  return '';
});

function scrollToList(): void {
  document.getElementById('track-list')?.scrollIntoView({ behavior: 'smooth', block: 'start' });
}

async function load(): Promise<void> {
  loading.value = true;
  try {
    const page = await trackApi.list(limit, offset.value, {
      provider: filterProvider.value || undefined,
      mood: filterMood.value || undefined,
      q: searchQuery.value || undefined,
    });
    tracks.value = page.items;
    total.value = page.total;
  } catch (err: unknown) {
    toast.error(errMessageOf(err), '음원 로드 실패');
  } finally {
    loading.value = false;
  }
}

function submitHero(): void {
  if (!heroQuery.value.trim()) {
    toast.info('검색어를 입력해 주세요. 비워 두면 전체 음원이 표시됩니다.');
    return;
  }
  searchQuery.value = heroQuery.value.trim();
  filterProvider.value = '';
  filterMood.value = '';
  offset.value = 0;
  void load();
  scrollToList();
}

function applyHint(hint: string): void {
  heroQuery.value = hint;
  submitHero();
}

function clearSearch(): void {
  heroQuery.value = '';
  searchQuery.value = '';
  offset.value = 0;
  void load();
}

const starterCards = [
  { key: 'starter-suno', kicker: 'AI 프로바이더', title: 'SUNO 프로 사운드', gradient: COVER_GRADIENTS[2], action: () => toggleProvider('SUNO') },
  { key: 'starter-udio', kicker: 'AI 프로바이더', title: 'UDIO 셀렉션', gradient: COVER_GRADIENTS[4], action: () => toggleProvider('UDIO') },
  { key: 'starter-focus', kicker: '무드', title: '집중을 위한 Focus', gradient: COVER_GRADIENTS[1], action: () => toggleMood('Focus') },
  { key: 'starter-cinema', kicker: '무드', title: '영상용 Cinematic', gradient: COVER_GRADIENTS[0], action: () => toggleMood('Cinematic') },
  { key: 'starter-shuffle', kicker: '스튜디오', title: '전체 셔플 재생', gradient: COVER_GRADIENTS[3], action: () => playAll() },
];

function toggleProvider(prov: string) {
  filterProvider.value = filterProvider.value === prov ? '' : prov;
  searchQuery.value = '';
  heroQuery.value = '';
  offset.value = 0;
  void load();
  scrollToList();
}

function toggleMood(mood: string) {
  filterMood.value = filterMood.value === mood ? '' : mood;
  searchQuery.value = '';
  heroQuery.value = '';
  offset.value = 0;
  void load();
  scrollToList();
}

function clearFilters() {
  filterProvider.value = '';
  filterMood.value = '';
  heroQuery.value = '';
  searchQuery.value = '';
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
  if (!auth.isAuthenticated) {
    toast.info('음원을 들으려면 먼저 로그인하세요.');
    void router.push({ name: 'login', query: { redirect: '/' } });
    return;
  }
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
    if (localStorage.getItem('avibe.onboarded') !== '1') {
      onboardingOpen.value = true;
      localStorage.setItem('avibe.onboarded', '1');
    }
  } catch {
    // Private mode: onboarding stays manual.
  }
});
</script>
