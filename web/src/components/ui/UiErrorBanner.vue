<template>
  <div
    v-if="message"
    role="alert"
    class="flex items-start gap-3 rounded-xl border border-accent-crimson/40 bg-red-950/40 p-4 text-xs leading-relaxed text-red-200"
  >
    <svg class="mt-0.5 h-4 w-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
      <circle cx="12" cy="12" r="9" />
      <path d="M12 8v4M12 16h.01" />
    </svg>
    <div class="min-w-0 flex-1">
      <div v-if="title" class="font-bold text-white">{{ title }}</div>
      <div>{{ message }}</div>
    </div>
    <button
      v-if="retryLabel"
      type="button"
      class="shrink-0 rounded-lg border border-red-700/60 px-2.5 py-1 font-semibold text-red-200 transition-colors hover:bg-red-900/50"
      @click="$emit('retry')"
    >
      {{ retryLabel }}
    </button>
  </div>
</template>

<script setup lang="ts">
interface Props {
  message?: string | null;
  title?: string;
  retryLabel?: string;
}

withDefaults(defineProps<Props>(), {
  message: null,
  title: '',
  retryLabel: '',
});

defineEmits<{
  (e: 'retry'): void;
}>();
</script>
