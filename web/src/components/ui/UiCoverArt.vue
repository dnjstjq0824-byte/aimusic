<template>
  <div
    class="relative flex shrink-0 items-center justify-center overflow-hidden border border-surface-border"
    :class="[sizeClass, roundedClass, gradientClass]"
    role="img"
    :aria-label="`${title} 커버 아트`"
  >
    <!-- Decorative wave bars -->
    <svg
      class="absolute inset-0 h-full w-full opacity-25"
      viewBox="0 0 48 48"
      preserveAspectRatio="xMidYMid slice"
      aria-hidden="true"
    >
      <g :transform="`translate(${barsOffset}, 0)`">
        <rect
          v-for="(h, i) in bars"
          :key="i"
          :x="4 + i * 6"
          :y="24 - h / 2"
          width="3"
          :height="h"
          rx="1.5"
          fill="white"
        />
      </g>
    </svg>
    <svg
      class="relative text-white/90"
      :class="iconClass"
      viewBox="0 0 24 24"
      fill="currentColor"
      aria-hidden="true"
    >
      <path d="M9 18.5A3.5 3.5 0 1 1 5.5 15c.54 0 1.05.12 1.5.34V6.6L19 4v11.5a3.5 3.5 0 1 1-2-3.16V7.7l-8 1.5v9.3z" />
    </svg>
    <span
      v-if="verified"
      class="absolute bottom-1 right-1 flex h-4 w-4 items-center justify-center rounded-full bg-accent-emerald text-surface-0"
      title="상업 라이선스 검증됨"
    >
      <svg class="h-2.5 w-2.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <path d="M20 6 9 17l-5-5" />
      </svg>
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';

interface Props {
  seed: string;
  title?: string;
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl';
  rounded?: 'md' | 'lg' | 'xl';
  verified?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  title: '음원',
  size: 'md',
  rounded: 'lg',
  verified: false,
});

const sizeClass = computed(() => {
  switch (props.size) {
    case 'xs': return 'h-9 w-9';
    case 'sm': return 'h-11 w-11';
    case 'md': return 'h-12 w-12';
    case 'lg': return 'h-20 w-20';
    case 'xl': return 'h-40 w-40';
    default: return 'h-12 w-12';
  }
});

const roundedClass = computed(() => {
  switch (props.rounded) {
    case 'md': return 'rounded-lg';
    case 'lg': return 'rounded-xl';
    case 'xl': return 'rounded-2xl';
    default: return 'rounded-xl';
  }
});

const iconClass = computed(() => {
  switch (props.size) {
    case 'xs': return 'h-4 w-4';
    case 'sm': return 'h-5 w-5';
    case 'lg': return 'h-8 w-8';
    case 'xl': return 'h-14 w-14';
    default: return 'h-5 w-5';
  }
});

const GRADIENTS = [
  'bg-gradient-to-tr from-brand-700 via-brand-600 to-amber-500',
  'bg-gradient-to-tr from-brandteal-raised via-brandteal to-accent-emerald/70',
  'bg-gradient-to-tr from-accent-indigo via-accent-purple to-accent-crimson',
  'bg-gradient-to-tr from-accent-emerald/80 via-teal-600 to-accent-cyan/80',
  'bg-gradient-to-tr from-accent-crimson/90 via-brand-600 to-gold/80',
  'bg-gradient-to-tr from-surface-3 via-brand-800 to-brand-600',
];

const hash = computed(() => {
  let h = 2166136261;
  for (const ch of props.seed) {
    h ^= ch.codePointAt(0) ?? 0;
    h = Math.imul(h, 16777619);
  }
  return h >>> 0;
});

const gradientClass = computed(() => GRADIENTS[hash.value % GRADIENTS.length] as string);

// Deterministic decorative bars derived from the same seed.
const bars = computed(() => {
  let x = hash.value;
  const out: number[] = [];
  for (let i = 0; i < 7; i += 1) {
    x = (Math.imul(x, 1664525) + 1013904223) >>> 0;
    out.push(10 + (x % 26));
  }
  return out;
});

const barsOffset = computed(() => -((hash.value >> 3) % 4));
</script>
