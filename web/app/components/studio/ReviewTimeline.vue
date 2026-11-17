<template>
  <ol class="space-y-0">
    <li v-for="(item, i) in items" :key="`${item.label}-${i}`" class="relative flex gap-3 pb-5 last:pb-0">
      <span
        v-if="i < items.length - 1"
        class="absolute left-[7px] top-5 h-[calc(100%-1rem)] w-px bg-surface-border"
        aria-hidden="true"
      />
      <span
        class="mt-1 h-[15px] w-[15px] shrink-0 rounded-full border-2"
        :class="dotClass(item.state)"
        aria-hidden="true"
      />
      <div class="min-w-0 flex-1">
        <div class="text-xs font-bold" :class="item.state === 'todo' ? 'text-text-muted' : 'text-white'">
          {{ item.label }}
        </div>
        <div v-if="item.at" class="mt-0.5 text-caption text-text-secondary">{{ item.at }}</div>
        <div v-if="item.note" class="mt-0.5 text-caption text-text-secondary">{{ item.note }}</div>
      </div>
    </li>
  </ol>
</template>

<script setup lang="ts">
export interface TimelineItem {
  label: string;
  at?: string;
  note?: string;
  state: 'done' | 'now' | 'todo';
}

defineProps<{ items: TimelineItem[] }>();

function dotClass(state: TimelineItem['state']): string {
  if (state === 'done') return 'border-success bg-success';
  if (state === 'now') return 'border-brand-400 bg-brand-500';
  return 'border-surface-border bg-surface-2';
}
</script>
