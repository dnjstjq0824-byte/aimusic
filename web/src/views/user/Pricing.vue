<template>
  <div class="space-y-8">
    <UiPageHeader
      eyebrow="Pricing"
      title="요금제"
      description="이용권은 1개월 단기 패스입니다. 자동결제 없이, 필요할 때만 결제하세요. 결제는 Toss Payments로 안전하게 처리됩니다."
    />

    <UiErrorBanner :message="error" title="요금제를 불러오지 못했습니다" retry-label="다시 시도" @retry="load" />

    <div v-if="loading" class="grid gap-4 md:grid-cols-3">
      <UiCard v-for="i in 3" :key="i" class="space-y-3">
        <UiSkeleton height="h-5" width="w-1/2" />
        <UiSkeleton height="h-8" width="w-2/3" />
        <UiSkeleton height="h-3" width="w-full" />
        <UiSkeleton height="h-3" width="w-full" />
      </UiCard>
    </div>

    <div v-else class="grid gap-4 md:grid-cols-3">
      <!-- Free tier (always available) -->
      <UiCard class="flex flex-col">
        <div class="text-xs font-bold uppercase tracking-wider text-neutral-400">Free</div>
        <div class="mt-2 flex items-baseline gap-1">
          <span class="text-3xl font-black text-white">₩0</span>
        </div>
        <p class="mt-2 text-xs leading-relaxed text-neutral-400">
          검증 음원 스트리밍과 플레이리스트를 무료로 이용하세요.
        </p>
        <ul class="mt-4 space-y-2 text-xs text-neutral-300">
          <li class="flex items-start gap-2"><span class="text-accent-emerald" aria-hidden="true">✓</span>전체 검증 음원 스트리밍</li>
          <li class="flex items-start gap-2"><span class="text-accent-emerald" aria-hidden="true">✓</span>플레이리스트 무제한 생성</li>
          <li class="flex items-start gap-2"><span class="text-accent-emerald" aria-hidden="true">✓</span>모바일·웹 동기화</li>
        </ul>
        <div class="mt-6 flex-1" />
        <RouterLink to="/explore">
          <UiButton variant="secondary" size="md" full-width>무료로 듣기</UiButton>
        </RouterLink>
      </UiCard>

      <!-- Paid plans from the server catalog -->
      <UiCard
        v-for="plan in plans"
        :key="plan.id"
        :border-variant="plan.code === 'LISTENER_PLUS_MONTHLY' ? 'brand' : 'default'"
        class="flex flex-col"
      >
        <div class="flex items-center justify-between">
          <div class="text-xs font-bold uppercase tracking-wider text-neutral-400">
            {{ plan.interval === 'MONTHLY' ? '월간 패스' : plan.interval }}
          </div>
          <UiBadge v-if="plan.code === 'LISTENER_PLUS_MONTHLY'" variant="brand" size="xs">인기</UiBadge>
        </div>
        <div class="mt-2 text-lg font-bold text-white">{{ plan.nameKo }}</div>
        <div class="mt-1 flex items-baseline gap-1">
          <span class="text-3xl font-black text-white">₩{{ plan.amountKrw.toLocaleString('ko-KR') }}</span>
          <span class="text-xs text-neutral-500">/ 1개월</span>
        </div>
        <ul class="mt-4 space-y-2 text-xs text-neutral-300">
          <li class="flex items-start gap-2"><span class="text-accent-emerald" aria-hidden="true">✓</span>Free의 모든 혜택</li>
          <li class="flex items-start gap-2"><span class="text-accent-emerald" aria-hidden="true">✓</span>320kbps 고음질 스트리밍</li>
          <li class="flex items-start gap-2"><span class="text-accent-emerald" aria-hidden="true">✓</span>우선 심사·정산 리포트</li>
          <li v-if="plan.code === 'STUDIO_PRO_MONTHLY'" class="flex items-start gap-2"><span class="text-accent-emerald" aria-hidden="true">✓</span>파트너 스튜디오 프로 도구</li>
        </ul>
        <div class="mt-6 flex-1" />
        <UiButton
          :variant="plan.code === 'LISTENER_PLUS_MONTHLY' ? 'brand' : 'secondary'"
          size="md"
          full-width
          @click="startCheckout(plan)"
        >
          {{ currentPlanCode === plan.code ? '이용 중' : '이용권 구매하기' }}
        </UiButton>
      </UiCard>
    </div>

    <UiCard>
      <div class="flex items-start gap-3">
        <svg class="h-5 w-5 shrink-0 text-brand-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <path d="M12 3 5 6v5c0 4.4 3 8.4 7 10 4-1.6 7-5.6 7-10V6l-7-3z" />
          <path d="m9 12 2 2 4-4" />
        </svg>
        <div class="text-xs leading-relaxed text-neutral-300">
          <div class="font-bold text-white">안심 결제 안내</div>
          <p class="mt-1">
            결제는 Toss Payments 결제창에서 직접 처리되며, AVIBE 서버는 카드 정보를 보관하지 않습니다.
            이용권은 구매일로부터 1개월간 유효한 단기 패스로 자동 연장되지 않습니다.
            결제 후 7일 이내 미사용 시 전액 환불이 가능합니다.
          </p>
        </div>
      </div>
    </UiCard>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { billingApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { useAuthStore } from '@/stores/authStore';
import { useToast } from '@/composables/useToast';
import type { BillingPlan, BillingSubscription } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';

const router = useRouter();
const auth = useAuthStore();
const toast = useToast();

const plans = ref<BillingPlan[]>([]);
const subscription = ref<BillingSubscription | null>(null);
const loading = ref(true);
const error = ref<string | null>(null);

const currentPlanCode = computed(() => subscription.value?.plan?.code ?? null);

async function load(): Promise<void> {
  loading.value = true;
  error.value = null;
  try {
    plans.value = await billingApi.plans();
    if (auth.isAuthenticated) {
      const res = await billingApi.mySubscription();
      subscription.value = res.subscription;
    }
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
}

function startCheckout(plan: BillingPlan): void {
  if (currentPlanCode.value === plan.code) {
    toast.info('이미 이용 중인 요금제입니다.');
    return;
  }
  if (!auth.isAuthenticated) {
    toast.info('구매하려면 먼저 로그인하세요.');
    void router.push({ name: 'login', query: { redirect: `/checkout?plan=${plan.code}` } });
    return;
  }
  void router.push({ name: 'checkout', query: { plan: plan.code } });
}

onMounted(() => {
  void load();
});
</script>
