<template>
  <span class="inline-flex items-center gap-2">
    <img :src="logoUrl" :width="size" :height="size" alt="AI Music logo" class="rounded-lg" />
    <span v-if="wordmark" class="font-bold tracking-tight"><slot>AI Music</slot></span>
  </span>
</template>

<script setup lang="ts">
import logoUrl from '@/assets/logo.svg';

withDefaults(defineProps<{ size?: number; wordmark?: boolean }>(), {
  size: 28,
  wordmark: true,
});
</script>
