<template>
  <div class="space-y-6">
    <RouterLink
      :to="localePath('/account')"
      class="inline-flex min-h-11 items-center text-sm font-semibold text-text-secondary hover:text-white md:hidden"
    >
      <span aria-hidden="true" class="mr-1">‹</span>
      {{ t('account.navBack') }}
    </RouterLink>
    <div class="flex flex-col gap-6 md:flex-row">
      <AccountNav class="hidden md:block" />
      <div class="min-w-0 flex-1 space-y-6">
        <UiPageHeader :title="t('account.billTitle')" :description="t('account.billSub')" />

        <UiCard v-if="!subscription" class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
          <div>
            <h3 class="text-sm font-bold text-white">{{ t('account.billNone') }}</h3>
            <p class="text-xs text-text-secondary mt-1">{{ t('account.billNoneHint') }}</p>
          </div>
          <RouterLink :to="localePath('/pricing')">
            <UiButton variant="brand" size="sm">{{ t('account.billViewPlans') }}</UiButton>
          </RouterLink>
        </UiCard>

        <UiCard v-else class="space-y-3">
          <div class="flex items-center justify-between">
            <h3 class="text-sm font-bold text-white">{{ subscription.plan?.nameKo || subscription.planId }}</h3>
            <UiBadge variant="success" size="xs">{{ t('account.billActive') }}</UiBadge>
          </div>
          <div class="space-y-1.5 text-xs">
            <div class="flex justify-between">
              <span class="text-text-secondary">{{ t('account.billValid') }}</span>
              <span class="font-medium text-text-primary">{{ fmtDate(subscription.currentPeriodStart) }} ~ {{ fmtDate(subscription.currentPeriodEnd) }}</span>
            </div>
            <div v-if="subscription.cancelRequestedAt" class="flex justify-between">
              <span class="text-text-secondary">{{ t('account.billCancelAt') }}</span>
              <span class="font-medium text-accent-amber">{{ t('account.billEndsOn', { date: fmtDate(subscription.currentPeriodEnd) }) }}</span>
            </div>
          </div>
          <div class="pt-1">
            <UiButton variant="danger" size="sm" @click="cancelOpen = true">
              {{ t('account.billCancel') }}
            </UiButton>
          </div>
        </UiCard>

        <UiCard class="space-y-2">
          <h3 class="text-sm font-bold text-white">{{ t('account.sumTitle') }}</h3>
          <div class="grid gap-2 text-xs sm:grid-cols-3">
            <div>
              <div class="text-text-muted">{{ t('account.sumId') }}</div>
              <div class="font-mono text-text-primary break-all">{{ auth.email }}</div>
            </div>
            <div>
              <div class="text-text-muted">{{ t('account.sumRole') }}</div>
              <div class="font-semibold text-text-primary">{{ roleLabel }}</div>
            </div>
            <div>
              <div class="text-text-muted">{{ t('account.sumConsent') }}</div>
              <div class="text-verified font-semibold">{{ t('account.sumConsentDone') }}</div>
            </div>
          </div>
        </UiCard>

        <UiModal v-model="cancelOpen" :title="t('account.cancelTitle')" :description="t('account.cancelDesc')" max-width="md">
          <div class="space-y-3 p-5">
            <UiCard>
              <div class="font-semibold text-white text-sm mb-1">{{ t('account.cancelEnd') }}</div>
              <div class="text-xs text-text-secondary mb-3">
                {{ subscription ? t('account.cancelEndDesc', { date: fmtDate(subscription.currentPeriodEnd) }) : '' }}
              </div>
              <UiButton
                v-if="!subscription?.cancelRequestedAt"
                variant="secondary"
                size="sm"
                :loading="cancelLoading"
                @click="requestCancelEnd"
              >
                {{ t('account.cancelDoEnd') }}
              </UiButton>
              <div v-else class="text-xs font-semibold text-accent-amber">
                {{ t('account.cancelEndDone') }}
              </div>
            </UiCard>

            <UiCard>
              <div class="font-semibold text-white text-sm mb-1">{{ t('account.cancelNow') }}</div>
              <div v-if="!refundPreview" class="text-xs text-text-muted">
                {{ t('account.cancelNowUnknown') }}
              </div>
              <div v-else class="text-xs text-brand-300 mb-3">{{ refundText }}</div>
              <UiButton variant="danger" size="sm" :loading="cancelLoading" @click="requestCancelNow">
                {{ t('account.cancelDoNow') }}
              </UiButton>
            </UiCard>

            <div class="flex justify-end">
              <UiButton variant="outline" size="sm" @click="cancelOpen = false">
                {{ t('account.cancelClose') }}
              </UiButton>
            </div>
          </div>
        </UiModal>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { computed, onMounted, ref } from 'vue';
