<template>
  <div class="pointer-events-none fixed bottom-28 right-4 z-50 flex max-w-sm flex-col gap-2.5 sm:bottom-24 sm:right-6">
    <TransitionGroup
      enter-active-class="transform ease-out duration-300 transition"
      enter-from-class="translate-y-2 opacity-0 sm:translate-y-0 sm:translate-x-4"
      enter-to-class="translate-y-0 opacity-100 sm:translate-x-0"
      leave-active-class="transition ease-in duration-200"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0 scale-95"
    >
      <div
        v-for="t in toasts"
        :key="t.id"
        class="pointer-events-auto flex items-start gap-3 rounded-xl border p-3.5 shadow-2xl backdrop-blur-xl"
        :class="bgClass(t.type)"
      >
        <span class="mt-0.5 text-base shrink-0">{{ iconOf(t.type) }}</span>
        <div class="min-w-0 flex-1">
          <div v-if="t.title" class="text-xs font-bold text-white">{{ t.title }}</div>
          <div class="text-xs text-neutral-200 leading-relaxed">{{ t.message }}</div>
        </div>
        <button
          class="shrink-0 text-neutral-400 hover:text-white transition-colors"
          title="닫기"
          @click="dismiss(t.id)"
        >
          ✕
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

function iconOf(type: string) {
  switch (type) {
    case 'success':
      return '✓';
    case 'error':
      return '✕';
    case 'warning':
      return '⚠';
    default:
      return 'ℹ';
  }
}
</script>
