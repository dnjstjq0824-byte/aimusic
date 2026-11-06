<template>
  <div>
    <h1 class="mb-4 text-xl font-bold">Curation</h1>
    <p class="mb-3 text-sm text-neutral-400">재생수 기준 READY 상위 트랙과 즉시 takedown 작업대입니다.</p>
    <p v-if="error" class="mb-2 text-sm text-red-400">{{ error }}</p>
    <ul class="space-y-2">
      <li v-for="t in tracks" :key="t.id" class="rounded bg-neutral-900 p-3 text-sm">
        <div class="flex items-center gap-2">
          <span class="min-w-0 flex-1 truncate font-semibold">{{ t.title }}</span>
          <AiBadge :track="t" />
        </div>
        <div class="mt-1 text-xs text-neutral-400">plays {{ t.playCount }} · {{ t.aiProvider }}</div>
        <div class="mt-2 flex gap-2">
          <input v-model="reason[t.id]" placeholder="takedown reason" class="min-w-0 flex-1 rounded bg-neutral-800 p-1 text-xs" />
          <button class="rounded bg-red-800 px-2 py-1 text-xs font-bold" @click="takedown(t.id)">Takedown</button>
        </div>
      </li>
    </ul>
    <p v-if="tracks.length === 0" class="text-sm text-neutral-500">큐레이션 대상이 없습니다.</p>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { adminApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import type { Track } from '@/api/types';
import AiBadge from '@/components/legal/AiBadge.vue';

const tracks = ref<Track[]>([]);
const reason = ref<Record<string, string>>({});
const error = ref<string | null>(null);

async function load(): Promise<void> {
  try {
    tracks.value = await adminApi.curation();
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  }
}

async function takedown(trackId: string): Promise<void> {
  try {
    await adminApi.takedown(trackId, reason.value[trackId] || undefined);
    await load();
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  }
}

onMounted(() => void load());
</script>
