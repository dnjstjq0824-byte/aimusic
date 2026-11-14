<template>
  <div class="space-y-12">
    <UiPageHeader
      eyebrow="Playlists"
      title="플레이리스트"
      description="담고, 모으고, 바로 재생하세요. 검증된 AI 음원으로 만든 나만의 믹스."
    >
      <template #actions>
        <ClientOnly>
          <UiButton v-if="auth.isAuthenticated" variant="brand" size="sm" @click="createModalOpen = true">
            새 플레이리스트
          </UiButton>
          <RouterLink v-else :to="{ name: 'login', query: { redirect: '/playlists' } }">
            <UiButton variant="brand" size="sm">로그인하고 만들기</UiButton>
          </RouterLink>
          <template #fallback>
            <RouterLink :to="{ name: 'login', query: { redirect: '/playlists' } }">
              <UiButton variant="brand" size="sm">로그인하고 만들기</UiButton>
            </RouterLink>
          </template>
        </ClientOnly>
      </template>
    </UiPageHeader>

    <!-- Featured mix banner -->
    <section v-if="featured" aria-label="오늘의 믹스">
      <div
        class="relative overflow-hidden rounded-3xl border border-surface-border p-6 sm:p-8"
        :class="gradientFor(featured.key)"
      >
        <span class="absolute inset-0 bg-gradient-to-r from-black/80 via-black/50 to-black/10" aria-hidden="true" />
        <div class="relative flex flex-col gap-6 md:flex-row md:items-end md:justify-between">
          <div class="min-w-0 space-y-3">
            <UiBadge variant="brand" size="sm" dot>오늘의 믹스</UiBadge>
            <h2 class="font-display text-3xl font-extrabold tracking-tight text-white sm:text-4xl">
              {{ featured.title }}
            </h2>
            <p class="max-w-xl text-sm text-white/70">{{ featured.subtitle }}</p>
            <ol class="flex flex-wrap gap-x-4 gap-y-1 text-xs text-white/60">
              <li v-for="(t, i) in featured.tracks.slice(0, 3)" :key="t.id" class="truncate">
                <span class="font-mono text-white/40">{{ String(i + 1).padStart(2, '0') }}</span>
                {{ t.title }}
              </li>
            </ol>
          </div>
          <div class="flex shrink-0 gap-2">
            <UiButton variant="brand" size="md" @click="playMix(featured, false)">재생</UiButton>
            <UiButton variant="secondary" size="md" @click="playMix(featured, true)">셔플</UiButton>
          </div>
        </div>
      </div>
    </section>

    <!-- My playlists -->
    <!-- Client-only: section depends on auth state + client-loaded store (SSR: login prompt) -->
    <ClientOnly>
    <template #fallback>
      <section aria-labelledby="my-playlists-heading" class="space-y-4">
        <div class="flex items-end justify-between gap-3">
          <div class="flex items-center gap-2">
            <h2 id="my-playlists-heading" class="font-display text-xl font-bold tracking-tight text-white">
              내 플레이리스트
            </h2>
          </div>
        </div>
        <div class="rounded-2xl border border-dashed border-surface-border p-8 text-center">
          <p class="text-sm font-semibold text-white">로그인하면 나만의 플레이리스트를 만들 수 있어요</p>
          <p class="mt-1 text-xs text-text-secondary">추천 믹스 재생에도 로그인이 필요합니다. 1분이면 가입할 수 있어요.</p>
        </div>
      </section>
    </template>
    <section aria-labelledby="my-playlists-heading" class="space-y-4">
      <div class="flex items-end justify-between gap-3">
        <div class="flex items-center gap-2">
          <h2 id="my-playlists-heading" class="font-display text-xl font-bold tracking-tight text-white">
            내 플레이리스트
          </h2>
          <span v-if="auth.isAuthenticated" class="rounded-full bg-surface-2 px-2.5 py-0.5 font-mono text-xs text-text-secondary">
            {{ playlists.playlists.length }}
          </span>
        </div>
        <RouterLink v-if="auth.isAuthenticated" to="/my-library" class="shrink-0 text-xs font-semibold text-brand-400 hover:underline">
          라이브러리 관리 →
        </RouterLink>
      </div>

      <div v-if="!auth.isAuthenticated" class="rounded-2xl border border-dashed border-surface-border p-8 text-center">
        <p class="text-sm font-semibold text-white">로그인하면 나만의 플레이리스트를 만들 수 있어요</p>
        <p class="mt-1 text-xs text-text-secondary">추천 믹스 재생에도 로그인이 필요합니다. 1분이면 가입할 수 있어요.</p>
      </div>

      <div v-else-if="playlists.loading && playlists.playlists.length === 0" :class="gridClass">
        <div v-for="i in 5" :key="i" class="space-y-2">
          <UiSkeleton height="h-auto" width="w-full" rounded="xl" class="aspect-square" />
          <UiSkeleton height="h-3" width="w-2/3" />
        </div>
      </div>

      <div v-else :class="gridClass">
        <!-- New playlist tile -->
        <button
          type="button"
          class="group flex aspect-square flex-col items-center justify-center gap-2 rounded-2xl border border-dashed border-surface-border bg-surface-1 text-text-secondary transition-colors hover:border-brand-500/60 hover:text-brand-300"
          @click="createModalOpen = true"
        >
          <span class="flex h-12 w-12 items-center justify-center rounded-full bg-surface-2 text-2xl transition-transform group-hover:scale-110" aria-hidden="true">+</span>
          <span class="text-xs font-semibold">새 플레이리스트</span>
        </button>

        <article v-for="p in playlists.playlists" :key="p.id" class="group min-w-0">
          <div class="relative aspect-square overflow-hidden rounded-2xl border border-surface-border transition-colors group-hover:border-brand-500/50" :class="gradientFor(p.id)">
            <RouterLink
              :to="{ name: 'playlist-detail', params: { id: p.id } }"
              class="absolute inset-0"
              :aria-label="`「${p.title}」 열기`"
            >
              <span class="absolute inset-0 bg-gradient-to-t from-black/70 via-black/10 to-transparent" aria-hidden="true" />
              <svg class="absolute left-3 top-3 h-6 w-6 text-white/70" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                <path d="M3 6h12v2H3zm0 4h12v2H3zm0 4h8v2H3zm14-4v8.2a3 3 0 1 1-2-2.83V8h5v2z" />
              </svg>
              <span class="absolute bottom-3 left-3 rounded-full bg-black/40 px-2 py-0.5 text-caption font-semibold text-white/80 backdrop-blur">
                {{ p.isPublic ? '공개' : '비공개' }}
              </span>
            </RouterLink>
            <button
              type="button"
              :class="playButtonClass"
              :disabled="(counts[p.id] ?? 0) === 0 || busyId === p.id"
              :aria-label="`「${p.title}」 재생`"
              @click="playPlaylist(p.id)"
            >
              <svg v-if="busyId === p.id" class="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z" />
              </svg>
              <svg v-else class="ml-0.5 h-4 w-4" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M8 5v14l11-7z" /></svg>
            </button>
          </div>
          <RouterLink :to="{ name: 'playlist-detail', params: { id: p.id } }" class="mt-2 block truncate text-sm font-bold text-white hover:text-brand-400">
            {{ p.title }}
          </RouterLink>
          <p class="text-xs text-text-muted">
            {{ counts[p.id] === undefined ? '불러오는 중…' : `${counts[p.id]}곡` }}
          </p>
        </article>
      </div>
    </section>
    </ClientOnly>

    <!-- Recommended mixes -->
    <section aria-labelledby="mixes-heading" class="space-y-4">
      <div class="flex items-end justify-between gap-3">
        <h2 id="mixes-heading" class="font-display text-xl font-bold tracking-tight text-white">추천 믹스</h2>
        <RouterLink to="/explore" class="shrink-0 text-xs font-semibold text-brand-400 hover:underline">전체 탐색 →</RouterLink>
      </div>

      <UiErrorBanner :message="mixError" title="추천 믹스를 불러오지 못했습니다" retry-label="다시 시도" @retry="loadMixes" />

      <div v-if="mixLoading" :class="gridClass">
        <div v-for="i in 5" :key="i" class="space-y-2">
          <UiSkeleton height="h-auto" width="w-full" rounded="xl" class="aspect-square" />
          <UiSkeleton height="h-3" width="w-2/3" />
        </div>
      </div>

      <UiEmptyState
        v-else-if="!mixError && mixes.length === 0"
        title="아직 추천할 음원이 없습니다"
        description="검증을 통과한 음원이 등록되면 자동으로 믹스가 만들어집니다."
      />

      <div v-else :class="gridClass">
        <article v-for="m in mixes" :key="m.key" class="group min-w-0">
          <div class="relative aspect-square overflow-hidden rounded-2xl border border-surface-border transition-colors group-hover:border-brand-500/50" :class="gradientFor(m.key)">
            <span class="absolute inset-0 bg-gradient-to-t from-black/75 via-black/10 to-transparent" aria-hidden="true" />
            <span class="absolute inset-x-3 bottom-3">
              <span class="block text-caption font-semibold uppercase tracking-wider text-white/60">{{ m.kicker }}</span>
              <span class="mt-0.5 block truncate text-sm font-bold text-white">{{ m.title }}</span>
            </span>
            <button type="button" :class="playButtonClass" :aria-label="`「${m.title}」 재생`" @click="playMix(m, false)">
              <svg class="ml-0.5 h-4 w-4" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M8 5v14l11-7z" /></svg>
            </button>
          </div>
          <p class="mt-2 truncate text-xs text-text-muted">{{ m.tracks.length }}곡 · {{ m.subtitle }}</p>
        </article>
      </div>
    </section>

    <!-- Create playlist -->
    <UiModal
      v-model="createModalOpen"
      title="새 플레이리스트"
      description="이름과 공개 범위를 정하세요."
      @close="createModalOpen = false"
    >
      <form class="space-y-4" @submit.prevent="submitCreate">
        <UiInput v-model="newTitle" label="플레이리스트 이름" placeholder="예: 심야 코딩용 Lo-Fi Synth" :maxlength="80" required />
        <UiCheckbox v-model="newIsPublic">공개 플레이리스트로 만들기</UiCheckbox>
        <div class="flex justify-end gap-2 pt-2">
          <UiButton variant="ghost" size="sm" @click="createModalOpen = false">취소</UiButton>
          <UiButton variant="brand" size="sm" type="submit" :loading="creating">만들기</UiButton>
        </div>
      </form>
    </UiModal>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { playlistApi, trackApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { useAuthStore } from '@/stores/authStore';
import { usePlayerStore } from '@/stores/playerStore';
import { usePlaylistStore } from '@/stores/playlistStore';
import { useToast } from '@/composables/useToast';
import type { Track } from '@/api/types';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiButton from '@/components/ui/UiButton.vue';
import UiCheckbox from '@/components/ui/UiCheckbox.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiInput from '@/components/ui/UiInput.vue';
import UiModal from '@/components/ui/UiModal.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import { GRADIENTS } from '@/components/ui/UiCoverArt.vue';

interface Mix {
  key: string;
  kicker: string;
  title: string;
  subtitle: string;
  tracks: Track[];
}

const auth = useAuthStore();
const player = usePlayerStore();
const playlists = usePlaylistStore();
const toast = useToast();
const router = useRouter();

const gridClass = 'grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5';
const playButtonClass =
  'absolute bottom-3 right-3 flex h-10 w-10 items-center justify-center rounded-full bg-brand-500 -on-brand shadow-glow-brand transition-all hover:bg-brand-400 active:scale-95 disabled:cursor-not-allowed disabled:opacity-40 ' +
  'opacity-0 translate-y-1 group-hover:translate-y-0 group-hover:opacity-100 focus-visible:opacity-100 [@media(hover:none)]:translate-y-0 [@media(hover:none)]:opacity-100';

const counts = ref<Record<string, number>>({});
const busyId = ref<string | null>(null);

const allTracks = ref<Track[]>([]);
const mixLoading = ref(false);
const mixError = ref<string | null>(null);

const createModalOpen = ref(false);
const newTitle = ref('');
const newIsPublic = ref(false);
const creating = ref(false);

function hashOf(s: string): number {
  let h = 2166136261;
  for (let i = 0; i < s.length; i += 1) h = Math.imul(h ^ s.charCodeAt(i), 16777619);
  return h >>> 0;
}

function gradientFor(seed: string): string {
  return GRADIENTS[hashOf(seed) % GRADIENTS.length] as string;
}

const mixes = computed<Mix[]>(() => {
  const tracks = allTracks.value;
  if (tracks.length === 0) return [];
  const out: Mix[] = [
    { key: 'mix-all', kicker: '스튜디오', title: '전체 검증 믹스', subtitle: '상업 검증을 통과한 모든 음원', tracks },
  ];

  const latest = [...tracks].sort((a, b) => b.createdAt.localeCompare(a.createdAt)).slice(0, 20);
  out.push({ key: 'mix-latest', kicker: '새로 나온', title: '최신 업로드', subtitle: '방금 게이트를 통과한 음원', tracks: latest });

  const byProvider = new Map<string, Track[]>();
  const byMood = new Map<string, Track[]>();
  for (const t of tracks) {
    if (t.aiProvider) byProvider.set(t.aiProvider, [...(byProvider.get(t.aiProvider) ?? []), t]);
    for (const mood of t.moodTags ?? []) byMood.set(mood, [...(byMood.get(mood) ?? []), t]);
  }
  for (const [prov, list] of byProvider) {
    out.push({ key: `mix-prov-${prov}`, kicker: 'AI 프로바이더', title: `${prov} 셀렉션`, subtitle: `${prov} 상업 플랜 생성 음원`, tracks: list });
  }
  for (const [mood, list] of byMood) {
    out.push({ key: `mix-mood-${mood}`, kicker: '무드', title: `#${mood}`, subtitle: `${mood} 무드 모음`, tracks: list });
  }
  const instrumental = tracks.filter((t) => t.isInstrumental);
  if (instrumental.length > 0) {
    out.push({ key: 'mix-inst', kicker: '작업용', title: '보컬 없는 연주곡', subtitle: '영상·작업 배경음', tracks: instrumental });
  }
  return out.slice(0, 10);
});

// Rotates daily so the banner feels alive without server-side curation.
const featured = computed<Mix | null>(() => {
  if (mixes.value.length === 0) return null;
  const day = Math.floor(Date.now() / 86_400_000);
  return mixes.value[day % mixes.value.length] ?? null;
});

async function loadMixes(): Promise<void> {
  mixLoading.value = true;
  mixError.value = null;
  try {
    const page = await trackApi.list(50, 0, {});
    allTracks.value = (page.items ?? []).filter((t) => t.status === 'READY');
  } catch (err) {
    mixError.value = errMessageOf(err);
  } finally {
    mixLoading.value = false;
  }
}

async function loadCounts(): Promise<void> {
  await Promise.all(
    playlists.playlists.map(async (p) => {
      try {
        counts.value[p.id] = ((await playlistApi.listTracks(p.id)) ?? []).length;
      } catch {
        counts.value[p.id] = 0;
      }
    }),
  );
}

function playMix(mix: Mix, shuffled: boolean): void {
  if (mix.tracks.length === 0) return;
  if (!auth.isAuthenticated) {
    toast.info('믹스를 들으려면 먼저 로그인하세요.');
    void router.push({ name: 'login', query: { redirect: '/playlists' } });
    return;
  }
  if (player.shuffle !== shuffled) player.toggleShuffle();
  player.setQueue(mix.tracks);
  toast.success(`「${mix.title}」 ${mix.tracks.length}곡을 재생합니다.`);
}

async function playPlaylist(id: string): Promise<void> {
  busyId.value = id;
  try {
    await playlists.fetchTracks(id);
    const tracks = (playlists.entryTracks[id] ?? []).filter((t) => t.status === 'READY');
    if (tracks.length === 0) {
      toast.info('재생 가능한 음원이 없습니다.');
      return;
    }
    player.setQueue(tracks);
  } catch (err) {
    toast.error(errMessageOf(err), '재생 실패');
  } finally {
    busyId.value = null;
  }
}

async function submitCreate(): Promise<void> {
  const title = newTitle.value.trim();
  if (!title) return;
  creating.value = true;
  try {
    const p = await playlists.create(title, newIsPublic.value);
    counts.value[p.id] = 0;
    createModalOpen.value = false;
    newTitle.value = '';
    newIsPublic.value = false;
    toast.success('플레이리스트를 만들었습니다.');
    void router.push({ name: 'playlist-detail', params: { id: p.id } });
  } catch (err) {
    toast.error(errMessageOf(err), '생성 실패');
  } finally {
    creating.value = false;
  }
}

onMounted(async () => {
  void loadMixes();
  if (auth.isAuthenticated) {
    await playlists.fetchMine();
    void loadCounts();
  }
});
</script>
