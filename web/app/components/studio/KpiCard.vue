<template>
  <!-- v3 flow (phase 3): quiet label, large light number. Tone is a small
       dot next to the label instead of a coloured value. -->
  <UiCard class="flex flex-col gap-3">
    <div class="flex items-center gap-2 text-sm text-text-secondary">
      <span v-if="tone !== 'default'" class="h-1.5 w-1.5 shrink-0 rounded-full" :class="dotClass" aria-hidden="true" />
      <span class="truncate">{{ title }}</span>
    </div>
    <div class="type-display tabular-nums text-white" :class="valueSize">{{ value }}</div>
    <div v-if="sub" class="text-xs leading-relaxed text-text-muted">{{ sub }}</div>
  </UiCard>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import UiCard from '@/components/ui/UiCard.vue';

const props = withDefaults(
  defineProps<{ title: string; value: string; sub?: string; tone?: 'default' | 'brand' | 'success' }>(),
  { sub: '', tone: 'default' },
);

// Numbers read large; worded values ("정산 주기에 확정") step down so they
// don't wrap into a two-line headline.
const valueSize = computed(() => (props.value.length > 8 ? 'text-xl sm:text-2xl' : 'text-3xl sm:text-4xl'));

const dotClass =computed(() => ({ default: '', brand: 'bg-brand-400', success: 'bg-success' })[props.tone]);
</script>
