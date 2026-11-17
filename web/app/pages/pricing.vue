<template>
  <div class="space-y-8">
    <UiPageHeader :title="t('pricing.title')" :description="t('pricing.sub')" />

    <UiErrorBanner :message="error" :title="t('pricing.errTitle')" :retry-label="t('track.retry')" @retry="load" />

    <div class="flex gap-1 rounded-xl border border-surface-border bg-surface-1 p-1" role="tablist" :aria-label="t('pricing.title')">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        type="button"
        role="tab"
        :aria-selected="activeTab === tab.id"
        class="flex-1 whitespace-nowrap rounded-lg px-3 py-2 text-xs font-bold transition-colors"
        :class="activeTab === tab.id ? 'bg-brand-500 text-on-brand' : 'text-text-secondary hover:text-white'"
        @click="activeTab = tab.id"
      >
        {{ t(tab.labelKey) }}
      </button>
    </div>

    <div v-if="loading" class="grid gap-4 md:grid-cols-3">
      <UiCard v-for="i in 3" :key="i" class="space-y-3">
        <UiSkeleton height="h-5" width="w-1/2" />
        <UiSkeleton height="h-8" width="w-2/3" />
        <UiSkeleton height="h-3" width="w-full" />
        <UiSkeleton height="h-3" width="w-full" />
      </UiCard>
    </div>

    <template v-else-if="!error">
      <!-- Listener tab -->
      <div v-if="activeTab === 'listener'" class="grid gap-4 md:grid-cols-3">
        <UiCard class="flex flex-col">
          <div class="text-xs font-bold uppercase tracking-wider text-text-secondary">{{ t('pricing.freeName') }}</div>
          <div class="mt-2 flex items-baseline gap-1">
            <span class="text-3xl font-semibold text-white">{{ fmt.currencyKRW(0) }}</span>
          </div>
          <p class="mt-2 text-xs leading-relaxed text-text-secondary">{{ t('pricing.freeDesc') }}</p>
          <ul class="mt-4 space-y-2 text-xs text-text-primary">
            <li class="flex items-start gap-2"><span class="text-success" aria-hidden="true">✓</span>{{ t('pricing.freeF1') }}</li>
            <li class="flex items-start gap-2"><span class="text-success" aria-hidden="true">✓</span>{{ t('pricing.freeF2') }}</li>
            <li class="flex items-start gap-2"><span class="text-success" aria-hidden="true">✓</span>{{ t('pricing.freeF3') }}</li>
          </ul>
          <div class="mt-6 flex-1" />
          <RouterLink :to="localePath('/explore')">
            <UiButton variant="secondary" size="md" full-width>{{ t('pricing.freeCta') }}</UiButton>
          </RouterLink>
        </UiCard>

        <UiCard
          v-for="plan in listenerPlans"
          :key="plan.id"
          :border-variant="plan.code === 'LISTENER_PLUS_MONTHLY' ? 'brand' : 'default'"
          class="flex flex-col"
        >
          <div class="flex items-center justify-between">
            <div class="text-xs font-bold uppercase tracking-wider text-text-secondary">
              {{ plan.interval === 'MONTHLY' ? t('pricing.perMonth').replace('/', '').trim() : plan.interval }}
            </div>
            <UiBadge v-if="plan.code === 'LISTENER_PLUS_MONTHLY'" variant="brand" size="xs">{{ t('pricing.popularBadge') }}</UiBadge>
          </div>
          <div class="mt-2 text-lg font-bold text-white">{{ plan.nameKo }}</div>
          <div class="mt-1 flex items-baseline gap-1">
            <span class="text-3xl font-semibold text-white">{{ fmt.currencyKRW(plan.amountKrw) }}</span>
            <span class="text-xs text-text-muted">{{ t('pricing.perMonth') }}</span>
          </div>
          <ul class="mt-4 space-y-2 text-xs text-text-primary">
            <li class="flex items-start gap-2"><span class="text-success" aria-hidden="true">✓</span>{{ t('pricing.plusF1') }}</li>
            <li class="flex items-start gap-2"><span class="text-success" aria-hidden="true">✓</span>{{ t('pricing.plusF2') }}</li>
          </ul>
          <div class="mt-6 flex-1" />
          <UiButton
            :variant="plan.code === 'LISTENER_PLUS_MONTHLY' ? 'brand' : 'secondary'"
            size="md"
            full-width
            @click="startCheckout(plan)"
          >
            {{ currentPlanCode === plan.code ? t('pricing.inUse') : t('pricing.buyCta') }}
          </UiButton>
        </UiCard>
      </div>

      <!-- Creator tab -->
      <div v-if="activeTab === 'creator'" class="space-y-4">
        <div class="grid gap-4 md:grid-cols-3">
          <UiCard class="flex flex-col">
            <div class="text-xs font-bold uppercase tracking-wider text-text-secondary">{{ t('pricing.creatorFreeName') }}</div>
            <div class="mt-2 flex items-baseline gap-1">
              <span class="text-3xl font-semibold text-white">{{ fmt.currencyKRW(0) }}</span>
            </div>
            <p class="mt-2 text-xs leading-relaxed text-text-secondary">{{ t('pricing.creatorFreeDesc') }}</p>
            <div class="mt-6 flex-1" />
            <RouterLink :to="localePath('/signup/creator')">
              <UiButton variant="secondary" size="md" full-width>{{ t('pricing.freeCta') }}</UiButton>
            </RouterLink>
          </UiCard>

          <UiCard
            v-for="plan in creatorPlans"
            :key="plan.id"
            class="flex flex-col"
            border-variant="brand"
          >
            <div class="text-xs font-bold uppercase tracking-wider text-text-secondary">
              {{ plan.interval === 'MONTHLY' ? t('pricing.perMonth').replace('/', '').trim() : plan.interval }}
            </div>
            <div class="mt-2 text-lg font-bold text-white">{{ plan.nameKo }}</div>
            <div class="mt-1 flex items-baseline gap-1">
              <span class="text-3xl font-semibold text-white">{{ fmt.currencyKRW(plan.amountKrw) }}</span>
              <span class="text-xs text-text-muted">{{ t('pricing.perMonth') }}</span>
            </div>
            <ul class="mt-4 space-y-2 text-xs text-text-primary">
              <li class="flex items-start gap-2"><span class="text-success" aria-hidden="true">✓</span>{{ t('pricing.studioF1') }}</li>
              <li class="flex items-start gap-2"><span class="text-success" aria-hidden="true">✓</span>{{ t('pricing.studioF2') }}</li>
              <li class="flex items-start gap-2"><span class="text-success" aria-hidden="true">✓</span>{{ t('pricing.studioF3') }}</li>
            </ul>
            <div class="mt-6 flex-1" />
            <UiButton variant="brand" size="md" full-width @click="startCheckout(plan)">
              {{ currentPlanCode === plan.code ? t('pricing.inUse') : t('pricing.buyCta') }}
            </UiButton>
          </UiCard>
        </div>
        <p class="text-center text-xs text-text-muted">{{ t('pricing.creditSoon') }}</p>
      </div>

      <!-- Business tab: no prices, inquiry only -->
      <div v-if="activeTab === 'business'" class="grid gap-4 md:grid-cols-3">
        <UiCard v-for="card in bizCards" :key="card.titleKey" class="flex flex-col">
          <div class="text-lg font-bold text-white">{{ t(card.titleKey) }}</div>
          <p class="mt-1 text-xs leading-relaxed text-text-secondary">{{ t(card.descKey) }}</p>
          <ul class="mt-4 space-y-2 text-xs text-text-primary">
            <li class="flex items-start gap-2"><span class="text-success" aria-hidden="true">✓</span>{{ t(card.l1Key) }}</li>
            <li class="flex items-start gap-2"><span class="text-success" aria-hidden="true">✓</span>{{ t(card.l2Key) }}</li>
            <li class="flex items-start gap-2"><span class="text-success" aria-hidden="true">✓</span>{{ t(card.l3Key) }}</li>
          </ul>
          <div class="mt-6 flex-1" />
          <UiButton variant="brand" size="md" full-width @click="inquiryOpen = true">
            {{ t('pricing.bizCta') }}
          </UiButton>
        </UiCard>
      </div>

      <!-- Comparison (listener + creator tabs) -->
      <section v-if="activeTab !== 'business'" :aria-label="t('pricing.cmpTitle')" class="overflow-x-auto">
        <h2 class="font-display mb-3 text-xl font-medium tracking-[-0.03em] text-white">{{ t('pricing.cmpTitle') }}</h2>
        <table class="w-full min-w-[480px] border-collapse text-xs">
          <thead>
            <tr class="border-b border-surface-border text-left text-text-secondary">
              <th scope="col" class="py-2 pr-4 font-semibold">{{ t('pricing.cmpH') }}</th>
              <th scope="col" class="py-2 pr-4 font-semibold">{{ t('pricing.freeName') }}</th>
              <th scope="col" class="py-2 pr-4 font-semibold">{{ plusName }}</th>
              <th scope="col" class="py-2 font-semibold">{{ studioName }}</th>
            </tr>
          </thead>
          <tbody class="text-text-primary">
            <tr v-for="row in cmpRows" :key="row.labelKey" class="border-b border-surface-border/50">
              <th scope="row" class="py-2.5 pr-4 text-left font-semibold text-text-secondary">{{ t(row.labelKey) }}</th>
              <td class="py-2.5 pr-4">{{ t(row.freeKey) }}</td>
              <td class="py-2.5 pr-4">{{ t(row.plusKey) }}</td>
              <td class="py-2.5">{{ t(row.studioKey) }}</td>
            </tr>
          </tbody>
        </table>
      </section>

      <!-- FAQ -->
      <section :aria-label="t('pricing.faqTitle')">
        <h2 class="font-display mb-3 text-xl font-medium tracking-[-0.03em] text-white">{{ t('pricing.faqTitle') }}</h2>
        <div class="space-y-2">
          <details v-for="item in faqItems" :key="item.qKey" class="group rounded-xl border border-surface-border bg-surface-1 px-4 py-3">
            <summary class="cursor-pointer text-sm font-bold text-white">{{ t(item.qKey) }}</summary>
            <p class="mt-2 text-xs leading-relaxed text-text-secondary">{{ t(item.aKey) }}</p>
          </details>
        </div>
      </section>

      <!-- Payment + refund (refundPolicy.ts is the verbatim source) -->
      <UiCard>
        <div class="flex items-start gap-3">
          <svg class="h-5 w-5 shrink-0 text-brand-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <path d="M12 3 5 6v5c0 4.4 3 8.4 7 10 4-1.6 7-5.6 7-10V6l-7-3z" />
            <path d="m9 12 2 2 4-4" />
          </svg>
          <div class="text-xs leading-relaxed text-text-primary">
            <div class="font-bold text-white">{{ t('pricing.payTitle') }}</div>
            <p class="mt-1">{{ t('pricing.payToss') }}</p>
            <p class="mt-1">{{ t('pricing.payPass') }}</p>
            <ul class="mt-2 list-disc space-y-1 pl-5">
              <li v-for="key in REFUND_POLICY_LINE_KEYS" :key="key">{{ t(key) }}</li>
            </ul>
          </div>
        </div>
      </UiCard>
    </template>

    <UiModal v-model="inquiryOpen" :title="t('business.inqTitle')" :description="t('business.inqSub')" max-width="lg">
      <div class="p-5">
        <InquiryForm />
      </div>
    </UiModal>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { billingApi } from '@/api/endpoints';
