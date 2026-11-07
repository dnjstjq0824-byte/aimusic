<template>
  <div class="space-y-6">
    <UiPageHeader
      eyebrow="Catalog"
      title="AI 음원 탐색"
      description="상업적 라이선스가 100% 검증된 AI 아티스트 음원을 다양한 필터로 탐색하세요."
    >
      <template #actions>
        <UiButton
          variant="brand"
          size="sm"
          :disabled="tracks.length === 0 || loading"
          @click="playAll"
        >
          전체 재생
        </UiButton>
      </template>
    </UiPageHeader>

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
        <UiSelect v-model="sortBy" label="정렬" @change="onSearch">
          <option value="recent">최신 등록순</option>
          <option value="popular">인기 재생순</option>
          <option value="oldest">오래된 등록순</option>
          <option value="title">제목순</option>
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
          :class="filterMood === tag ? 'bg-brand-500 text-[#04121a] font-bold' : 'bg-surface-2 text-neutral-400 hover:text-white hover:bg-surface-3'"
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
        title="일치하는 음원이 없습니다"
        description="다른 검색어나 필터를 선택해 보세요. AIVIBE에는 지속적으로 검증된 상업용 음원이 등록됩니다."
      >
        <template #action>
          <UiButton variant="secondary" size="sm" @click="resetFilters">필터 초기화</UiButton>
        </template>
      </UiEmptyState>
    </div>

    <div v-else class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
      <TrackCard
        v-for="t in tracks"
        :key="t.id"
        :track="t"
        @play="playTrack"
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
      @prev="changePage(-1)"
      @next="changePage(1)"
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

    <!-- Playlist Add Picker Modal -->
    <UiPlaylistPicker
      :model-value="Boolean(playlistTargetTrack)"
      :track-title="playlistTargetTrack?.title ?? ''"
      :playlists="playlists.playlists"
      :create-loading="creatingPlaylist"
      @update:model-value="playlistTargetTrack = $event ? playlistTargetTrack : null"
      @select="addTrackToPlaylist"
      @create="createNewPlaylist"
    />
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
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSelect from '@/components/ui/UiSelect.vue';
import UiPlaylistPicker from '@/components/ui/UiPlaylistPicker.vue';
import UiPagination from '@/components/ui/UiPagination.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';
import TrackCard from '@/components/TrackCard.vue';
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
const total = ref(0);

const searchQuery = ref('');
const filterProvider = ref('');
const filterMood = ref((route.query.mood as string) || '');
const sortBy = ref('recent');

const reportTarget = ref<Track | null>(null);
const playlistTargetTrack = ref<Track | null>(null);
const creatingPlaylist = ref(false);

async function load() {
  loading.value = true;
  try {
    const res = await trackApi.list(limit, offset.value, {
      q: searchQuery.value.trim() || undefined,
      provider: filterProvider.value || undefined,
      mood: filterMood.value || undefined,
      sort: sortBy.value || undefined,
    });
    tracks.value = res.items;
    total.value = res.total;
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

function goPage(p: number): void {
  offset.value = Math.max(0, (p - 1) * limit);
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
