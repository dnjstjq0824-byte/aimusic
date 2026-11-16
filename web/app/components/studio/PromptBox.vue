<template>
  <div class="space-y-1">
    <UiTextarea
      :model-value="modelValue"
      :label="t('studio.create.v2.promptLabel')"
      :placeholder="t('studio.create.v2.promptPh')"
      :hint="t('studio.create.v2.promptHint')"
      :rows="4"
      :maxlength="2000"
      required
      @update:model-value="$emit('update:modelValue', $event)"
    />
  </div>
</template>

<script setup lang="ts">
import UiTextarea from '@/components/ui/UiTextarea.vue';

defineProps<{ modelValue: string }>();
defineEmits<{ (e: 'update:modelValue', val: string): void }>();

const { t } = useI18n();
</script>