import { REFUND_POLICY_LINE_KEYS } from '@/api/refundPolicy';
import { useAuthStore } from '@/stores/authStore';
import { useToast } from '@/composables/useToast';
import { useApiError } from '@/composables/useApiError';
import { useFormat } from '@/composables/useFormat';
import type { BillingPlan, BillingSubscription } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiModal from '@/components/ui/UiModal.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import InquiryForm from '@/components/business/InquiryForm.vue';
const localePath = useLocalePath();

const { t } = useI18n();
const router = useRouter();
const auth = useAuthStore();
const toast = useToast();
const apiError = useApiError();
const fmt = useFormat();

type Tab = 'listener' | 'creator' | 'business';
// NOTE: tab/card/row/faq labelKeys resolve dynamically (invisible to
// check-i18n usage scan — they are used, do not delete).
const tabs: { id: Tab; labelKey: string }[] = [
  { id: 'listener', labelKey: 'pricing.tabListener' },
  { id: 'creator', labelKey: 'pricing.tabCreator' },
  { id: 'business', labelKey: 'pricing.tabBiz' },
];
const activeTab = ref<Tab>('listener');

const bizCards = [
  { titleKey: 'pricing.biz1T', descKey: 'pricing.biz1D', l1Key: 'pricing.biz1L1', l2Key: 'pricing.biz1L2', l3Key: 'pricing.biz1L3' },
  { titleKey: 'pricing.biz2T', descKey: 'pricing.biz2D', l1Key: 'pricing.biz2L1', l2Key: 'pricing.biz2L2', l3Key: 'pricing.biz2L3' },
  { titleKey: 'pricing.biz3T', descKey: 'pricing.biz3D', l1Key: 'pricing.biz3L1', l2Key: 'pricing.biz3L2', l3Key: 'pricing.biz3L3' },
];

