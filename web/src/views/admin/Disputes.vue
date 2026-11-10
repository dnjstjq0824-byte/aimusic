<template>
  <div class="space-y-6">
    <UiPageHeader
      eyebrow="OSP Governance"
      title="저작권법 제103조 분쟁 관리 &amp; 임시조치"
      description="신고 접수된 음원에 대해 스트리밍 차단(임시조치), 소명 심사, 원상 복구를 처리합니다."
    >
      <template #actions>
        <UiButton variant="secondary" size="sm" :loading="loading" @click="load">새로고침</UiButton>
      </template>
    </UiPageHeader>

    <UiErrorBanner :message="error" title="분쟁 목록을 불러오지 못했습니다" retry-label="다시 시도" @retry="load" />

    <!-- Disputes List -->
    <div v-if="loading" class="space-y-3">
      <UiSkeleton height="h-24" width="w-full" />
      <UiSkeleton height="h-24" width="w-full" />
    </div>

    <div v-else-if="disputes.length === 0" class="py-12">
      <UiEmptyState
        title="접수된 권리침해 분쟁이 없습니다"
        description="현재 진행 중인 저작권 침해 또는 보이스 클로닝 신고가 없습니다."
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
            <span class="font-bold text-white text-sm">신고 ID: {{ d.id }}</span>
            <UiBadge
              :variant="disputeVariant(d.status)"
              size="xs"
            >
              {{ disputeStatusKo(d.status) }}
            </UiBadge>
          </div>
          <div class="flex flex-wrap items-center gap-2 text-[11px] text-neutral-400">
            <span>신고 접수: <strong>{{ d.reporterUserId ? `회원(${d.reporterUserId.slice(0, 8)})` : '외부/익명 접수' }}</strong></span>
            <UiBadge :variant="slaBreached(d) ? 'danger' : 'neutral'" size="xs">
              {{ slaBreached(d) ? `SLA 초과 · ${slaLabel(d)}` : slaLabel(d) }}
            </UiBadge>
            <span v-if="d.slaNotifiedAt" class="text-neutral-500">SLA 알림 발송됨</span>
          </div>
        </div>

        <div class="grid gap-2 text-xs sm:grid-cols-2">
          <div>
            <span class="text-neutral-500 text-[10px] uppercase font-bold block">대상 트랙 ID</span>
            <span class="font-mono text-neutral-200">{{ d.trackId }}</span>
          </div>
          <div>
            <span class="text-neutral-500 text-[10px] uppercase font-bold block">신고 사유 / 침해 유형</span>
            <span class="text-neutral-200 font-semibold">{{ disputeReasonKo(d.reason) }}</span>
          </div>
        </div>

        <div v-if="d.description" class="rounded-xl bg-surface-0 p-3 text-xs border border-surface-border">
          <span class="text-accent-amber font-bold text-[10px] uppercase block">신고 상세 소명 및 진술</span>
          <p class="text-neutral-300 mt-1 whitespace-pre-wrap">{{ d.description }}</p>
        </div>

        <div class="flex flex-wrap items-center justify-end gap-2 pt-2 border-t border-surface-border">
          <UiButton
            variant="danger"
            size="xs"
            @click="askResolve(d.id, true)"
          >
            침해 인정 (임시조치 &amp; 스트라이크)
          </UiButton>
          <UiButton
            variant="success"
            size="xs"
            @click="askResolve(d.id, false)"
          >
            신고 기각 / 소명 인용 (정상 유지)
          </UiButton>
        </div>
      </UiCard>
    </div>

    <UiConfirmDialog
      :model-value="pendingResolve !== null"
      :title="pendingResolve?.uphold ? '침해 인정 및 임시조치' : '신고 기각'"
      :description="pendingResolve?.uphold
        ? '해당 음원을 즉시 임시조치하고 파트너에게 스트라이크를 부과합니다.'
        : '신고를 기각하고 음원을 정상 상태로 유지/복원합니다.'"
      :confirm-label="pendingResolve?.uphold ? '인정하고 조치하기' : '기각하기'"
      :danger="pendingResolve?.uphold ?? false"
      :loading="resolving"
      @update:model-value="onResolveDialogToggle"
      @confirm="doResolve"
    />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { adminApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { useToast } from '@/composables/useToast';
import type { CopyrightDispute } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiConfirmDialog from '@/components/ui/UiConfirmDialog.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';

const toast = useToast();

const disputes = ref<CopyrightDispute[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);
const pendingResolve = ref<{ id: string; uphold: boolean } | null>(null);
const resolving = ref(false);

const DISPUTE_STATUS_KO: Record<string, string> = {
  OPEN: '접수됨',
  UNDER_REVIEW: '심사 중',
  UPHELD_TAKEDOWN: '침해 인정·임시조치',
  DISMISSED: '기각됨',
};

const DISPUTE_REASON_KO: Record<string, string> = {
  AI_POLICY_VIOLATION: '비상업·무료 요금제 생성물 의심',
  VOICE_CLONE: '음성·퍼블리시티권 침해',
  UNAUTHORIZED_SAMPLE: '저작권 침해·무단 샘플링',
  OTHER: '기타',
};

function disputeStatusKo(status: string): string {
  return DISPUTE_STATUS_KO[status] ?? status;
}

function disputeReasonKo(reason: string): string {
  return DISPUTE_REASON_KO[reason] ?? reason;
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
  if (h < 1) return '접수 1시간 이내';
  if (h < 24) return `접수 후 ${h}시간 경과`;
  return `접수 후 ${Math.floor(h / 24)}일 ${h % 24}시간 경과`;
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
    disputes.value = await adminApi.listDisputes();
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
    toast.success(uphold ? '침해 인정: 즉시 임시조치 및 스트라이크가 부과되었습니다.' : '신고 기각: 정상 상태로 유지/복원되었습니다.');
    pendingResolve.value = null;
    await load();
  } catch (err) {
    toast.error(errMessageOf(err), '조치 실패');
  } finally {
    resolving.value = false;
  }
}

onMounted(() => {
  void load();
});
</script>
