<template>
  <div class="space-y-6">
    <UiPageHeader
      :eyebrow="t('admin.iq.eyebrow')"
      :title="t('admin.iq.title')"
      :description="t('admin.iq.desc')"
    >
      <template #actions>
        <UiButton variant="secondary" size="sm" :loading="loading" @click="load">{{ t('admin.iq.refresh') }}</UiButton>
      </template>
    </UiPageHeader>

    <UiErrorBanner :message="error" :title="t('admin.iq.loadFail')" :retry-label="t('common.actions.retry')" @retry="load" />

    <UiTabs v-model="status" :tabs="statusTabs" @update:model-value="load" />

    <div v-if="loading" class="space-y-2">
      <UiSkeleton height="h-14" width="w-full" />
      <UiSkeleton height="h-14" width="w-full" />
    </div>

    <div v-else-if="items.length === 0">
      <UiEmptyState :title="t('admin.iq.empty')" :description="t('admin.iq.emptyDesc')" />
    </div>

    <div v-else class="space-y-3">
      <UiCard v-for="q in items" :key="q.id" class="space-y-2">
        <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <span class="text-sm font-bold text-white">{{ q.company }}</span>
              <UiBadge :variant="q.status === 'CLOSED' ? 'success' : q.status === 'CONTACTED' ? 'info' : 'warning'" size="xs">
                {{ inquiryStatus(q.status) }}
              </UiBadge>
            </div>
            <div class="mt-1 text-caption text-text-secondary">
              {{ q.contactName }} · {{ q.contactEmail }}<span v-if="q.contactPhone"> · {{ q.contactPhone }}</span>
            </div>
            <div class="mt-0.5 font-mono text-caption text-text-muted">
              {{ q.country || '—' }} · {{ t('admin.iq.stores', { n: q.storeCount }) }} · {{ q.useCase || '—' }}
            </div>
          </div>
          <div class="flex shrink-0 gap-2">
            <UiButton
              v-for="s in ['NEW', 'CONTACTED', 'CLOSED']"
              :key="s"
              :variant="q.status === s ? 'brand' : 'ghost'"
              size="xs"
              :disabled="busyId === q.id"
              @click="setStatus(q, s)"
            >
              {{ inquiryStatus(s) }}
            </UiButton>
          </div>
        </div>
        <p class="whitespace-pre-wrap rounded-lg bg-surface-0 p-3 text-xs text-text-primary border border-surface-border">
          {{ q.message }}
        </p>
        <div class="font-mono text-caption text-text-muted">
          {{ fmtDateTime(q.createdAt) }}
        </div>
      </UiCard>
    </div>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { computed, onMounted, ref } from 'vue';
import { adminApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { useApiError } from '@/composables/useApiError';
import type { BusinessInquiry } from '@/api/types';
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
  { value: '', label: t('admin.iq.tabAll') },
  { value: 'NEW', label: t('admin.iq.tabNew') },
  { value: 'CONTACTED', label: t('admin.iq.tabContacted') },
  { value: 'CLOSED', label: t('admin.iq.tabClosed') },
]);

function inquiryStatus(status: string): string {
  const key = `admin.iq.inquiryStatus.${status}`;
  return te(key) ? String(t(key)) : status;
}

function fmtDateTime(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString(locale.value === 'en' ? 'en-US' : 'ko-KR', { hour12: false });
}

const status = ref('');
const items = ref<BusinessInquiry[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);
const busyId = ref<string | null>(null);

async function load(): Promise<void> {
  loading.value = true;
  error.value = null;
  try {
    items.value = (await adminApi.listInquiries(status.value || undefined, 50)) ?? [];
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
}

async function setStatus(q: BusinessInquiry, next: string): Promise<void> {
  if (q.status === next) return;
  busyId.value = q.id;
  try {
    const updated = await adminApi.setInquiryStatus(q.id, next);
    const i = items.value.findIndex((x) => x.id === q.id);
    if (i >= 0) items.value[i] = updated;
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
