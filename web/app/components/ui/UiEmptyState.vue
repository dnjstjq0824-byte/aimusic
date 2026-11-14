<template>
  <div class="flex flex-col items-center justify-center p-8 text-center rounded-2xl border border-dashed border-surface-border bg-surface-1/40">
    <div class="mb-3 flex h-12 w-12 items-center justify-center rounded-2xl bg-surface-2 border border-surface-border text-text-secondary">
      <slot name="icon">
        <svg v-if="!icon" class="h-6 w-6" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
          <path d="M9 18.5A3.5 3.5 0 1 1 5.5 15c.54 0 1.05.12 1.5.34V6.6L19 4v11.5a3.5 3.5 0 1 1-2-3.16V7.7l-8 1.5v9.3z" />
        </svg>
        <span v-else aria-hidden="true">{{ icon }}</span>
      </slot>
    </div>
    <h3 class="text-sm font-bold text-white">{{ title }}</h3>
    <p v-if="description" class="mt-1 max-w-sm text-xs text-text-secondary leading-relaxed">{{ description }}</p>
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
  icon: '',
  description: '',
});
</script>
