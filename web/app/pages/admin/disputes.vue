<template>
  <div class="space-y-6">
    <UiPageHeader
      :eyebrow="t('admin.dp.eyebrow')"
      :title="t('admin.dp.title')"
      :description="t('admin.dp.desc')"
    >
      <template #actions>
        <UiButton variant="secondary" size="sm" :loading="loading" @click="load">{{ t('admin.dp.refresh') }}</UiButton>
      </template>
    </UiPageHeader>

    <UiErrorBanner :message="error" :title="t('admin.dp.loadFail')" :retry-label="t('common.actions.retry')" @retry="load" />

    <div v-if="loading" class="space-y-3">
      <UiSkeleton height="h-24" width="w-full" />
      <UiSkeleton height="h-24" width="w-full" />
    </div>

    <div v-else-if="disputes.length === 0" class="py-12">
      <UiEmptyState
        :title="t('admin.dp.empty')"
        :description="t('admin.dp.emptyDesc')"
      />
    </div>

    <div v-else class="space-y-3">
      <UiCard
        v-for="d in disputes"
        :key="d.id"
        class="space-y-3"
      >
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2 border-b border-surface-border pb-3">
          <div class="flex items-center gap-2">
            <span class="font-bold text-white text-sm">{{ t('admin.dp.reportId', { id: d.id }) }}</span>
            <UiBadge
              :variant="disputeVariant(d.status)"
              size="xs"
            >
              {{ disputeStatus(d.status) }}
            </UiBadge>
          </div>
          <div class="flex flex-wrap items-center gap-2 text-caption text-text-secondary">
            <span>{{ t('admin.dp.reporter', { who: d.reporterUserId ? t('admin.dp.member', { id: d.reporterUserId.slice(0, 8) }) : t('admin.dp.external') }) }}</span>
            <UiBadge :variant="slaBreached(d) ? 'danger' : 'neutral'" size="xs">
              {{ slaBreached(d) ? t('admin.dp.slaOver', { label: slaLabel(d) }) : slaLabel(d) }}
            </UiBadge>
            <span v-if="d.slaNotifiedAt" class="text-text-muted">{{ t('admin.dp.slaNotified') }}</span>
          </div>
        </div>

        <div class="grid gap-2 text-xs sm:grid-cols-2">
          <div>
            <span class="text-text-muted text-caption uppercase font-bold block">{{ t('admin.dp.trackId') }}</span>
            <span class="font-mono text-text-primary">{{ d.trackId }}</span>
          </div>
          <div>
            <span class="text-text-muted text-caption uppercase font-bold block">{{ t('admin.dp.reason') }}</span>
            <span class="text-text-primary font-semibold">{{ disputeReason(d.reason) }}</span>
          </div>
        </div>

        <div v-if="d.description" class="rounded-xl bg-surface-0 p-3 text-xs border border-surface-border">
          <span class="text-accent-amber font-bold text-caption uppercase block">{{ t('admin.dp.detail') }}</span>
          <p class="text-text-primary mt-1 whitespace-pre-wrap">{{ d.description }}</p>
        </div>

        <div class="flex flex-wrap items-center justify-end gap-2 pt-2 border-t border-surface-border">
          <UiButton
            variant="danger"
            size="xs"
            @click="askResolve(d.id, true)"
          >
            {{ t('admin.dp.uphold') }}
          </UiButton>
          <UiButton
            variant="success"
            size="xs"
            @click="askResolve(d.id, false)"
          >
            {{ t('admin.dp.dismiss') }}
          </UiButton>
        </div>
      </UiCard>
    </div>

    <UiConfirmDialog
      :model-value="pendingResolve !== null"
      :title="pendingResolve?.uphold ? t('admin.dp.confirmUpholdTitle') : t('admin.dp.confirmDismissTitle')"
      :description="pendingResolve?.uphold ? t('admin.dp.confirmUpholdDesc') : t('admin.dp.confirmDismissDesc')"
      :confirm-label="pendingResolve?.uphold ? t('admin.dp.confirmUpholdGo') : t('admin.dp.confirmDismissGo')"
      :danger="pendingResolve?.uphold ?? false"
      :loading="resolving"
      @update:model-value="onResolveDialogToggle"
      @confirm="doResolve"
    />
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { onMounted, ref } from 'vue';
import { adminApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { useToast } from '@/composables/useToast';
import { useApiError } from '@/composables/useApiError';
import type { CopyrightDispute } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiConfirmDialog from '@/components/ui/UiConfirmDialog.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';

const { t, te } = useI18n();
const toast = useToast();
const apiError = useApiError();

const disputes = ref<CopyrightDispute[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);
const pendingResolve = ref<{ id: string; uphold: boolean } | null>(null);
const resolving = ref(false);

function disputeStatus(status: string): string {
  const key = `admin.dp.status.${status}`;
  return te(key) ? String(t(key)) : status;
}

function disputeReason(reason: string): string {
  const key = `admin.dp.reasons.${reason}`;
  return te(key) ? String(t(key)) : reason;
}

function disputeVariant(status: string): 'danger' | 'success' | 'warning' | 'neutral' {
  if (status === 'UPHELD_TAKEDOWN') return 'danger';
  if (status === 'DISMISSED') return 'success';
  if (status === 'UNDER_REVIEW' || status === 'OPEN') return 'warning';
  return 'neutral';
}

// SLA: open disputes older than 72h trigger the worker sweeper alert.
const SLA_HOURS = 72;

function slaElapsedHours(d: CopyrightDispute): number {
  const ms = Date.now() - new Date(d.createdAt).getTime();
  return Number.isFinite(ms) ? ms / 3_600_000 : 0;
}

function slaBreached(d: CopyrightDispute): boolean {
  return (d.status === 'OPEN' || d.status === 'UNDER_REVIEW') && slaElapsedHours(d) >= SLA_HOURS;
}

function slaLabel(d: CopyrightDispute): string {
  const h = Math.floor(slaElapsedHours(d));
  if (h < 1) return t('admin.dp.slaFresh');
  if (h < 24) return t('admin.dp.slaHours', { n: h });
  return t('admin.dp.slaDays', { d: Math.floor(h / 24), h: h % 24 });
}

function askResolve(id: string, uphold: boolean): void {
  pendingResolve.value = { id, uphold };
}

function onResolveDialogToggle(open: boolean): void {
  if (!open) pendingResolve.value = null;
}

async function load() {
  loading.value = true;
  error.value = null;
  try {
    disputes.value = (await adminApi.listDisputes()) ?? [];
  } catch (err) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
}

async function doResolve() {
  if (!pendingResolve.value) return;
  const { id, uphold } = pendingResolve.value;
  resolving.value = true;
  try {
    await adminApi.resolveDispute(id, uphold);
    toast.success(t(uphold ? 'admin.dp.upheld' : 'admin.dp.dismissed'));
    pendingResolve.value = null;
    await load();
  } catch (err) {
    toast.error(apiError(err), t('admin.dp.resolveFail'));
  } finally {
    resolving.value = false;
  }
}

onMounted(() => {
  void load();
});
</script>
