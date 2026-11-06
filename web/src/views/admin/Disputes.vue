<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
      <div>
        <div class="inline-flex items-center gap-1.5 text-xs text-accent-purple font-bold uppercase tracking-wider mb-1">
          <span>OSP Governance</span>
        </div>
        <h1 class="text-2xl font-black text-white tracking-tight">저작권법 제103조 분쟁 관리 &amp; 임시조치</h1>
        <p class="text-xs text-neutral-400 mt-1">
          신고 접수된 음원에 대해 원클릭 스트리밍 차단(임시조치), 소명 심사, 원상 복구 및 영구 삭제를 처리합니다.
        </p>
      </div>
      <UiButton variant="secondary" size="sm" @click="load">새로고침</UiButton>
    </div>

    <!-- Error Alert -->
    <div v-if="error" class="rounded-xl border border-red-800 bg-red-950/40 p-4 text-xs text-red-200">
      {{ error }}
    </div>

    <!-- Disputes List -->
    <div v-if="loading" class="space-y-3">
      <UiSkeleton height="h-24" width="w-full" />
      <UiSkeleton height="h-24" width="w-full" />
    </div>

    <div v-else-if="disputes.length === 0" class="py-12">
      <UiEmptyState
        icon="⚖️"
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
              :variant="d.status === 'UPHELD' ? 'danger' : d.status === 'DISMISSED' ? 'success' : 'warning'"
              size="xs"
            >
              {{ d.status }}
            </UiBadge>
          </div>
          <div class="text-[11px] text-neutral-400">
            신고 접수: <strong>{{ d.reporterUserId ? `회원(${d.reporterUserId.slice(0, 8)})` : '외부/익명 접수' }}</strong>
          </div>
        </div>

        <div class="grid gap-2 text-xs sm:grid-cols-2">
          <div>
            <span class="text-neutral-500 text-[10px] uppercase font-bold block">대상 트랙 ID</span>
            <span class="font-mono text-neutral-200">{{ d.trackId }}</span>
          </div>
          <div>
            <span class="text-neutral-500 text-[10px] uppercase font-bold block">신고 사유 / 침해 유형</span>
            <span class="text-neutral-200 font-semibold">{{ d.reason }}</span>
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
            @click="resolve(d.id, true)"
          >
            침해 인정 (임시조치 &amp; 스트라이크)
          </UiButton>
          <UiButton
            variant="success"
            size="xs"
            @click="resolve(d.id, false)"
          >
            신고 기각 / 소명 인용 (정상 유지)
          </UiButton>
        </div>
      </UiCard>
    </div>
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
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';

const toast = useToast();

const disputes = ref<CopyrightDispute[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);

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

async function resolve(disputeId: string, uphold: boolean) {
  try {
    await adminApi.resolveDispute(disputeId, uphold);
    toast.success(uphold ? '침해 인정: 즉시 임시조치 및 스트라이크가 부과되었습니다.' : '신고 기각: 정상 상태로 유지/복원되었습니다.');
    await load();
  } catch (err) {
    toast.error(errMessageOf(err), '조치 실패');
  }
}

onMounted(() => {
  void load();
});
</script>
