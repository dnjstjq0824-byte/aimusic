<template>
  <div class="space-y-6">
    <UiPageHeader :title="t('explore.title')" :description="t('explore.sub')">
      <template #actions>
        <UiButton
          variant="brand"
          size="sm"
          :disabled="tracks.length === 0 || loading"
          @click="playAll"
        >
          {{ t('explore.playAll') }}
        </UiButton>
      </template>
    </UiPageHeader>

    <UiErrorBanner :message="error" :title="t('track.errTitle')" :retry-label="t('track.retry')" @retry="load" />

    <!-- Search + sort -->
    <div class="flex flex-col gap-3 sm:flex-row sm:items-end">
      <form class="flex-1" role="search" @submit.prevent="submitSearch">
        <UiInput
          v-model="searchQuery"
          :label="t('explore.searchLabel')"
          :placeholder="t('explore.searchPh')"
          @enter="submitSearch"
        />
      </form>
      <UiSelect v-model="sortBy" :label="t('explore.sortLabel')" class="sm:w-44" @change="onSortChange">
        <option value="recent">{{ t('explore.sortRecent') }}</option>
        <option value="popular">{{ t('explore.sortPopular') }}</option>
        <option value="relevant">{{ t('explore.sortRelevant') }}</option>
        <option value="oldest">{{ t('explore.sortOldest') }}</option>
        <option value="title">{{ t('explore.sortTitle') }}</option>
      </UiSelect>
      <UiButton variant="secondary" size="sm" class="lg:hidden" @click="filtersOpen = true">
        {{ t('explore.moodLabel') }} ({{ activeFilterCount }})
      </UiButton>
    </div>

    <!-- v3 flow: photo shortcuts while nothing is filtered yet. -->
    <StarterRail v-if="activeFilterCount === 0 && !hasSearchQuery" heading-id="explore-start-h" compact />

    <div class="grid gap-6 lg:grid-cols-[240px_minmax(0,1fr)]">
      <!-- Filter panel (desktop) -->
      <aside class="hidden lg:block">
        <UiCard>
          <ExploreFilters :model="filters" @update="applyFilters" />
        </UiCard>
      </aside>

      <!-- Results -->
      <section class="min-w-0 space-y-4">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <p class="text-xs text-text-secondary" role="status">
            {{ t('explore.results', { count: fmt.number(total) }) }}
          </p>
          <div class="flex items-center gap-2">
            <div class="flex gap-1 rounded-lg border border-surface-border bg-surface-1 p-1" role="group" :aria-label="t('explore.viewList')">
              <button
                type="button"
                class="whitespace-nowrap rounded-md px-2.5 py-1 text-xs font-bold transition-colors"
                :class="view === 'list' ? 'bg-brand-500 text-on-brand' : 'text-text-secondary hover:text-white'"
                :aria-pressed="view === 'list'"
                @click="view = 'list'"
              >
                {{ t('explore.viewList') }}
              </button>
              <button
                type="button"
                class="whitespace-nowrap rounded-md px-2.5 py-1 text-xs font-bold transition-colors"
                :class="view === 'grid' ? 'bg-brand-500 text-on-brand' : 'text-text-secondary hover:text-white'"
                :aria-pressed="view === 'grid'"
                @click="view = 'grid'"
              >
                {{ t('explore.viewGrid') }}
              </button>
            </div>
            <button
              v-if="hasActiveFilters"
              type="button"
              class="text-caption text-accent-crimson hover:underline"
              @click="resetFilters"
            >
              {{ t('explore.resetFilters') }}
            </button>
          </div>
        </div>

        <div v-if="chips.length > 0" class="flex flex-wrap gap-1.5">
          <button
            v-for="chip in chips"
            :key="chip.id"
            type="button"
            class="inline-flex items-center gap-1 rounded-full border border-brand-500/50 bg-brand-950 px-2.5 py-1 text-xs font-semibold text-brand-300 transition-colors hover:bg-brand-900"
            @click="chip.clear"
          >
            {{ chip.label }} ✕
          </button>
        </div>

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

        <div v-else-if="!error && tracks.length === 0" class="py-12">
          <UiEmptyState :title="t('explore.emptyTitle')" :description="t('explore.emptyDesc')">
            <template #action>
              <UiButton variant="secondary" size="sm" @click="resetFilters">{{ t('explore.resetFilters') }}</UiButton>
            </template>
          </UiEmptyState>
        </div>

        <ol v-else-if="view === 'list'" class="space-y-1">
          <li v-for="tr in tracks" :key="tr.id">
            <TrackRow
              :track="tr"
              @play="playTrack"
            />
          </li>
        </ol>

        <div v-else class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          <TrackCard
            v-for="tr in tracks"
            :key="tr.id"
            :track="tr"
            @play="playTrack"
          />
        </div>

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
      </section>
    </div>

    <BottomSheet v-model="filtersOpen" :title="t('explore.moodLabel')">
      <div class="space-y-4">
        <ExploreFilters :model="filters" @update="applyFilters" />
        <UiButton variant="brand" size="md" full-width @click="filtersOpen = false">
          {{ t('explore.results', { count: fmt.number(total) }) }}
        </UiButton>
      </div>
    </BottomSheet>

  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { trackApi } from '@/api/endpoints';