const cmpRows = [
  { labelKey: 'pricing.cmpR1', freeKey: 'pricing.cmpR1Free', plusKey: 'pricing.cmpR1Plus', studioKey: 'pricing.cmpR1Studio' },
  { labelKey: 'pricing.cmpR2', freeKey: 'pricing.cmpR2Free', plusKey: 'pricing.cmpR2Plus', studioKey: 'pricing.cmpR2Studio' },
  { labelKey: 'pricing.cmpR3', freeKey: 'pricing.cmpR3Free', plusKey: 'pricing.cmpR3Plus', studioKey: 'pricing.cmpR3Studio' },
  { labelKey: 'pricing.cmpR4', freeKey: 'pricing.cmpR4Free', plusKey: 'pricing.cmpR4Plus', studioKey: 'pricing.cmpR4Studio' },
];

const faqItems = [
  { qKey: 'pricing.faq1Q', aKey: 'pricing.faq1A' },
  { qKey: 'pricing.faq2Q', aKey: 'pricing.faq2A' },
  { qKey: 'pricing.faq3Q', aKey: 'pricing.faq3A' },
  { qKey: 'pricing.faq4Q', aKey: 'pricing.faq4A' },
];

const plans = ref<BillingPlan[]>([]);
const subscription = ref<BillingSubscription | null>(null);
const loading = ref(true);
const error = ref<string | null>(null);
const inquiryOpen = ref(false);

