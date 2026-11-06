<template>
  <div class="fixed inset-0 z-40 flex justify-end bg-black/60" @click.self="emit('close')">
    <aside class="flex h-full w-80 flex-col bg-neutral-900 p-4">
      <div class="mb-2 flex items-center justify-between">
        <h2 class="font-bold">Queue ({{ player.queue.length }})</h2>
        <div class="flex gap-2 text-xs">
          <button class="rounded bg-neutral-800 px-2 py-1" @click="player.clearQueue()">Clear</button>
          <button class="rounded bg-neutral-800 px-2 py-1" @click="emit('close')">Close</button>
        </div>
      </div>
      <ol class="flex-1 space-y-1 overflow-y-auto">
        <li
          v-for="(t, i) in orderedQueue"
          :key="t.id"
          class="flex items-center gap-2 rounded px-2 py-1 text-sm"
          :class="t.id === player.currentTrack?.id ? 'bg-brand-600/30' : 'bg-neutral-800/50'"
        >
          <span class="w-6 text-neutral-500">{{ i + 1 }}</span>
          <span class="min-w-0 flex-1 truncate">{{ t.title }}</span>
          <button class="text-xs text-neutral-400 hover:text-white" @click="playAt(i)">▶</button>
          <button class="text-xs text-neutral-400 hover:text-red-400" @click="player.removeFromQueue(t.id)">✕</button>
        </li>
      </ol>
      <p v-if="orderedQueue.length === 0" class="py-8 text-center text-sm text-neutral-500">큐가 비어 있습니다.</p>
    </aside>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { usePlayerStore } from '@/stores/playerStore';

const emit = defineEmits<{ close: [] }>();
const player = usePlayerStore();

const orderedQueue = computed(() => player.order.map((qi) => player.queue[qi]).filter((t) => t !== undefined));

function playAt(orderIndex: number): void {
  player.orderPos = orderIndex;
  void player.playCurrent();
}
</script>
