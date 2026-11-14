<template>
  <div
    class="animate-pulse rounded bg-surface-2/80"
    :class="[
      roundedClass,
      widthClass,
      heightClass,
    ]"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue';

interface Props {
  width?: string;
  height?: string;
  rounded?: 'sm' | 'md' | 'lg' | 'xl' | 'full';
}

const props = withDefaults(defineProps<Props>(), {
  width: 'w-full',
  height: 'h-4',
  rounded: 'md',
});

const roundedClass = computed(() => {
  switch (props.rounded) {
    case 'sm': return 'rounded';
    case 'md': return 'rounded-md';
    case 'lg': return 'rounded-xl';
    case 'xl': return 'rounded-2xl';
    case 'full': return 'rounded-full';
    default: return 'rounded-md';
  }
});

const widthClass = computed(() => props.width);
const heightClass = computed(() => props.height);
</script>
