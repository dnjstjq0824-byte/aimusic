<template>
  <div
    class="relative inline-flex items-center justify-center overflow-hidden rounded-full font-bold select-none shrink-0"
    :class="[
      sizeClasses[size],
      gradientClass,
    ]"
  >
    <img v-if="src" :src="src" :alt="alt" class="h-full w-full object-cover" />
    <span v-else class="text-white tracking-wider">{{ initials }}</span>
    <span
      v-if="status"
      class="absolute bottom-0 right-0 h-2.5 w-2.5 rounded-full ring-2 ring-surface-0"
      :class="status === 'online' ? 'bg-accent-emerald' : 'bg-text-muted'"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';

interface Props {
  src?: string;
  name?: string;
  alt?: string;
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl';
  status?: 'online' | 'offline';
}

const props = withDefaults(defineProps<Props>(), {
  src: '',
  name: 'User',
  alt: 'Avatar',
  size: 'md',
  status: undefined,
});

const sizeClasses = {
  xs: 'h-6 w-6 text-caption',
  sm: 'h-8 w-8 text-xs',
  md: 'h-10 w-10 text-sm',
  lg: 'h-12 w-12 text-base',
  xl: 'h-16 w-16 text-xl',
};

const initials = computed(() => {
  if (!props.name) return 'U';
  return props.name
    .split(' ')
    .map((w) => w[0])
    .join('')
    .substring(0, 2)
    .toUpperCase();
});

const gradientClass = computed(() => {
  const hash = (props.name || 'User').split('').reduce((acc, char) => acc + char.charCodeAt(0), 0);
  const gradients = [
    'bg-gradient-to-tr from-brand-600 to-amber-500',
    'bg-gradient-to-tr from-accent-cyan to-blue-700',
    'bg-gradient-to-tr from-teal-500 to-brandteal',
    'bg-gradient-to-tr from-accent-cyan to-blue-600',
    'bg-gradient-to-tr from-accent-crimson to-pink-600',
  ];
  return gradients[hash % gradients.length];
});
</script>
