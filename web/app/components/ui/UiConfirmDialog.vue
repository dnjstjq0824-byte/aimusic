<template>
  <UiModal
    :model-value="modelValue"
    :title="title"
    :description="description"
    max-width="sm"
    @close="$emit('update:modelValue', false)"
  >
    <p v-if="body" class="text-xs leading-relaxed text-text-secondary">{{ body }}</p>
    <template #footer>
      <UiButton variant="ghost" size="sm" @click="$emit('update:modelValue', false)">
        {{ cancelLabel }}
      </UiButton>
      <UiButton :variant="danger ? 'danger' : 'brand'" size="sm" :loading="loading" @click="$emit('confirm')">
        {{ confirmLabel }}
      </UiButton>
    </template>
  </UiModal>
</template>

<script setup lang="ts">
import UiModal from './UiModal.vue';
import UiButton from './UiButton.vue';

interface Props {
  modelValue: boolean;
  title: string;
  description?: string;
  body?: string;
  confirmLabel?: string;
  cancelLabel?: string;
  danger?: boolean;
  loading?: boolean;
}

withDefaults(defineProps<Props>(), {
  description: '',
  body: '',
  confirmLabel: '확인',
  cancelLabel: '취소',
  danger: false,
  loading: false,
});

defineEmits<{
  (e: 'update:modelValue', val: boolean): void;
  (e: 'confirm'): void;
}>();
</script>
