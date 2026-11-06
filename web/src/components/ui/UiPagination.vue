<template>
  <div class="flex items-center justify-between gap-4 py-2">
    <div class="text-xs text-neutral-400">
      <slot name="info">
        <span>페이지 {{ page }}</span>
      </slot>
    </div>
    <div class="flex items-center gap-1.5">
      <UiButton
        variant="secondary"
        size="sm"
        :disabled="page <= 1 || disabled"
        @click="$emit('prev')"
      >
        ← 이전
      </UiButton>
      <UiButton
        variant="secondary"
        size="sm"
        :disabled="!hasMore || disabled"
        @click="$emit('next')"
      >
        다음 →
      </UiButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import UiButton from './UiButton.vue';

interface Props {
  page: number;
  hasMore?: boolean;
  disabled?: boolean;
}

withDefaults(defineProps<Props>(), {
  hasMore: true,
  disabled: false,
});

defineEmits<{
  (e: 'prev'): void;
  (e: 'next'): void;
}>();
</script>
