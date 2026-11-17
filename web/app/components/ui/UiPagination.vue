<template>
  <nav class="flex flex-wrap items-center justify-between gap-x-4 gap-y-2 py-2" :aria-label="t('common.paginator.label')">
    <div class="text-xs text-text-secondary">
      <slot name="info">
        <span v-if="total !== null">{{ t('common.paginator.ofTotal', { range: rangeText, total: fmt.number(total) }) }}</span>
        <span v-else>{{ t('common.paginator.pageOnly', { page }) }}</span>
      </slot>
    </div>
    <div class="flex flex-wrap items-center gap-1.5">
      <UiButton
        variant="secondary"
        size="sm"
        :disabled="page <= 1 || disabled"
        :aria-label="t('common.paginator.prev')"
        @click="$emit('prev')"
      >
        ← {{ t('common.paginator.prevShort') }}
      </UiButton>
      <template v-if="pageNumbers.length > 0">
        <button
          v-for="p in pageNumbers"
          :key="p"
          type="button"
          :aria-label="t('common.paginator.pageNum', { p })"
          :aria-current="p === page ? 'page' : undefined"
          :disabled="disabled"
          class="min-w-[2rem] rounded-lg px-2 py-1.5 text-xs font-semibold transition-colors disabled:opacity-50"
          :class="p === page
            ? 'bg-brand-500 text-on-brand shadow-glow-brand'
            : 'bg-surface-2 text-text-primary hover:bg-surface-3 hover:text-white border border-surface-border'"
          @click="goPage(p)"
        >
          {{ p }}
        </button>
      </template>
      <UiButton
        variant="secondary"
        size="sm"
        :disabled="!hasMore || disabled"
        :aria-label="t('common.paginator.next')"
        @click="$emit('next')"
      >
        {{ t('common.paginator.nextShort') }} →
      </UiButton>
    </div>
  </nav>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { useFormat } from '@/composables/useFormat';
import UiButton from './UiButton.vue';

const { t } = useI18n();
const fmt = useFormat();

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
  if (limit <= 0 || total === 0) return t('common.paginator.empty');
  const from = (props.page - 1) * limit + 1;
  const to = Math.min(total, props.page * limit);
  return t('common.paginator.range', { from: fmt.number(from), to: fmt.number(to) });
});

function goPage(p: number): void {
  if (p !== props.page) emit('page', p);
}
</script>
