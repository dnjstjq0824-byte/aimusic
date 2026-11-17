<template>
  <div class="w-full">
    <label v-if="label" :for="areaId" class="mb-1.5 block text-xs font-medium text-text-primary">
      {{ label }}
      <span v-if="required" class="text-accent-crimson" aria-hidden="true">*</span>
    </label>
    <textarea
      :id="areaId"
      :value="modelValue"
      :placeholder="placeholder"
      :required="required"
      :disabled="disabled"
      :rows="rows"
      :maxlength="maxlength"
      :aria-invalid="error ? 'true' : undefined"
      :aria-describedby="describedBy || undefined"
      class="w-full rounded-lg border bg-surface-1 px-3 py-2 text-sm text-text-primary placeholder-text-muted transition-all duration-150 focus:outline-none focus:ring-2 disabled:opacity-50 disabled:cursor-not-allowed resize-y"
      :class="[
        error
          ? 'border-accent-crimson focus:border-accent-crimson focus:ring-accent-crimson/40'
          : 'border-surface-border focus:border-brand-500 focus:ring-brand-500/30',
      ]"
      @input="$emit('update:modelValue', ($event.target as HTMLTextAreaElement).value)"
      @blur="$emit('blur', $event)"
    />
    <p v-if="error" :id="errorId" class="mt-1 text-xs text-accent-crimson">{{ error }}</p>
    <p v-else-if="hint" :id="hintId" class="mt-1 text-xs text-text-muted">{{ hint }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, useId } from 'vue';

interface Props {
  modelValue?: string;
  label?: string;
  placeholder?: string;
  error?: string | null;
  hint?: string;
  required?: boolean;
  disabled?: boolean;
  rows?: number;
  maxlength?: number;
  id?: string;
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: '',
  label: '',
  placeholder: '',
  error: null,
  hint: '',
  required: false,
  disabled: false,
  rows: 3,
  maxlength: undefined,
  id: undefined,
});

defineEmits<{
  (e: 'update:modelValue', val: string): void;
  (e: 'blur', event: FocusEvent): void;
}>();

const autoId = useId();
const areaId = computed(() => props.id ?? `ui-textarea-${autoId}`);
const errorId = computed(() => `${areaId.value}-error`);
const hintId = computed(() => `${areaId.value}-hint`);
const describedBy = computed(() => {
  if (props.error) return errorId.value;
  if (props.hint) return hintId.value;
  return '';
});
</script>
