<template>
  <div>
    <h1 class="mb-4 text-xl font-bold">AI Subscriptions</h1>
    <p class="mb-3 text-sm text-neutral-400">
      상업용 플랜 증빙을 등록하면 관리자가 검증합니다. Gate1-2 사전검사는 VERIFIED 구독 기준으로 수행됩니다.
    </p>
    <form class="mb-6 grid gap-2 rounded bg-neutral-900 p-4 sm:grid-cols-4" @submit.prevent="create">
      <select v-model="provider" required class="rounded bg-neutral-800 p-2 text-sm">
        <option value="">provider…</option>
        <option v-for="p in providers" :key="p" :value="p">{{ p }}</option>
      </select>
      <input v-model="planName" required placeholder="plan name" class="rounded bg-neutral-800 p-2 text-sm" />
      <input v-model="proofKey" placeholder="proof object key (optional)" class="rounded bg-neutral-800 p-2 text-sm" />
      <button class="rounded bg-brand-600 px-3 py-1.5 text-sm font-bold disabled:opacity-50" :disabled="busy">
        {{ busy ? '등록 중…' : '증빙 등록' }}
      </button>
    </form>
    <p v-if="error" class="mb-2 text-sm text-red-400">{{ error }}</p>
    <ul class="space-y-2">
      <li v-for="s in subs" :key="s.id" class="rounded bg-neutral-900 p-3 text-sm">
        <div class="flex items-center gap-2">
          <span class="font-semibold">{{ s.provider }} — {{ s.planName }}</span>
          <span
            class="rounded px-1.5 py-0.5 text-[11px] font-bold"
            :class="s.status === 'VERIFIED' ? 'bg-emerald-700' : s.status === 'REJECTED' ? 'bg-red-800' : 'bg-neutral-700'"
          >
            {{ s.status }}
          </span>
        </div>
        <div class="mt-1 font-mono text-xs text-neutral-500">{{ s.id }}</div>
        <div v-if="s.expiresAt" class="text-xs text-neutral-400">만료: {{ s.expiresAt }}</div>
      </li>
    </ul>
    <p v-if="subs.length === 0" class="text-sm text-neutral-500">등록된 구독이 없습니다.</p>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { partnerApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import type { PartnerAISubscription } from '@/api/types';

const providers = ['SUNO', 'UDIO', 'AIVA', 'STABLE_AUDIO', 'SOUNDRAW', 'MUBERT', 'CUSTOM_OPEN_MODEL'];
const subs = ref<PartnerAISubscription[]>([]);
const error = ref<string | null>(null);
const busy = ref(false);
const provider = ref('');
const planName = ref('');
const proofKey = ref('');

async function load(): Promise<void> {
  try {
    subs.value = await partnerApi.listSubscriptions();
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  }
}

async function create(): Promise<void> {
  busy.value = true;
  error.value = null;
  try {
    await partnerApi.createSubscription({
      provider: provider.value,
      planName: planName.value,
      proofObjectKey: proofKey.value || undefined,
    });
    provider.value = '';
    planName.value = '';
    proofKey.value = '';
    await load();
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    busy.value = false;
  }
}

onMounted(() => void load());
</script>
