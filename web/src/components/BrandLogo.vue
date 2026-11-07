<template>
  <span class="inline-flex items-center gap-2">
    <img :src="logoUrl" :width="size" :height="size" alt="AVIBE 로고" class="rounded-lg" />
    <span v-if="wordmark" class="flex flex-col leading-none">
      <slot>
        <span class="font-extrabold tracking-tight text-white" :class="wordmarkClass">AVIBE</span>
      </slot>
      <span v-if="tagline" class="mt-0.5 text-[9px] font-semibold uppercase tracking-wider text-brand-400">
        {{ tagline }}
      </span>
    </span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import logoUrl from '@/assets/logo.svg';

const props = withDefaults(defineProps<{ size?: number; wordmark?: boolean; tagline?: string }>(), {
  size: 28,
  wordmark: true,
  tagline: '',
});

const wordmarkClass = computed(() => (props.size >= 32 ? 'text-base' : 'text-sm'));
</script>
