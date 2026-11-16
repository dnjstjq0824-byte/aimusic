<template>
  <div class="space-y-6">
    <UiPageHeader
      :eyebrow="t('business.console.ovEyebrow')"
      :title="t('business.console.ovTitle')"
      :description="t('business.console.ovDesc')"
    >
      <template #actions>
        <RouterLink to="/business/console/stores">
          <UiButton variant="brand" size="sm">
            {{ t('business.console.ovStoresCta') }}
          </UiButton>
        </RouterLink>
      </template>
    </UiPageHeader>

    <UiErrorBanner :message="error" :title="t('business.console.ovLoadFail')" :retry-label="t('common.actions.retry')" @retry="load" />

    <div v-if="profile && profile.status !== 'APPROVED'" class="rounded-xl border border-amber-500/40 bg-amber-500/10 px-4 py-3 text-xs text-amber-200">
      {{ t('business.console.ovPending') }}
    </div>

    <div v-if="loading" class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <UiCard v-for="i in 4" :key="i" class="space-y-2">
        <UiSkeleton height="h-3" width="w-1/2" />
        <UiSkeleton height="h-8" width="w-2/3" />
      </UiCard>
    </div>
    <div v-else class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <KpiCard :title="t('business.console.kpiStores')" :value="String(stores.length)" :sub="approvedSub" />
      <KpiCard :title="t('business.console.kpiPlayers')" :value="String(activeDevices)" :sub="t('business.console.kpiPlayersSub', { n: devices.length })" tone="brand" />
      <KpiCard :title="t('business.console.kpiPlays')" :value="countedPlays.toLocaleString()" :sub="t('business.console.kpiPlaysSub')" tone="success" />
      <KpiCard :title="t('business.console.kpiLicenses')" :value="String(totalLicenses)" />
    </div>

    <UiCard v-if="!loading">
      <template #header>
        <h2 class="text-xs font-bold text-text-primary uppercase tracking-wider">{{ t('business.console.alertTitle') }}</h2>
      </template>
      <ul v-if="offlineDevices.length > 0" class="space-y-2 text-xs">
        <li v-for="d in offlineDevices" :key="d.id" class="flex items-center justify-between gap-2 rounded-xl border border-surface-border bg-surface-2 px-3 py-2">
          <span class="text-text-primary">{{ d.deviceName }} <span class="text-text-muted">· {{ storeNameOf(d.businessId) }}</span></span>
          <UiBadge variant="warning" size="xs">{{ t('business.console.offline') }}</UiBadge>
        </li>
      </ul>
      <p v-else class="text-xs text-text-secondary">{{ t('business.console.alertEmpty') }}</p>
    </UiCard>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { computed, onMounted, ref } from 'vue';
import { consoleApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import type { BusinessProfile, StoreBusiness, StoreDevice } from '@/api/types';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import KpiCard from '@/components/studio/KpiCard.vue';

const { t } = useI18n();

const OFFLINE_AFTER_MS = 15 * 60_000;

const profile = ref<BusinessProfile | null>(null);
const stores = ref<StoreBusiness[]>([]);
const devices = ref<(StoreDevice & { businessId: string })[]>([]);
const countedPlays = ref(0);
const loading = ref(true);
const error = ref<string | null>(null);

const activeDevices = computed(() => devices.value.filter((d) => d.status === 'ACTIVE').length);
const totalLicenses = computed(() => stores.value.reduce((acc, s) => acc + (s.licenseCount || 0), 0));
const approvedSub = computed(() => {
  const n = stores.value.filter((s) => s.status === 'APPROVED').length;
  return t('business.console.kpiStoresSub', { n });
});
const offlineDevices = computed(() => {
  const cutoff = Date.now() - OFFLINE_AFTER_MS;
  return devices.value.filter(
    (d) => d.status === 'ACTIVE' && (!d.lastSeenAt || new Date(d.lastSeenAt).getTime() < cutoff),
  );
});

function storeNameOf(businessId: string): string {
  return stores.value.find((s) => s.id === businessId)?.storeName ?? '';
}

async function load(): Promise<void> {
  loading.value = true;
  error.value = null;
  try {
    profile.value = await consoleApi.profile().catch(() => null);
    const list = (await consoleApi.listStores()) ?? [];
    stores.value = list;
    const perStore = await Promise.all(
      list.map(async (s) => {
        try {
          return await consoleApi.listStoreDevices(s.id);
        } catch {
          return [];
        }
      }),
    );
    devices.value = perStore.flat();
    let counted = 0;
    await Promise.all(
      devices.value.slice(0, 20).map(async (d) => {
        try {
          const logs = (await consoleApi.storeDeviceLogs(d.businessId, d.id, 50)) ?? [];
          counted += logs.filter((l) => l.countedAsPlay).length;
        } catch {
          // A failing device must not break the overview.
        }
      }),
    );
    countedPlays.value = counted;
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
}

onMounted(() => void load());
</script>
