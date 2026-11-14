<template>
  <div class="fixed inset-0 z-50 flex justify-end bg-black/60" @click.self="emit('close')">
    <aside
      ref="panelRef"
      role="dialog"
      aria-modal="true"
      aria-label="재생 대기열"
      tabindex="-1"
      class="flex h-full w-full max-w-[90vw] flex-col border-l border-surface-border bg-surface-1 p-4 pb-safe sm:w-80"
    >
      <div class="mb-2 flex items-center justify-between">
        <h2 class="text-sm font-bold text-white">재생 대기열 ({{ player.queue.length }})</h2>
        <div class="flex gap-2 text-xs">
          <UiButton variant="secondary" size="xs" @click="player.clearQueue()">비우기</UiButton>
          <UiButton variant="ghost" size="xs" @click="emit('close')">닫기</UiButton>
        </div>
      </div>
      <ol v-if="orderedQueue.length > 0" class="flex-1 space-y-1 overflow-y-auto">
        <li
          v-for="(t, i) in orderedQueue"
          :key="t.id"
          class="flex items-center gap-2 rounded-lg px-2 py-1.5 text-sm"
          :class="t.id === player.currentTrack?.id ? 'bg-brand-950/60 border border-brand-500/30' : 'bg-surface-2/60'"
          :aria-current="t.id === player.currentTrack?.id ? 'true' : undefined"
        >
          <span class="w-6 shrink-0 text-xs text-neutral-500">{{ i + 1 }}</span>
          <span class="min-w-0 flex-1 truncate text-xs text-neutral-200">{{ t.title }}</span>
          <button
            type="button"
            class="rounded p-1 text-xs text-neutral-400 hover:text-white"
            :aria-label="`「${t.title}」 재생`"
            @click="playAt(i)"
          >
            ▶
          </button>
          <button
            type="button"
            class="rounded p-1 text-xs text-neutral-400 hover:text-accent-crimson"
            :aria-label="`「${t.title}」 대기열에서 제거`"
            @click="player.removeFromQueue(t.id)"
          >
            ✕
          </button>
        </li>
      </ol>
      <UiEmptyState
        v-else
        title="대기열이 비어 있습니다"
        description="음원 목록에서 곡을 담으면 여기에 표시됩니다."
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