import { useAuthStore } from '@/stores/authStore';
import { usePlayerStore } from '@/stores/playerStore';
import { useToast } from '@/composables/useToast';
import { useApiError } from '@/composables/useApiError';
import { useFormat } from '@/composables/useFormat';
import { tagDisplayName } from '@/utils/tags';
import { providerDisplayName } from '@/utils/providerDisplay';
import { fetchPublic } from '@/utils/publicApi';
import { usePublicSeo } from '@/composables/usePublicSeo';
import type { Track, TrackPage } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiInput from '@/components/ui/UiInput.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import StarterRail from '@/components/landing/StarterRail.vue';
import UiSelect from '@/components/ui/UiSelect.vue';
import UiPagination from '@/components/ui/UiPagination.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';
import BottomSheet from '@/components/ui/BottomSheet.vue';
import TrackCard from '@/components/TrackCard.vue';
import TrackRow from '@/components/TrackRow.vue';
import ExploreFilters, { type ExploreFilterModel } from '@/components/explore/ExploreFilters.vue';
const localePath = useLocalePath();

const { t, te, locale } = useI18n();
const route = useRoute();
const router = useRouter();
// WI-S2: search-result URLs (?q=) are noindex,follow; the bare explore page indexes.
const hasSearchQuery = computed(() => typeof route.query.q === 'string' && route.query.q.trim() !== '');
usePublicSeo({
  title: `${t('common.titles.explore')} · AVIBE`,
  description: t('seo.exploreDescription'),
  path: '/explore',
  locale: locale.value,
  noindex: hasSearchQuery,
});
const auth = useAuthStore();
const player = usePlayerStore();
const toast = useToast();
const apiError = useApiError();
const fmt = useFormat();

const tracks = ref<Track[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);
const offset = ref(0);
const limit = 18;
const total = ref(0);
const view = ref<'list' | 'grid'>('list');
const filtersOpen = ref(false);

const searchQuery = ref((route.query.q as string) || '');
// Until the listener picks a sort, a text search ranks by relevance (exact
// title / creator matches first) and browsing ranks by recency.
const sortExplicit = ref(['recent', 'oldest', 'popular', 'title', 'relevant'].includes(route.query.sort as string));
const sortBy = ref(
  sortExplicit.value ? (route.query.sort as string) : searchQuery.value.trim() ? 'relevant' : 'recent',
);
function onSortChange(): void {
  sortExplicit.value = true;
  onSearch();
}
const filters = reactive<ExploreFilterModel>({
  mood: (route.query.mood as string) || '',
  genre: '',
  provider: '',
  certified: false,
  bpm: '',
  dur: '',
});


const BPM_RANGES: Record<string, { minBpm?: number; maxBpm?: number }> = {
  slow: { maxBpm: 90 },
  mid: { minBpm: 90, maxBpm: 120 },
  fast: { minBpm: 120 },
};
const DUR_RANGES: Record<string, { minDur?: number; maxDur?: number }> = {
  short: { maxDur: 180000 },
  mid: { minDur: 180000, maxDur: 300000 },
  long: { minDur: 300000 },
};

// SSR: render the initial result set into HTML so crawlers (and first paint)
// see tracks instead of an empty shell. Client interactions keep using load().
const { data: initialData, error: initialError } = await useAsyncData(
  () => `explore-initial-${searchQuery.value.trim()}-${sortBy.value}-${filters.mood}`,
  () => fetchPublic<TrackPage>('/tracks', {
    limit,
    offset: 0,
    q: searchQuery.value.trim() || undefined,
    sort: sortBy.value || undefined,
    mood: filters.mood || undefined,
  }),
);
if (initialData.value) {
  tracks.value = initialData.value.items ?? [];
  total.value = initialData.value.total ?? 0;
  loading.value = false;
} else if (initialError.value) {
  error.value = apiError(initialError.value);
  loading.value = false;
} else {
  loading.value = false;
}

const activeFilterCount = computed(() => {
  let n = 0;
  if (filters.mood) n++;
  if (filters.genre) n++;
  if (filters.provider) n++;
  if (filters.certified) n++;
  if (filters.bpm) n++;
  if (filters.dur) n++;
  return n;
});

