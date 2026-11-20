<template>
  <div class="space-y-6">
    <UiPageHeader
      :eyebrow="t('studio.dash.eyebrow')"
      :title="t('studio.dash.title')"
      :description="t('studio.dash.desc')"
    >
      <template #actions>
        <RouterLink to="/studio/upload">
          <UiButton variant="brand" size="sm">
            {{ t('studio.dash.uploadCta') }}
          </UiButton>
        </RouterLink>
        <RouterLink to="/studio/catalog">
          <UiButton variant="secondary" size="sm">
            {{ t('studio.dash.catalogCta') }}
          </UiButton>
        </RouterLink>
      </template>
    </UiPageHeader>

    <UiErrorBanner :message="error" :title="t('studio.dash.loadFail')" />

    <div v-if="loading" class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <UiCard v-for="i in 4" :key="i" class="space-y-2">
        <UiSkeleton height="h-3" width="w-1/2" />
        <UiSkeleton height="h-8" width="w-2/3" />
        <UiSkeleton height="h-3" width="w-3/4" />
      </UiCard>
    </div>
    <div v-else class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <KpiCard :title="t('studio.dash.kpiPlays')" :value="t('studio.dash.kpiPlaysValue', { n: totalPlays.toLocaleString() })" :sub="t('studio.dash.kpiPlaysSub')" />
      <KpiCard :title="t('studio.dash.kpiCertified')" :value="t('studio.dash.kpiCertifiedValue', { n: certifiedCount })" :sub="t('studio.dash.kpiCertifiedSub')" tone="success" />
      <KpiCard :title="t('studio.dash.kpiReviewing')" :value="t('studio.dash.kpiReviewingValue', { n: reviewingCount })" :sub="t('studio.dash.kpiReviewingSub')" tone="brand" />
      <KpiCard :title="t('studio.dash.kpiSettlement')" :value="t('studio.dash.kpiSettlementValue')" :sub="t('studio.dash.kpiSettlementSub')" />
    </div>

    <UiCard v-if="!loading" border-variant="brand">
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div class="space-y-1">
          <h3 class="text-base font-medium text-white">{{ t('studio.dash.settleTitle') }}</h3>
          <p class="text-xs text-text-primary">
            {{ t('studio.dash.settleDesc', { plays: totalPlays.toLocaleString() }) }}
          </p>
        </div>
        <RouterLink to="/studio/earnings" class="text-xs text-brand-400 hover:underline shrink-0">
          {{ t('studio.dash.settleCta') }}
        </RouterLink>
      </div>
    </UiCard>

    <div v-if="!loading" class="grid gap-4 lg:grid-cols-2">
      <UiCard>
        <template #header>
          <h2 class="text-base font-medium text-white">{{ t('studio.dash.todosTitle') }}</h2>
        </template>
        <ul v-if="todos.length > 0" class="space-y-2 text-sm">
          <li v-for="todo in todos" :key="todo.key" class="flex items-center justify-between gap-2 rounded-2xl bg-white/[0.04] px-4 py-3">
            <span class="text-text-primary">{{ todo.label }}</span>
            <RouterLink :to="todo.to" class="shrink-0 rounded-full border border-white/10 px-3 py-1 text-xs text-white transition-colors hover:border-white/25">
              {{ t('studio.dash.todoGo') }}
            </RouterLink>
          </li>
        </ul>
        <p v-else class="text-xs text-text-secondary">{{ t('studio.dash.todosEmpty') }}</p>
      </UiCard>

      <UiCard>
        <template #header>
          <div class="flex items-center justify-between">
            <h2 class="text-base font-medium text-white">{{ t('studio.dash.recentTitle') }}</h2>
            <RouterLink to="/studio/catalog" class="text-xs font-semibold text-brand-400 hover:underline">
              {{ t('studio.dash.recentAll') }}
            </RouterLink>
          </div>
        </template>
        <ReviewTimeline v-if="recentItems.length > 0" :items="recentItems" />
        <UiEmptyState
          v-else
          :title="t('studio.dash.recentEmpty')"
          :description="t('studio.dash.recentEmptyDesc')"
        >
          <template #action>
            <RouterLink to="/studio/upload">
              <UiButton variant="brand" size="sm">{{ t('studio.dash.uploadCta') }}</UiButton>
            </RouterLink>
          </template>
        </UiEmptyState>
      </UiCard>
    </div>

    <UiCard v-if="!loading" padding="none">
      <template #header>
        <div class="px-5 pt-3 flex items-center justify-between">
          <h2 class="text-base font-medium text-white">{{ t('studio.dash.topTitle') }}</h2>
          <span class="text-xs text-text-muted font-mono">{{ topTracks.length }}</span>
        </div>
      </template>
      <p v-if="topTracks.length === 0" class="px-5 py-6 text-xs text-text-muted">
        {{ t('studio.dash.topEmpty') }}
      </p>
      <ul v-else class="divide-y divide-white/[0.06]">
        <li
          v-for="(trk, i) in topTracks"
          :key="trk.id"
          class="flex items-center gap-3 px-5 py-3 text-sm transition-colors hover:bg-white/[0.03]"
        >
          <span class="w-6 shrink-0 tabular-nums text-text-muted">{{ i + 1 }}</span>
          <RouterLink
            :to="localePath({ name: 'track-detail', params: { id: trk.id } })"
            class="min-w-0 flex-1 truncate font-medium text-white hover:text-brand-400"
          >
            {{ trk.title }}
          </RouterLink>
          <span class="shrink-0 text-text-secondary">{{ t('studio.dash.topPlays', { n: (trk.playCount || 0).toLocaleString() }) }}</span>
        </li>
      </ul>
    </UiCard>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { computed, onMounted, ref } from 'vue';
