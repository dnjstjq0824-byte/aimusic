<template>
  <Teleport to="body">
    <Transition
      enter-active-class="ease-out duration-200"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="ease-in duration-150"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-if="modelValue"
        class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/75 backdrop-blur-md"
        @click="onBackdropClick"
      >
        <Transition
          enter-active-class="ease-out duration-200"
          enter-from-class="opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
          enter-to-class="opacity-100 translate-y-0 sm:scale-100"
          leave-active-class="ease-in duration-150"
          leave-from-class="opacity-100 translate-y-0 sm:scale-100"
          leave-to-class="opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
        >
          <div
            class="relative w-full rounded-2xl border border-surface-border bg-surface-1 shadow-2xl overflow-hidden flex flex-col max-h-[90vh]"
            :class="maxWidthClasses[maxWidth]"
            @click.stop
          >
            <!-- Header -->
            <div class="flex items-center justify-between border-b border-surface-border px-5 py-4 shrink-0">
              <div class="min-w-0 pr-4">
                <h3 class="text-base font-bold text-white truncate">{{ title }}</h3>
                <p v-if="description" class="mt-0.5 text-xs text-neutral-400 truncate">{{ description }}</p>
              </div>
              <button
                type="button"
                class="rounded-lg p-1.5 text-neutral-400 hover:text-white hover:bg-surface-2 transition-colors"
                title="닫기"
                @click="close"
              >
                ✕
              </button>
            </div>

            <!-- Body -->
            <div class="overflow-y-auto px-5 py-4 text-sm text-neutral-200 flex-1">
              <slot />
            </div>

            <!-- Footer -->
            <div v-if="$slots.footer" class="border-t border-surface-border px-5 py-3.5 bg-surface-0/50 shrink-0 flex items-center justify-end gap-2.5">
              <slot name="footer" />
            </div>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue';

interface Props {
  modelValue: boolean;
  title: string;
  description?: string;
  maxWidth?: 'sm' | 'md' | 'lg' | 'xl' | '2xl';
  closeOnBackdrop?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  description: '',
  maxWidth: 'md',
  closeOnBackdrop: true,
});

const emit = defineEmits<{
  (e: 'update:modelValue', val: boolean): void;
  (e: 'close'): void;
}>();

const maxWidthClasses = {
  sm: 'max-w-sm',
  md: 'max-w-md',
  lg: 'max-w-lg',
  xl: 'max-w-xl',
  '2xl': 'max-w-2xl',
};

function close() {
  emit('update:modelValue', false);
  emit('close');
}

function onBackdropClick() {
  if (props.closeOnBackdrop) {
    close();
  }
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && props.modelValue) {
    close();
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleKeydown);
});

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown);
});
</script>
