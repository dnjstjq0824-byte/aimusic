<template>
  <div>
    <h1 class="mb-4 text-xl font-bold">Disputes</h1>
    <p v-if="error" class="mb-2 text-sm text-red-400">{{ error }}</p>
    <ul class="space-y-2">
      <li v-for="d in disputes" :key="d.id" class="rounded bg-neutral-900 p-3 text-sm">
        <div class="flex items-center gap-2">
          <span class="font-semibold">{{ d.reason }}</span>
          <span class="rounded bg-neutral-700 px-1.5 py-0.5 text-[11px]">{{ d.status }}</span>
          <span v-if="d.strikeApplied" class="rounded bg-red-800 px-1.5 py-0.5 text-[11px]">strike</span>
        </div>
        <div class="mt-1 font-mono text-xs text-neutral-500">dispute {{ d.id }} · track {{ d.trackId }}</div>
        <p class="mt-1 text-neutral-300">{{ d.description }}</p>
        <div class="mt-2 flex gap-2">
          <button class="rounded bg-red-800 px-2 py-1 text-xs font-bold" @click="resolve(d.id, true)">인용 (takedown+strike)</button>
          <button class="rounded bg-neutral-700 px-2 py-1 text-xs font-bold" @click="resolve(d.id, false)">기각</button>
        </div>
      </li>
    </ul>
    <p v-if="disputes.length === 0" class="text-sm text-neutral-500">열린 분쟁이 없습니다.</p>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { adminApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import type { CopyrightDispute } from '@/api/types';

const disputes = ref<CopyrightDispute[]>([]);
const error = ref<string | null>(null);

async function load(): Promise<void> {
  try {
    disputes.value = await adminApi.listDisputes();
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  }
}

async function resolve(id: string, uphold: boolean): Promise<void> {
  try {
    await adminApi.resolveDispute(id, uphold);
    await load();
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  }
}

onMounted(() => void load());
</script>
