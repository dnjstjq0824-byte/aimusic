<template>
  <div>
    <h1 class="mb-4 text-xl font-bold">Review Workbench</h1>
    <p v-if="error" class="mb-2 text-sm text-red-400">{{ error }}</p>
    <div class="grid gap-6 lg:grid-cols-3">
      <section>
        <h2 class="mb-2 font-bold">Tracks ({{ queues.tracks?.length ?? 0 }})</h2>
        <ul class="space-y-2">
          <li v-for="t in queues.tracks ?? []" :key="t.id" class="rounded bg-neutral-900 p-3 text-sm">
            <div class="font-semibold">{{ t.title }}</div>
            <div class="font-mono text-xs text-neutral-500">{{ t.id }}</div>
            <div class="text-xs text-neutral-400">{{ t.aiProvider }} · {{ t.status }}</div>
            <div class="mt-2 flex gap-2">
              <input v-model="trackReason[t.id]" placeholder="reason (optional)" class="min-w-0 flex-1 rounded bg-neutral-800 p-1 text-xs" />
              <button class="rounded bg-emerald-700 px-2 py-1 text-xs font-bold" @click="reviewTrack(t.id, true)">승인</button>
              <button class="rounded bg-red-800 px-2 py-1 text-xs font-bold" @click="reviewTrack(t.id, false)">거절</button>
            </div>
          </li>
        </ul>
      </section>
      <section>
        <h2 class="mb-2 font-bold">Partners ({{ queues.partners?.length ?? 0 }})</h2>
        <ul class="space-y-2">
          <li v-for="p in queues.partners ?? []" :key="p.id" class="rounded bg-neutral-900 p-3 text-sm">
            <div class="font-semibold">{{ p.companyName }}</div>
            <div class="font-mono text-xs text-neutral-500">{{ p.id }}</div>
            <div class="text-xs text-neutral-400">{{ p.status }}</div>
            <div class="mt-2 flex gap-2">
              <button class="rounded bg-emerald-700 px-2 py-1 text-xs font-bold" @click="reviewPartner(p.id, true)">승인</button>
              <button class="rounded bg-red-800 px-2 py-1 text-xs font-bold" @click="reviewPartner(p.id, false)">정지</button>
            </div>
          </li>
        </ul>
      </section>
      <section>
        <h2 class="mb-2 font-bold">Subscriptions ({{ queues.subscriptions?.length ?? 0 }})</h2>
        <ul class="space-y-2">
          <li v-for="s in queues.subscriptions ?? []" :key="s.id" class="rounded bg-neutral-900 p-3 text-sm">
            <div class="font-semibold">{{ s.provider }} / {{ s.planName }}</div>
            <div class="font-mono text-xs text-neutral-500">{{ s.id }}</div>
            <div class="mt-2 flex gap-2">
              <input v-model="subExpiry[s.id]" type="date" class="min-w-0 flex-1 rounded bg-neutral-800 p-1 text-xs" />
              <button class="rounded bg-emerald-700 px-2 py-1 text-xs font-bold" @click="reviewSub(s.id, true)">검증</button>
              <button class="rounded bg-red-800 px-2 py-1 text-xs font-bold" @click="reviewSub(s.id, false)">거절</button>
            </div>
          </li>
        </ul>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { adminApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import type { ReviewQueues } from '@/api/types';

const queues = ref<ReviewQueues>({});
const error = ref<string | null>(null);
const trackReason = ref<Record<string, string>>({});
const subExpiry = ref<Record<string, string>>({});

async function load(): Promise<void> {
  try {
    queues.value = await adminApi.pending();
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  }
}

async function reviewTrack(id: string, approve: boolean): Promise<void> {
  try {
    await adminApi.reviewTrack(id, approve, trackReason.value[id] || undefined);
    await load();
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  }
}

async function reviewPartner(id: string, approve: boolean): Promise<void> {
  try {
    await adminApi.reviewPartner(id, approve);
    await load();
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  }
}

async function reviewSub(id: string, approve: boolean): Promise<void> {
  try {
    const raw = subExpiry.value[id];
    const expiresAt = raw ? new Date(raw).toISOString() : undefined;
    await adminApi.reviewSubscription(id, approve, expiresAt);
    await load();
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  }
}

onMounted(() => void load());
</script>
