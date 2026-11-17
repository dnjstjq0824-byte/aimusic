<template>
  <div class="space-y-6">
    <UiPageHeader
      :eyebrow="t('business.console.plEyebrow')"
      :title="t('business.console.plTitle')"
      :description="t('business.console.plDesc')"
    >
      <template #actions>
        <UiButton variant="brand" size="sm" :disabled="stores.length === 0" @click="issueOpen = true">
          {{ t('business.console.plIssue') }}
        </UiButton>
      </template>
    </UiPageHeader>

    <UiErrorBanner :message="error" :title="t('business.console.plLoadFail')" :retry-label="t('common.actions.retry')" @retry="load" />

    <div v-if="loading" class="space-y-2">
      <UiSkeleton height="h-16" width="w-full" />
      <UiSkeleton height="h-16" width="w-full" />
    </div>

    <div v-else-if="rows.length === 0">
      <UiEmptyState
        :title="t('business.console.plEmpty')"
        :description="t('business.console.plEmptyDesc')"
      >
        <template #action>
          <UiButton variant="brand" size="sm" :disabled="stores.length === 0" @click="issueOpen = true">
            {{ t('business.console.plIssue') }}
          </UiButton>
        </template>
      </UiEmptyState>
    </div>

    <UiCard v-else padding="none">
      <div class="overflow-x-auto">
        <table class="w-full min-w-[680px] text-xs">
          <thead>
            <tr class="bg-surface-2/60 text-left text-text-secondary">
              <th class="px-4 py-2 font-semibold">{{ t('business.console.plColDevice') }}</th>
              <th class="px-4 py-2 font-semibold">{{ t('business.console.plColStore') }}</th>
              <th class="px-4 py-2 font-semibold">{{ t('business.console.plColStatus') }}</th>
              <th class="px-4 py-2 font-semibold">{{ t('business.console.plColLastSeen') }}</th>
              <th class="px-4 py-2 font-semibold">{{ t('business.console.plColNow') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-surface-border">
            <tr v-for="r in rows" :key="r.device.id" class="hover:bg-surface-2/40">
              <td class="px-4 py-3 font-bold text-white">{{ r.device.deviceName }}</td>
              <td class="px-4 py-3 text-text-primary">{{ r.storeName }}</td>
              <td class="px-4 py-3">
                <UiBadge :variant="r.online ? 'success' : 'warning'" size="xs">
                  {{ r.online ? t('business.console.online') : t('business.console.offline') }}
                </UiBadge>
              </td>
              <td class="px-4 py-3 text-text-secondary">{{ r.lastSeenText }}</td>
              <td class="px-4 py-3 text-text-primary">{{ r.nowPlaying ?? '—' }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </UiCard>

    <UiModal v-model="issueOpen" :title="t('business.console.issueTitle')" :description="t('business.console.issueDesc')" @close="issueOpen = false">
      <form class="space-y-4" @submit.prevent="issue">
        <UiSelect v-model="issueStoreId" :label="t('business.console.issueStore')" required>
          <option value="">{{ t('business.console.issueStorePh') }}</option>
          <option v-for="s in stores" :key="s.id" :value="s.id">{{ s.storeName }}</option>
        </UiSelect>
        <UiInput v-model="issueName" :label="t('business.console.issueName')" :placeholder="t('business.console.issueNamePh')" required />
        <div v-if="issuedCode" class="rounded-lg bg-surface-2 px-4 py-3 text-center">
          <p class="text-caption text-text-secondary">{{ t('business.console.devCode') }}</p>
          <p class="font-mono text-2xl font-bold tracking-[0.3em] text-brand-300">{{ issuedCode }}</p>
          <p class="mt-1 text-caption text-text-muted">{{ t('business.console.issueHint') }}</p>
        </div>
        <UiErrorBanner :message="formError" :title="t('business.console.issueFail')" />
        <div class="flex justify-end gap-2">
          <UiButton variant="ghost" size="sm" type="button" @click="issueOpen = false">{{ t('common.actions.close') }}</UiButton>
          <UiButton v-if="!issuedCode" variant="brand" size="sm" type="submit" :disabled="formBusy">{{ t('business.console.devIssue') }}</UiButton>
        </div>
      </form>
    </UiModal>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { onMounted, ref } from 'vue';
import { consoleApi, trackApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { useApiError } from '@/composables/useApiError';
import type { StoreBusiness, StoreDevice } from '@/api/types';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiInput from '@/components/ui/UiInput.vue';
import UiModal from '@/components/ui/UiModal.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSelect from '@/components/ui/UiSelect.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';

const { t, locale } = useI18n();
const apiError = useApiError();

const ONLINE_AFTER_MS = 15 * 60_000;

interface PlayerRow {
  device: StoreDevice;
  storeName: string;
  online: boolean;
  lastSeenText: string;
  nowPlaying: string | null;
}

const stores = ref<StoreBusiness[]>([]);
const rows = ref<PlayerRow[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);

const issueOpen = ref(false);
const formBusy = ref(false);
const formError = ref<string | null>(null);
const issueStoreId = ref('');
const issueName = ref('');
const issuedCode = ref<string | null>(null);

function fmtDateTime(iso: string): string {
  try {
    return new Date(iso).toLocaleString(locale.value === 'zh-cn' ? 'zh-CN' : locale.value);
  } catch {
    return iso;
  }
}

async function load(): Promise<void> {
  loading.value = true;
  error.value = null;
  try {
    const list = (await consoleApi.listStores()) ?? [];
    stores.value = list;
    const out: PlayerRow[] = [];
    for (const s of list) {
      let devs: StoreDevice[] = [];
      try {
        devs = (await consoleApi.listStoreDevices(s.id)) ?? [];
      } catch {
        continue;
      }
      for (const d of devs.slice(0, 20)) {
        const seen = d.lastSeenAt ? new Date(d.lastSeenAt).getTime() : 0;
        const online = d.status === 'ACTIVE' && seen > 0 && Date.now() - seen < ONLINE_AFTER_MS;
        let nowPlaying: string | null = null;
        try {
          const logs = (await consoleApi.storeDeviceLogs(s.id, d.id, 1)) ?? [];
          const trackId = logs[0]?.trackId;
          if (trackId) {
            try {
              nowPlaying = (await trackApi.get(trackId))?.title ?? trackId;
            } catch {
              nowPlaying = trackId;
            }
          }
        } catch {
          // Now-playing is best-effort.
        }
        out.push({
          device: d,
          storeName: s.storeName,
          online,
          lastSeenText: d.lastSeenAt ? fmtDateTime(d.lastSeenAt) : t('business.console.neverSeen'),
          nowPlaying,
        });
      }
    }
    rows.value = out;
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
}

async function issue(): Promise<void> {
  formBusy.value = true;
  formError.value = null;
  try {
    const created = await consoleApi.createStoreDevice(issueStoreId.value, issueName.value.trim());
    issuedCode.value = created.pairingCode ?? null;
    await load();
  } catch (err: unknown) {
    formError.value = apiError(err);
  } finally {
    formBusy.value = false;
  }
}

onMounted(() => void load());
</script>
