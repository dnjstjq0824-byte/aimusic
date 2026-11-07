<template>
  <div v-if="visible" role="status">
    <UiCard :border-variant="borderVariant" padding="sm">
      <div class="flex items-center gap-2">
        <UiBadge :variant="badgeVariant" size="sm" :dot="status === 'checking'">{{ title }}</UiBadge>
      </div>
      <p class="mt-2 text-xs leading-relaxed text-neutral-300">{{ message }}</p>
      <p v-if="code" class="mt-1 font-mono text-[11px] text-neutral-500">{{ code }}</p>
    </UiCard>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { ERR_MESSAGES } from '@/api/client';
import UiCard from '@/components/ui/UiCard.vue';
import UiBadge from '@/components/ui/UiBadge.vue';

const props = defineProps<{ status: 'idle' | 'checking' | 'pass' | 'fail'; code?: string | null }>();

const visible = computed(() => props.status === 'checking' || props.status === 'pass' || props.status === 'fail');
const title = computed(() =>
  props.status === 'checking' ? '라이선스 사전검사 진행 중' : props.status === 'pass' ? 'Gate1-2 통과' : '업로드 차단',
);
const message = computed(() => {
  if (props.status === 'checking') return '상업 라이선스 적격성을 확인하고 있습니다.';
  if (props.status === 'pass') return '상업용 구독·생성일·제공자 정책 검사를 통과했습니다. 다음 단계로 진행하세요.';
  if (props.code && props.code in ERR_MESSAGES) return (ERR_MESSAGES[props.code] as string);
  return props.code ?? '검사에 실패했습니다.';
});
const badgeVariant = computed(() =>
  props.status === 'pass' ? 'success' : props.status === 'fail' ? 'danger' : 'info',
);
const borderVariant = computed(() =>
  props.status === 'pass' ? 'success' : props.status === 'fail' ? 'danger' : 'default',
);
</script>
