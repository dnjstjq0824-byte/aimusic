<template>
  <component
    :is="certNumber ? 'RouterLink' : 'span'"
    :to="certNumber ? `/certificate/${certNumber}` : undefined"
    class="inline-flex shrink-0 items-center gap-1 font-semibold text-verified"
    :class="sizeClass"
    :title="FULL_LABEL"
    :aria-label="FULL_LABEL"
  >
    <svg :class="iconClass" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <path d="M12 2 4 5.5v5.2c0 5 3.4 9.3 8 10.8 4.6-1.5 8-5.8 8-10.8V5.5L12 2z" opacity="0.25" />
      <path d="M12 2 4 5.5v5.2c0 5 3.4 9.3 8 10.8 4.6-1.5 8-5.8 8-10.8V5.5L12 2zm-1.2 13.6-3.3-3.3 1.4-1.4 1.9 1.9 4.7-4.7 1.4 1.4-5.1 5.1z" />
    </svg>
    <span v-if="showLabel">{{ label }}</span>
  </component>
</template>

<script setup lang="ts">
import { computed } from 'vue';

// Verified Seal (v2 §4.6): rendered ONLY for tracks with an issued certificate.
// Labels move to i18n in WI-R2; ko is the source.
const FULL_LABEL = '상업적 이용권 보유 인증';

const props = withDefaults(
  defineProps<{
    size?: 16 | 20 | 28;
    label?: string;
    showLabel?: boolean;
    certNumber?: string | null;
  }>(),
  { size: 16, label: '이용권 인증', showLabel: true, certNumber: null },
);

const sizeClass = computed(() => ({ 16: 'text-caption', 20: 'text-body-sm', 28: 'text-body' })[props.size]);
const iconClass = computed(() => ({ 16: 'h-4 w-4', 20: 'h-5 w-5', 28: 'h-7 w-7' })[props.size]);
</script>
