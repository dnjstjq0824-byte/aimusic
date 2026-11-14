<template>
  <div class="mx-auto max-w-2xl space-y-6 py-8">
    <UiPageHeader
      eyebrow="Certificate Verification"
      title="상업적 이용권 보유 인증"
      description="인증서 번호로 진위를 확인합니다. 인쇄 버튼으로 PDF 보관이 가능합니다."
    >
      <template #actions>
        <UiButton v-if="view" variant="secondary" size="sm" class="print:hidden" @click="printPage">
          인쇄 / PDF 저장
        </UiButton>
      </template>
    </UiPageHeader>

    <UiErrorBanner :message="error" title="인증서를 확인하지 못했습니다" />

    <div v-if="loading" class="space-y-2">
      <UiSkeleton height="h-10" width="w-full" />
      <UiSkeleton height="h-40" width="w-full" />
    </div>

    <UiCard v-else-if="view" class="space-y-4">
      <div class="flex flex-wrap items-center justify-between gap-2 border-b border-surface-border pb-3">
        <div class="font-mono text-lg font-black text-white">{{ view.certificate.certNumber }}</div>
        <UiBadge variant="success" size="sm">진위 확인됨</UiBadge>
      </div>
      <dl class="grid gap-3 text-xs sm:grid-cols-2">
        <div>
          <dt class="font-bold uppercase tracking-wider text-neutral-500 text-[10px]">곡명</dt>
          <dd class="mt-0.5 font-semibold text-white">{{ view.title }}</dd>
        </div>
        <div>
          <dt class="font-bold uppercase tracking-wider text-neutral-500 text-[10px]">AI 플랫폼</dt>
          <dd class="mt-0.5 text-neutral-200">
            {{ view.aiProvider }}<span v-if="view.aiModel"> ({{ view.aiModel }})</span>
          </dd>
        </div>
        <div>
          <dt class="font-bold uppercase tracking-wider text-neutral-500 text-[10px]">사용 요금제</dt>
          <dd class="mt-0.5 text-neutral-200">{{ view.planTier ?? '미기재' }}</dd>
        </div>
        <div>
          <dt class="font-bold uppercase tracking-wider text-neutral-500 text-[10px]">생성일</dt>
          <dd class="mt-0.5 text-neutral-200">{{ view.generatedAt ?? '미기재' }}</dd>
        </div>
        <div>
          <dt class="font-bold uppercase tracking-wider text-neutral-500 text-[10px]">게이트 통과</dt>
          <dd class="mt-0.5 font-semibold" :class="view.gatesPassed ? 'text-accent-emerald' : 'text-accent-amber'">
            {{ view.gatesPassed ? 'Gate 1-3 통과' : '일부 미통과' }}
          </dd>
        </div>
        <div>
          <dt class="font-bold uppercase tracking-wider text-neutral-500 text-[10px]">발급일</dt>
          <dd class="mt-0.5 text-neutral-200">{{ fmtDate(view.certificate.issuedAt) }}</dd>
        </div>
        <div v-if="view.isrc">
          <dt class="font-bold uppercase tracking-wider text-neutral-500 text-[10px]">ISRC</dt>
          <dd class="mt-0.5 font-mono text-neutral-200">{{ view.isrc }}</dd>
        </div>
      </dl>
      <p class="border-t border-surface-border pt-3 text-[11px] leading-relaxed text-neutral-500">
        본 인증은 생성 시점의 유료 상업 플랜 보유, 당시 정책상 상업 이용 허용, 중복·재등록 아님을
        파트너 신고와 자동 검사에 근거해 확인한 절차이며, 권리 관계를 보증하지 않습니다.
      </p>
    </UiCard>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useRoute } from 'vue-router';
import { trackApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import type { CertificateView } from '@/api/types';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';

const route = useRoute();
const view = ref<CertificateView | null>(null);
const loading = ref(true);
const error = ref<string | null>(null);

function fmtDate(iso: string): string {
  return new Date(iso).toLocaleDateString('ko-KR');
}

function printPage(): void {
  window.print();
}

onMounted(async () => {
  const number = route.params.number as string;
  if (!number) {
    loading.value = false;
    error.value = '인증서 번호가 없습니다.';
    return;
  }
  try {
    view.value = await trackApi.verifyCertificate(number);
  } catch (err) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
});
</script>
