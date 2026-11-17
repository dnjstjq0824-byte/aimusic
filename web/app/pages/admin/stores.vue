<template>
  <div class="space-y-6">
    <UiPageHeader
      :eyebrow="t('admin.st.eyebrow')"
      :title="t('admin.st.title')"
      :description="t('admin.st.desc')"
    />

    <UiErrorBanner :message="error" :title="t('admin.st.loadFail')" :retry-label="t('common.actions.retry')" @retry="load" />

    <UiTabs v-model="status" :tabs="statusTabs" @update:model-value="load" />

    <div v-if="loading" class="space-y-2">
      <UiSkeleton height="h-14" width="w-full" />
      <UiSkeleton height="h-14" width="w-full" />
    </div>

    <div v-else-if="stores.length === 0">
      <UiEmptyState :title="t('admin.st.empty')" :description="t('admin.st.emptyDesc')" />
    </div>

    <UiCard v-else padding="none">
      <ul class="divide-y divide-surface-border">
        <li v-for="s in stores" :key="s.id" class="flex flex-col gap-2 px-5 py-4 text-xs sm:flex-row sm:items-center sm:justify-between">
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <span class="text-sm font-bold text-white">{{ s.storeName }}</span>
              <UiBadge :variant="s.status === 'APPROVED' ? 'success' : s.status === 'SUSPENDED' ? 'danger' : 'warning'" size="xs">
                {{ storeStatus(s.status) }}
              </UiBadge>
            </div>
            <p class="mt-1 font-mono text-caption text-text-muted">
              {{ t('admin.st.meta', { bizNo: s.businessNumber, n: s.licenseCount, at: fmtDate(s.createdAt) }) }}
            </p>
          </div>
          <div class="flex shrink-0 gap-2">
            <UiButton v-if="s.status !== 'APPROVED'" variant="brand" size="sm" :disabled="busyId === s.id" @click="verify(s, true)">
              {{ t('admin.st.approve') }}
            </UiButton>
            <UiButton v-if="s.status !== 'SUSPENDED'" variant="ghost" size="sm" :disabled="busyId === s.id" @click="verify(s, false)">
              {{ t('admin.st.suspend') }}
            </UiButton>
          </div>
        </li>
      </ul>
    </UiCard>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { computed, onMounted, ref } from 'vue';
import { adminApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { useApiError } from '@/composables/useApiError';
import type { StoreBusiness } from '@/api/types';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import UiTabs, { type TabOption } from '@/components/ui/UiTabs.vue';

const { t, te, locale } = useI18n();
const apiError = useApiError();

const statusTabs = computed<TabOption[]>(() => [
  { value: 'PENDING', label: t('admin.st.tabPending') },
  { value: 'APPROVED', label: t('admin.st.tabApproved') },
  { value: 'SUSPENDED', label: t('admin.st.tabSuspended') },
]);

function storeStatus(status: string): string {
  const key = `admin.st.storeStatus.${status}`;
  return te(key) ? String(t(key)) : status;
}

function fmtDate(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleDateString(locale.value === 'en' ? 'en-US' : 'ko-KR');
}

const status = ref('PENDING');
const stores = ref<StoreBusiness[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);
const busyId = ref<string | null>(null);

async function load(): Promise<void> {
  loading.value = true;
  error.value = null;
  try {
    stores.value = (await adminApi.listStores(status.value, 50, 0)) ?? [];
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
}

async function verify(s: StoreBusiness, approve: boolean): Promise<void> {
  busyId.value = s.id;
  try {
    await adminApi.verifyStore(s.id, approve);
    await load();
  } catch (err: unknown) {
    error.value = apiError(err);
  } finally {
    busyId.value = null;
  }
}

onMounted(() => {
  void load();
});
</script>
