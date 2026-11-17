<!-- Creator profile (WI-S4, UI §3.8): SSR public page with follow + share. -->
<template>
  <div class="space-y-6">
    <h1 v-if="error" class="sr-only">{{ t('seo.creatorTitle', { name: handle }) }}</h1>
    <UiErrorBanner
      :message="error"
      :title="t('seo.creatorNotFound')"
      :retry-label="t('track.retry')"
      @retry="reload"
    />

    <div v-if="!error && loading" class="space-y-4">
      <UiSkeleton height="h-32" width="w-full" rounded="xl" />
      <UiSkeleton height="h-64" width="w-full" rounded="xl" />
    </div>

    <template v-else-if="!error && profile">
      <UiCard>
        <div class="flex flex-col gap-4 sm:flex-row sm:items-center">
          <UiAvatar :name="profile.displayName" size="xl" />
          <div class="min-w-0 flex-1">
            <h1 class="font-display truncate text-2xl font-black text-white">{{ profile.displayName }}</h1>
            <p class="mt-1 text-xs text-text-secondary">
              @{{ profile.handle }} · {{ t('seo.creatorStats', { tracks: fmt.number(profile.trackCount), plays: fmt.number(profile.totalPlays) }) }}
            </p>
          </div>
          <div class="flex items-center gap-2">
            <UiButton variant="secondary" size="sm" :aria-pressed="following" @click="toggleFollow">
              {{ following ? t('library.unfollow') : t('library.follow') }}
            </UiButton>
            <UiButton variant="outline" size="sm" @click="copyLink">
              {{ t('common.actions.share') }}
            </UiButton>
          </div>
        </div>
        <div class="mt-4 border-t border-surface-border pt-4">
          <ShareButtons :url="shareUrl" :title="profile.displayName" />
        </div>
      </UiCard>

      <UiTabs v-model="tab" :tabs="tabs" />

      <section v-if="tab === 'top'" :aria-label="tabs[0]?.label ?? ''">
        <div v-if="topTracks.length === 0" class="py-8 text-center text-xs text-text-muted">
          {{ t('seo.creatorNoTracks') }}
        </div>
        <ol v-else class="space-y-1">
          <li v-for="(tr, i) in topTracks" :key="tr.id">
            <TrackRow :track="tr" :rank="i + 1" @play="playOther" @enqueue="enqueueOther" />
          </li>
        </ol>
      </section>

      <section v-else-if="tab === 'all'" :aria-label="tabs[1]?.label ?? ''">
        <div v-if="allTracks.length === 0" class="py-8 text-center text-xs text-text-muted">
          {{ t('seo.creatorNoTracks') }}
        </div>
        <ol v-else class="space-y-1">
          <li v-for="tr in allTracks" :key="tr.id">
            <TrackRow :track="tr" @play="playOther" @enqueue="enqueueOther" />
          </li>
        </ol>
      </section>

      <section v-else-if="tab === 'playlists'" :aria-label="tabs[2]?.label ?? ''">
        <div v-if="creatorPlaylists.length === 0" class="py-8 text-center text-xs text-text-muted">
          {{ t('seo.creatorNoPlaylists') }}
        </div>
        <div v-else class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          <RouterLink
            v-for="pl in creatorPlaylists"
            :key="pl.id"
            :to="localePath({ name: 'playlist-detail', params: { id: detailParam(pl) } })"
            class="group"
          >
            <CoverMosaic :seeds="[pl.id]" :title="pl.title" size="md" />
            <span class="mt-1.5 block truncate text-xs font-bold text-white group-hover:text-brand-400">{{ pl.title }}</span>
          </RouterLink>
        </div>
      </section>

      <section v-else :aria-label="tabs[3]?.label ?? ''">
        <UiCard class="space-y-2 text-xs text-text-primary">
          <p>{{ t('seo.creatorAbout', { name: profile.displayName, tracks: fmt.number(profile.trackCount) }) }}</p>
          <p class="text-text-secondary">{{ t('seo.creatorAboutRights') }}</p>
        </UiCard>
      </section>
    </template>

  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { engagementApi } from '@/api/endpoints';
import { useAuthStore } from '@/stores/authStore';
import { usePlayerStore } from '@/stores/playerStore';
import { useToast } from '@/composables/useToast';
import { useApiError } from '@/composables/useApiError';
import { useFormat } from '@/composables/useFormat';
import { usePublicSeo } from '@/composables/usePublicSeo';
import { fetchPublic } from '@/utils/publicApi';
import {
  breadcrumbLd,
  canonicalFor,
  creatorTitle,
  describeCreator,
  detailParam,
  needsCjkThaiFont,
  personLd,
} from '@/utils/seo';
import { trackEvent } from '@/utils/analytics';
import type { Playlist, Track } from '@/api/types';
import UiAvatar from '@/components/ui/UiAvatar.vue';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import UiTabs from '@/components/ui/UiTabs.vue';
import CoverMosaic from '@/components/ui/CoverMosaic.vue';
import TrackRow from '@/components/TrackRow.vue';
import ShareButtons from '@/components/seo/ShareButtons.vue';

