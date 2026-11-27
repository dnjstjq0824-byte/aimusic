<template>
  <div class="space-y-6">
    <UiPageHeader :title="t('library.title')">
      <template #actions>
        <UiButton
          v-if="canPlayAll"
          variant="brand"
          size="sm"
          :disabled="loading"
          @click="playAll"
        >
          {{ t('library.playAll') }}
        </UiButton>
      </template>
    </UiPageHeader>

    <UiErrorBanner :message="error" :title="t('track.errTitle')" :retry-label="t('track.retry')" @retry="load" />

    <div class="flex gap-1 overflow-x-auto rounded-xl border border-surface-border bg-surface-1 p-1 scrollbar-none" role="tablist" :aria-label="t('library.title')">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        type="button"
        role="tab"
        :aria-selected="activeTab === tab.id"
        class="flex-1 whitespace-nowrap rounded-lg px-3 py-2 text-xs font-bold transition-colors"
        :class="activeTab === tab.id ? 'bg-brand-500 text-on-brand' : 'text-text-secondary hover:text-white'"
        @click="activeTab = tab.id"
      >
        {{ t(tab.labelKey) }}
      </button>
    </div>

    <div v-if="loading" class="space-y-2">
      <UiSkeleton v-for="i in 6" :key="i" height="h-12" width="w-full" rounded="lg" />
    </div>

    <template v-else-if="!error">
      <!-- My playlists -->
      <div v-if="activeTab === 'playlists'" class="space-y-3">
        <div class="flex items-center justify-between gap-2">
          <p class="text-xs text-text-secondary" role="status">{{ t('playlist.metaTracks', { count: playlists.playlists.length }) }}</p>
          <RouterLink :to="localePath('/me/playlists')" class="text-xs font-semibold text-brand-400 hover:underline">
            {{ t('library.managePlaylists') }} →
          </RouterLink>
        </div>
        <div v-if="playlists.playlists.length === 0" class="py-10">
          <UiEmptyState :title="t('playlist.meEmptyT')" :description="t('playlist.meEmptyD')">
            <template #action>
              <RouterLink :to="localePath('/me/playlists')">
                <UiButton variant="brand" size="sm">{{ t('playlist.meEmptyCta') }}</UiButton>
              </RouterLink>
            </template>
          </UiEmptyState>
        </div>
        <div v-else class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
          <RouterLink
            v-for="pl in playlists.playlists"
            :key="pl.id"
            :to="localePath({ name: 'playlist-detail', params: { id: detailParam(pl) } })"
            class="group block min-w-0"
          >
            <CoverMosaic :seeds="pl.coverSeeds ?? []" :title="pl.title" size="md" class="!h-auto !w-full aspect-square" />
            <span class="mt-2 block truncate text-sm font-bold text-white group-hover:text-brand-400">{{ pl.title }}</span>
            <span class="block text-caption text-text-muted">
              {{ pl.isPublic ? t('playlist.detailPublic') : t('playlist.detailPrivate') }} · {{ t('playlist.metaTracks', { count: pl.trackCount ?? 0 }) }}
            </span>
          </RouterLink>
        </div>
      </div>

      <!-- Liked tracks -->
      <div v-if="activeTab === 'liked'" class="space-y-1">
        <div v-if="liked.length === 0" class="py-10">
          <UiEmptyState :title="t('library.likedEmptyT')" :description="t('library.likedEmptyD')">
            <template #action>
              <RouterLink :to="localePath('/explore')">
                <UiButton variant="secondary" size="sm">{{ t('library.likedEmptyCta') }}</UiButton>
              </RouterLink>
            </template>
          </UiEmptyState>
        </div>
        <TrackRow
          v-for="tr in liked"
          :key="tr.id"
          :track="tr"
          @play="(x) => playFrom(liked, x)"
        />
      </div>

      <!-- Recent plays -->
      <div v-if="activeTab === 'recent'" class="space-y-1">
        <div v-if="recent.length === 0" class="py-10">
          <UiEmptyState :title="t('library.recentEmptyT')" :description="t('library.recentEmptyD')" />
        </div>
        <TrackRow
          v-for="tr in recent"
          :key="tr.id"
          :track="tr"
          @play="(x) => playFrom(recent, x)"
        />
      </div>

      <!-- Saved playlists -->
      <div v-if="activeTab === 'saved'" class="space-y-3">
        <div v-if="saved.length === 0" class="py-10">
          <UiEmptyState :title="t('library.savedEmptyT')" :description="t('library.savedEmptyD')" />
        </div>
        <div v-else class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          <UiCard v-for="pl in saved" :key="pl.id" class="group">
            <RouterLink :to="localePath({ name: 'playlist-detail', params: { id: detailParam(pl) } })" class="block">
              <CoverMosaic :seeds="pl.coverSeeds?.length ? pl.coverSeeds : [pl.id]" :title="pl.title" size="md" />
              <span class="mt-2 block truncate text-sm font-bold text-white group-hover:text-brand-400">{{ pl.title }}</span>
            </RouterLink>
            <button
              type="button"
              class="mt-1 text-caption text-text-muted hover:text-accent-crimson hover:underline"
              @click="unsavePlaylist(pl.id)"
            >
              {{ t('library.unsave') }}
            </button>
          </UiCard>
        </div>
      </div>

      <!-- Follows -->
      <div v-if="activeTab === 'follows'" class="space-y-6">
        <div v-if="follows.length === 0" class="py-10">
          <UiEmptyState :title="t('library.followsEmptyT')" :description="t('library.followsEmptyD')" />
        </div>
        <section v-if="followCreators.length > 0">
          <h3 class="mb-2 text-sm font-bold text-white">{{ t('library.followsCreatorsT') }}</h3>
          <ul class="space-y-1">
            <li
              v-for="f in followCreators"
              :key="f.targetId"
              class="flex items-center justify-between gap-2 rounded-xl border border-surface-border bg-surface-1 px-3 py-2"
            >
              <RouterLink
                v-if="f.handle"
                :to="localePath({ name: 'creators-handle', params: { handle: f.handle } })"
                class="truncate text-xs font-bold text-white hover:text-brand-400"
              >
                {{ f.displayName }}
              </RouterLink>
              <span v-else class="truncate text-xs font-bold text-white">{{ f.displayName }}</span>
              <button
                type="button"
                class="shrink-0 text-caption text-text-muted hover:text-accent-crimson hover:underline"
                @click="unfollow('creator', f.targetId)"
              >
                {{ t('library.unfollow') }}
              </button>
            </li>
          </ul>
        </section>
        <section v-if="followPlaylists.length > 0">
          <h3 class="mb-2 text-sm font-bold text-white">{{ t('library.followsPlaylistsT') }}</h3>
          <ul class="space-y-1">
            <li
              v-for="f in followPlaylists"
              :key="f.targetId"
              class="flex items-center justify-between gap-2 rounded-xl border border-surface-border bg-surface-1 px-3 py-2"
            >
              <RouterLink
                :to="localePath({ name: 'playlist-detail', params: { id: f.targetId } })"
                class="truncate text-xs font-bold text-white hover:text-brand-400"
              >
                {{ f.displayName }}
              </RouterLink>
              <button
                type="button"
                class="shrink-0 text-caption text-text-muted hover:text-accent-crimson hover:underline"
                @click="unfollow('playlist', f.targetId)"
              >
                {{ t('library.unfollow') }}
              </button>
            </li>
          </ul>
        </section>
      </div>

    </template>

  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { computed, onMounted, ref } from 'vue';
