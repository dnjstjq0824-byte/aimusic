<template>
  <ol class="flex items-center gap-1" :aria-label="label || $t('common.stepper.label')">
    <li v-for="(step, i) in steps" :key="i" class="flex flex-1 items-center gap-1 last:flex-none" :aria-current="i === current ? 'step' : undefined">
      <span
        class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-xs font-bold"
        :class="i < current ? 'bg-brand-500 text-on-brand' : i === current ? 'border-2 border-brand-500 text-brand-400' : 'border border-surface-border text-text-muted'"
      >
        <svg v-if="i < current" class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <path d="M20 6 9 17l-5-5" />
        </svg>
        <span v-else>{{ i + 1 }}</span>
      </span>
      <span class="mr-1 hidden text-xs font-semibold sm:inline" :class="i === current ? 'text-text-primary' : 'text-text-muted'">
        {{ step }}
      </span>
      <span v-if="i < steps.length - 1" class="h-px min-w-3 flex-1" :class="i < current ? 'bg-brand-500' : 'bg-surface-border'" aria-hidden="true" />
    </li>
  </ol>
</template>

<script setup lang="ts">
// Multi-step progress (signup flows, upload — UI §4).
withDefaults(defineProps<{ steps: string[]; current?: number; label?: string }>(), {
  current: 0,
  label: '',
});
</script>
