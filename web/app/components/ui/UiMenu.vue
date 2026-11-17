<template>
  <div class="relative inline-block">
    <button
      ref="buttonRef"
      type="button"
      class="flex h-9 w-9 items-center justify-center rounded-lg text-text-secondary transition-colors hover:bg-surface-2 hover:text-text-primary"
      aria-haspopup="menu"
      :aria-expanded="open"
      :aria-label="label || $t('common.actions.more')"
      @click="open = !open"
    >
      <slot name="trigger">
        <svg class="h-5 w-5" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
          <circle cx="5" cy="12" r="1.8" />
          <circle cx="12" cy="12" r="1.8" />
          <circle cx="19" cy="12" r="1.8" />
        </svg>
      </slot>
    </button>
    <div
      v-if="open"
      role="menu"
      :aria-label="label || $t('common.actions.more')"
      class="absolute z-50 mt-1.5 w-52 overflow-hidden rounded-xl border border-surface-border bg-surface-2 py-1.5 shadow-2xl"
      :class="align === 'left' ? 'left-0' : 'right-0'"
    >
      <button
        v-for="(item, i) in items"
        :key="i"
        type="button"
        role="menuitem"
        :disabled="item.disabled"
        class="flex w-full items-center gap-2 px-3 py-2 text-left text-sm transition-colors disabled:cursor-not-allowed disabled:opacity-40"
        :class="item.danger ? 'text-accent-crimson hover:bg-surface-3' : 'text-text-secondary hover:bg-surface-3 hover:text-text-primary'"
        @click="pick(i)"
      >
        {{ item.label }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue';

// ⋯ overflow menu (track cards/rows — UI §4.1).
export interface MenuItem {
  label: string;
  danger?: boolean;
  disabled?: boolean;
}

withDefaults(defineProps<{ items: MenuItem[]; label?: string; align?: 'left' | 'right' }>(), {
  label: '',
  align: 'right',
});
const emit = defineEmits<{ (e: 'select', index: number): void }>();

const open = ref(false);
const buttonRef = ref<HTMLElement | null>(null);

function pick(i: number): void {
  open.value = false;
  emit('select', i);
}

function onPointerDown(e: Event): void {
  if (open.value && buttonRef.value && !buttonRef.value.parentElement?.contains(e.target as Node)) {
    open.value = false;
  }
}

function onKeydown(e: KeyboardEvent): void {
  if (e.key === 'Escape') open.value = false;
}

onMounted(() => {
  document.addEventListener('pointerdown', onPointerDown);
  window.addEventListener('keydown', onKeydown);
});

onUnmounted(() => {
  document.removeEventListener('pointerdown', onPointerDown);
  window.removeEventListener('keydown', onKeydown);
});
</script>
