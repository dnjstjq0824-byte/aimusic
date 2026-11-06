<template>
  <div>
    <h1 class="mb-4 text-xl font-bold">Partner Dashboard</h1>
    <p v-if="error" class="mb-2 text-sm text-red-400">{{ error }}</p>
    <div class="mb-6 grid gap-3 sm:grid-cols-3">
      <div class="rounded bg-neutral-900 p-3">
        <div class="text-xs text-neutral-400">내 음원</div>
        <div class="text-2xl font-bold">{{ tracks.length }}</div>
      </div>
      <div class="rounded bg-neutral-900 p-3">
        <div class="text-xs text-neutral-400">READY</div>
        <div class="text-2xl font-bold text-emerald-400">{{ readyCount }}</div>
      </div>
      <div class="rounded bg-neutral-900 p-3">
        <div class="text-xs text-neutral-400">검증된 구독</div>
        <div class="text-2xl font-bold">{{ verifiedSubs }}</div>
      </div>
    </div>
    <h2 class="mb-2 font-bold">내 음원 상태</h2>
    <ul class="space-y-2">
      <li v-for="t in tracks" :key="t.id" class="rounded bg-neutral-900 p-3 text-sm">
        <div class="flex items-center gap-2">
          <span class="min-w-0 flex-1 truncate font-semibold">{{ t.title }}</span>
          <AiBadge :track="t" />
        </div>
        <div class="mt-1 text-xs text-neutral-400">
          {{ t.aiProvider }} · plays {{ t.playCount }}
          <span v-if="t.rejectionReasonKo" class="text-red-400"> · {{ t.rejectionReasonKo }}</span>
        </div>
      </li>
    </ul>
    <p v-if="tracks.length === 0" class="text-sm text-neutral-500">등록된 음원이 없습니다. Upload에서 등록하세요.</p>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { partnerApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import type { PartnerAISubscription, Track } from '@/api/types';
import AiBadge from '@/components/legal/AiBadge.vue';

const tracks = ref<Track[]>([]);
const subs = ref<PartnerAISubscription[]>([]);
const error = ref<string | null>(null);

const readyCount = computed(() => tracks.value.filter((t) => t.status === 'READY').length);
const verifiedSubs = computed(() => subs.value.filter((s) => s.status === 'VERIFIED').length);

onMounted(async () => {
  try {
    const [t, s] = await Promise.all([partnerApi.listMyTracks(50, 0), partnerApi.listSubscriptions()]);
    tracks.value = t;
    subs.value = s;
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  }
});
</script>
