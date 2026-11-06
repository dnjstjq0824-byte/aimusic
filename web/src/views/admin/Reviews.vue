<template>
  <div>
    <h1 class="mb-4 text-xl font-bold">Review Workbench (Gate 4 상업 라이선스 심사)</h1>
    <p v-if="error" class="mb-2 text-sm text-red-400">{{ error }}</p>
    <div class="grid gap-6 lg:grid-cols-3">
      <section>
        <h2 class="mb-2 font-bold">Tracks ({{ pendingTracks.length }})</h2>
        <ul class="space-y-2">
          <li v-for="t in pendingTracks" :key="t.id" class="space-y-1.5 rounded bg-neutral-900 p-3 text-sm">
            <div class="flex items-center justify-between gap-2">
              <div class="font-semibold">{{ t.title }}</div>
              <button
                class="rounded bg-brand-600 px-2 py-0.5 text-xs font-bold"
                @click="previewTrack(t)"
              >
                ▶ 미리듣기
              </button>
            </div>
            <div class="font-mono text-xs text-neutral-500">{{ t.id }}</div>
            <div class="text-xs text-neutral-300">
              제공자: <span class="font-semibold">{{ t.aiProvider }}</span> · 상태: {{ t.status }} ·
              {{ t.bitrateKbps ? `${t.bitrateKbps}kbps` : '' }}
            </div>
            <!-- Gate 1-3 Rights & Forensic Summary -->
            <div v-if="rightsOf(t.id)" class="rounded bg-neutral-800/80 p-2 text-xs text-neutral-300">
              <div>
                요금제: <span class="font-semibold">{{ rightsOf(t.id)?.planTierUsed || '구독 기본' }}</span> ·
                생성일: {{ fmtDate(rightsOf(t.id)?.generatedAt) }}
              </div>
              <div v-if="subOf(t.id)">
                구독 증빙: {{ subOf(t.id)?.planName }} ({{ subOf(t.id)?.status }}) ·
                증빙키: <span class="font-mono">{{ subOf(t.id)?.proofObjectKey || '없음' }}</span>
              </div>
              <div v-if="rightsOf(t.id)?.providerTrackUrl" class="truncate">
                원본 URL:
                <a :href="rightsOf(t.id)?.providerTrackUrl ?? '#'" target="_blank" rel="noreferrer" class="text-brand-400 underline">
                  {{ rightsOf(t.id)?.providerTrackUrl }}
                </a>
              </div>
              <div class="mt-1 font-mono text-[11px] text-emerald-300">
                Gate1={{ rightsOf(t.id)?.gate1PolicyPassed ? 'PASS' : 'FAIL' }} ·
                Gate2={{ rightsOf(t.id)?.gate2DatePassed ? 'PASS' : 'FAIL' }} ·
                Gate3(FFprobe)={{ rightsOf(t.id)?.gate3FfprobePassed ? 'PASS' : 'PENDING/FAIL' }}
              </div>
              <div v-if="rightsOf(t.id)?.ffprobeReportJson" class="mt-1 break-all font-mono text-[11px] text-neutral-400">
                FFprobe: {{ rightsOf(t.id)?.ffprobeReportJson }}
              </div>
            </div>
            <div class="mt-2 flex gap-2">
              <input v-model="trackReason[t.id]" placeholder="반려 사유 (거절 시)" class="min-w-0 flex-1 rounded bg-neutral-800 p-1 text-xs" />
              <button class="rounded bg-emerald-700 px-2 py-1 text-xs font-bold" @click="reviewTrack(t.id, true)">승인 (READY)</button>
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
            <div v-if="s.proofObjectKey" class="text-xs text-neutral-400">증빙: {{ s.proofObjectKey }}</div>
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
import { computed, onMounted, ref } from 'vue';
import { adminApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { usePlayerStore } from '@/stores/playerStore';
import type { PartnerAISubscription, ReviewQueues, Track, TrackRightsDeclaration } from '@/api/types';

const player = usePlayerStore();
const queues = ref<ReviewQueues>({});
const error = ref<string | null>(null);
const trackReason = ref<Record<string, string>>({});
const subExpiry = ref<Record<string, string>>({});

const pendingTracks = computed<Track[]>(() => queues.value.tracks ?? queues.value.licenseGateTracks ?? []);

function rightsOf(trackId: string): TrackRightsDeclaration | undefined {
  return queues.value.rightsByTrack?.[trackId];
}

function subOf(trackId: string): PartnerAISubscription | undefined {
  const subId = rightsOf(trackId)?.subscriptionId;
  return subId ? queues.value.subsById?.[subId] : undefined;
}

function fmtDate(iso?: string | null): string {
  if (!iso) return '–';
  return iso.slice(0, 10);
}

function previewTrack(t: Track): void {
  player.setQueue([t], t.id);
}

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
