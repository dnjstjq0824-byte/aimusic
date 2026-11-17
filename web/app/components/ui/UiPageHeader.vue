<template>
  <div class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
    <div class="min-w-0">
      <div v-if="eyebrow || $slots.eyebrow" class="mb-1.5 flex items-center gap-2">
        <slot name="eyebrow">
          <span class="inline-flex items-center gap-1.5 text-xs font-bold uppercase tracking-wider text-brand-400">
            <span class="h-1.5 w-1.5 rounded-full bg-brand-500" aria-hidden="true" />
            {{ eyebrow }}
          </span>
        </slot>
      </div>
      <h1 class="font-display text-2xl font-black tracking-tight text-white sm:text-3xl">{{ title }}</h1>
      <p v-if="description" class="mt-1 max-w-2xl text-xs leading-relaxed text-text-secondary sm:text-sm">
        {{ description }}
      </p>
    </div>
    <div v-if="$slots.actions" class="flex shrink-0 flex-wrap items-center gap-2">
      <slot name="actions" />
    </div>
  </div>
</template>

<script setup lang="ts">
interface Props {
  eyebrow?: string;
  title: string;
  description?: string;
}

withDefaults(defineProps<Props>(), {
  eyebrow: '',
  description: '',
});
</script>
