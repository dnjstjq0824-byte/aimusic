<template>
  <span class="inline-flex items-center gap-2">
    <img :src="logoUrl" :width="size" :height="size" alt="AVIBE 로고" class="rounded-lg" />
    <span v-if="wordmark" class="flex flex-col leading-none">
      <slot>
        <span class="font-extrabold tracking-tight text-white" :class="wordmarkClass">AVIBE</span>
      </slot>
      <span
        v-if="tagline"
        class="mt-1 whitespace-nowrap text-caption font-semibold tracking-wider text-brand-400"
        :class="tagline === SLOGAN ? 'tracking-wide' : 'uppercase'"
      >
        {{ tagline }}
      </span>
    </span>
  </span>
</template>

<script lang="ts">
/** Brand slogan, shown under the logo, in the home hero and in the footer. */
export const SLOGAN = 'Beyond Your Music';
</script>

<script setup lang="ts">
import { computed } from 'vue';
import logoUrl from '@/assets/img/logo.svg';

const props = withDefaults(defineProps<{ size?: number; wordmark?: boolean; tagline?: string }>(), {
  size: 28,
  wordmark: true,
  tagline: SLOGAN,
});

const wordmarkClass = computed(() => (props.size >= 32 ? 'text-base' : 'text-sm'));
</script>
