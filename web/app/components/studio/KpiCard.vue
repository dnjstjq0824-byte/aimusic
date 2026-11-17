<template>
  <UiCard class="space-y-1">
    <div class="text-caption font-bold text-text-muted uppercase tracking-wider">{{ title }}</div>
    <div class="text-2xl font-semibold tabular-nums" :class="valueClass">{{ value }}</div>
    <div v-if="sub" class="text-caption text-text-secondary">{{ sub }}</div>
  </UiCard>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import UiCard from '@/components/ui/UiCard.vue';

const props = withDefaults(
  defineProps<{ title: string; value: string; sub?: string; tone?: 'default' | 'brand' | 'success' }>(),
  { sub: '', tone: 'default' },
);

const valueClass = computed(
  () => ({ default: 'text-white', brand: 'text-brand-400', success: 'text-success' })[props.tone],
);
</script>
