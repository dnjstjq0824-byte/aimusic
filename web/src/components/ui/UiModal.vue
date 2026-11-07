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
            ref="panelRef"
            role="dialog"
            aria-modal="true"
            :aria-labelledby="titleId"
            :aria-describedby="description ? descId : undefined"
            tabindex="-1"
            class="relative w-full rounded-2xl border border-surface-border bg-surface-1 shadow-2xl overflow-hidden flex flex-col max-h-[90vh]"
            :class="maxWidthClasses[maxWidth]"
            @click.stop
            @keydown.tab="trapTab"
          >
            <!-- Header -->
            <div class="flex items-center justify-between border-b border-surface-border px-5 py-4 shrink-0">
              <div class="min-w-0 pr-4">
                <h3 :id="titleId" class="text-base font-bold text-white truncate">{{ title }}</h3>
                <p v-if="description" :id="descId" class="mt-0.5 text-xs text-neutral-400 truncate">{{ description }}</p>
              </div>
              <button
                type="button"
                class="rounded-lg p-1.5 text-neutral-400 hover:text-white hover:bg-surface-2 transition-colors"
                aria-label="닫기"
                @click="close"
              >
                <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
                  <path d="M18 6 6 18M6 6l12 12" />
                </svg>
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
import { nextTick, onMounted, onUnmounted, ref, useId, watch } from 'vue';

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

const titleId = `modal-title-${useId()}`;
const descId = `modal-desc-${useId()}`;
const panelRef = ref<HTMLElement | null>(null);
let lastFocused: HTMLElement | null = null;

// Process-wide scroll-lock counter shared by nested modals via the DOM.
function scrollLocks(): number {
  return Number(document.body.dataset.modalLocks ?? '0');
}

function lockBodyScroll(): void {
  const next = scrollLocks() + 1;
  document.body.dataset.modalLocks = String(next);
  if (next === 1) {
    document.body.style.overflow = 'hidden';
  }
}

function unlockBodyScroll(): void {
  const next = Math.max(0, scrollLocks() - 1);
  document.body.dataset.modalLocks = String(next);
  if (next === 0) {
    document.body.style.overflow = '';
  }
}

function focusables(): HTMLElement[] {
  if (!panelRef.value) return [];
  const nodes = panelRef.value.querySelectorAll<HTMLElement>(
    'a[href], button:not([disabled]), textarea, input, select, [tabindex]:not([tabindex="-1"])',
  );
  return Array.from(nodes).filter((el) => !el.hasAttribute('disabled') && el.offsetParent !== null);
}

function trapTab(e: KeyboardEvent): void {
  const items = focusables();
  if (items.length === 0) {
    e.preventDefault();
    panelRef.value?.focus();
    return;
  }
  const first = items[0] as HTMLElement;
  const last = items[items.length - 1] as HTMLElement;
  if (e.shiftKey && document.activeElement === first) {
    e.preventDefault();
    last.focus();
  } else if (!e.shiftKey && document.activeElement === last) {
    e.preventDefault();
    first.focus();
  }
}

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

watch(
  () => props.modelValue,
  (open) => {
    if (open) {
      lastFocused = document.activeElement as HTMLElement | null;
      lockBodyScroll();
      void nextTick(() => {
        const items = focusables();
        if (items.length > 0) {
          (items[0] as HTMLElement).focus();
        } else {
          panelRef.value?.focus();
        }
      });
    } else {
      unlockBodyScroll();
      lastFocused?.focus?.();
      lastFocused = null;
    }
  },
  { immediate: true },
);

onMounted(() => {
  window.addEventListener('keydown', handleKeydown);
});

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown);
  if (props.modelValue) {
    unlockBodyScroll();
  }
});
</script>