import { partnerApi, trackApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import type { PartnerAISubscription, Track } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';
import KpiCard from '@/components/studio/KpiCard.vue';
import ReviewTimeline, { type TimelineItem } from '@/components/studio/ReviewTimeline.vue';

const { t, te, locale } = useI18n();
const localePath = useLocalePath();

function statusLabel(status: string): string {
  const key = `studio.status.${status}`;
  return te(key) ? String(t(key)) : status;
}

const tracks = ref<Track[]>([]);
const subs = ref<PartnerAISubscription[]>([]);
const certifiedIds = ref<Set<string>>(new Set());
const loading = ref(true);
const error = ref<string | null>(null);

const REVIEWING = new Set(['UPLOADED', 'TRANSCODING', 'LICENSE_GATE_PENDING']);
const readyTracks = computed(() => tracks.value.filter((x) => x.status === 'READY'));
const certifiedCount = computed(() => readyTracks.value.filter((x) => certifiedIds.value.has(x.id)).length);
const reviewingCount = computed(() => tracks.value.filter((x) => REVIEWING.has(x.status)).length);
const totalPlays = computed(() => tracks.value.reduce((acc, x) => acc + (x.playCount || 0), 0));

const todos = computed(() => {
  const out: { key: string; label: string; to: string }[] = [];
  const soon = Date.now() + 30 * 86400_000;
  const expiring = subs.value.filter((s) => s.status === 'VERIFIED' && s.expiresAt && new Date(s.expiresAt).getTime() < soon).length;
  if (expiring > 0) out.push({ key: 'proof', label: t('studio.dash.todoProof', { n: expiring }), to: '/studio/proofs' });
  const rejected = tracks.value.filter((x) => x.status === 'LICENSE_GATE_REJECTED').length;
  if (rejected > 0) out.push({ key: 'rejected', label: t('studio.dash.todoRejected', { n: rejected }), to: '/studio/catalog' });
  return out;
});

function fmtDate(iso: string): string {
  try {
    return new Date(iso).toLocaleDateString(locale.value === 'zh-cn' ? 'zh-CN' : locale.value);
  } catch {
    return iso;
  }
}

const recentItems = computed<TimelineItem[]>(() =>
  [...tracks.value]
    .sort((a, b) => (a.createdAt < b.createdAt ? 1 : -1))
    .slice(0, 5)
    .map((x) => ({
      label: x.title,
      at: fmtDate(x.createdAt),
      note: statusLabel(x.status),
      state: (x.status === 'READY' ? 'done' : REVIEWING.has(x.status) ? 'now' : 'todo') as TimelineItem['state'],
    })),
);

const topTracks = computed(() =>
  [...tracks.value].sort((a, b) => (b.playCount || 0) - (a.playCount || 0)).slice(0, 5),
);

onMounted(async () => {
  loading.value = true;
  try {
    const [list, subscriptions] = await Promise.all([
      partnerApi.listMyTracks(50, 0),
      partnerApi.listSubscriptions(),
    ]);
    tracks.value = list ?? [];
    subs.value = subscriptions ?? [];
    const ready = tracks.value.filter((x) => x.status === 'READY').slice(0, 20);
    const found = await Promise.all(
      ready.map(async (x) => {
        try {
          const cert = await trackApi.getCertificate(x.id);
          return cert ? x.id : null;
        } catch {
          return null;
        }
      }),
    );
    certifiedIds.value = new Set(found.filter((id): id is string => id !== null));
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
});
</script>
