<template>
  <label class="inline-flex items-start gap-2.5 cursor-pointer select-none">
    <div class="relative flex items-center justify-center mt-0.5">
      <input
        type="checkbox"
        :checked="modelValue"
        :disabled="disabled"
        :required="required"
        class="peer sr-only"
        @change="$emit('update:modelValue', ($event.target as HTMLInputElement).checked)"
      />
      <div
        class="h-4 w-4 rounded border border-surface-border bg-surface-1 transition-all duration-150 peer-checked:border-brand-500 peer-checked:bg-brand-500 peer-focus-visible:ring-2 peer-focus-visible:ring-brand-500/50"
        :class="disabled ? 'opacity-50 cursor-not-allowed' : ''"
      />
      <svg
        class="pointer-events-none absolute h-3 w-3 text-[#04121a] opacity-0 transition-opacity peer-checked:opacity-100"
        viewBox="0 0 14 10"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
      >
        <path d="M1 5l4 4L13 1" />
      </svg>
    </div>
    <div class="text-xs text-neutral-300 leading-snug">
      <slot />
    </div>
  </label>
</template>

<script setup lang="ts">
interface Props {
  modelValue?: boolean;
  disabled?: boolean;
  required?: boolean;
}

withDefaults(defineProps<Props>(), {
  modelValue: false,
  disabled: false,
  required: false,
});

defineEmits<{
  (e: 'update:modelValue', val: boolean): void;
}>();
</script>
