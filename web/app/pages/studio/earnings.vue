<template>
  <div class="space-y-6">
    <UiPageHeader
      :eyebrow="t('studio.earnings.eyebrow')"
      :title="t('studio.earnings.title')"
      :description="t('studio.earnings.desc')"
    />

    <UiErrorBanner :message="error" :title="t('studio.earnings.loadFail')" :retry-label="t('common.actions.retry')" @retry="load" />

    <div v-if="loading" class="grid gap-4 sm:grid-cols-3">
      <UiCard v-for="i in 3" :key="i" class="space-y-2">
        <UiSkeleton height="h-3" width="w-1/2" />
        <UiSkeleton height="h-8" width="w-2/3" />
      </UiCard>
    </div>
    <div v-else class="grid gap-4 sm:grid-cols-3">
      <KpiCard :title="t('studio.earnings.kpiPlays')" :value="totalPlays.toLocaleString()" :sub="t('studio.earnings.kpiPlaysSub')" />
      <KpiCard :title="t('studio.earnings.kpiTracks')" :value="String(readyCount)" :sub="t('studio.earnings.kpiTracksSub')" tone="success" />
      <KpiCard :title="t('studio.earnings.kpiShare')" :value="t('studio.earnings.kpiShareValue')" :sub="t('studio.earnings.kpiShareSub')" tone="brand" />
    </div>

    <UiCard v-if="!loading" border-variant="brand">
      <div class="space-y-1">
        <h3 class="text-base font-medium text-white">{{ t('studio.earnings.ruleTitle') }}</h3>
        <p class="text-xs text-text-primary">{{ t('studio.earnings.ruleDesc') }}</p>
        <p class="text-xs text-text-secondary">{{ t('studio.earnings.ruleNote') }}</p>
        <RouterLink to="/legal/PARTNER_CONTRACT" target="_blank" class="inline-block text-xs text-brand-400 hover:underline">
          {{ t('studio.earnings.contract') }}
        </RouterLink>
      </div>
    </UiCard>

    <UiCard v-if="!loading" padding="none">
      <template #header>
        <div class="px-5 pt-3 flex items-center justify-between">
          <h2 class="text-base font-medium text-white">{{ t('studio.earnings.tableTitle') }}</h2>
          <span class="text-xs text-text-muted font-mono">{{ tracks.length }}</span>
        </div>
      </template>
      <p v-if="tracks.length === 0" class="px-5 py-6 text-xs text-text-muted">
        {{ t('studio.earnings.empty') }}
      </p>
      <div v-else class="overflow-x-auto">
        <table class="w-full min-w-[560px] text-xs">
          <thead>
            <tr class="bg-surface-2/60 text-left text-text-secondary">
              <th class="px-4 py-2 font-semibold">{{ t('studio.earnings.colTitle') }}</th>
              <th class="px-4 py-2 font-semibold text-right">{{ t('studio.earnings.colPlays') }}</th>
              <th class="px-4 py-2 font-semibold text-right">{{ t('studio.earnings.colShare') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-surface-border">
            <tr v-for="trk in sorted" :key="trk.id" class="hover:bg-surface-2/40">
              <td class="px-4 py-3 font-bold text-white">{{ trk.title }}</td>
              <td class="px-4 py-3 text-right font-mono text-text-primary">{{ (trk.playCount || 0).toLocaleString() }}</td>
              <td class="px-4 py-3 text-right font-mono text-text-secondary">{{ sharePct(trk.playCount || 0) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </UiCard>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { computed, onMounted, ref } from 'vue';
import { partnerApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import type { Track } from '@/api/types';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import KpiCard from '@/components/studio/KpiCard.vue';

const { t } = useI18n();

const tracks = ref<Track[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);

const totalPlays = computed(() => tracks.value.reduce((acc, x) => acc + (x.playCount || 0), 0));
const readyCount = computed(() => tracks.value.filter((x) => x.status === 'READY').length);
const sorted = computed(() => [...tracks.value].sort((a, b) => (b.playCount || 0) - (a.playCount || 0)));

function sharePct(plays: number): string {
  const total = totalPlays.value;
  if (total <= 0) return '—';
  return `${((plays / total) * 100).toFixed(1)}%`;
}

async function load(): Promise<void> {
  loading.value = true;
  error.value = null;
  try {
    tracks.value = (await partnerApi.listMyTracks(50, 0)) ?? [];
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
}

onMounted(() => void load());
</script>
