<template>
  <span class="inline-flex flex-wrap items-center gap-1">
    <UiBadge v-if="track.aiGenerated" variant="brand" size="xs" title="AI 생성 음원">AI</UiBadge>
    <UiBadge variant="neutral" size="xs" :title="track.aiDisclosureText ?? ''">
      {{ disclosureLabel }}
    </UiBadge>
    <UiBadge
      v-if="isCertified"
      variant="commercial"
      size="xs"
      title="상업 라이선스 검증 완료 (Gate1-3 통과 + 심사 승인)"
    >
      상업적 이용권 보유 인증
    </UiBadge>
    <UiBadge v-else :variant="statusVariant" size="xs">
      {{ statusLabel }}
    </UiBadge>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import UiBadge from '@/components/ui/UiBadge.vue';
import type { Track } from '@/api/types';

const props = defineProps<{ track: Track }>();

const STATUS_KO: Record<string, string> = {
  UPLOADED: '업로드됨',
  TRANSCODING: '변환 중',
  LICENSE_GATE_PENDING: '심사 대기',
  LICENSE_GATE_REJECTED: '게이트 반려',
  READY: '재생 가능',
  TAKEN_DOWN: '임시조치 중',
  REJECTED: '반려됨',
};

const disclosureLabel = computed(() => {
  const genType = props.track.aiGenerationType ?? '';
  if (genType === 'FULL_AI') return '순수 AI 생성';
  if (genType === 'AI_ASSISTED_HUMAN') return 'AI 어시스트';
  return props.track.aiProvider;
});

const statusLabel = computed(() => STATUS_KO[props.track.status] ?? '처리 중');
const statusVariant = computed(() => {
  const st = props.track.status;
  if (st === 'READY') return 'success';
  if (st === 'TAKEN_DOWN' || st === 'REJECTED' || st === 'LICENSE_GATE_REJECTED') return 'danger';
  return 'warning';
});

// Demo/launch tracks carry "(Launch Demo)" in the title and must never show
// the certification badge. The track API exposes no per-track cert field, so
// the title marker is the only frontend-distinguishable signal (a backend
// cert/demo field is the structural fix, engine scope).
const isDemo = computed(() => props.track.title.includes('(Launch Demo)'));
const isCertified = computed(() => props.track.status === 'READY' && !isDemo.value);
</script>
