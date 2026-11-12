<template>
  <div class="mx-auto max-w-2xl space-y-6 py-6">
    <div v-if="loading" class="space-y-3 text-center">
      <div class="flex justify-center" aria-hidden="true">
        <svg class="h-8 w-8 animate-spin text-brand-500" viewBox="0 0 24 24" fill="none">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z" />
        </svg>
      </div>
      <h1 class="text-xl font-black text-white">결제 승인 중…</h1>
      <p class="text-xs text-neutral-400">창을 닫지 마세요.</p>
    </div>

    <template v-else-if="subscription">
      <UiPageHeader eyebrow="Payment" title="결제가 완료되었습니다" />
      <UiCard border-variant="success" class="space-y-3">
        <div class="flex items-center justify-between">
          <span class="text-sm font-bold text-white">{{ subscription.plan?.nameKo ?? '이용권' }}</span>
          <UiBadge variant="success" size="sm">ACTIVE</UiBadge>
        </div>
        <dl class="space-y-2 text-xs">
          <div class="flex justify-between">
            <dt class="text-neutral-500">유효기간</dt>
            <dd class="text-neutral-200">
              {{ fmtDate(subscription.currentPeriodStart) }} ~ {{ fmtDate(subscription.currentPeriodEnd) }}
            </dd>
          </div>
          <div class="flex justify-between">
            <dt class="text-neutral-500">주문번호</dt>
            <dd class="font-mono text-neutral-300">{{ orderId }}</dd>
          </div>
        </dl>
        <p class="border-t border-surface-border pt-3 text-[11px] leading-relaxed text-neutral-400">
          환불 기준: {{ REFUND_POLICY_SUMMARY }} 해지·환불은 내 라이브러리에서 신청할 수 있습니다.
        </p>
      </UiCard>
      <div class="flex justify-end gap-2">
        <RouterLink to="/my-library">
          <UiButton variant="brand" size="md">내 라이브러리로 이동</UiButton>
        </RouterLink>
      </div>
    </template>

    <template v-else>
      <UiPageHeader eyebrow="Payment" title="결제 승인 실패" />
      <UiErrorBanner :message="error" title="승인 실패" />
      <div class="flex justify-end gap-2">
        <RouterLink to="/pricing">
          <UiButton variant="secondary" size="md">요금제로 돌아가기</UiButton>
        </RouterLink>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useRoute } from 'vue-router';
import { billingApi } from '@/api/endpoints';
import { REFUND_POLICY_SUMMARY } from '@/api/refundPolicy';
import { errMessageOf } from '@/api/client';
import type { BillingSubscription } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';

const route = useRoute();

const loading = ref(true);
const subscription = ref<BillingSubscription | null>(null);
const orderId = ref('');
const error = ref<string | null>(null);

function fmtDate(iso: string): string {
  return new Date(iso).toLocaleDateString('ko-KR');
}

onMounted(async () => {
  const paymentKey = route.query.paymentKey;
  const oid = route.query.orderId;
  const amount = route.query.amount;
  if (typeof paymentKey !== 'string' || typeof oid !== 'string' || typeof amount !== 'string') {
    loading.value = false;
    error.value = 'Toss 결제 응답이 올바르지 않습니다. (paymentKey/orderId/amount 누락)';
    return;
  }
  orderId.value = oid;
  try {
    const res = await billingApi.confirmPayment(paymentKey, oid, Number(amount));
    subscription.value = res.subscription;
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
});
</script>