const currentPlanCode = computed(() => subscription.value?.plan?.code ?? null);
const listenerPlans = computed(() => plans.value.filter(
  (p) => !p.code.startsWith('STUDIO_') && !p.code.startsWith('CREATOR_'),
));
const creatorPlans = computed(() => plans.value.filter(
  (p) => p.code.startsWith('STUDIO_') || p.code.startsWith('CREATOR_'),
));
const plusName = computed(
  () => listenerPlans.value.find((p) => p.code === 'LISTENER_PLUS_MONTHLY')?.nameKo
    ?? listenerPlans.value[0]?.nameKo ?? '',
);
const studioName = computed(() => creatorPlans.value[0]?.nameKo ?? '');

async function load(): Promise<void> {
  loading.value = true;
  error.value = null;
  try {
    plans.value = (await billingApi.plans()) ?? [];
    if (auth.isAuthenticated) {
      const res = await billingApi.mySubscription();
      subscription.value = res.subscription;
    }
  } catch (err: unknown) {
    error.value = apiError(err);
  } finally {
    loading.value = false;
  }
}

function startCheckout(plan: BillingPlan): void {
  if (currentPlanCode.value === plan.code) {
    toast.info(t('pricing.alreadyUsing'));
    return;
  }
  if (!auth.isAuthenticated) {
    toast.info(t('pricing.needLogin'));
    void router.push(localePath({ name: 'login', query: { redirect: `/checkout?plan=${plan.code}` } }));
    return;
  }
  void router.push(localePath({ name: 'checkout', query: { plan: plan.code } }));
}

onMounted(() => {
  void load();
});
</script>
