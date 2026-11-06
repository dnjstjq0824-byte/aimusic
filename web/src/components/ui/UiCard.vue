<template>
  <div
    class="rounded-2xl border bg-surface-1/90 transition-all duration-200"
    :class="[
      borderClass,
      paddingClasses[padding],
      hoverable ? 'hover:border-surface-border-hover hover:bg-surface-1 hover:shadow-subtle-elevated' : '',
      glass ? 'glass-panel' : '',
    ]"
  >
    <div v-if="$slots.header" class="mb-4 pb-3 border-b border-surface-border">
      <slot name="header" />
    </div>
    <slot />
    <div v-if="$slots.footer" class="mt-4 pt-3 border-t border-surface-border">
      <slot name="footer" />
    </div>
  </div>
</template>

<script setup lang="ts">
interface Props {
  padding?: 'none' | 'sm' | 'md' | 'lg';
  hoverable?: boolean;
  glass?: boolean;
  borderVariant?: 'default' | 'brand' | 'success' | 'danger';
}

const props = withDefaults(defineProps<Props>(), {
  padding: 'md',
  hoverable: false,
  glass: false,
  borderVariant: 'default',
});

const paddingClasses = {
  none: 'p-0',
  sm: 'p-3',
  md: 'p-4 sm:p-5',
  lg: 'p-6 sm:p-8',
};

const borderClass = {
  default: 'border-surface-border',
  brand: 'border-brand-500/40 shadow-glow-brand',
  success: 'border-accent-emerald/40',
  danger: 'border-accent-crimson/40',
}[props.borderVariant];
</script>
