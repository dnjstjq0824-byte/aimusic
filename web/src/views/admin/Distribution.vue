<template>
  <div class="space-y-6">
    <UiPageHeader
      eyebrow="Audit Console"
      title="DSP 유통 키트"
      description="선별한 READY 곡의 유통 메타데이터(ISRC·BPM·키·권리 요약·마스터 다운로드)를 JSON/CSV로 내보냅니다. ISRC 미지정·미승인 곡은 제외 사유와 함께 skipped에 표시됩니다."
    />

    <UiCard padding="md">
      <form class="space-y-4" @submit.prevent="preview">
        <UiTextarea
          v-model="idsText"
          label="트랙 ID 목록 (쉼표·공백·줄바꿈 구분, 최대 100곡)"
          placeholder="예: 550e8400-e29b-41d4-a716-446655440000"
          :rows="4"
        />
        <UiErrorBanner :message="error" title="내보내기 실패" />
        <div class="flex flex-wrap gap-2">
          <UiButton variant="brand" size="sm" type="submit" :disabled="busy">미리보기</UiButton>
          <UiButton variant="secondary" size="sm" type="button" :disabled="busy || trackIds.length === 0" @click="downloadCsv">
            CSV 다운로드
          </UiButton>
        </div>
      </form>
    </UiCard>

    <UiCard v-if="kit" padding="none">
      <template #header>
        <div class="flex items-center justify-between px-5 pt-3">
          <h2 class="text-xs font-bold uppercase tracking-wider text-neutral-400">유통 가능 {{ kit.tracks.length }}곡 · 제외 {{ kit.skipped.length }}곡</h2>
          <span class="font-mono text-[11px] text-neutral-500">{{ kit.generatedAt }}</span>
        </div>
      </template>
      <div class="overflow-x-auto">
        <table class="w-full min-w-[720px] text-xs">
          <thead>
            <tr class="bg-surface-2/60 text-left text-neutral-400">
              <th class="px-3 py-2 font-semibold">ISRC</th>
              <th class="px-3 py-2 font-semibold">곡명</th>
              <th class="px-3 py-2 font-semibold">아티스트</th>
              <th class="px-3 py-2 font-semibold">BPM/키</th>
              <th class="px-3 py-2 font-semibold">제공자/티어</th>
              <th class="px-3 py-2 font-semibold">마스터</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-surface-border">
            <tr v-for="t in kit.tracks" :key="t.trackId" class="hover:bg-surface-2/40">
              <td class="px-3 py-2 font-mono text-neutral-200">{{ t.isrc }}</td>
              <td class="px-3 py-2 font-bold text-white">{{ t.title }}</td>
              <td class="px-3 py-2 text-neutral-300">{{ t.artistName }}</td>
              <td class="px-3 py-2 font-mono text-neutral-300">{{ t.bpm ?? '-' }} / {{ t.musicalKey || '-' }}</td>
              <td class="px-3 py-2 text-neutral-300">{{ t.aiProvider }} {{ t.planTierUsed }}</td>
              <td class="px-3 py-2">
                <a :href="t.masterDownloadUrl" target="_blank" rel="noopener" class="text-brand-300 underline hover:text-brand-200">
                  다운로드
                </a>
                <span class="ml-1 text-[10px] text-neutral-500">({{ masterLeft(t.masterUrlExpiry) }})</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div v-if="kit.skipped.length > 0" class="border-t border-surface-border px-5 py-3">
        <h3 class="mb-1 text-xs font-bold text-neutral-400">제외된 곡</h3>
        <ul class="space-y-1 text-[11px]">
          <li v-for="s in kit.skipped" :key="s.trackId" class="font-mono text-neutral-500">
            {{ s.trackId }} — {{ skipKo(s.reason) }}
          </li>
        </ul>
      </div>
    </UiCard>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { adminApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import type { DSPKit } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiTextarea from '@/components/ui/UiTextarea.vue';

const idsText = ref('');
const kit = ref<DSPKit | null>(null);
const busy = ref(false);
const error = ref<string | null>(null);

const trackIds = computed(() =>
  idsText.value.split(/[\s,]+/).map((s) => s.trim()).filter((s) => s.length > 0),
);

function skipKo(reason: string): string {
  if (reason === 'track_not_found') return '트랙 없음';
  if (reason === 'missing_isrc') return 'ISRC 미지정';
  if (reason === 'rights_missing') return '권리선언 누락';
  if (reason === 'partner_missing') return '파트너 누락';
  if (reason.startsWith('not_ready')) return `미승인(${reason.slice('not_ready:'.length)})`;
  return reason;
}

function masterLeft(expiry: string): string {
  const ms = new Date(expiry).getTime() - Date.now();
  if (ms <= 0) return '만료됨';
  return `${Math.ceil(ms / 60000)}분 남음`;
}

async function preview(): Promise<void> {
  busy.value = true;
  error.value = null;
  try {
    kit.value = await adminApi.dspKit(trackIds.value);
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    busy.value = false;
  }
}

async function downloadCsv(): Promise<void> {
  busy.value = true;
  error.value = null;
  try {
    const blob = await adminApi.dspKitCsv(trackIds.value);
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `dsp-kit-${new Date().toISOString().slice(0, 10)}.csv`;
    document.body.appendChild(a);
    a.click();
    a.remove();
    URL.revokeObjectURL(url);
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    busy.value = false;
  }
}
</script>
