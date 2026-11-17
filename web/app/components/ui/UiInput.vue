<template>
  <div class="w-full">
    <div v-if="label || $slots.labelRight" class="mb-1.5 flex items-center justify-between text-xs">
      <label v-if="label" :for="inputId" class="font-medium text-text-primary">
        {{ label }}
        <span v-if="required" class="text-accent-crimson" aria-hidden="true">*</span>
      </label>
      <slot name="labelRight" />
    </div>
    <div
      class="relative flex min-h-11 items-center rounded-lg border bg-surface-1 transition-all duration-150"
      :class="[
        error ? 'border-accent-crimson focus-within:ring-2 focus-within:ring-accent-crimson/40' : 'border-surface-border focus-within:border-brand-500 focus-within:ring-2 focus-within:ring-brand-500/30',
        disabled ? 'opacity-50 cursor-not-allowed bg-surface-0' : '',
      ]"
    >
      <div v-if="$slots.prefix" class="pl-3 text-text-secondary">
        <slot name="prefix" />
      </div>
      <input
        :id="inputId"
        ref="inputRef"
        :value="modelValue"
        :type="computedType"
        :placeholder="placeholder"
        :required="required"
        :disabled="disabled"
        :minlength="minlength"
        :maxlength="maxlength"
        :step="step"
        :min="min"
        :max="max"
        :aria-invalid="error ? 'true' : undefined"
        :aria-describedby="describedBy || undefined"
        class="w-full min-w-0 flex-1 bg-transparent px-3 py-2 text-sm text-text-primary placeholder-text-muted focus:outline-none"
        @input="$emit('update:modelValue', ($event.target as HTMLInputElement).value)"
        @blur="$emit('blur', $event)"
        @focus="$emit('focus', $event)"
        @keydown.enter="$emit('enter', $event)"
      />
      <div v-if="type === 'password'" class="shrink-0 pr-3">
        <button
          type="button"
          class="whitespace-nowrap text-xs text-text-secondary hover:text-text-primary transition-colors"
          :aria-label="showPassword ? $t('common.input.hidePassword') : $t('common.input.showPassword')"
          :aria-pressed="showPassword"
          @click="showPassword = !showPassword"
        >
          {{ showPassword ? $t('common.input.hide') : $t('common.input.show') }}
        </button>
      </div>
      <div v-else-if="isPickerType" class="shrink-0 pr-3">
        <button
          type="button"
          class="rounded p-0.5 text-text-secondary transition-colors hover:text-text-primary"
          :aria-label="$t('common.input.openPicker')"
          @click="openPicker"
        >
          <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
            <rect x="3" y="5" width="18" height="16" rx="2" />
            <path d="M8 3v4M16 3v4M3 10h18" />
          </svg>
        </button>
      </div>
      <div v-else-if="$slots.suffix" class="pr-3 text-text-secondary">
        <slot name="suffix" />
      </div>
    </div>
    <p v-if="error" :id="errorId" class="mt-1 text-xs text-accent-crimson">{{ error }}</p>
    <p v-else-if="hint" :id="hintId" class="mt-1 text-xs text-text-muted">{{ hint }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, useId } from 'vue';

interface Props {
  modelValue?: string | number;
  label?: string;
  placeholder?: string;
  type?: string;
  error?: string | null;
  hint?: string;
  required?: boolean;
  disabled?: boolean;
  id?: string;
  minlength?: number;
  maxlength?: number;
  step?: string | number;
  min?: string | number;
  max?: string | number;
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: '',
  label: '',
  placeholder: '',
  type: 'text',
  error: null,
  hint: '',
  required: false,
  disabled: false,
  id: undefined,
});

defineEmits<{
  (e: 'update:modelValue', val: string): void;
  (e: 'blur', event: FocusEvent): void;
  (e: 'focus', event: FocusEvent): void;
  (e: 'enter', event: KeyboardEvent): void;
}>();

const showPassword = ref(false);
const autoId = useId();
const inputRef = ref<HTMLInputElement | null>(null);

const inputId = computed(() => props.id ?? `ui-input-${autoId}`);
const errorId = computed(() => `${inputId.value}-error`);
const hintId = computed(() => `${inputId.value}-hint`);
const describedBy = computed(() => {
  if (props.error) return errorId.value;
  if (props.hint) return hintId.value;
  return '';
});

const computedType = computed(() => {
  if (props.type === 'password') {
    return showPassword.value ? 'text' : 'password';
  }
  return props.type;
});

// EV-P02: the native calendar indicator alone is unreliable in dark theme,
// so date inputs get an explicit button that opens the picker; the picked
// value flows back through the native input event into v-model.
const isPickerType = computed(() => props.type === 'date' || props.type === 'datetime-local');

function openPicker(): void {
  inputRef.value?.showPicker?.();
}
</script>
