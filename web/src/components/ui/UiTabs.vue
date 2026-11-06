<template>
  <div class="flex items-center gap-1 rounded-xl bg-surface-0 p-1 border border-surface-border">
    <button
      v-for="tab in tabs"
      :key="tab.value"
      type="button"
      class="flex-1 rounded-lg px-3 py-1.5 text-xs font-semibold transition-all duration-150 text-center select-none"
      :class="[
        modelValue === tab.value
          ? 'bg-surface-2 text-white shadow-sm border border-surface-border/60'
          : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-1/40',
      ]"
      @click="$emit('update:modelValue', tab.value)"
    >
      <span class="inline-flex items-center justify-center gap-1.5">
        <span v-if="tab.badge" class="h-1.5 w-1.5 rounded-full bg-brand-500" />
        {{ tab.label }}
        <span v-if="tab.count !== undefined" class="rounded-full bg-surface-3 px-1.5 text-[10px] text-neutral-300">
          {{ tab.count }}
        </span>
      </span>
    </button>
  </div>
</template>

<script setup lang="ts">
export interface TabOption {
  label: string;
  value: string;
  badge?: boolean;
  count?: number;
}

interface Props {
  modelValue: string;
  tabs: TabOption[];
}

defineProps<Props>();

defineEmits<{
  (e: 'update:modelValue', val: string): void;
}>();
</script>
