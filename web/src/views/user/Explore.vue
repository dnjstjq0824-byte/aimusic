<template>
  <div class="space-y-6">
    <!-- Explore Header -->
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
      <div>
        <h1 class="text-2xl font-black text-white tracking-tight">AI 음원 탐색</h1>
        <p class="text-xs text-neutral-400 mt-1">
          상업적 라이선스가 100% 검증된 AI 아티스트 음원을 다양한 필터로 탐색하세요.
        </p>
      </div>
      <div class="flex items-center gap-2">
        <UiButton
          variant="brand"
          size="sm"
          :disabled="tracks.length === 0"
          @click="playAll"
        >
          ▶ 전체 재생
        </UiButton>
      </div>
    </div>

    <!-- Filter Toolbar -->
    <UiCard class="space-y-4">
      <div class="grid gap-3 sm:grid-cols-2 md:grid-cols-4">
        <!-- Search Keyword -->
        <UiInput
          v-model="searchQuery"
          placeholder="곡 제목, 아티스트, 키워드 검색…"
          @enter="onSearch"
        />

        <!-- AI Provider -->
        <UiSelect v-model="filterProvider" @change="onSearch">
          <option value="">모든 AI 플랫폼</option>
          <option value="SUNO">Suno (Pro / Premier)</option>
          <option value="UDIO">Udio (Standard / Pro)</option>
          <option value="AIVA">AIVA (Pro)</option>
          <option value="STABLE_AUDIO">Stable Audio (Creator)</option>
          <option value="SOUNDRAW">Soundraw (Distribute)</option>
          <option value="MUBERT">Mubert (Studio)</option>
          <option value="CUSTOM_OPEN_MODEL">Custom Open Model</option>
        </UiSelect>

        <!-- Mood & Genre -->
        <UiSelect v-model="filterMood" @change="onSearch">
          <option value="">모든 무드 / 분위기</option>
          <option value="Focus">Focus &amp; Work (집중/업무)</option>
          <option value="Chill">Chill &amp; Lo-Fi (휴식/라운지)</option>
          <option value="Cyberpunk">Cyberpunk &amp; Synth (사이버펑크)</option>
          <option value="Cinematic">Cinematic &amp; Epic (영화/웅장)</option>
          <option value="Workout">Workout &amp; Energy (운동/에너지)</option>
          <option value="Sleep">Sleep &amp; Ambient (수면/명상)</option>
        </UiSelect>

        <!-- Sort Option -->
        <UiSelect v-model="sortBy" @change="onSearch">
          <option value="recent">최신 등록순</option>
          <option value="popular">인기 재생순</option>
        </UiSelect>
      </div>

      <!-- Quick Mood Chips -->
      <div class="flex flex-wrap items-center gap-1.5 pt-1 border-t border-surface-border/50 text-xs">
        <span class="text-neutral-500 font-medium mr-1 text-[11px]">추천 태그:</span>
        <button
          v-for="tag in popularTags"
          :key="tag"
          type="button"
          class="rounded-full px-2.5 py-1 transition-colors"
          :class="filterMood === tag ? 'bg-brand-500 text-white font-bold' : 'bg-surface-2 text-neutral-400 hover:text-white hover:bg-surface-3'"
          @click="selectMood(tag)"
        >
          #{{ tag }}
        </button>
        <button
          v-if="filterMood || filterProvider || searchQuery"
          type="button"
          class="ml-auto text-[11px] text-accent-crimson hover:underline"
          @click="resetFilters"
        >
          필터 초기화
        </button>
      </div>
    </UiCard>

    <!-- Track Grid / List -->
    <div v-if="loading" class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
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

    <div v-else-if="tracks.length === 0" class="py-12">
      <UiEmptyState
        icon="🔍"
        title="일치하는 음원이 없습니다"
        description="다른 검색어나 필터를 선택해 보세요. AIVIBE에는 지속적으로 검증된 상업용 음원이 등록됩니다."
      >
        <template #action>
          <UiButton variant="secondary" size="sm" @click="resetFilters">필터 초기화</UiButton>
        </template>
      </UiEmptyState>
    </div>

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
              class="relative flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-surface-2 group-hover:bg-brand-500 text-neutral-300 group-hover:text-white transition-all shadow-md active:scale-95"
              title="재생"
              @click="playTrack(t)"
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
                <UiBadge variant="commercial" size="xs">
                  ✓ 상업 검증
                </UiBadge>
                <span class="text-[11px] font-medium text-neutral-400">{{ t.aiProvider }}</span>
              </div>
            </div>
          </div>

          <!-- Metadata info -->
          <div class="mt-3 flex items-center justify-between text-[11px] text-neutral-400 pt-2 border-t border-surface-border">
            <span>재생 {{ t.playCount.toLocaleString() }}회</span>
            <span v-if="t.durationMs">{{ fmtDuration(t.durationMs) }}</span>
            <span v-if="t.bitrateKbps">{{ t.bitrateKbps }}kbps</span>
          </div>
        </div>

        <!-- Action strip -->
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
      @prev="changePage(-1)"
      @next="changePage(1)"
    />

    <!-- Dispute Modal -->
    <DisputeReportModal
      v-if="reportTarget"
      :track-id="reportTarget.id"
      :track-title="reportTarget.title"
      @close="reportTarget = null"
      @reported="onReported"
    />

    <!-- Playlist Add Picker Modal -->
    <UiModal
      :model-value="Boolean(playlistTargetTrack)"
      title="플레이리스트에 담기"
      description="선택한 음원을 내 플레이리스트에 저장합니다."
      @close="playlistTargetTrack = null"
    >
      <div v-if="playlists.playlists.length === 0" class="py-4 text-center">
        <p class="text-xs text-neutral-400 mb-3">생성된 플레이리스트가 없습니다.</p>
        <UiButton variant="brand" size="sm" @click="createNewPlaylist">새 플레이리스트 만들기</UiButton>
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
          <span class="text-neutral-400">선택 →</span>
        </button>
      </div>
    </UiModal>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useRoute } from 'vue-router';
