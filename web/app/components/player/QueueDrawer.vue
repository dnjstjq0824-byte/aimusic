<template>
  <div class="fixed inset-0 z-50 flex justify-end bg-black/60" @click.self="emit('close')">
    <aside
      ref="panelRef"
      role="dialog"
      aria-modal="true"
      :aria-label="t('player.queueTitle', { count: player.queue.length })"
      tabindex="-1"
      class="flex h-full w-full max-w-[90vw] flex-col border-l border-surface-border bg-surface-1 p-4 pb-safe sm:w-96"
      @keydown.tab="trapTabIn(panelRef, $event)"
    >
      <div class="mb-2 flex items-center justify-between">
        <h2 class="text-sm font-bold text-white">{{ t('player.queueTitle', { count: player.queue.length }) }}</h2>
        <div class="flex gap-2 text-xs">
          <UiButton variant="secondary" size="xs" :disabled="player.queue.length === 0" @click="player.clearQueue()">
            {{ t('player.queueClear') }}
          </UiButton>
          <UiButton variant="ghost" size="xs" @click="emit('close')">{{ t('player.queueClose') }}</UiButton>
        </div>
      </div>
      <p v-if="player.queue.length > 1" class="mb-2 text-caption text-text-muted">{{ t('player.queueHint') }}</p>
      <ol v-if="player.orderedQueue.length > 0" class="flex-1 space-y-1 overflow-y-auto">
        <li
          v-for="(track, i) in player.orderedQueue"
          :key="track.id"
          class="flex items-center gap-1.5 rounded-lg border px-2 py-1.5 text-sm"
          :class="[
            i === player.orderPos ? 'border-brand-500/30 bg-brand-950/60' : 'border-transparent bg-surface-2/60',
            dragOver === i && dragFrom !== null && dragFrom !== i ? '!border-brand-500/60' : '',
          ]"
          @dragover.prevent="dragOver = i"
          @drop.prevent="onDrop(i)"
        >
          <span
            class="cursor-grab select-none text-text-muted active:cursor-grabbing"
            draggable="true"
            :aria-label="t('playlist.dragHandle', { title: track.title })"
            @dragstart="dragFrom = i"
            @dragend="dragFrom = null; dragOver = null"
          >
            <svg class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
              <circle cx="9" cy="6" r="1.4" /><circle cx="15" cy="6" r="1.4" />
              <circle cx="9" cy="12" r="1.4" /><circle cx="15" cy="12" r="1.4" />
              <circle cx="9" cy="18" r="1.4" /><circle cx="15" cy="18" r="1.4" />
            </svg>
          </span>
          <span class="w-5 shrink-0 text-center text-xs text-text-muted">
            <span v-if="i === player.orderPos" class="text-brand-400" :aria-label="t('player.nowPlaying')">▶</span>
            <span v-else>{{ i + 1 }}</span>
          </span>
          <button
            type="button"
            class="min-w-0 flex-1 truncate text-left text-xs text-text-primary hover:text-white"
            :aria-label="t('player.queuePlay', { title: track.title })"
            :aria-current="i === player.orderPos ? 'true' : undefined"
            @click="playAt(i)"
          >
            {{ track.title }}
            <span v-if="track.creatorName" class="block truncate text-caption text-text-muted">{{ track.creatorName }}</span>
          </button>
          <button
            type="button"
            class="rounded p-1 text-text-secondary hover:text-white disabled:opacity-30"
            :disabled="i === 0"
            :aria-label="t('playlist.trackUpAria', { title: track.title })"
            @click="player.moveInQueue(i, i - 1)"
          >
            <svg class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="m18 15-6-6-6 6" /></svg>
          </button>
          <button
            type="button"
            class="rounded p-1 text-text-secondary hover:text-white disabled:opacity-30"
            :disabled="i === player.orderedQueue.length - 1"
            :aria-label="t('playlist.trackDownAria', { title: track.title })"
            @click="player.moveInQueue(i, i + 1)"
          >
            <svg class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="m6 9 6 6 6-6" /></svg>
          </button>
          <button
            type="button"
            class="rounded p-1 text-xs text-text-secondary hover:text-accent-crimson"
            :aria-label="t('player.queueRemove', { title: track.title })"
            @click="player.removeFromQueue(track.id)"
          >
            ✕
          </button>
        </li>
      </ol>
      <UiEmptyState
        v-else
        :title="t('player.queueEmptyTitle')"
        :description="t('player.queueEmptyHint')"
      />
    </aside>
  </div>
</template>

<script setup lang="ts">
import { nextTick, onMounted, onUnmounted, ref } from 'vue';
import { lockBodyScroll, trapTabIn, unlockBodyScroll } from '@/composables/useDialog';
import { usePlayerStore } from '@/stores/playerStore';
import UiButton from '@/components/ui/UiButton.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';

const emit = defineEmits<{ close: [] }>();
const player = usePlayerStore();
const { t } = useI18n();
const panelRef = ref<HTMLElement | null>(null);
const dragFrom = ref<number | null>(null);
const dragOver = ref<number | null>(null);

function playAt(orderIndex: number): void {
  void player.playAt(orderIndex);
}

function onDrop(index: number): void {
  const from = dragFrom.value;
  dragFrom.value = null;
  dragOver.value = null;
  if (from !== null) player.moveInQueue(from, index);
}

function onKeydown(e: KeyboardEvent): void {
  if (e.key === 'Escape') emit('close');
}

let lastFocused: HTMLElement | null = null;

onMounted(() => {
  window.addEventListener('keydown', onKeydown);
  lastFocused = document.activeElement as HTMLElement | null;
  lockBodyScroll();
  void nextTick(() => panelRef.value?.focus());
});

onUnmounted(() => {
  window.removeEventListener('keydown', onKeydown);
  unlockBodyScroll();
  lastFocused?.focus?.();
  lastFocused = null;
});
</script>
