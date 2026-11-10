<template>
  <div
    class="pointer-events-none fixed left-4 right-4 top-16 z-50 flex flex-col gap-2.5 sm:left-auto sm:right-6 sm:top-20 sm:max-w-sm"
    aria-live="polite"
    aria-atomic="false"
  >
    <TransitionGroup
      enter-active-class="transform ease-out duration-300 transition"
      enter-from-class="-translate-y-2 opacity-0 sm:translate-y-0 sm:translate-x-4"
      enter-to-class="translate-y-0 opacity-100 sm:translate-x-0"
      leave-active-class="transition ease-in duration-200"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0 scale-95"
    >
      <div
        v-for="t in toasts"
        :key="t.id"
        :role="t.type === 'error' ? 'alert' : 'status'"
        class="pointer-events-auto flex items-start gap-3 rounded-xl border p-3.5 shadow-2xl backdrop-blur-xl"
        :class="bgClass(t.type)"
      >
        <span class="mt-0.5 shrink-0" aria-hidden="true" v-html="iconSvg(t.type)" />
        <div class="min-w-0 flex-1">
          <div v-if="t.title" class="text-xs font-bold text-white">{{ t.title }}</div>
          <div class="text-xs text-neutral-200 leading-relaxed">{{ t.message }}</div>
        </div>
        <button
          type="button"
          class="shrink-0 rounded p-0.5 text-neutral-400 hover:text-white transition-colors"
          aria-label="알림 닫기"
          @click="dismiss(t.id)"
        >
          <svg class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
            <path d="M18 6 6 18M6 6l12 12" />
          </svg>
        </button>
      </div>
    </TransitionGroup>
  </div>
</template>

<script setup lang="ts">
import { useToast } from '@/composables/useToast';

const { toasts, dismiss } = useToast();

function bgClass(type: string) {
  switch (type) {
    case 'success':
      return 'border-accent-emerald/40 bg-surface-1/95 text-accent-emerald';
    case 'error':
      return 'border-accent-crimson/40 bg-surface-1/95 text-accent-crimson';
    case 'warning':
      return 'border-accent-amber/40 bg-surface-1/95 text-accent-amber';
    default:
      return 'border-brand-500/40 bg-surface-1/95 text-brand-400';
  }
}

const ICON_PATHS: Record<string, string> = {
  success: '<path d="M20 6 9 17l-5-5"/>',
  error: '<circle cx="12" cy="12" r="9"/><path d="M12 8v4M12 16h.01"/>',
  warning: '<path d="M12 3 2 20h20L12 3z"/><path d="M12 10v4M12 17.5h.01"/>',
  info: '<circle cx="12" cy="12" r="9"/><path d="M12 11v5M12 7.5h.01"/>',
};

function iconSvg(type: string): string {
  const path = ICON_PATHS[type] ?? ICON_PATHS.info;
  return `<svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${path}</svg>`;
}
</script>
