<template>
  <div class="w-full">
    <label v-if="label" :for="selectId" class="mb-1.5 block text-xs font-medium text-neutral-300">
      {{ label }}
      <span v-if="required" class="text-accent-crimson" aria-hidden="true">*</span>
    </label>
    <div
      class="relative flex items-center rounded-lg border bg-surface-1 transition-all duration-150"
      :class="[
        error ? 'border-accent-crimson focus-within:ring-2 focus-within:ring-accent-crimson/40' : 'border-surface-border focus-within:border-brand-500 focus-within:ring-2 focus-within:ring-brand-500/30',
        disabled ? 'opacity-50 cursor-not-allowed bg-surface-0' : '',
      ]"
    >
      <select
        :id="selectId"
        :value="modelValue"
        :disabled="disabled"
        :required="required"
        :aria-invalid="error ? 'true' : undefined"
        :aria-describedby="describedBy || undefined"
        class="w-full appearance-none bg-transparent px-3 py-2 pr-8 text-sm text-neutral-100 focus:outline-none cursor-pointer"
        @change="$emit('update:modelValue', ($event.target as HTMLSelectElement).value); $emit('change', ($event.target as HTMLSelectElement).value)"
      >
        <slot />
      </select>
      <div class="pointer-events-none absolute right-3 flex items-center text-neutral-400">
        <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
        </svg>
      </div>
    </div>
    <p v-if="error" :id="errorId" class="mt-1 text-xs text-accent-crimson">{{ error }}</p>
    <p v-else-if="hint" :id="hintId" class="mt-1 text-xs text-neutral-500">{{ hint }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, useId } from 'vue';

interface Props {
  modelValue?: string | number;
  label?: string;
  error?: string | null;
  hint?: string;
  required?: boolean;
  disabled?: boolean;
  id?: string;
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: '',
  label: '',
  error: null,
  hint: '',
  required: false,
  disabled: false,
  id: undefined,
});

const autoId = useId();
const selectId = computed(() => props.id ?? `ui-select-${autoId}`);
const errorId = computed(() => `${selectId.value}-error`);
const hintId = computed(() => `${selectId.value}-hint`);
const describedBy = computed(() => {
  if (props.error) return errorId.value;
  if (props.hint) return hintId.value;
  return '';
});

defineEmits<{
  (e: 'update:modelValue', val: string): void;
  (e: 'change', val: string): void;
}>();
</script>
