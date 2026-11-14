<template>
  <UiModal
    :model-value="modelValue"
    title="플레이리스트에 담기"
    :description="trackTitle ? `「${trackTitle}」을(를) 보관할 플레이리스트를 선택하세요.` : '보관할 플레이리스트를 선택하세요.'"
    @close="$emit('update:modelValue', false)"
  >
    <div v-if="playlists.length === 0 && !creating" class="py-4 text-center">
      <p class="mb-3 text-xs text-neutral-400">생성된 플레이리스트가 없습니다.</p>
      <UiButton variant="brand" size="sm" @click="creating = true">새 플레이리스트 만들기</UiButton>
    </div>
    <div v-else class="space-y-2">
      <button
        v-for="p in playlists"
        :key="p.id"
        type="button"
        class="flex w-full items-center justify-between rounded-xl border border-surface-border bg-surface-2 p-3 text-left text-xs transition-colors hover:bg-surface-3"
        @click="$emit('select', p.id)"
      >
        <span class="min-w-0">
          <span class="block truncate font-bold text-white">{{ p.title }}</span>
          <span class="mt-0.5 block text-[11px] text-neutral-500">{{ p.isPublic ? '공개' : '비공개' }}</span>
        </span>
        <span class="shrink-0 text-neutral-400">담기 →</span>
      </button>
      <button
        v-if="!creating"
        type="button"
        class="flex w-full items-center justify-center gap-1.5 rounded-xl border border-dashed border-surface-border p-3 text-xs font-semibold text-neutral-400 transition-colors hover:border-brand-500/50 hover:text-brand-300"
        @click="creating = true"
      >
        + 새로 만들기
      </button>
      <form v-else class="flex items-center gap-2 rounded-xl border border-brand-500/40 bg-brand-950/20 p-2" @submit.prevent="submitCreate">
        <UiInput
          v-model="newTitle"
          placeholder="새 플레이리스트 제목"
          :maxlength="80"
          class="flex-1"
        />
        <UiButton variant="brand" size="sm" type="submit" :loading="createLoading" :disabled="newTitle.trim() === ''">
          생성
        </UiButton>
      </form>
    </div>
  </UiModal>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';
import UiModal from './UiModal.vue';
import UiButton from './UiButton.vue';
import UiInput from './UiInput.vue';
import type { Playlist } from '@/api/types';

interface Props {
  modelValue: boolean;
  trackTitle?: string;
  playlists: Playlist[];
  createLoading?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  trackTitle: '',
  createLoading: false,
});

const emit = defineEmits<{
  (e: 'update:modelValue', val: boolean): void;
  (e: 'select', playlistId: string): void;
  (e: 'create', title: string): void;
}>();

const creating = ref(false);
const newTitle = ref('');

watch(
  () => props.modelValue,
  (open) => {
    if (!open) {
      creating.value = false;
      newTitle.value = '';
    }
  },
);

function submitCreate(): void {
  const title = newTitle.value.trim();
  if (title === '') return;
  emit('create', title);
}
</script>
