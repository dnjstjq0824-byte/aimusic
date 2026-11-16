<template>
  <div class="mx-auto max-w-2xl space-y-6 py-6">
    <div v-if="loading" class="space-y-3 text-center">
      <div class="flex justify-center" aria-hidden="true">
        <svg class="h-8 w-8 animate-spin text-brand-500" viewBox="0 0 24 24" fill="none">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z" />
        </svg>
      </div>
      <h1 class="text-xl font-black text-white">{{ t('checkout.cfApproving') }}</h1>
      <p class="text-xs text-text-secondary">{{ t('checkout.cfDontClose') }}</p>
    </div>

    <template v-else-if="subscription">
      <UiPageHeader eyebrow="Payment" :title="t('checkout.cfDone')" />
      <UiCard border-variant="success" class="space-y-3">
        <div class="flex items-center justify-between">
          <span class="text-sm font-bold text-white">{{ subscription.plan?.nameKo ?? t('checkout.cfPass') }}</span>
          <UiBadge variant="success" size="sm">ACTIVE</UiBadge>
        </div>
        <dl class="space-y-2 text-xs">
          <div class="flex justify-between">
            <dt class="text-text-muted">{{ t('checkout.fPeriod') }}</dt>
            <dd class="text-text-primary">
              {{ fmt.dateISO(subscription.currentPeriodStart) }} ~ {{ fmt.dateISO(subscription.currentPeriodEnd) }}
            </dd>
          </div>
          <div class="flex justify-between">
            <dt class="text-text-muted">{{ t('checkout.fOrder') }}</dt>
            <dd class="font-mono text-text-primary">{{ orderId }}</dd>
          </div>
        </dl>
        <p class="border-t border-surface-border pt-3 text-caption leading-relaxed text-text-secondary">
          {{ t('checkout.cfRefundLabel') }}: {{ t(REFUND_POLICY_SUMMARY_KEY) }} {{ t('checkout.cfRefundNote') }}
        </p>
      </UiCard>
      <div class="flex justify-end gap-2">
        <RouterLink to="/library">
          <UiButton variant="brand" size="md">{{ t('checkout.cfGoLibrary') }}</UiButton>
        </RouterLink>
      </div>
    </template>

    <template v-else>
      <UiPageHeader eyebrow="Payment" :title="t('checkout.cfFailed')" />
      <UiErrorBanner :message="error" :title="t('checkout.cfFailBanner')" />
      <div class="flex justify-end gap-2">
        <RouterLink to="/pricing">
          <UiButton variant="secondary" size="md">{{ t('checkout.backToPricing') }}</UiButton>
        </RouterLink>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute } from 'vue-router';
import { billingApi } from '@/api/endpoints';
import { REFUND_POLICY_SUMMARY_KEY } from '@/api/refundPolicy';
import { errMessageOf } from '@/api/client';
import { useFormat } from '@/composables/useFormat';
import { trackEvent } from '@/utils/analytics';
import type { BillingSubscription } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';

const route = useRoute();
const { t } = useI18n();
const fmt = useFormat();

const loading = ref(true);
const subscription = ref<BillingSubscription | null>(null);
const orderId = ref('');
const error = ref<string | null>(null);

onMounted(async () => {
  const paymentKey = route.query.paymentKey;
  const oid = route.query.orderId;
  const amount = route.query.amount;
  if (typeof paymentKey !== 'string' || typeof oid !== 'string' || typeof amount !== 'string') {
    loading.value = false;
    error.value = t('checkout.cfBadResponse');
    return;
  }
  orderId.value = oid;
  try {
    const res = await billingApi.confirmPayment(paymentKey, oid, Number(amount));
    trackEvent({ name: 'checkout_completed' });
    subscription.value = res.subscription;
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
});
</script>