import { engagementApi, playlistApi, playbackApi } from '@/api/endpoints';
import { detailParam } from '@/utils/seo';
import { usePlayerStore } from '@/stores/playerStore';
import { usePlaylistStore } from '@/stores/playlistStore';
import { useToast } from '@/composables/useToast';
import { useApiError } from '@/composables/useApiError';
import type { Track, Playlist, Follow } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';
import CoverMosaic from '@/components/ui/CoverMosaic.vue';
import TrackRow from '@/components/TrackRow.vue';

const { t } = useI18n();
const localePath = useLocalePath();
const player = usePlayerStore();
const playlists = usePlaylistStore();
const toast = useToast();
const apiError = useApiError();

// NOTE: tab labelKeys resolve dynamically (invisible to check-i18n usage
// scan — they are used, do not delete).
const tabs = [
  { id: 'playlists', labelKey: 'library.tabPlaylists' },
  { id: 'liked', labelKey: 'library.tabLiked' },
  { id: 'recent', labelKey: 'library.tabRecent' },
  { id: 'saved', labelKey: 'library.tabSaved' },
  { id: 'follows', labelKey: 'library.tabFollows' },
] as const;
const activeTab = ref<(typeof tabs)[number]['id']>('playlists');

const liked = ref<Track[]>([]);
const recent = ref<Track[]>([]);
const saved = ref<Playlist[]>([]);
const follows = ref<Follow[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);


const followCreators = computed(() => follows.value.filter((f) => f.targetType === 'creator'));
const followPlaylists = computed(() => follows.value.filter((f) => f.targetType === 'playlist'));

const canPlayAll = computed(
  () => (activeTab.value === 'liked' && liked.value.length > 0) ||
    (activeTab.value === 'recent' && recent.value.length > 0),
);

async function load(): Promise<void> {
  loading.value = true;
  error.value = null;
  try {
    const [likedRes, recentRes, savedRes, followsRes] = await Promise.all([
      engagementApi.likedTracks(50),
      playbackApi.recent(20),
      engagementApi.savedPlaylists(50),
      engagementApi.follows(undefined, 100),
      playlists.fetchMine(),
    ]);
    liked.value = likedRes ?? [];
    recent.value = recentRes ?? [];
    saved.value = savedRes ?? [];
    follows.value = followsRes ?? [];
  } catch (err: unknown) {
    error.value = apiError(err);
  } finally {
    loading.value = false;
  }
}

// A row plays its whole list from that row, so the next one follows.
function playFrom(list: Track[], tr: Track): void {
  player.setQueue(list, tr.id);
}

function playAll(): void {
  const list = activeTab.value === 'liked' ? liked.value : recent.value;
  if (list.length === 0) return;
  player.setQueue(list, list[0]?.id);
}

async function unsavePlaylist(id: string): Promise<void> {
  try {
    await playlistApi.unsave(id);
    saved.value = saved.value.filter((p) => p.id !== id);
    toast.info(t('library.toastUnfollowed'));
  } catch (err: unknown) {
    toast.error(apiError(err));
  }
}

async function unfollow(targetType: 'creator' | 'playlist', targetId: string): Promise<void> {
  try {
    await engagementApi.unfollow(targetType, targetId);
    follows.value = follows.value.filter((f) => !(f.targetType === targetType && f.targetId === targetId));
    toast.info(t('library.toastUnfollowed'));
  } catch (err: unknown) {
    toast.error(apiError(err));
  }
}

onMounted(() => {
  void load();
});
</script>