const hasActiveFilters = computed(
  () => activeFilterCount.value > 0 || searchQuery.value.trim() !== '',
);

interface Chip { id: string; label: string; clear: () => void }
const chips = computed<Chip[]>(() => {
  const out: Chip[] = [];
  if (searchQuery.value.trim()) {
    out.push({ id: 'q', label: `“${searchQuery.value.trim()}”`, clear: () => { searchQuery.value = ''; onSearch(); } });
  }
  if (filters.mood) {
    out.push({ id: 'mood', label: tagDisplayName(filters.mood, t, te), clear: () => { filters.mood = ''; onSearch(); } });
  }
  if (filters.genre) {
    out.push({ id: 'genre', label: tagDisplayName(filters.genre, t, te), clear: () => { filters.genre = ''; onSearch(); } });
  }
  if (filters.provider) {
    out.push({ id: 'provider', label: providerDisplayName(filters.provider), clear: () => { filters.provider = ''; onSearch(); } });
  }
  if (filters.certified) {
    out.push({ id: 'cert', label: t('explore.certOnly'), clear: () => { filters.certified = false; onSearch(); } });
  }
  if (filters.bpm) {
    const label = filters.bpm === 'slow' ? t('explore.bpmSlow') : filters.bpm === 'mid' ? t('explore.bpmMid') : t('explore.bpmFast');
    out.push({ id: 'bpm', label, clear: () => { filters.bpm = ''; onSearch(); } });
  }
  if (filters.dur) {
    const label = filters.dur === 'short' ? t('explore.durShort') : filters.dur === 'mid' ? t('explore.durMid') : t('explore.durLong');
    out.push({ id: 'dur', label, clear: () => { filters.dur = ''; onSearch(); } });
  }
  return out;
});

async function load(): Promise<void> {
  loading.value = true;
  error.value = null;
  try {
    const res = await trackApi.list(limit, offset.value, {
      q: searchQuery.value.trim() || undefined,
      provider: filters.provider || undefined,
      genre: filters.genre || undefined,
      mood: filters.mood || undefined,
      certified: filters.certified || undefined,
      sort: sortBy.value || undefined,
      ...(filters.bpm ? BPM_RANGES[filters.bpm] : {}),
      ...(filters.dur ? DUR_RANGES[filters.dur] : {}),
    });
    tracks.value = res.items ?? [];
    total.value = res.total;
  } catch (err: unknown) {
    error.value = apiError(err);
  } finally {
    loading.value = false;
  }
}

function onSearch(): void {
  if (!sortExplicit.value) sortBy.value = searchQuery.value.trim() ? 'relevant' : 'recent';
  offset.value = 0;
  void load();
}

function applyFilters(patch: Partial<ExploreFilterModel>): void {
  Object.assign(filters, patch);
  onSearch();
}

function submitSearch(): void {
  if (!searchQuery.value.trim() && activeFilterCount.value === 0) {
    toast.info(t('explore.enterHint'));
    return;
  }
  onSearch();
}

function resetFilters(): void {
  searchQuery.value = '';
  filters.mood = '';
  filters.genre = '';
  filters.provider = '';
  filters.certified = false;
  filters.bpm = '';
  filters.dur = '';
  sortBy.value = 'recent';
  sortExplicit.value = false;
  onSearch();
}

function changePage(direction: number): void {
  offset.value = Math.max(0, offset.value + direction * limit);
  void load();
  window.scrollTo({ top: 0, behavior: 'smooth' });
}

function goPage(p: number): void {
  offset.value = Math.max(0, (p - 1) * limit);
  void load();
  window.scrollTo({ top: 0, behavior: 'smooth' });
}

// Results become the queue so the next result plays after this one.
function playTrack(tr: Track): void {
  player.setQueue(tracks.value, tr.id);
}

function playAll(): void {
  if (tracks.value.length === 0) return;
  if (!auth.isAuthenticated) {
    toast.info(t('track.toastNeedLogin'));
    void router.push(localePath({ name: 'login', query: { redirect: '/explore' } }));
    return;
  }
  player.setQueue(tracks.value, tracks.value[0]?.id);
}

watch(
  () => route.query.q,
  (q) => {
    const next = typeof q === 'string' ? q : '';
    if (next !== searchQuery.value) {
      searchQuery.value = next;
      onSearch();
    }
  },
);

watch(
  () => route.query.mood,
  (mood) => {
    const next = typeof mood === 'string' ? mood : '';
    if (next !== filters.mood) {
      filters.mood = next;
      onSearch();
    }
  },
);

</script>
