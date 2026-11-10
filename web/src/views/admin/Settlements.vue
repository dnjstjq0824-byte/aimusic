<template>
  <div class="space-y-6">
    <UiPageHeader
      eyebrow="Admin Finance"
      title="파트너 정산 관리"
      description="유통 순매출 70:30 정산 계산, 승인, 지급 완료 처리를 수행합니다. 계좌는 마스킹된 미리보기만 노출됩니다."
    >
      <template #actions>
        <UiButton variant="secondary" size="sm" :loading="loading" @click="reload">
          새로고침
        </UiButton>
      </template>
    </UiPageHeader>

    <UiErrorBanner :message="error" title="정산 내역을 불러오지 못했습니다" retry-label="다시 시도" @retry="reload" />

    <UiCard class="space-y-3">
      <div class="flex flex-col gap-3 lg:flex-row lg:items-end">
        <label class="flex-[2] text-xs text-neutral-400">
          <span class="mb-1 block font-semibold">파트너 ID</span>
          <input
            v-model="partnerId"
            type="text"
            placeholder="partner UUID"
            class="w-full rounded-lg border border-surface-border bg-surface-1 px-3 py-2 font-mono text-xs text-white placeholder:text-neutral-600"
            @keyup.enter="reload"
          />
        </label>
        <label class="flex-1 text-xs text-neutral-400">
          <span class="mb-1 block font-semibold">정산 시작 (YYYY-MM-DD)</span>
          <input
            v-model="periodStart"
            type="text"
            placeholder="2026-11-01"
            class="w-full rounded-lg border border-surface-border bg-surface-1 px-3 py-2 font-mono text-xs text-white placeholder:text-neutral-600"
          />
        </label>
        <label class="flex-1 text-xs text-neutral-400">
          <span class="mb-1 block font-semibold">정산 종료 (YYYY-MM-DD)</span>
          <input
            v-model="periodEnd"
            type="text"
            placeholder="2026-11-30"
            class="w-full rounded-lg border border-surface-border bg-surface-1 px-3 py-2 font-mono text-xs text-white placeholder:text-neutral-600"
          />
        </label>
      </div>
      <div class="flex flex-col gap-3 lg:flex-row lg:items-end">
        <label class="flex-1 text-xs text-neutral-400">
          <span class="mb-1 block font-semibold">매장 구독료 풀 KRW (기간 합산)</span>
          <input
            v-model="storePool"
            type="number"
            min="0"
            placeholder="0"
            class="w-full rounded-lg border border-surface-border bg-surface-1 px-3 py-2 text-sm text-white placeholder:text-neutral-600"
          />
        </label>
        <label class="flex-1 text-xs text-neutral-400">
          <span class="mb-1 block font-semibold">직접 귀속 매출 KRW (이 파트너)</span>
          <input
            v-model="directRevenue"
            type="number"
            min="0"
            placeholder="0"
            class="w-full rounded-lg border border-surface-border bg-surface-1 px-3 py-2 text-sm text-white placeholder:text-neutral-600"
          />
        </label>
        <label class="flex-1 text-xs text-neutral-400">
          <span class="mb-1 block font-semibold">소득 구분 (선택)</span>
          <input
            v-model="incomeType"
            type="text"
            placeholder="예: BUSINESS_33"
            class="w-full rounded-lg border border-surface-border bg-surface-1 px-3 py-2 font-mono text-xs text-white placeholder:text-neutral-600"
          />
        </label>
        <div class="flex gap-2">
          <UiButton variant="secondary" size="sm" :loading="loading" @click="reload">조회</UiButton>
          <UiButton variant="brand" size="sm" :loading="calcBusy" @click="calculate">정산 계산</UiButton>
        </div>
      </div>
      <div v-if="account" class="rounded-lg border border-surface-border bg-surface-0 px-3 py-2 font-mono text-xs text-neutral-300">
        정산 계좌: {{ account.bank }} {{ account.accountMasked }}
      </div>
    </UiCard>

    <div v-if="loading && settlements.length === 0" class="space-y-3">
      <UiSkeleton height="h-16" width="w-full" />
      <UiSkeleton height="h-16" width="w-full" />
    </div>

    <div v-else-if="settlements.length === 0" class="py-12">
      <UiEmptyState title="정산 내역이 없습니다" description="파트너 ID를 입력하고 조회하거나 정산 계산을 실행하세요." />
    </div>

    <div v-else class="space-y-3">
      <UiCard v-for="s in settlements" :key="s.id" class="space-y-2">
        <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <span class="font-mono text-xs font-bold text-white">{{ ymd(s.periodStart) }} ~ {{ ymd(s.periodEnd) }}</span>
              <UiBadge :variant="statusVariant(s.status)" size="xs">{{ statusKo(s.status) }}</UiBadge>
            </div>
            <div class="mt-1 font-mono text-[11px] text-neutral-400">
              매장 {{ s.playCount.toLocaleString() }}회·{{ s.storePlaySecs.toLocaleString() }}초(전체 {{ s.totalStorePlaySecs.toLocaleString() }}초)
              · 풀 {{ s.storePoolKrw.toLocaleString() }}원·직접 {{ s.directKrw.toLocaleString() }}원
              · 총액(70%) {{ s.grossAmountKrw.toLocaleString() }}원
              · 원천징수 {{ s.withholdingKrw.toLocaleString() }}원
              · 실지급 <strong class="text-accent-emerald">{{ s.netAmountKrw.toLocaleString() }}원</strong>
            </div>
            <div v-if="s.forfeitureReason" class="mt-1 text-[11px] font-semibold text-accent-crimson">
              몰수 사유: {{ s.forfeitureReason }}
            </div>
            <div v-if="s.status === 'HELD'" class="mt-1 text-[11px] font-semibold text-accent-amber">
              지급 보류 중: {{ s.holdReason }}
            </div>
          </div>
          <div class="flex shrink-0 flex-wrap items-center gap-2">
            <input
              v-model="reference[s.id]"
              type="text"
              placeholder="지급 참조번호 (선택)"
              class="w-44 rounded-lg border border-surface-border bg-surface-1 px-2 py-1 font-mono text-[11px] text-white placeholder:text-neutral-600"
            />
            <UiButton
              variant="secondary"
              size="xs"
              :disabled="s.status !== 'PENDING'"
              :loading="busyId === `approve-${s.id}`"
              @click="approve(s)"
            >
              승인
            </UiButton>
            <UiButton
              variant="brand"
              size="xs"
              :disabled="s.status !== 'APPROVED'"
              :loading="busyId === `paid-${s.id}`"
              @click="markPaid(s)"
            >
              지급 완료
            </UiButton>
            <input
              v-if="s.status === 'PENDING'"
              v-model="holdReason[s.id]"
              type="text"
              placeholder="보류 사유 (필수)"
              class="w-44 rounded-lg border border-surface-border bg-surface-1 px-2 py-1 text-[11px] text-white placeholder:text-neutral-600"
            />
            <UiButton
              v-if="s.status === 'PENDING'"
              variant="secondary"
              size="xs"
              :loading="busyId === `hold-${s.id}`"
              @click="hold(s)"
            >
              보류
            </UiButton>
            <UiButton
              v-if="s.status === 'HELD'"
              variant="secondary"
              size="xs"
              :loading="busyId === `release-${s.id}`"
              @click="release(s)"
            >
              보류 해제
            </UiButton>
          </div>
        </div>
      </UiCard>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { adminApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { useToast } from '@/composables/useToast';
import type { PartnerSettlement } from '@/api/types';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';

const toast = useToast();

const STATUS_KO: Record<string, string> = {
  PENDING: '계산됨',
  APPROVED: '승인됨',
  PAID: '지급 완료',
  FORFEITED: '몰수됨',
  HELD: '지급 보류',
};

function statusKo(status: string): string {
  return STATUS_KO[status] ?? status;
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
    toast.error('파트너 ID를 입력하세요.', '입력 필요');
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
    toast.error('정산 시작/종료일(YYYY-MM-DD)을 입력하세요.', '입력 필요');
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
    toast.success('정산이 계산되었습니다.');
    await reload();
  } catch (err: unknown) {
    toast.error(errMessageOf(err), '정산 계산 실패');
  } finally {
    calcBusy.value = false;
  }
}

