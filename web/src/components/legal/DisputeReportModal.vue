<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60" @click.self="emit('close')">
    <div class="w-full max-w-md rounded bg-neutral-900 p-5">
      <h2 class="mb-1 font-bold">저작권 분쟁 신고</h2>
      <p class="mb-3 truncate text-sm text-neutral-400">{{ trackTitle }}</p>
      <label class="mb-2 block text-sm">
        사유
        <select v-model="reason" class="mt-1 w-full rounded bg-neutral-800 p-2">
          <option value="AI_POLICY_VIOLATION">AI_POLICY_VIOLATION — AI 정책 위반</option>
          <option value="VOICE_CLONE">VOICE_CLONE — 무단 목소리 클론</option>
          <option value="UNAUTHORIZED_SAMPLE">UNAUTHORIZED_SAMPLE — 무단 샘플 사용</option>
          <option value="OTHER">OTHER — 기타</option>
        </select>
      </label>
      <label class="mb-3 block text-sm">
        상세 설명
        <textarea v-model="description" rows="3" class="mt-1 w-full rounded bg-neutral-800 p-2" placeholder="권리 관계를 구체적으로 적어 주세요" />
      </label>
      <p v-if="error" class="mb-2 text-sm text-red-400">{{ error }}</p>
      <div class="flex justify-end gap-2">
        <button class="rounded bg-neutral-800 px-3 py-1.5 text-sm" @click="emit('close')">취소</button>
        <button class="rounded bg-brand-600 px-3 py-1.5 text-sm font-bold disabled:opacity-50" :disabled="submitting" @click="submit">
          {{ submitting ? '신고 중…' : '신고하기' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { disputeApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';

const props = defineProps<{ trackId: string; trackTitle: string }>();
const emit = defineEmits<{ close: []; reported: [id: string] }>();

const reason = ref('AI_POLICY_VIOLATION');
const description = ref('');
const submitting = ref(false);
const error = ref<string | null>(null);

async function submit(): Promise<void> {
  submitting.value = true;
  error.value = null;
  try {
    const d = await disputeApi.report(props.trackId, reason.value, description.value || undefined);
    emit('reported', d.id);
    emit('close');
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    submitting.value = false;
  }
}
</script>
