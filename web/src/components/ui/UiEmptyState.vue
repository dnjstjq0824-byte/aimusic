<template>
  <div class="flex flex-col items-center justify-center p-8 text-center rounded-2xl border border-dashed border-surface-border bg-surface-1/40">
    <div class="mb-3 flex h-12 w-12 items-center justify-center rounded-2xl bg-surface-2 text-2xl border border-surface-border text-neutral-400">
      <slot name="icon">
        <span>{{ icon }}</span>
      </slot>
    </div>
    <h3 class="text-sm font-bold text-white">{{ title }}</h3>
    <p v-if="description" class="mt-1 max-w-sm text-xs text-neutral-400 leading-relaxed">{{ description }}</p>
    <div v-if="$slots.action" class="mt-4">
      <slot name="action" />
    </div>
  </div>
</template>

<script setup lang="ts">
interface Props {
  icon?: string;
  title: string;
  description?: string;
}

withDefaults(defineProps<Props>(), {
  icon: '🎵',
  description: '',
});
</script>
