<template>
  <!-- v3 flow (phase 3): the same frosted prompt surface as the home hero. -->
  <div class="space-y-2">
    <label :for="inputId" class="block text-sm text-text-secondary">
      {{ t('studio.create.v2.promptLabel') }}
    </label>
    <div class="prompt-box rounded-3xl px-5 pb-3 pt-4 transition-colors">
      <textarea
        :id="inputId"
        :value="modelValue"
        :placeholder="t('studio.create.v2.promptPh')"
        :aria-describedby="hintId"
        rows="4"
        maxlength="2000"
        required
        class="block w-full resize-none border-0 bg-transparent text-base leading-relaxed text-white placeholder:text-text-muted focus:outline-none focus-visible:shadow-none"
        @input="$emit('update:modelValue', ($event.target as HTMLTextAreaElement).value)"
      />
      <div class="mt-2 flex items-center justify-between gap-3 text-xs text-text-muted">
        <span :id="hintId">{{ t('studio.create.v2.promptHint') }}</span>
        <span class="shrink-0 tabular-nums">{{ modelValue.length }}/2000</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useId } from 'vue';

defineProps<{ modelValue: string }>();
defineEmits<{ (e: 'update:modelValue', val: string): void }>();

const { t } = useI18n();
const inputId = useId();
const hintId = `${inputId}-hint`;
</script>
