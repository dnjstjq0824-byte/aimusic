<template>
  <div class="space-y-6">
    <UiPageHeader
      eyebrow="Audit Console"
      title="플랫폼 개요"
      description="유저·음원·검수·분쟁·재생·정산 상태를 한눈에 확인하세요."
    >
      <template #actions>
        <UiButton variant="secondary" size="sm" :loading="loading" @click="load">새로고침</UiButton>
      </template>
    </UiPageHeader>

    <UiErrorBanner :message="error" title="개요를 불러오지 못했습니다" retry-label="다시 시도" @retry="load" />

    <div v-if="loading" class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <UiCard v-for="i in 4" :key="i" class="space-y-2">
        <UiSkeleton height="h-3" width="w-1/2" />
        <UiSkeleton height="h-8" width="w-2/3" />
      </UiCard>
    </div>

    <template v-else-if="overview">
      <!-- Metric cards -->
      <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <UiCard class="space-y-1">
          <div class="text-caption font-bold text-text-muted uppercase tracking-wider">전체 유저</div>
          <div class="text-2xl font-black text-white">{{ overview.users.total.toLocaleString() }}명</div>
          <div class="text-caption text-text-secondary">
            리스너 {{ overview.users.users.toLocaleString() }} · 파트너 {{ overview.users.partners.toLocaleString() }} · 관리자 {{ overview.users.admins.toLocaleString() }}
          </div>
        </UiCard>
        <UiCard class="space-y-1">
          <div class="text-caption font-bold text-text-muted uppercase tracking-wider">재생 가능 음원</div>
          <div class="text-2xl font-black text-success">{{ overview.tracks.ready.toLocaleString() }}곡</div>
          <div class="text-caption text-text-secondary">
            심사 대기 {{ overview.tracks.pendingGate.toLocaleString() }} · 임시조치 {{ overview.tracks.takenDown.toLocaleString() }}
          </div>
        </UiCard>
        <UiCard class="space-y-1">
          <div class="text-caption font-bold text-text-muted uppercase tracking-wider">24시간 유효 재생</div>
          <div class="text-2xl font-black text-white">{{ overview.plays.last24h.toLocaleString() }}회</div>
          <div class="text-caption text-text-secondary">누적 {{ overview.plays.total.toLocaleString() }}회</div>
        </UiCard>
        <UiCard class="space-y-1">
          <div class="text-caption font-bold text-text-muted uppercase tracking-wider">정산 대기</div>
          <div class="text-2xl font-black text-brand-300">{{ overview.settlementsPending.count.toLocaleString() }}건</div>
          <div class="text-caption text-text-secondary">순액 합계 ₩{{ overview.settlementsPending.netKrw.toLocaleString() }}</div>
        </UiCard>
      </div>

      <!-- Queues + health -->
      <div class="grid gap-4 lg:grid-cols-2">
        <UiCard>
          <template #header>
            <h2 class="text-xs font-bold text-text-primary uppercase tracking-wider">검수 대기열</h2>
          </template>
          <ul class="space-y-2 text-xs">
            <li>
              <RouterLink to="/admin/reviews" class="flex items-center justify-between rounded-xl border border-surface-border bg-surface-2 p-3 transition-colors hover:bg-surface-3">
                <span class="font-semibold text-white">Gate 4 음원 검수</span>
                <UiBadge :variant="overview.reviews.tracks > 0 ? 'warning' : 'success'" size="sm">
                  {{ overview.reviews.tracks }}건
                </UiBadge>
              </RouterLink>
            </li>
            <li>
              <RouterLink to="/admin/reviews" class="flex items-center justify-between rounded-xl border border-surface-border bg-surface-2 p-3 transition-colors hover:bg-surface-3">
                <span class="font-semibold text-white">파트너 승인</span>
                <UiBadge :variant="overview.reviews.partners > 0 ? 'warning' : 'success'" size="sm">
                  {{ overview.reviews.partners }}건
                </UiBadge>
              </RouterLink>
            </li>
            <li>
              <RouterLink to="/admin/reviews" class="flex items-center justify-between rounded-xl border border-surface-border bg-surface-2 p-3 transition-colors hover:bg-surface-3">
                <span class="font-semibold text-white">구독 증빙 검증</span>
                <UiBadge :variant="overview.reviews.subscriptions > 0 ? 'warning' : 'success'" size="sm">
                  {{ overview.reviews.subscriptions }}건
                </UiBadge>
              </RouterLink>
            </li>
            <li>
              <RouterLink to="/admin/disputes" class="flex items-center justify-between rounded-xl border border-surface-border bg-surface-2 p-3 transition-colors hover:bg-surface-3">
                <span class="font-semibold text-white">미해결 분쟁</span>
                <UiBadge :variant="overview.disputesOpen > 0 ? 'danger' : 'success'" size="sm">
                  {{ overview.disputesOpen }}건
                </UiBadge>
              </RouterLink>
            </li>
          </ul>
        </UiCard>

        <UiCard>
          <template #header>
            <h2 class="text-xs font-bold text-text-primary uppercase tracking-wider">파이프라인 상태</h2>
          </template>
          <dl class="space-y-3 text-xs">
            <div class="flex items-center justify-between">
              <dt class="text-text-secondary">트랜스코딩 큐 깊이</dt>
              <dd class="font-mono font-bold text-white">
                {{ overview.queueDepth === null ? 'Redis 연결 불가' : `${overview.queueDepth}건` }}
              </dd>
            </div>
            <div class="flex items-center justify-between">
              <dt class="text-text-secondary">24시간 변환 실패</dt>
              <dd>
                <UiBadge :variant="overview.jobsFailed24h > 0 ? 'danger' : 'success'" size="sm">
                  {{ overview.jobsFailed24h }}건
                </UiBadge>
              </dd>
            </div>
            <div class="border-t border-surface-border pt-3">
              <RouterLink to="/admin/curation" class="text-brand-400 hover:underline font-semibold">
                큐레이션 &amp; 정산배치로 이동 →
              </RouterLink>
            </div>
          </dl>
        </UiCard>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { onMounted, ref } from 'vue';
import { adminApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import type { AdminOverview } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';

const overview = ref<AdminOverview | null>(null);
const loading = ref(true);
const error = ref<string | null>(null);

async function load(): Promise<void> {
  loading.value = true;
  error.value = null;
  try {
    overview.value = await adminApi.overview();
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
}

onMounted(() => {
  void load();
});
</script>