async function approve(s: PartnerSettlement): Promise<void> {
  busyId.value = `approve-${s.id}`;
  try {
    await adminApi.approveSettlement(s.partnerId, ymd(s.periodStart), ymd(s.periodEnd));
    toast.success('정산이 승인되었습니다.');
    await reload();
  } catch (err: unknown) {
    toast.error(errMessageOf(err), '승인 실패');
  } finally {
    busyId.value = null;
  }
}

async function markPaid(s: PartnerSettlement): Promise<void> {
  busyId.value = `paid-${s.id}`;
  try {
    await adminApi.markSettlementPaid(s.partnerId, ymd(s.periodStart), ymd(s.periodEnd), reference.value[s.id] || undefined);
    toast.success('지급 완료 처리되었습니다.');
    await reload();
  } catch (err: unknown) {
    toast.error(errMessageOf(err), '지급 처리 실패');
  } finally {
    busyId.value = null;
  }
}

async function hold(s: PartnerSettlement): Promise<void> {
  const reason = (holdReason.value[s.id] ?? '').trim();
  if (!reason) {
    toast.error('보류 사유를 입력하세요.', '입력 필요');
    return;
  }
  busyId.value = `hold-${s.id}`;
  try {
    await adminApi.holdSettlement(s.partnerId, ymd(s.periodStart), ymd(s.periodEnd), reason);
    toast.success('정산이 보류되었습니다.');
    holdReason.value[s.id] = '';
    await reload();
  } catch (err: unknown) {
    toast.error(errMessageOf(err), '보류 실패');
  } finally {
    busyId.value = null;
  }
}

async function release(s: PartnerSettlement): Promise<void> {
  busyId.value = `release-${s.id}`;
  try {
    await adminApi.releaseSettlement(s.partnerId, ymd(s.periodStart), ymd(s.periodEnd));
    toast.success('보류가 해제되었습니다.');
    await reload();
  } catch (err: unknown) {
    toast.error(errMessageOf(err), '해제 실패');
  } finally {
    busyId.value = null;
  }
}
</script>
