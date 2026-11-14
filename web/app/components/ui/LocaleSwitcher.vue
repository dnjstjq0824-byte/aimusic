<template>
  <div class="relative inline-block">
    <button
      ref="buttonRef"
      type="button"
      class="flex min-h-11 min-w-11 items-center justify-center rounded-lg border border-surface-border bg-surface-1 px-2.5 text-text-secondary transition-colors hover:text-text-primary"
      aria-haspopup="listbox"
      :aria-expanded="open"
      aria-label="표시 언어"
      @click="open = !open"
    >
      <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
        <circle cx="12" cy="12" r="9" />
        <path d="M3 12h18M12 3c2.5 2.6 3.8 5.7 3.8 9S14.5 18.4 12 21c-2.5-2.6-3.8-5.7-3.8-9S9.5 5.6 12 3z" />
      </svg>
      <span class="text-xs font-semibold">{{ currentName }}</span>
    </button>
    <ul
      v-if="open"
      role="listbox"
      aria-label="언어 선택"
      class="absolute right-0 z-50 mt-2 w-44 overflow-hidden rounded-xl border border-surface-border bg-surface-2 py-1.5 shadow-2xl"
    >
      <li v-for="lang in LANGS" :key="lang.code" role="option" :aria-selected="lang.code === modelValue">
        <button
          type="button"
          class="flex w-full items-center justify-between px-3 py-2 text-left text-sm transition-colors hover:bg-surface-3"
          :class="lang.code === modelValue ? 'font-bold text-text-primary' : 'text-text-secondary'"
          @click="select(lang.code)"
        >
          <span>{{ lang.name }}</span>
          <span class="flex items-center gap-1.5">
            <span v-if="lang.beta" class="rounded bg-surface-3 px-1.5 py-0.5 text-caption text-text-muted">Beta</span>
            <span v-if="lang.code === modelValue" class="text-brand-400" aria-hidden="true">✓</span>
          </span>
        </button>
      </li>
    </ul>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue';

// Locale switcher (v2 §4.5): globe + self-name, no flags. Wired to @nuxtjs/i18n in WI-R2.
const LANGS = [
  { code: 'ko', name: '한국어', beta: false },
  { code: 'en', name: 'English', beta: false },
  { code: 'ja', name: '日本語', beta: true },
  { code: 'zh-CN', name: '简体中文', beta: true },
  { code: 'th', name: 'ไทย', beta: true },
] as const;

const props = withDefaults(defineProps<{ modelValue?: string }>(), { modelValue: 'ko' });
const emit = defineEmits<{ (e: 'update:modelValue', code: string): void }>();

const open = ref(false);
const buttonRef = ref<HTMLElement | null>(null);
const currentName = computed(() => LANGS.find((l) => l.code === props.modelValue)?.name ?? '한국어');

function select(code: string): void {
  open.value = false;
  emit('update:modelValue', code);
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
