<template>
  <div class="space-y-1">
    <UiTextarea
      :model-value="modelValue"
      :label="t('studio.create.v2.lyricsLabel')"
      :placeholder="t('studio.create.v2.lyricsPh')"
      :rows="5"
      :maxlength="10000"
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
