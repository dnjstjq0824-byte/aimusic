<template>
  <div class="mx-auto max-w-2xl space-y-6">
    <UiPageHeader
      eyebrow="Checkout"
      :title="t('checkout.title')"
      :description="t('checkout.desc')"
    />

    <UiErrorBanner :message="error" :title="t('checkout.failTitle')" :retry-label="t('common.actions.retry')" @retry="init" />

    <div v-if="loading" class="space-y-3">
      <UiSkeleton height="h-40" width="w-full" rounded="xl" />
    </div>

    <template v-else-if="plan">
      <UiCard class="space-y-4">
        <div class="flex items-center justify-between border-b border-surface-border pb-3">
          <h2 class="text-sm font-bold text-white">{{ t('checkout.sheet') }}</h2>
          <UiBadge variant="warning" size="sm">{{ t('checkout.pending') }}</UiBadge>
        </div>
        <dl class="space-y-2.5 text-xs">
          <div class="flex justify-between">
            <dt class="text-text-muted">{{ t('checkout.fItem') }}</dt>
            <dd class="font-bold text-white">{{ plan.nameKo }}</dd>
          </div>
          <div v-if="order" class="flex justify-between">
            <dt class="text-text-muted">{{ t('checkout.fOrder') }}</dt>
            <dd class="font-mono text-text-primary">{{ order.orderId }}</dd>
          </div>
          <div class="flex justify-between">
            <dt class="text-text-muted">{{ t('checkout.fPeriod') }}</dt>
            <dd class="text-text-primary">{{ t('checkout.periodValue') }}</dd>
          </div>
          <div class="flex justify-between border-t border-surface-border pt-2.5">
            <dt class="font-bold text-white">{{ t('checkout.fAmount') }}</dt>
            <dd class="text-lg font-semibold text-brand-400">{{ fmt.currencyKRW(plan.amountKrw) }}</dd>
          </div>
        </dl>
      </UiCard>

      <!-- C6: withdrawal/refund consent gate (required before the order exists) -->
      <UiCard class="space-y-3">
        <h2 class="text-sm font-bold text-white">{{ t('checkout.consentTitle') }}</h2>
        <ul class="list-disc space-y-1.5 pl-5 text-xs leading-relaxed text-text-primary">
          <li v-for="key in REFUND_POLICY_LINE_KEYS" :key="key">{{ t(key) }}</li>
        </ul>
        <UiCheckbox v-model="refundAgreed" required>
          {{ t('checkout.consentAgree') }}
        </UiCheckbox>
      </UiCard>

      <UiCard v-if="!tossReady">
        <div class="flex items-start gap-3">
          <UiBadge variant="neutral" size="sm">{{ t('checkout.preparing') }}</UiBadge>
          <p class="text-xs leading-relaxed text-text-secondary">
            {{ t('checkout.preparingDesc') }}
          </p>
        </div>
      </UiCard>

      <div class="flex flex-col gap-2 sm:flex-row sm:justify-end">
        <RouterLink to="/pricing">
          <UiButton variant="ghost" size="md">{{ t('checkout.backToPricing') }}</UiButton>
        </RouterLink>
        <UiButton variant="brand" size="md" :loading="paying" :disabled="!tossReady || !refundAgreed" @click="pay">
          {{ t('checkout.payCta', { amount: fmt.currencyKRW(plan.amountKrw) }) }}
        </UiButton>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute } from 'vue-router';
import { loadTossPayments } from '@tosspayments/tosspayments-sdk';
import { billingApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { REFUND_POLICY_LINE_KEYS } from '@/api/refundPolicy';
import { useAuthStore } from '@/stores/authStore';
import { useFormat } from '@/composables/useFormat';
import { useToast } from '@/composables/useToast';
import { trackEvent } from '@/utils/analytics';
import type { BillingOrder, BillingPlan } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiCheckbox from '@/components/ui/UiCheckbox.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';

const route = useRoute();
const auth = useAuthStore();
const toast = useToast();
const { t } = useI18n();
const fmt = useFormat();

const plan = ref<BillingPlan | null>(null);
const order = ref<BillingOrder | null>(null);
const loading = ref(true);
const paying = ref(false);
const error = ref<string | null>(null);
const refundAgreed = ref(false);

const tossClientKey = (import.meta.env.VITE_TOSS_CLIENT_KEY as string | undefined) ?? '';
const tossReady = tossClientKey !== '';

async function init(): Promise<void> {
  const code = route.query.plan;
  if (typeof code !== 'string' || code === '') {
    error.value = t('checkout.errNoPlan');
    loading.value = false;
    return;
  }
  loading.value = true;
  error.value = null;
  try {
    // The merchant order is created at pay time (with the refund consent
    // timestamp), so init only resolves the plan from the public catalog.
    const plans = await billingApi.plans();
    const found = plans.find((p) => p.code === code);
    if (!found) {
      error.value = t('checkout.errPlanNotFound');
      return;
    }
    plan.value = found;
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
}

async function pay(): Promise<void> {
  if (!plan.value || !tossReady || !refundAgreed.value) return;
  paying.value = true;
  try {
    if (!order.value) {
      const res = await billingApi.createOrder(plan.value.code, new Date().toISOString());
      plan.value = res.plan;
      order.value = res.order;
    }
    trackEvent({ name: 'checkout_started', props: { plan: plan.value.code } });
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
    toast.error(errMessageOf(err), t('checkout.payWindowErr'));
  } finally {
    paying.value = false;
  }
}

onMounted(() => {
  if (route.query.fail === '1') {
    toast.warning(t('checkout.payCancelled'));
  }
  void init();
});
</script>
