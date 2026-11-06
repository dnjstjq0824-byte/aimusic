<template>
  <div>
    <h1 class="mb-4 text-xl font-bold">Dispute Center</h1>
    <p class="mb-4 text-sm text-neutral-400">
      분쟁 신고는 Home의 각 음원 옆 신고 버튼으로 접수합니다. 아래는 내 계정의 스트라이크/차단 내역입니다.
    </p>
    <p v-if="error" class="mb-2 text-sm text-red-400">{{ error }}</p>
    <div v-if="loading" class="text-sm text-neutral-400">불러오는 중…</div>
    <ul v-else class="space-y-2">
      <li v-for="b in blocks" :key="b.id" class="rounded bg-neutral-900 p-3 text-sm">
        <div class="font-semibold">Strike #{{ b.strikeNumber }} — {{ b.reason }}</div>
        <div class="text-xs text-neutral-400">
          {{ b.createdAt }}<span v-if="b.expiresAt"> · 만료 {{ b.expiresAt }}</span>
        </div>
      </li>
    </ul>
    <p v-if="!loading && blocks.length === 0" class="text-sm text-neutral-500">차단 내역이 없습니다.</p>
    <div class="mt-6 rounded border border-neutral-800 p-3">
      <h2 class="mb-2 font-bold">파트너 이의제기 (Counter-claim)</h2>
      <p class="mb-2 text-xs text-neutral-400">분쟁 ID와 소명 내용을 입력하면 POST /disputes/:id/counter 로 전송됩니다.</p>
      <div class="flex flex-col gap-2 sm:flex-row">
        <input v-model="counterId" placeholder="dispute id" class="flex-1 rounded bg-neutral-800 p-2 text-sm" />
        <input v-model="counterNote" placeholder="소명 내용" class="flex-[2] rounded bg-neutral-800 p-2 text-sm" />
        <button class="rounded bg-brand-600 px-3 py-1.5 text-sm font-bold disabled:opacity-50" :disabled="counterBusy" @click="counter">
          {{ counterBusy ? '전송 중…' : '제출' }}
        </button>
      </div>
      <p v-if="counterError" class="mt-2 text-sm text-red-400">{{ counterError }}</p>
      <p v-if="counterDone" class="mt-2 text-sm text-emerald-400">이의제기가 접수되었습니다.</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { disputeApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import type { UserBlock } from '@/api/types';

const blocks = ref<UserBlock[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);

const counterId = ref('');
const counterNote = ref('');
const counterBusy = ref(false);
const counterError = ref<string | null>(null);
const counterDone = ref(false);

async function load(): Promise<void> {
  loading.value = true;
  error.value = null;
  try {
    blocks.value = await disputeApi.myBlocks();
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
}

async function counter(): Promise<void> {
  counterBusy.value = true;
  counterError.value = null;
  counterDone.value = false;
  try {
    await disputeApi.counter(counterId.value.trim(), counterNote.value.trim());
    counterDone.value = true;
  } catch (err: unknown) {
    counterError.value = errMessageOf(err);
  } finally {
    counterBusy.value = false;
  }
}

onMounted(() => void load());
</script>
