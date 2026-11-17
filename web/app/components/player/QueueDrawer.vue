<template>
  <div class="fixed inset-0 z-50 flex justify-end bg-black/60" @click.self="emit('close')">
    <aside
      ref="panelRef"
      role="dialog"
      aria-modal="true"
      :aria-label="t('player.queueTitle', { count: player.queue.length })"
      tabindex="-1"
      class="flex h-full w-full max-w-[90vw] flex-col border-l border-surface-border bg-surface-1 p-4 pb-safe sm:w-80"
    >
      <div class="mb-2 flex items-center justify-between">
        <h2 class="text-sm font-bold text-white">{{ t('player.queueTitle', { count: player.queue.length }) }}</h2>
        <div class="flex gap-2 text-xs">
          <UiButton variant="secondary" size="xs" @click="player.clearQueue()">{{ t('player.queueClear') }}</UiButton>
          <UiButton variant="ghost" size="xs" @click="emit('close')">{{ t('player.queueClose') }}</UiButton>
        </div>
      </div>
      <ol v-if="orderedQueue.length > 0" class="flex-1 space-y-1 overflow-y-auto">
        <li
          v-for="(track, i) in orderedQueue"
          :key="track.id"
          class="flex items-center gap-2 rounded-lg px-2 py-1.5 text-sm"
          :class="track.id === player.currentTrack?.id ? 'bg-brand-950/60 border border-brand-500/30' : 'bg-surface-2/60'"
          :aria-current="track.id === player.currentTrack?.id ? 'true' : undefined"
        >
          <span class="w-6 shrink-0 text-xs text-text-muted">{{ i + 1 }}</span>
          <span class="min-w-0 flex-1 truncate text-xs text-text-primary">{{ track.title }}</span>
          <button
            type="button"
            class="rounded p-1 text-xs text-text-secondary hover:text-white"
            :aria-label="t('player.queuePlay', { title: track.title })"
            @click="playAt(i)"
          >
            ▶
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
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue';
import { usePlayerStore } from '@/stores/playerStore';
import UiButton from '@/components/ui/UiButton.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';

const emit = defineEmits<{ close: [] }>();
const player = usePlayerStore();
const { t } = useI18n();
const panelRef = ref<HTMLElement | null>(null);

const orderedQueue = computed(() => player.order.map((qi) => player.queue[qi]).filter((t) => t !== undefined));

function playAt(orderIndex: number): void {
  void player.playAt(orderIndex);
}

function onKeydown(e: KeyboardEvent): void {
  if (e.key === 'Escape') emit('close');
}

onMounted(() => {
  window.addEventListener('keydown', onKeydown);
  void nextTick(() => panelRef.value?.focus());
});

onUnmounted(() => {
  window.removeEventListener('keydown', onKeydown);
});
</script>
