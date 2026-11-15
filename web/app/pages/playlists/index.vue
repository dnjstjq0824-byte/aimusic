<template>
  <div class="space-y-6">
    <UiPageHeader :title="t('playlist.title')" :description="t('playlist.sub')" />

    <UiErrorBanner :message="error" :title="t('playlist.errTitle')" :retry-label="t('track.retry')" @retry="load" />

    <div class="flex flex-wrap items-center gap-2">
      <div class="flex gap-1 rounded-lg border border-surface-border bg-surface-1 p-1" role="tablist" :aria-label="t('playlist.title')">
        <button
          type="button"
          role="tab"
          :aria-selected="tab === 'featured'"
          class="rounded-md px-3 py-1.5 text-xs font-bold transition-colors"
          :class="tab === 'featured' ? 'bg-brand-500 text-on-brand' : 'text-text-secondary hover:text-white'"
          @click="tab = 'featured'"
        >
          {{ t('playlist.tabFeatured') }}
        </button>
        <button
          type="button"
          role="tab"
          :aria-selected="tab === 'latest'"
          class="rounded-md px-3 py-1.5 text-xs font-bold transition-colors"
          :class="tab === 'latest' ? 'bg-brand-500 text-on-brand' : 'text-text-secondary hover:text-white'"
          @click="tab = 'latest'"
        >
          {{ t('playlist.tabLatest') }}
        </button>
      </div>
      <ClientOnly>
        <RouterLink
          v-if="auth.isAuthenticated"
          :to="localePath('/me/playlists')"
          class="text-xs font-semibold text-brand-400 hover:underline"
        >
          {{ t('playlist.tabMine') }} →
        </RouterLink>
      </ClientOnly>
    </div>

    <div v-if="loading" class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
      <UiSkeleton v-for="i in 8" :key="i" height="h-44" width="w-full" rounded="xl" />
    </div>

    <div v-else-if="!error && shown.length === 0" class="py-10">
      <UiEmptyState :title="t('playlist.emptyT')" :description="t('playlist.emptyD')" />
    </div>

    <template v-else>
      <section v-if="hero" :aria-label="t('playlist.editorTitle')">
        <RouterLink
          :to="localePath({ name: 'playlist-detail', params: { id: hero.id } })"
          class="group flex flex-col gap-4 rounded-2xl border border-surface-border bg-surface-1 p-4 transition-colors hover:border-brand-500/50 sm:flex-row sm:items-center"
        >
          <CoverMosaic :seeds="heroSeeds" :title="hero.title" size="lg" />
          <div class="min-w-0">
            <p class="text-caption font-bold uppercase tracking-wider text-brand-400">{{ t('playlist.editorTitle') }}</p>
            <h2 class="font-display mt-1 truncate text-xl font-black text-white group-hover:text-brand-300">{{ hero.title }}</h2>
          </div>
        </RouterLink>
      </section>

      <section :aria-label="t('playlist.gridTitle')">
        <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <RouterLink
            v-for="pl in grid"
            :key="pl.id"
            :to="localePath({ name: 'playlist-detail', params: { id: pl.id } })"
            class="group"
          >
            <CoverMosaic :seeds="[pl.id]" :title="pl.title" size="md" />
            <span class="mt-1.5 block truncate text-xs font-bold text-white group-hover:text-brand-400">{{ pl.title }}</span>
          </RouterLink>
        </div>
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { playlistApi } from '@/api/endpoints';
import { useAuthStore } from '@/stores/authStore';
import { useApiError } from '@/composables/useApiError';
import type { Playlist } from '@/api/types';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import CoverMosaic from '@/components/ui/CoverMosaic.vue';

const { t } = useI18n();
const localePath = useLocalePath();
const auth = useAuthStore();
const apiError = useApiError();

const tab = ref<'featured' | 'latest'>('featured');
const items = ref<Playlist[]>([]);
const heroSeeds = ref<string[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);

const shown = computed(() => {
  const list = [...items.value];
  if (tab.value === 'latest') {
    list.sort((a, b) => +new Date(b.createdAt) - +new Date(a.createdAt));
  }
  return list;
});

const hero = computed(() => shown.value[0] ?? null);
const grid = computed(() => shown.value.slice(1));

async function load(): Promise<void> {
  loading.value = true;
  error.value = null;
  try {
    items.value = (await playlistApi.listPublic(24)) ?? [];
    const first = items.value[0];
    if (first) {
      try {
        const tracks = await playlistApi.listTracks(first.id);
        heroSeeds.value = tracks.slice(0, 4).map((x) => x.trackId);
      } catch {
        heroSeeds.value = [first.id];
      }
    }
  } catch (err: unknown) {
    error.value = apiError(err);
  } finally {
    loading.value = false;
  }
}

onMounted(() => {
  void load();
});
</script>