import { trackApi, playlistApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { usePlayerStore } from '@/stores/playerStore';
import { usePlaylistStore } from '@/stores/playlistStore';
import { useToast } from '@/composables/useToast';
import type { Track } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiInput from '@/components/ui/UiInput.vue';
import UiSelect from '@/components/ui/UiSelect.vue';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiModal from '@/components/ui/UiModal.vue';
import UiPagination from '@/components/ui/UiPagination.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';
import DisputeReportModal from '@/components/legal/DisputeReportModal.vue';

const route = useRoute();
const player = usePlayerStore();
const playlists = usePlaylistStore();
const toast = useToast();

const popularTags = ['Focus', 'Chill', 'Cyberpunk', 'Cinematic', 'Workout', 'Sleep'];

const tracks = ref<Track[]>([]);
const loading = ref(false);
const offset = ref(0);
const limit = 18;

const searchQuery = ref('');
const filterProvider = ref('');
const filterMood = ref((route.query.mood as string) || '');
const sortBy = ref('recent');

const reportTarget = ref<Track | null>(null);
const playlistTargetTrack = ref<Track | null>(null);

function fmtDuration(ms?: number | null) {
  if (!ms) return '0:00';
  const totalSec = Math.floor(ms / 1000);
  const min = Math.floor(totalSec / 60);
  const sec = totalSec % 60;
  return `${min}:${sec < 10 ? '0' : ''}${sec}`;
}

async function load() {
  loading.value = true;
  try {
    const res = await trackApi.list(limit, offset.value, {
      provider: filterProvider.value || undefined,
      mood: filterMood.value || undefined,
    });
    tracks.value = res;
  } catch (err) {
    toast.error(errMessageOf(err), '음원 목록 로딩 실패');
  } finally {
    loading.value = false;
  }
}

function onSearch() {
  offset.value = 0;
  void load();
}

function selectMood(mood: string) {
  if (filterMood.value === mood) {
    filterMood.value = '';
  } else {
    filterMood.value = mood;
  }
  onSearch();
}

function resetFilters() {
  searchQuery.value = '';
  filterProvider.value = '';
  filterMood.value = '';
  sortBy.value = 'recent';
  onSearch();
}

function changePage(direction: number) {
  offset.value = Math.max(0, offset.value + direction * limit);
  void load();
  window.scrollTo({ top: 0, behavior: 'smooth' });
}

function playTrack(t: Track) {
  player.setQueue([t], t.id);
}

function playAll() {
  if (tracks.value.length === 0) return;
  player.setQueue(tracks.value, tracks.value[0]?.id);
  toast.success(`${tracks.value.length}개 트랙을 재생 목록에 담았습니다.`);
}

function enqueue(t: Track) {
  player.enqueue(t);
  toast.info(`"${t.title}"을(를) 재생 대기열에 추가했습니다.`);
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

function openReport(t: Track) {
  reportTarget.value = t;
}

function onReported(id: string) {
  reportTarget.value = null;
  toast.success(`신고가 접수되었습니다. (ID: ${id})`);
}

onMounted(() => {
  void load();
  void playlists.fetchMine();
});
</script>
