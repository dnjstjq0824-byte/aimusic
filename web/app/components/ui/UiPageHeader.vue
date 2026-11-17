<template>
  <div class="relative flex flex-col gap-5 pt-2 sm:flex-row sm:items-end sm:justify-between sm:pt-6">
    <!-- v3 flow: soft light behind every page title. -->
    <div class="hero-glow pointer-events-none absolute -left-40 -top-40 h-80 w-[44rem] max-w-[100vw] opacity-60" aria-hidden="true" />
    <div class="relative min-w-0">
      <div v-if="eyebrow || $slots.eyebrow" class="mb-2 flex items-center gap-2">
        <slot name="eyebrow">
          <span class="text-sm font-medium text-brand-400">{{ eyebrow }}</span>
        </slot>
      </div>
      <h1 class="type-display text-3xl text-white sm:text-5xl">{{ title }}</h1>
      <p v-if="description" class="mt-3 max-w-2xl text-sm leading-relaxed text-text-secondary sm:text-base">
        {{ description }}
      </p>
    </div>
    <div v-if="$slots.actions" class="relative flex shrink-0 flex-wrap items-center gap-2">
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
