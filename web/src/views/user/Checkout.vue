<template>
  <div class="mx-auto max-w-2xl space-y-6">
    <UiPageHeader
      eyebrow="Checkout"
      title="이용권 결제"
      description="선택한 요금제의 주문서를 확인하고 Toss Payments로 안전하게 결제하세요."
    />

    <UiErrorBanner :message="error" title="주문 처리 실패" retry-label="다시 시도" @retry="init" />

    <div v-if="loading" class="space-y-3">
      <UiSkeleton height="h-40" width="w-full" rounded="xl" />
    </div>

    <template v-else-if="plan && order">
      <UiCard class="space-y-4">
        <div class="flex items-center justify-between border-b border-surface-border pb-3">
          <h2 class="text-sm font-bold text-white">주문서</h2>
          <UiBadge variant="warning" size="sm">결제 대기</UiBadge>
        </div>
        <dl class="space-y-2.5 text-xs">
          <div class="flex justify-between">
            <dt class="text-neutral-500">상품</dt>
            <dd class="font-bold text-white">{{ plan.nameKo }}</dd>
          </div>
          <div class="flex justify-between">
            <dt class="text-neutral-500">주문번호</dt>
            <dd class="font-mono text-neutral-300">{{ order.orderId }}</dd>
          </div>
          <div class="flex justify-between">
            <dt class="text-neutral-500">유효기간</dt>
            <dd class="text-neutral-300">결제일로부터 1개월 (자동연장 없음)</dd>
          </div>
          <div class="flex justify-between border-t border-surface-border pt-2.5">
            <dt class="font-bold text-white">결제금액</dt>
            <dd class="text-lg font-black text-brand-400">₩{{ order.amountKrw.toLocaleString('ko-KR') }}</dd>
          </div>
        </dl>
      </UiCard>

      <UiCard v-if="!tossReady">
        <div class="flex items-start gap-3">
          <UiBadge variant="neutral" size="sm">준비 중</UiBadge>
          <p class="text-xs leading-relaxed text-neutral-400">
            결제 클라이언트 키가 설정되지 않아 결제창을 열 수 없습니다.
            관리자에게 <span class="font-mono">VITE_TOSS_CLIENT_KEY</span> 설정을 요청하세요.
          </p>
        </div>
      </UiCard>

      <div class="flex flex-col gap-2 sm:flex-row sm:justify-end">
        <RouterLink to="/pricing">
          <UiButton variant="ghost" size="md">요금제로 돌아가기</UiButton>
        </RouterLink>
        <UiButton variant="brand" size="md" :loading="paying" :disabled="!tossReady" @click="pay">
          Toss로 {{ order.amountKrw.toLocaleString('ko-KR') }}원 결제하기
        </UiButton>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useRoute } from 'vue-router';
import { loadTossPayments } from '@tosspayments/tosspayments-sdk';
import { billingApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { useAuthStore } from '@/stores/authStore';
import { useToast } from '@/composables/useToast';
import type { BillingOrder, BillingPlan } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';

const route = useRoute();
const auth = useAuthStore();
const toast = useToast();

const plan = ref<BillingPlan | null>(null);
const order = ref<BillingOrder | null>(null);
const loading = ref(true);
const paying = ref(false);
const error = ref<string | null>(null);

const tossClientKey = (import.meta.env.VITE_TOSS_CLIENT_KEY as string | undefined) ?? '';
const tossReady = tossClientKey !== '';

async function init(): Promise<void> {
  const code = route.query.plan;
  if (typeof code !== 'string' || code === '') {
    error.value = '요금제 코드가 없습니다. 요금제에서 다시 선택해 주세요.';
    loading.value = false;
    return;
  }
  loading.value = true;
  error.value = null;
  try {
    const res = await billingApi.createOrder(code);
    plan.value = res.plan;
    order.value = res.order;
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
}

async function pay(): Promise<void> {
  if (!plan.value || !order.value || !tossReady) return;
  paying.value = true;
  try {
    const toss = await loadTossPayments(tossClientKey);
    const payment = toss.payment({ customerKey: auth.userId ?? 'guest' });
    const origin = window.location.origin;
    await payment.requestPayment({
      method: 'CARD',
      amount: { currency: 'KRW', value: order.value.amountKrw },
      orderId: order.value.orderId,
      orderName: plan.value.nameKo,
      successUrl: `${origin}/checkout/confirm`,
      failUrl: `${origin}/checkout?plan=${plan.value.code}&fail=1`,
    });
  } catch (err: unknown) {
    // User-cancelled windows surface here too; keep the order for retry.
    toast.error(errMessageOf(err), '결제창 오류');
  } finally {
    paying.value = false;
  }
}

onMounted(() => {
  if (route.query.fail === '1') {
    toast.warning('결제가 취소되었거나 실패했습니다. 다시 시도해 주세요.');
  }
  void init();
});
</script>
