<template>
  <span class="inline-flex items-center gap-1">
    <span v-if="track.aiGenerated" class="rounded bg-brand-600 px-1.5 py-0.5 text-[11px] font-semibold text-white">
      AI
    </span>
    <span class="rounded bg-neutral-700 px-1.5 py-0.5 text-[11px] text-neutral-200" :title="track.aiDisclosureText ?? ''">
      {{ disclosureLabel }}
    </span>
    <span
      v-if="track.status === 'READY'"
      class="rounded bg-emerald-700 px-1.5 py-0.5 text-[11px] font-semibold text-white"
      title="상업 라이선스 검증 완료 (Gate1-3 통과 + 심사 승인)"
    >
      상업라이선스 검증됨 ✓
    </span>
    <span v-else class="rounded bg-neutral-800 px-1.5 py-0.5 text-[11px] text-neutral-400">
      {{ track.status }}
    </span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { Track } from '@/api/types';

const props = defineProps<{ track: Track }>();

const disclosureLabel = computed(() => {
  const text = props.track.aiDisclosureText ?? '';
  if (text.startsWith('FULL_AI')) return 'FULL_AI';
  if (text.startsWith('AI_ASSISTED_HUMAN')) return 'AI_ASSISTED_HUMAN';
  return props.track.aiProvider;
});
</script>
