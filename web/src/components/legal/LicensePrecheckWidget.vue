<template>
  <div v-if="visible" class="rounded border p-3 text-sm" :class="tone">
    <div class="font-bold">{{ title }}</div>
    <div class="mt-1">{{ message }}</div>
    <div v-if="code" class="mt-1 font-mono text-xs opacity-80">{{ code }}</div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { ERR_MESSAGES } from '@/api/client';

const props = defineProps<{ status: 'idle' | 'checking' | 'pass' | 'fail'; code?: string | null }>();

const visible = computed(() => props.status === 'checking' || props.status === 'pass' || props.status === 'fail');
const title = computed(() =>
  props.status === 'checking' ? '라이선스 사전검사 진행 중…' : props.status === 'pass' ? 'Gate1-2 통과' : '업로드 차단',
);
const message = computed(() => {
  if (props.status === 'checking') return '상업 라이선스 적격성을 확인하고 있습니다.';
  if (props.status === 'pass') return '상업용 구독·생성일·제공자 정책 검사를 통과했습니다. 다음 단계로 진행하세요.';
  if (props.code && props.code in ERR_MESSAGES) return (ERR_MESSAGES[props.code] as string);
  return props.code ?? '검사에 실패했습니다.';
});
const tone = computed(() =>
  props.status === 'pass'
    ? 'border-emerald-700 bg-emerald-900/30 text-emerald-200'
    : props.status === 'fail'
      ? 'border-red-700 bg-red-900/30 text-red-200'
      : 'border-neutral-700 bg-neutral-800 text-neutral-200',
);
</script>
