<template>
  <div class="space-y-6">
    <UiPageHeader
      :eyebrow="t('business.console.liEyebrow')"
      :title="t('business.console.liTitle')"
      :description="t('business.console.liDesc')"
    />

    <UiErrorBanner :message="error" :title="t('business.console.liLoadFail')" :retry-label="t('common.actions.retry')" @retry="load" />

    <UiCard v-if="!loading" padding="none">
      <template #header>
        <div class="px-5 pt-3">
          <h2 class="text-xs font-bold text-text-secondary uppercase tracking-wider">{{ t('business.console.liStoresTitle') }}</h2>
        </div>
      </template>
      <p v-if="stores.length === 0" class="px-5 py-6 text-xs text-text-muted">
        {{ t('business.console.liStoresEmpty') }}
      </p>
      <div v-else class="overflow-x-auto">
        <table class="w-full min-w-[560px] text-xs">
          <thead>
            <tr class="bg-surface-2/60 text-left text-text-secondary">
              <th class="px-4 py-2 font-semibold">{{ t('business.console.liColStore') }}</th>
              <th class="px-4 py-2 font-semibold">{{ t('business.console.liColStatus') }}</th>
              <th class="px-4 py-2 font-semibold text-right">{{ t('business.console.liColLicenses') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-surface-border">
            <tr v-for="s in stores" :key="s.id" class="hover:bg-surface-2/40">
              <td class="px-4 py-3 font-bold text-white">{{ s.storeName }}</td>
              <td class="px-4 py-3">
                <UiBadge :variant="s.status === 'APPROVED' ? 'success' : s.status === 'SUSPENDED' ? 'danger' : 'warning'" size="xs">
                  {{ storeStatus(s.status) }}
                </UiBadge>
              </td>
              <td class="px-4 py-3 text-right font-mono text-text-primary">{{ s.licenseCount }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </UiCard>

    <UiCard v-if="!loading" padding="none">
      <template #header>
        <div class="px-5 pt-3">
          <h2 class="text-xs font-bold text-text-secondary uppercase tracking-wider">{{ t('business.console.liCertsTitle') }}</h2>
          <p class="mt-1 text-caption text-text-muted">{{ t('business.console.liCertsDesc') }}</p>
        </div>
      </template>
      <div v-if="certsLoading" class="space-y-2 p-4">
        <UiSkeleton height="h-10" width="w-full" />
      </div>
      <p v-else-if="certs.length === 0" class="px-5 py-6 text-xs text-text-muted">
        {{ t('business.console.liCertsEmpty') }}
      </p>
      <ul v-else class="divide-y divide-surface-border">
        <li v-for="c in certs" :key="c.number" class="flex flex-col gap-1 px-5 py-3 text-xs sm:flex-row sm:items-center sm:justify-between">
          <span class="font-bold text-white">{{ c.title }}</span>
          <RouterLink :to="`/certificate/${c.number}`" class="font-mono text-verified hover:underline">
            {{ c.number }}
          </RouterLink>
        </li>
      </ul>
    </UiCard>

    <UiCard v-if="!loading">
      <p class="text-xs text-text-secondary">{{ t('business.console.liVideoNote') }}</p>
      <RouterLink to="/business" class="mt-2 inline-block text-xs font-semibold text-brand-400 hover:underline">
        {{ t('business.console.liVideoCta') }}
      </RouterLink>
    </UiCard>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { onMounted, ref } from 'vue';
import { consoleApi, trackApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import type { StoreBusiness } from '@/api/types';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';

const { t, te } = useI18n();

function storeStatus(s: string): string {
  const key = `business.console.storeStatus.${s}`;
  return te(key) ? String(t(key)) : s;
}

interface CertifiedRow {
  number: string;
  title: string;
}

const stores = ref<StoreBusiness[]>([]);
const certs = ref<CertifiedRow[]>([]);
const loading = ref(true);
const certsLoading = ref(false);
const error = ref<string | null>(null);

async function load(): Promise<void> {
  loading.value = true;
  error.value = null;
  try {
    stores.value = (await consoleApi.listStores()) ?? [];
  } catch (err: unknown) {
    error.value = errMessageOf(err);
    loading.value = false;
    return;
  }
  loading.value = false;
  certsLoading.value = true;
  try {
    const trackIds = new Set<string>();
    for (const s of stores.value.slice(0, 5)) {
      let devs = [];
      try {
        devs = (await consoleApi.listStoreDevices(s.id)) ?? [];
      } catch {
        continue;
      }
      for (const d of devs.slice(0, 3)) {
        try {
          const logs = (await consoleApi.storeDeviceLogs(s.id, d.id, 20)) ?? [];
          for (const l of logs) {
            if (trackIds.size >= 20) break;
            trackIds.add(l.trackId);
          }
        } catch {
          // Per-device failures must not break the list.
        }
      }
    }
    const rows: CertifiedRow[] = [];
    await Promise.all(
      [...trackIds].map(async (trackId) => {
        try {
          const cert = await trackApi.getCertificate(trackId);
          if (cert) rows.push({ number: cert.certificate.certNumber, title: cert.title });
        } catch {
          // Uncertified tracks are skipped.
        }
      }),
    );
    rows.sort((a, b) => (a.title < b.title ? -1 : 1));
    certs.value = rows;
  } finally {
    certsLoading.value = false;
  }
}

onMounted(() => void load());
</script>
