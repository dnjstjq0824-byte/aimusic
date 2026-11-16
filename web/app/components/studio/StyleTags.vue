<template>
  <fieldset class="space-y-2">
    <legend class="text-xs font-bold text-text-primary">{{ t('studio.create.v2.styleLabel') }}</legend>
    <p class="text-caption text-text-muted">{{ t('studio.create.v2.styleHint', { n: MAX_STYLE_TAGS }) }}</p>
    <div v-for="group in STYLE_TAG_GROUPS" :key="group.id" class="flex flex-wrap gap-1.5">
      <button
        v-for="key in group.keys"
        :key="key"
        type="button"
        :aria-pressed="modelValue.includes(key)"
        class="rounded-full border px-2.5 py-1 text-xs font-medium transition-all"
        :class="
          modelValue.includes(key)
            ? 'border-brand-500 bg-brand-500/15 text-brand-300'
            : 'border-surface-border text-text-secondary hover:text-white'
        "
        @click="$emit('update:modelValue', toggleTag(modelValue, key))"
      >
        {{ t(`tags.${key}`) }}
      </button>
    </div>
  </fieldset>
</template>

<script setup lang="ts">
import { MAX_STYLE_TAGS, STYLE_TAG_GROUPS, toggleTag } from './styleTagOptions';

defineProps<{ modelValue: string[] }>();
defineEmits<{ (e: 'update:modelValue', val: string[]): void }>();

// tags.* keys ship in all 5 locales (R5).
const { t } = useI18n();
</script>
