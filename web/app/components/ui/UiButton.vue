<template>
  <button
    :type="type"
    :disabled="disabled || loading"
    class="inline-flex items-center justify-center font-medium transition-all duration-200 select-none relative disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer active:scale-[0.98]"
    :class="[
      sizeClasses[size],
      variantClasses[variant],
      roundedClasses[rounded],
      fullWidth ? 'w-full' : '',
    ]"
    @click="$emit('click', $event)"
  >
    <svg
      v-if="loading"
      class="animate-spin -ml-1 mr-2 h-4 w-4"
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
    >
      <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
      <path
        class="opacity-75"
        fill="currentColor"
        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
      ></path>
    </svg>
    <slot name="prefix" />
    <span :class="{ 'opacity-90': loading }"><slot /></span>
    <slot name="suffix" />
  </button>
</template>

<script setup lang="ts">
interface Props {
  variant?: 'primary' | 'secondary' | 'outline' | 'ghost' | 'danger' | 'success' | 'brand';
  size?: 'xs' | 'sm' | 'md' | 'lg';
  type?: 'button' | 'submit' | 'reset';
  disabled?: boolean;
  loading?: boolean;
  fullWidth?: boolean;
  rounded?: 'sm' | 'md' | 'lg' | 'full';
}

withDefaults(defineProps<Props>(), {
  variant: 'primary',
  size: 'md',
  type: 'button',
  disabled: false,
  loading: false,
  fullWidth: false,
  rounded: 'md',
});

defineEmits<{
  (e: 'click', event: MouseEvent): void;
}>();

const sizeClasses = {
  xs: 'min-h-6 px-2 py-1 text-xs gap-1.5',
  sm: 'min-h-9 px-3 py-1.5 text-xs gap-2',
  md: 'min-h-11 px-4 py-2 text-sm gap-2',
  lg: 'min-h-[52px] px-5 py-2.5 text-base gap-2.5',
};

const roundedClasses = {
  sm: 'rounded',
  md: 'rounded-lg',
  lg: 'rounded-xl',
  full: 'rounded-full',
};

const variantClasses = {
  brand: 'bg-brand-500 hover:bg-brand-400 text-on-brand font-bold shadow-glow-brand hover:shadow-glow-brand-lg border border-brand-300/40',
  primary: 'bg-brand-500 hover:bg-brand-400 text-on-brand font-bold shadow-glow-brand hover:shadow-glow-brand-lg border border-brand-300/40',
  secondary: 'bg-surface-2 hover:bg-surface-3 text-text-primary border border-surface-border hover:border-surface-border-hover',
  outline: 'bg-transparent hover:bg-surface-2 text-text-secondary border border-surface-border hover:border-surface-border-hover',
  ghost: 'bg-transparent hover:bg-surface-2 text-text-secondary hover:text-text-primary',
  danger: 'bg-accent-crimson/90 hover:bg-accent-crimson text-white border border-red-500/30 shadow',
  success: 'bg-success hover:brightness-110 text-on-brand font-semibold shadow',
};
</script>
