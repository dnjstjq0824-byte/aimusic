<template>
  <div class="space-y-6">
    <UiPageHeader
      :eyebrow="t('admin.se.eyebrow')"
      :title="t('admin.se.title')"
      :description="t('admin.se.desc')"
    >
      <template #actions>
        <UiButton variant="secondary" size="sm" :loading="loading" @click="reload">
          {{ t('admin.se.refresh') }}
        </UiButton>
      </template>
    </UiPageHeader>

    <UiErrorBanner :message="error" :title="t('admin.se.loadFail')" :retry-label="t('common.actions.retry')" @retry="reload" />

    <UiCard class="space-y-3">
      <div class="flex flex-col gap-3 lg:flex-row lg:items-end">
        <label class="flex-[2] text-xs text-text-secondary">
          <span class="mb-1 block font-semibold">{{ t('admin.se.fPartner') }}</span>
          <input
            v-model="partnerId"
            type="text"
            placeholder="partner UUID"
            class="w-full rounded-lg border border-surface-border bg-surface-1 px-3 py-2 font-mono text-xs text-white placeholder:text-text-muted"
            @keyup.enter="reload"
          />
        </label>
        <label class="flex-1 text-xs text-text-secondary">
          <span class="mb-1 block font-semibold">{{ t('admin.se.fStart') }}</span>
          <input
            v-model="periodStart"
            type="text"
            placeholder="2026-11-01"
            class="w-full rounded-lg border border-surface-border bg-surface-1 px-3 py-2 font-mono text-xs text-white placeholder:text-text-muted"
          />
        </label>
        <label class="flex-1 text-xs text-text-secondary">
          <span class="mb-1 block font-semibold">{{ t('admin.se.fEnd') }}</span>
          <input
            v-model="periodEnd"
            type="text"
            placeholder="2026-11-30"
            class="w-full rounded-lg border border-surface-border bg-surface-1 px-3 py-2 font-mono text-xs text-white placeholder:text-text-muted"
          />
        </label>
      </div>
      <div class="flex flex-col gap-3 lg:flex-row lg:items-end">
        <label class="flex-1 text-xs text-text-secondary">
          <span class="mb-1 block font-semibold">{{ t('admin.se.fPool') }}</span>
          <input
            v-model="storePool"
            type="number"
            min="0"
            placeholder="0"
            class="w-full rounded-lg border border-surface-border bg-surface-1 px-3 py-2 text-sm text-white placeholder:text-text-muted"
          />
        </label>
        <label class="flex-1 text-xs text-text-secondary">
          <span class="mb-1 block font-semibold">{{ t('admin.se.fDirect') }}</span>
          <input
            v-model="directRevenue"
            type="number"
            min="0"
            placeholder="0"
            class="w-full rounded-lg border border-surface-border bg-surface-1 px-3 py-2 text-sm text-white placeholder:text-text-muted"
          />
        </label>
        <label class="flex-1 text-xs text-text-secondary">
          <span class="mb-1 block font-semibold">{{ t('admin.se.fIncome') }}</span>
          <input
            v-model="incomeType"
            type="text"
            :placeholder="t('admin.se.fIncomePh')"
            class="w-full rounded-lg border border-surface-border bg-surface-1 px-3 py-2 font-mono text-xs text-white placeholder:text-text-muted"
          />
        </label>
        <div class="flex gap-2">
          <UiButton variant="secondary" size="sm" :loading="loading" @click="reload">{{ t('admin.se.query') }}</UiButton>
          <UiButton variant="brand" size="sm" :loading="calcBusy" @click="calculate">{{ t('admin.se.calc') }}</UiButton>
        </div>
      </div>
      <div v-if="account" class="rounded-lg border border-surface-border bg-surface-0 px-3 py-2 font-mono text-xs text-text-primary">
        {{ t('admin.se.account', { bank: account.bank, masked: account.accountMasked }) }}
      </div>
    </UiCard>

    <div v-if="loading && settlements.length === 0" class="space-y-3">
      <UiSkeleton height="h-16" width="w-full" />
      <UiSkeleton height="h-16" width="w-full" />
    </div>

    <div v-else-if="settlements.length === 0" class="py-12">
      <UiEmptyState :title="t('admin.se.empty')" :description="t('admin.se.emptyDesc')" />
    </div>

    <div v-else class="space-y-3">
      <UiCard v-for="s in settlements" :key="s.id" class="space-y-2">
        <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <span class="font-mono text-xs font-bold text-white">{{ ymd(s.periodStart) }} ~ {{ ymd(s.periodEnd) }}</span>
              <UiBadge :variant="statusVariant(s.status)" size="xs">{{ settleStatus(s.status) }}</UiBadge>
            </div>
            <div class="mt-1 font-mono text-caption text-text-secondary">
              {{ t('admin.se.row', { plays: s.playCount.toLocaleString(), secs: s.storePlaySecs.toLocaleString(), total: s.totalStorePlaySecs.toLocaleString(), pool: s.storePoolKrw.toLocaleString(), direct: s.directKrw.toLocaleString(), gross: s.grossAmountKrw.toLocaleString(), tax: s.withholdingKrw.toLocaleString(), net: s.netAmountKrw.toLocaleString() }) }}
            </div>
            <div v-if="s.forfeitureReason" class="mt-1 text-caption font-semibold text-accent-crimson">
              {{ t('admin.se.forfeit', { reason: s.forfeitureReason }) }}
            </div>
            <div v-if="s.status === 'HELD'" class="mt-1 text-caption font-semibold text-accent-amber">
              {{ t('admin.se.held', { reason: s.holdReason }) }}
            </div>
          </div>
          <div class="flex shrink-0 flex-wrap items-center gap-2">
            <input
              v-model="reference[s.id]"
              type="text"
              :placeholder="t('admin.se.refPh')"
              class="w-44 rounded-lg border border-surface-border bg-surface-1 px-2 py-1 font-mono text-caption text-white placeholder:text-text-muted"
            />
            <UiButton
              variant="secondary"
              size="xs"
              :disabled="s.status !== 'PENDING'"
              :loading="busyId === `approve-${s.id}`"
              @click="approve(s)"
            >
              {{ t('admin.se.approve') }}
            </UiButton>
            <UiButton
              variant="brand"
              size="xs"
              :disabled="s.status !== 'APPROVED'"
              :loading="busyId === `paid-${s.id}`"
              @click="markPaid(s)"
            >
              {{ t('admin.se.markPaid') }}
            </UiButton>
            <input
              v-if="s.status === 'PENDING'"
              v-model="holdReason[s.id]"
              type="text"
              :placeholder="t('admin.se.holdReasonPh')"
              class="w-44 rounded-lg border border-surface-border bg-surface-1 px-2 py-1 text-caption text-white placeholder:text-text-muted"
            />
            <UiButton
              v-if="s.status === 'PENDING'"
              variant="secondary"
              size="xs"
              :loading="busyId === `hold-${s.id}`"
              @click="hold(s)"
            >
              {{ t('admin.se.hold') }}
            </UiButton>
            <UiButton
              v-if="s.status === 'HELD'"
              variant="secondary"
              size="xs"
              :loading="busyId === `release-${s.id}`"
              @click="release(s)"
            >
              {{ t('admin.se.release') }}
            </UiButton>
          </div>
        </div>
      </UiCard>
    </div>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { ref } from 'vue';
import { adminApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { useToast } from '@/composables/useToast';
import { useApiError } from '@/composables/useApiError';
import type { PartnerSettlement } from '@/api/types';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';

const { t, te } = useI18n();
const toast = useToast();
const apiError = useApiError();

function settleStatus(status: string): string {
  const key = `admin.se.settleStatus.${status}`;
  return te(key) ? String(t(key)) : status;
}

// Backend returns RFC3339 timestamps but approve/mark-paid parse YYYY-MM-DD.
function ymd(iso: string): string {
  return iso.slice(0, 10);
}

function statusVariant(status: string): 'success' | 'warning' | 'danger' | 'info' {
  if (status === 'PAID') return 'success';
  if (status === 'FORFEITED') return 'danger';
  if (status === 'APPROVED') return 'info';
  return 'warning';
}

const settlements = ref<PartnerSettlement[]>([]);
const account = ref<{ bank: string; accountMasked: string } | null>(null);
const loading = ref(false);
const calcBusy = ref(false);
const error = ref<string | null>(null);
const busyId = ref<string | null>(null);

const partnerId = ref('');
const periodStart = ref('');
const periodEnd = ref('');
const storePool = ref('');
const directRevenue = ref('');
const incomeType = ref('');
const reference = ref<Record<string, string>>({});
const holdReason = ref<Record<string, string>>({});

function requirePartner(): boolean {
  if (!partnerId.value.trim()) {
    toast.error(t('admin.se.needPartner'), t('admin.se.inputNeeded'));
    return false;
  }
  return true;
}

async function reload(): Promise<void> {
  if (!requirePartner()) return;
  loading.value = true;
  error.value = null;
  account.value = null;
  try {
    const [rows, acct] = await Promise.all([
      adminApi.listSettlements(partnerId.value.trim()),
      adminApi.getPartnerSettlementAccount(partnerId.value.trim()).catch(() => null),
    ]);
    settlements.value = rows ?? [];
    account.value = acct;
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
}

async function calculate(): Promise<void> {
  if (!requirePartner()) return;
  if (!periodStart.value.trim() || !periodEnd.value.trim()) {
    toast.error(t('admin.se.needPeriod'), t('admin.se.inputNeeded'));
    return;
  }
  calcBusy.value = true;
  try {
    const pool = storePool.value === '' ? undefined : Number(storePool.value);
    const direct = directRevenue.value === '' ? undefined : Number(directRevenue.value);
    await adminApi.calculateSettlement(
      partnerId.value.trim(),
      periodStart.value.trim(),
      periodEnd.value.trim(),
      pool,
      direct,
      incomeType.value.trim() || undefined,
    );
    toast.success(t('admin.se.calculated'));
    await reload();
  } catch (err: unknown) {
    toast.error(apiError(err), t('admin.se.calcFail'));
  } finally {
    calcBusy.value = false;
  }
}

async function approve(s: PartnerSettlement): Promise<void> {
  busyId.value = `approve-${s.id}`;
  try {
    await adminApi.approveSettlement(s.partnerId, ymd(s.periodStart), ymd(s.periodEnd));
    toast.success(t('admin.se.approved'));
    await reload();
  } catch (err: unknown) {
    toast.error(apiError(err), t('admin.se.approveFail'));
  } finally {
    busyId.value = null;
  }
}

async function markPaid(s: PartnerSettlement): Promise<void> {
  busyId.value = `paid-${s.id}`;
  try {
    await adminApi.markSettlementPaid(s.partnerId, ymd(s.periodStart), ymd(s.periodEnd), reference.value[s.id] || undefined);
    toast.success(t('admin.se.paid'));
    await reload();
  } catch (err: unknown) {
    toast.error(apiError(err), t('admin.se.paidFail'));
  } finally {
    busyId.value = null;
  }
}

async function hold(s: PartnerSettlement): Promise<void> {
  const reason = (holdReason.value[s.id] ?? '').trim();
  if (!reason) {
    toast.error(t('admin.se.needHoldReason'), t('admin.se.inputNeeded'));
    return;
  }
  busyId.value = `hold-${s.id}`;
  try {
    await adminApi.holdSettlement(s.partnerId, ymd(s.periodStart), ymd(s.periodEnd), reason);
    toast.success(t('admin.se.heldToast'));
    holdReason.value[s.id] = '';
    await reload();
  } catch (err: unknown) {
    toast.error(apiError(err), t('admin.se.holdFail'));
  } finally {
    busyId.value = null;
  }
}

async function release(s: PartnerSettlement): Promise<void> {
  busyId.value = `release-${s.id}`;
  try {
    await adminApi.releaseSettlement(s.partnerId, ymd(s.periodStart), ymd(s.periodEnd));
    toast.success(t('admin.se.released'));
    await reload();
  } catch (err: unknown) {
    toast.error(apiError(err), t('admin.se.releaseFail'));
  } finally {
    busyId.value = null;
  }
}
</script>
