<template>
  <div role="tablist" aria-label="탭" class="flex items-center gap-1 rounded-xl bg-surface-0 p-1 border border-surface-border">
    <button
      v-for="(tab, i) in tabs"
      :key="tab.value"
      ref="tabRefs"
      type="button"
      role="tab"
      :aria-selected="modelValue === tab.value"
      :tabindex="modelValue === tab.value ? 0 : -1"
      class="flex-1 whitespace-nowrap rounded-lg px-2 py-1.5 text-[11px] font-semibold sm:px-3 sm:text-xs transition-all duration-150 text-center select-none"
      :class="[
        modelValue === tab.value
          ? 'bg-surface-2 text-white shadow-sm border border-surface-border/60'
          : 'text-neutral-400 hover:text-neutral-200 hover:bg-surface-1/40',
      ]"
      @click="$emit('update:modelValue', tab.value)"
      @keydown.left.prevent="moveFocus(i, -1)"
      @keydown.right.prevent="moveFocus(i, 1)"
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
import { ref } from 'vue';

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

const props = defineProps<Props>();

const emit = defineEmits<{
  (e: 'update:modelValue', val: string): void;
}>();

const tabRefs = ref<HTMLButtonElement[]>([]);

function moveFocus(index: number, delta: number): void {
  const next = (index + delta + props.tabs.length) % props.tabs.length;
  const tab = props.tabs[next];
  if (tab) {
    emit('update:modelValue', tab.value);
    tabRefs.value[next]?.focus();
  }
}
</script>