interface CreatorPayload {
  profile: { handle: string; displayName: string; userId: string; partnerId: string; trackCount: number; totalPlays: number };
  tracks: Track[];
  playlists: Playlist[];
}

const localePath = useLocalePath();
const { t, locale } = useI18n();
const route = useRoute();
const router = useRouter();
const auth = useAuthStore();
const player = usePlayerStore();
const toast = useToast();
const apiError = useApiError();
const fmt = useFormat();

const handle = computed(() => String(route.params.handle ?? ''));
const { data: pageData, pending, error: pageError, refresh } = await useAsyncData(
  () => `creator-${handle.value}`,
  () => fetchPublic<CreatorPayload>(`/creators/${encodeURIComponent(handle.value)}`),
);
const profile = computed(() => pageData.value?.profile ?? null);
const allTracks = computed(() => pageData.value?.tracks ?? []);
const topTracks = computed(() => [...allTracks.value].sort((a, b) => b.playCount - a.playCount));
const creatorPlaylists = computed(() => pageData.value?.playlists ?? []);
const loading = computed(() => pending.value);
const error = ref<string | null>(null);

if (pageError.value) {
  const st = (pageError.value as { statusCode?: number }).statusCode ?? 0;
  if (st === 404) {
    throw showError({ statusCode: 404, statusMessage: 'Creator not found' });
  }
  error.value = apiError(pageError.value);
}

async function reload(): Promise<void> {
  error.value = null;
  await refresh();
  if (pageError.value) error.value = apiError(pageError.value);
}

const pagePath = computed(() => `/creators/${handle.value}`);
const shareUrl = computed(() => canonicalFor(pagePath.value, locale.value));

if (profile.value && !pageError.value) {
  const p = profile.value;
  const description = describeCreator(t, p.displayName, p.trackCount);
  usePublicSeo({
    title: creatorTitle(t, p.displayName),
    description,
    path: pagePath.value,
    locale: locale.value,
    ogType: 'profile',
    image: null,
  });
  useSchemaOrg([
    personLd(
      p.displayName,
      shareUrl.value,
      allTracks.value
        .slice(0, 10)
        .map((x) => canonicalFor(`/tracks/${x.slug && x.shortId ? `${x.slug}-${x.shortId}` : x.id}`, locale.value)),
    ),
    breadcrumbLd([
      { name: 'AVIBE', url: canonicalFor('/', locale.value) },
      { name: p.displayName, url: shareUrl.value },
    ]),
  ]);
  defineOgImageComponent('AvibeOg', {
    title: p.displayName,
    creator: `@${p.handle}`,
    verified: false,
    plain: needsCjkThaiFont(`${p.displayName} ${p.handle}`),
  });
}

const tab = ref('top');
const tabs = computed(() => [
  { value: 'top', label: t('seo.creatorTabTop') },
  { value: 'all', label: t('seo.creatorTabAll') },
  { value: 'playlists', label: t('seo.creatorTabPlaylists') },
  { value: 'about', label: t('seo.creatorTabAbout') },
]);

const following = ref(false);

function requireLogin(): boolean {
  if (auth.isAuthenticated) return true;
  toast.info(t('playlist.toastNeedLogin'));
  void router.push(localePath({ name: 'login', query: { redirect: route.fullPath } }));
  return false;
}

async function toggleFollow(): Promise<void> {
  if (!requireLogin() || !profile.value) return;
  try {
    if (following.value) {
      await engagementApi.unfollow('creator', profile.value.partnerId);
      following.value = false;
      toast.info(t('library.toastUnfollowed'));
    } else {
      await engagementApi.follow('creator', profile.value.partnerId);
      following.value = true;
      toast.success(t('library.toastFollowed'));
    }
  } catch (err: unknown) {
    toast.error(apiError(err));
  }
}

async function copyLink(): Promise<void> {
  try {
    await navigator.clipboard.writeText(shareUrl.value);
    trackEvent({ name: 'share_clicked', props: { channel: 'copy' } });
    toast.success(t('track.toastLinkCopied'));
  } catch {
    toast.error(t('track.toastCopyFailed'));
  }
}

function playOther(tr: Track): void {
  player.setQueue([tr], tr.id);
}

function enqueueOther(tr: Track): void {
  player.enqueue(tr);
  toast.info(t('track.toastQueued'));
}

onMounted(() => {
  if (auth.isAuthenticated && profile.value) {
    const pid = profile.value.partnerId;
    void engagementApi
      .follows('creator', 100)
      .then((mine) => {
        following.value = (mine ?? []).some((f) => f.targetId === pid);
      })
      .catch(() => false);
  }
});
</script>
