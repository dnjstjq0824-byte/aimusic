<template>
  <span
    class="inline-flex shrink-0 items-center gap-1.5 whitespace-nowrap font-medium select-none"
    :class="[
      sizeClasses[size],
      variantClasses[variant],
      roundedClasses[rounded],
    ]"
  >
    <span v-if="dot" class="h-1.5 w-1.5 rounded-full" :class="dotColorClass" />
    <slot name="prefix" />
    <slot />
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue';

interface Props {
  variant?: 'commercial' | 'syncOnly' | 'blocked' | 'brand' | 'verified' | 'success' | 'warning' | 'danger' | 'info' | 'neutral';
  size?: 'xs' | 'sm' | 'md';
  rounded?: 'sm' | 'md' | 'full';
  dot?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'neutral',
  size: 'sm',
  rounded: 'md',
  dot: false,
});

const sizeClasses = {
  xs: 'px-1.5 py-0.5 text-caption tracking-tight',
  sm: 'px-2 py-0.5 text-xs',
  md: 'px-2.5 py-1 text-xs',
};

const roundedClasses = {
  sm: 'rounded',
  md: 'rounded-md',
  full: 'rounded-full',
};

const variantClasses = {
  commercial: 'bg-emerald-950/80 text-emerald-300 border border-emerald-600/40 shadow-sm font-semibold',
  syncOnly: 'bg-amber-950/80 text-amber-300 border border-amber-600/40 font-semibold',
  blocked: 'bg-red-950/80 text-red-300 border border-red-600/40 font-semibold',
  brand: 'bg-brand-950/80 text-brand-300 border border-brand-500/40 font-semibold',
  verified: 'bg-brand-950/80 text-brand-300 border border-brand-500/40 font-semibold',
  success: 'bg-emerald-950/80 text-emerald-400 border border-emerald-700/50',
  warning: 'bg-amber-950/80 text-amber-400 border border-amber-700/50',
  danger: 'bg-red-950/80 text-red-400 border border-red-700/50',
  info: 'bg-blue-950/80 text-blue-400 border border-blue-700/50',
  neutral: 'bg-surface-2 text-text-primary border border-surface-border',
};

const dotColorClass = computed(() => {
  switch (props.variant) {
    case 'commercial':
    case 'success':
      return 'bg-emerald-400 animate-pulse';
    case 'syncOnly':
    case 'warning':
      return 'bg-amber-400';
    case 'blocked':
    case 'danger':
      return 'bg-red-500';
    case 'brand':
      return 'bg-brand-500';
    case 'verified':
      return 'bg-brand-400';
    case 'info':
      return 'bg-blue-400';
    default:
      return 'bg-surface-3';
  }
});
</script>
