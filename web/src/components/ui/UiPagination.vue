<template>
  <nav class="flex items-center justify-between gap-4 py-2" aria-label="페이지네이션">
    <div class="text-xs text-neutral-400">
      <slot name="info">
        <span v-if="total !== null">전체 {{ total.toLocaleString('ko-KR') }}개 중 {{ rangeText }}</span>
        <span v-else>페이지 {{ page }}</span>
      </slot>
    </div>
    <div class="flex items-center gap-1.5">
      <UiButton
        variant="secondary"
        size="sm"
        :disabled="page <= 1 || disabled"
        aria-label="이전 페이지"
        @click="$emit('prev')"
      >
        ← 이전
      </UiButton>
      <template v-if="pageNumbers.length > 0">
        <button
          v-for="p in pageNumbers"
          :key="p"
          type="button"
          :aria-label="`페이지 ${p}`"
          :aria-current="p === page ? 'page' : undefined"
          :disabled="disabled"
          class="min-w-[2rem] rounded-lg px-2 py-1.5 text-xs font-semibold transition-colors disabled:opacity-50"
          :class="p === page
            ? 'bg-brand-500 text-[#04121a] shadow-glow-brand'
            : 'bg-surface-2 text-neutral-300 hover:bg-surface-3 hover:text-white border border-surface-border'"
          @click="goPage(p)"
        >
          {{ p }}
        </button>
      </template>
      <UiButton
        variant="secondary"
        size="sm"
        :disabled="!hasMore || disabled"
        aria-label="다음 페이지"
        @click="$emit('next')"
      >
        다음 →
      </UiButton>
    </div>
  </nav>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import UiButton from './UiButton.vue';

interface Props {
  page: number;
  hasMore?: boolean;
  disabled?: boolean;
  // When total+limit are provided, numbered pages render and `page` events fire.
  total?: number | null;
  limit?: number | null;
}

const props = withDefaults(defineProps<Props>(), {
  hasMore: true,
  disabled: false,
  total: null,
  limit: null,
});

const emit = defineEmits<{
  (e: 'prev'): void;
  (e: 'next'): void;
  (e: 'page', page: number): void;
}>();

const pageCount = computed(() => {
  if (props.total === null || props.limit === null || props.limit <= 0) return 0;
  return Math.max(1, Math.ceil(props.total / props.limit));
});

const pageNumbers = computed(() => {
  const count = pageCount.value;
  if (count <= 1) return [];
  // Compact window: current ± 2 clamped into range, max 5 numbers.
  const start = Math.min(Math.max(1, props.page - 2), Math.max(1, count - 4));
  const end = Math.min(count, start + 4);
  const out: number[] = [];
  for (let p = start; p <= end; p += 1) out.push(p);
  return out;
});

const rangeText = computed(() => {
  const limit = props.limit ?? 0;
  const total = props.total ?? 0;
  if (limit <= 0 || total === 0) return '0개';
  const from = (props.page - 1) * limit + 1;
  const to = Math.min(total, props.page * limit);
  return `${from.toLocaleString('ko-KR')}–${to.toLocaleString('ko-KR')}개`;
});

function goPage(p: number): void {
  if (p !== props.page) emit('page', p);
}
</script>
