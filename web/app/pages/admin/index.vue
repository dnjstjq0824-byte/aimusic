<template>
  <div class="space-y-6">
    <UiPageHeader
      :eyebrow="t('admin.ov.eyebrow')"
      :title="t('admin.ov.title')"
      :description="t('admin.ov.desc')"
    >
      <template #actions>
        <UiButton variant="secondary" size="sm" :loading="loading" @click="load">{{ t('admin.ov.refresh') }}</UiButton>
      </template>
    </UiPageHeader>

    <UiErrorBanner :message="error" :title="t('admin.ov.loadFail')" :retry-label="t('common.actions.retry')" @retry="load" />

    <div v-if="loading" class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <UiCard v-for="i in 4" :key="i" class="space-y-2">
        <UiSkeleton height="h-3" width="w-1/2" />
        <UiSkeleton height="h-8" width="w-2/3" />
      </UiCard>
    </div>

    <template v-else-if="overview">
      <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <UiCard class="space-y-1">
          <div class="text-caption font-bold text-text-muted uppercase tracking-wider">{{ t('admin.ov.users') }}</div>
          <div class="text-2xl font-semibold text-white">{{ t('admin.ov.usersValue', { n: overview.users.total.toLocaleString() }) }}</div>
          <div class="text-caption text-text-secondary">
            {{ t('admin.ov.usersBreak', { u: overview.users.users.toLocaleString(), p: overview.users.partners.toLocaleString(), a: overview.users.admins.toLocaleString() }) }}
          </div>
        </UiCard>
        <UiCard class="space-y-1">
          <div class="text-caption font-bold text-text-muted uppercase tracking-wider">{{ t('admin.ov.tracks') }}</div>
          <div class="text-2xl font-semibold text-success">{{ t('admin.ov.tracksValue', { n: overview.tracks.ready.toLocaleString() }) }}</div>
          <div class="text-caption text-text-secondary">
            {{ t('admin.ov.tracksBreak', { p: overview.tracks.pendingGate.toLocaleString(), d: overview.tracks.takenDown.toLocaleString() }) }}
          </div>
        </UiCard>
        <UiCard class="space-y-1">
          <div class="text-caption font-bold text-text-muted uppercase tracking-wider">{{ t('admin.ov.plays') }}</div>
          <div class="text-2xl font-semibold text-white">{{ t('admin.ov.playsValue', { n: overview.plays.last24h.toLocaleString() }) }}</div>
          <div class="text-caption text-text-secondary">{{ t('admin.ov.playsTotal', { n: overview.plays.total.toLocaleString() }) }}</div>
        </UiCard>
        <UiCard class="space-y-1">
          <div class="text-caption font-bold text-text-muted uppercase tracking-wider">{{ t('admin.ov.settles') }}</div>
          <div class="text-2xl font-semibold text-brand-300">{{ t('admin.ov.settlesValue', { n: overview.settlementsPending.count.toLocaleString() }) }}</div>
          <div class="text-caption text-text-secondary">{{ t('admin.ov.settlesNet', { n: overview.settlementsPending.netKrw.toLocaleString() }) }}</div>
        </UiCard>
      </div>

      <div class="grid gap-4 lg:grid-cols-2">
        <UiCard>
          <template #header>
            <h2 class="text-xs font-bold text-text-primary uppercase tracking-wider">{{ t('admin.ov.queueTitle') }}</h2>
          </template>
          <ul class="space-y-2 text-xs">
            <li>
              <RouterLink to="/admin/reviews" class="flex items-center justify-between rounded-xl border border-surface-border bg-surface-2 p-3 transition-colors hover:bg-surface-3">
                <span class="font-semibold text-white">{{ t('admin.ov.qTracks') }}</span>
                <UiBadge :variant="overview.reviews.tracks > 0 ? 'warning' : 'success'" size="sm">
                  {{ t('admin.ov.count', { n: overview.reviews.tracks }) }}
                </UiBadge>
              </RouterLink>
            </li>
            <li>
              <RouterLink to="/admin/reviews" class="flex items-center justify-between rounded-xl border border-surface-border bg-surface-2 p-3 transition-colors hover:bg-surface-3">
                <span class="font-semibold text-white">{{ t('admin.ov.qPartners') }}</span>
                <UiBadge :variant="overview.reviews.partners > 0 ? 'warning' : 'success'" size="sm">
                  {{ t('admin.ov.count', { n: overview.reviews.partners }) }}
                </UiBadge>
              </RouterLink>
            </li>
            <li>
              <RouterLink to="/admin/reviews" class="flex items-center justify-between rounded-xl border border-surface-border bg-surface-2 p-3 transition-colors hover:bg-surface-3">
                <span class="font-semibold text-white">{{ t('admin.ov.qSubs') }}</span>
                <UiBadge :variant="overview.reviews.subscriptions > 0 ? 'warning' : 'success'" size="sm">
                  {{ t('admin.ov.count', { n: overview.reviews.subscriptions }) }}
                </UiBadge>
              </RouterLink>
            </li>
            <li>
              <RouterLink to="/admin/disputes" class="flex items-center justify-between rounded-xl border border-surface-border bg-surface-2 p-3 transition-colors hover:bg-surface-3">
                <span class="font-semibold text-white">{{ t('admin.ov.qDisputes') }}</span>
                <UiBadge :variant="overview.disputesOpen > 0 ? 'danger' : 'success'" size="sm">
                  {{ t('admin.ov.count', { n: overview.disputesOpen }) }}
                </UiBadge>
              </RouterLink>
            </li>
          </ul>
        </UiCard>

        <UiCard>
          <template #header>
            <h2 class="text-xs font-bold text-text-primary uppercase tracking-wider">{{ t('admin.ov.pipeTitle') }}</h2>
          </template>
          <dl class="space-y-3 text-xs">
            <div class="flex items-center justify-between">
              <dt class="text-text-secondary">{{ t('admin.ov.queueDepth') }}</dt>
              <dd class="font-mono font-bold text-white">
                {{ overview.queueDepth === null ? t('admin.ov.redisDown') : t('admin.ov.count', { n: overview.queueDepth }) }}
              </dd>
            </div>
            <div class="flex items-center justify-between">
              <dt class="text-text-secondary">{{ t('admin.ov.fail24') }}</dt>
              <dd>
                <UiBadge :variant="overview.jobsFailed24h > 0 ? 'danger' : 'success'" size="sm">
                  {{ t('admin.ov.count', { n: overview.jobsFailed24h }) }}
                </UiBadge>
              </dd>
            </div>
            <div class="border-t border-surface-border pt-3">
              <RouterLink to="/admin/curation" class="text-brand-400 hover:underline font-semibold">
                {{ t('admin.ov.curationCta') }}
              </RouterLink>
            </div>
          </dl>
        </UiCard>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { onMounted, ref } from 'vue';
import { adminApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import type { AdminOverview } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';

const { t } = useI18n();
const overview = ref<AdminOverview | null>(null);
const loading = ref(true);
const error = ref<string | null>(null);

async function load(): Promise<void> {
  loading.value = true;
  error.value = null;
  try {
    overview.value = await adminApi.overview();
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
}

onMounted(() => {
  void load();
});
</script>