import { billingApi } from '@/api/endpoints';
import { useAuthStore } from '@/stores/authStore';
import { useToast } from '@/composables/useToast';
import { useApiError } from '@/composables/useApiError';
import { useFormat } from '@/composables/useFormat';
import type { BillingSubscription, RefundQuote } from '@/api/types';
import AccountNav from '@/components/account/AccountNav.vue';
import UiButton from '@/components/ui/UiButton.vue';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiModal from '@/components/ui/UiModal.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';

const { t } = useI18n();
const localePath = useLocalePath();
const auth = useAuthStore();
const toast = useToast();
const apiError = useApiError();
const fmt = useFormat();

const subscription = ref<BillingSubscription | null>(null);
const refundPreview = ref<RefundQuote | null>(null);
const cancelOpen = ref(false);
const cancelLoading = ref(false);

const roleLabel = computed(() => {
  if (auth.role === 'ADMIN') return t('account.roleAdmin');
  if (auth.role === 'PARTNER') return t('account.rolePartner');
  return t('account.roleUser');
});

const refundText = computed(() => {
  const q = refundPreview.value;
  if (!q) return '';
  if (q.type === 'WITHDRAWAL') {
    return t('account.refundFull', { amount: fmt.currencyKRW(q.refundKrw) });
  }
  if (q.refundKrw <= 0) {
    return t('account.refundZero', { days: fmt.number(q.usedDays) });
  }
  return t('account.refundPart', {
    used: fmt.number(q.usedDays),
    total: fmt.number(q.totalDays),
    deduct: fmt.currencyKRW(q.amountKrw - q.refundKrw),
    refund: fmt.currencyKRW(q.refundKrw),
  });
});

function fmtDate(v?: string | null): string {
  if (!v) return '';
  const d = new Date(v);
  return Number.isNaN(d.getTime()) ? '' : d.toLocaleDateString();
}

async function loadBilling(): Promise<void> {
  if (!auth.isAuthenticated) return;
  try {
    const res = await billingApi.mySubscription();
    subscription.value = res.subscription;
    if (res.subscription) {
      refundPreview.value = await billingApi.refundQuote().catch(() => null);
    }
  } catch (err: unknown) {
    toast.error(apiError(err));
  }
}

async function requestCancelEnd(): Promise<void> {
  cancelLoading.value = true;
  try {
    const res = await billingApi.cancelSubscription('END_OF_PERIOD');
    if (subscription.value && res.subscription) {
      subscription.value = { ...subscription.value, cancelRequestedAt: res.subscription.cancelRequestedAt };
    }
    toast.success(t('account.cancelDoneEnd'));
    cancelOpen.value = false;
  } catch (err: unknown) {
    toast.error(apiError(err));
  } finally {
    cancelLoading.value = false;
  }
}

async function requestCancelNow(): Promise<void> {
  cancelLoading.value = true;
  try {
    const res = await billingApi.cancelSubscription('IMMEDIATE');
    const amount = res.quote?.refundKrw ?? 0;
    if (res.mode === 'IMMEDIATE') {
      subscription.value = null;
    }
    toast.success(amount > 0
      ? t('account.cancelDoneNow', { amount: fmt.currencyKRW(amount) })
      : t('account.cancelDoneNowZero'));
    cancelOpen.value = false;
  } catch (err: unknown) {
    toast.error(apiError(err));
  } finally {
    cancelLoading.value = false;
  }
}

onMounted(() => {
  void loadBilling();
});
</script>
