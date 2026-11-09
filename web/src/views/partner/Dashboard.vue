<template>
  <div class="space-y-6">
    <UiPageHeader
      eyebrow="Studio Console"
      title="대시보드 &amp; 정산 관리"
      description="상업용 라이선스 검증 현황과 스트리밍 정산 내역을 관리합니다."
    >
      <template #actions>
        <RouterLink to="/partner/upload">
          <UiButton variant="brand" size="sm">
            새 음원 업로드 (Gatekeeper)
          </UiButton>
        </RouterLink>
        <RouterLink to="/partner/subscriptions">
          <UiButton variant="secondary" size="sm">
            AI 구독 증빙 관리
          </UiButton>
        </RouterLink>
      </template>
    </UiPageHeader>

    <UiErrorBanner :message="error" title="대시보드를 불러오지 못했습니다" />

    <!-- Metrics Cards -->
    <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <UiCard class="space-y-1">
        <div class="text-[11px] font-bold text-neutral-500 uppercase tracking-wider">등록된 전체 음원</div>
        <div class="text-2xl font-black text-white">{{ tracks.length }}곡</div>
        <div class="text-[11px] text-neutral-400">포렌식 검증 진행 포함</div>
      </UiCard>

      <UiCard class="space-y-1">
        <div class="text-[11px] font-bold text-neutral-500 uppercase tracking-wider">상업 유통 승인 (READY)</div>
        <div class="text-2xl font-black text-accent-emerald">{{ readyCount }}곡</div>
        <div class="text-[11px] text-emerald-400/80">Gate 1-4 전체 통과</div>
      </UiCard>

      <UiCard class="space-y-1">
        <div class="text-[11px] font-bold text-neutral-500 uppercase tracking-wider">검증된 AI 요금제</div>
        <div class="text-2xl font-black text-brand-400">{{ verifiedSubs }}개</div>
        <div class="text-[11px] text-neutral-400">Suno, AIVA Pro 등</div>
      </UiCard>

      <UiCard class="space-y-1">
        <div class="text-[11px] font-bold text-neutral-500 uppercase tracking-wider">누적 유효 스트리밍</div>
        <div class="text-2xl font-black text-white">{{ totalPlays.toLocaleString() }}회</div>
        <div class="text-[11px] text-neutral-400">30초 이상 재생 집계</div>
      </UiCard>
    </div>

    <!-- Settlement Projection & Tax Card -->
    <UiCard border-variant="brand">
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div class="space-y-1">
          <div class="flex items-center gap-2">
            <svg class="h-4 w-4 text-brand-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
              <rect x="3" y="6" width="18" height="13" rx="2" />
              <path d="M3 10h18M16 15h2" />
            </svg>
            <h3 class="text-xs font-bold text-white uppercase tracking-wider">예상 정산 (30초 유효 스트리밍)</h3>
          </div>
          <p class="text-xs text-neutral-300">
            총 누적 유효 스트리밍: <strong class="text-white font-mono">{{ totalPlays.toLocaleString() }}회</strong>
            · 추정 정산금액: <strong class="text-brand-400 font-mono">약 ₩{{ estimatedNet.toLocaleString() }}원</strong>
            (재생당 10원 · 소득세 3.3% 원천징수 후 추정치)
          </p>
        </div>
        <RouterLink to="/legal/PARTNER_CONTRACT" target="_blank" class="text-xs text-brand-400 hover:underline shrink-0">
          정산 계약 규정 확인 →
        </RouterLink>
      </div>
    </UiCard>

    <!-- Track Status Table -->
    <UiCard padding="none">
      <template #header>
        <div class="px-5 pt-3 flex items-center justify-between">
          <h2 class="text-xs font-bold text-neutral-400 uppercase tracking-wider">내 음원 게이트키퍼 통과 현황</h2>
          <span class="text-xs text-neutral-500 font-mono">{{ tracks.length }} 트랙</span>
        </div>
      </template>

      <div v-if="loading" class="space-y-2 p-4">
        <UiSkeleton height="h-10" width="w-full" />
        <UiSkeleton height="h-10" width="w-full" />
      </div>

      <div v-else-if="tracks.length === 0" class="py-12">
        <UiEmptyState
          title="등록된 음원이 없습니다"
          description="Suno Pro, AIVA Pro 등 유료 상업 플랜에서 생성한 음원을 업로드해 보세요."
        >
          <template #action>
            <RouterLink to="/partner/upload">
              <UiButton variant="brand" size="sm">첫 음원 등록하기</UiButton>
            </RouterLink>
          </template>
        </UiEmptyState>
      </div>

      <ul v-else class="divide-y divide-surface-border">
        <li
          v-for="t in tracks"
          :key="t.id"
          class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 px-5 py-3 hover:bg-surface-2/50 transition-colors text-xs"
        >
          <div class="min-w-0 flex-1">
            <div class="flex flex-col items-start gap-1.5 sm:flex-row sm:items-center sm:gap-2">
              <RouterLink
                :to="{ name: 'track-detail', params: { id: t.id } }"
                class="block min-w-0 max-w-full font-bold text-white hover:text-brand-400 truncate"
              >
                {{ t.title }}
              </RouterLink>
              <AiBadge :track="t" />
            </div>
            <div class="mt-1 flex flex-wrap items-center gap-2 text-[11px] text-neutral-400">
              <span>플랫폼: {{ t.aiProvider }}</span>
              <span>·</span>
              <span>재생: {{ t.playCount.toLocaleString() }}회</span>
              <span v-if="t.rejectionReasonKo" class="text-accent-crimson font-medium">
                · 반려 사유: {{ t.rejectionReasonKo }}
              </span>
            </div>
          </div>

          <div class="flex items-center gap-2 shrink-0">
            <UiBadge :variant="statusVariant(t.status)" size="sm">
              {{ statusKo(t.status) }}
            </UiBadge>
            <RouterLink :to="{ name: 'track-detail', params: { id: t.id } }">
              <UiButton variant="secondary" size="xs">상세보기</UiButton>
            </RouterLink>
          </div>
        </li>
      </ul>
    </UiCard>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { partnerApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import type { PartnerAISubscription, Track } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';
import AiBadge from '@/components/legal/AiBadge.vue';

const tracks = ref<Track[]>([]);
const subs = ref<PartnerAISubscription[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);

const readyCount = computed(() => tracks.value.filter((t) => t.status === 'READY').length);
const verifiedSubs = computed(() => subs.value.filter((s) => s.status === 'VERIFIED').length);
const totalPlays = computed(() => tracks.value.reduce((acc, t) => acc + (t.playCount || 0), 0));
// Mirrors backend settlement math: gross = plays × ₩10, withholding 3.3%.
const estimatedNet = computed(() => {
  const gross = totalPlays.value * 10;
  return gross - Math.floor((gross * 33) / 1000);
});

const STATUS_KO: Record<string, string> = {
  UPLOADED: '업로드됨',
  TRANSCODING: '변환 중',
  LICENSE_GATE_PENDING: '심사 대기',
  LICENSE_GATE_REJECTED: '게이트 반려',
  READY: '재생 가능',
  TAKEN_DOWN: '임시조치 중',
};

function statusKo(status: string): string {
  return STATUS_KO[status] ?? '처리 중';
}

function statusVariant(status: string): 'success' | 'danger' | 'warning' | 'neutral' {
  if (status === 'READY') return 'success';
  if (status === 'LICENSE_GATE_REJECTED' || status === 'TAKEN_DOWN') return 'danger';
  if (status === 'LICENSE_GATE_PENDING' || status === 'TRANSCODING') return 'warning';
  return 'neutral';
}

onMounted(async () => {
  loading.value = true;
  try {
    const [t, s] = await Promise.all([
      partnerApi.listMyTracks(50, 0),
      partnerApi.listSubscriptions(),
    ]);
    tracks.value = t;
    subs.value = s;
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
});
</script>
