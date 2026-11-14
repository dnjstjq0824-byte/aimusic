<template>
  <Teleport to="body">
    <div v-if="modelValue" class="fixed inset-0 z-50 sm:flex sm:items-center sm:justify-center" role="dialog" aria-modal="true" :aria-label="title">
      <div class="absolute inset-0 bg-black/70" @click="closeOnBackdrop && close()" />
      <div
        class="absolute inset-x-0 bottom-0 max-h-[88vh] overflow-y-auto rounded-t-2xl border-t border-surface-border bg-surface-1 p-5 pb-safe sm:relative sm:inset-auto sm:w-full sm:max-w-lg sm:rounded-2xl sm:border sm:p-6"
      >
        <div class="mx-auto mb-4 h-1 w-10 rounded-full bg-surface-3 sm:hidden" aria-hidden="true" />
        <div class="mb-4 flex items-start justify-between gap-3">
          <h2 class="font-display text-h3 font-bold text-text-primary">{{ title }}</h2>
          <button
            type="button"
            class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg text-text-secondary hover:bg-surface-2 hover:text-text-primary"
            :aria-label="$t('common.actions.close')"
            @click="close()"
          >
            <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
              <path d="M6 6l12 12M18 6 6 18" />
            </svg>
          </button>
        </div>
        <slot />
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, watch } from 'vue';

// Bottom sheet on mobile, centered dialog on desktop (UI §5).
const props = withDefaults(
  defineProps<{ modelValue: boolean; title: string; closeOnBackdrop?: boolean }>(),
  { closeOnBackdrop: true },
);
const emit = defineEmits<{ (e: 'update:modelValue', open: boolean): void; (e: 'close'): void }>();

function close(): void {
  emit('update:modelValue', false);
  emit('close');
}

function onKeydown(e: KeyboardEvent): void {
  if (e.key === 'Escape' && props.modelValue) close();
}

watch(
  () => props.modelValue,
  (open) => {
    if (typeof document === 'undefined') return;
    document.body.style.overflow = open ? 'hidden' : '';
  },
);

onMounted(() => window.addEventListener('keydown', onKeydown));
onUnmounted(() => {
  window.removeEventListener('keydown', onKeydown);
  if (typeof document !== 'undefined') document.body.style.overflow = '';
});
</script>
