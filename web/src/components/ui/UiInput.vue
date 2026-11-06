<template>
  <div class="w-full">
    <div v-if="label || $slots.labelRight" class="mb-1.5 flex items-center justify-between text-xs">
      <label v-if="label" class="font-medium text-neutral-300">
        {{ label }}
        <span v-if="required" class="text-accent-crimson">*</span>
      </label>
      <slot name="labelRight" />
    </div>
    <div
      class="relative flex items-center rounded-lg border bg-surface-1 transition-all duration-150"
      :class="[
        error ? 'border-accent-crimson focus-within:ring-2 focus-within:ring-accent-crimson/40' : 'border-surface-border focus-within:border-brand-500 focus-within:ring-2 focus-within:ring-brand-500/30',
        disabled ? 'opacity-50 cursor-not-allowed bg-surface-0' : '',
      ]"
    >
      <div v-if="$slots.prefix" class="pl-3 text-neutral-400">
        <slot name="prefix" />
      </div>
      <input
        :id="id"
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
        class="w-full bg-transparent px-3 py-2 text-sm text-neutral-100 placeholder-neutral-500 focus:outline-none"
        @input="$emit('update:modelValue', ($event.target as HTMLInputElement).value)"
        @blur="$emit('blur', $event)"
        @focus="$emit('focus', $event)"
        @keydown.enter="$emit('enter', $event)"
      />
      <div v-if="type === 'password'" class="pr-3">
        <button
          type="button"
          class="text-xs text-neutral-400 hover:text-neutral-200 transition-colors"
          @click="showPassword = !showPassword"
        >
          {{ showPassword ? '숨김' : '표시' }}
        </button>
      </div>
      <div v-else-if="$slots.suffix" class="pr-3 text-neutral-400">
        <slot name="suffix" />
      </div>
    </div>
    <p v-if="error" class="mt-1 text-xs text-accent-crimson">{{ error }}</p>
    <p v-else-if="hint" class="mt-1 text-xs text-neutral-500">{{ hint }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';

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

const computedType = computed(() => {
  if (props.type === 'password') {
    return showPassword.value ? 'text' : 'password';
  }
  return props.type;
});
</script>
