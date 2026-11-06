<template>
  <div>
    <h1 class="mb-4 text-xl font-bold">License Policies</h1>
    <p class="mb-3 text-sm text-neutral-400">제공자별 상업 라이선스 정책을 실시간으로 수정합니다 (PUT /admin/policies/:provider).</p>
    <p v-if="error" class="mb-2 text-sm text-red-400">{{ error }}</p>
    <p v-if="saved" class="mb-2 text-sm text-emerald-400">{{ saved }}</p>
    <div class="space-y-3">
      <div v-for="p in policies" :key="p.provider" class="rounded bg-neutral-900 p-4">
        <div class="mb-2 flex items-center gap-2">
          <h2 class="font-bold">{{ p.provider }}</h2>
        </div>
        <div class="grid gap-2 text-sm sm:grid-cols-2">
          <label class="flex items-center gap-2">
            <input v-model="draft[p.provider]!.allowsStandaloneStreaming" type="checkbox" />
            단독 스트리밍 허용
          </label>
          <label>최소 비트레이트 (kbps)
            <input v-model.number="draft[p.provider]!.minBitrateKbps" type="number" min="0" class="ml-2 w-28 rounded bg-neutral-800 p-1.5" />
          </label>
          <label class="sm:col-span-2">허용 상업 요금제 (쉼표 구분, 예: Pro, Premier)
            <input v-model="allowedTiers[p.provider]" class="mt-1 w-full rounded bg-neutral-800 p-1.5" />
          </label>
          <label class="sm:col-span-2">차단 비상업 요금제 (쉼표 구분, 예: Free, Basic)
            <input v-model="blockedTiers[p.provider]" class="mt-1 w-full rounded bg-neutral-800 p-1.5" />
          </label>
          <label class="sm:col-span-2">Sync 전용 (단독 스트리밍 차단) 요금제 (쉼표 구분, 예: Standard, Creator)
            <input v-model="syncOnlyTiers[p.provider]" class="mt-1 w-full rounded bg-neutral-800 p-1.5" />
          </label>
          <label class="sm:col-span-2">차단 메타데이터 키워드 (쉼표 구분)
            <input v-model="keywords[p.provider]" class="mt-1 w-full rounded bg-neutral-800 p-1.5" />
          </label>
          <label class="sm:col-span-2">거절 메시지 (KO)
            <input v-model="draft[p.provider]!.rejectionMessageKo" class="mt-1 w-full rounded bg-neutral-800 p-1.5" />
          </label>
          <label class="sm:col-span-2">상업 약관 URL
            <input v-model="termsUrl[p.provider]" placeholder="https://…" class="mt-1 w-full rounded bg-neutral-800 p-1.5" />
          </label>
        </div>
        <button
          class="mt-2 rounded bg-brand-600 px-3 py-1.5 text-sm font-bold disabled:opacity-50"
          :disabled="saving === p.provider"
          @click="save(p.provider)"
        >
          {{ saving === p.provider ? '저장 중…' : '저장' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { adminApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import type { AIProviderPolicy } from '@/api/types';

const policies = ref<AIProviderPolicy[]>([]);
const draft = ref<Record<string, { allowsStandaloneStreaming: boolean; minBitrateKbps: number; rejectionMessageKo: string }>>({});
const allowedTiers = ref<Record<string, string>>({});
const blockedTiers = ref<Record<string, string>>({});
const syncOnlyTiers = ref<Record<string, string>>({});
const keywords = ref<Record<string, string>>({});
const termsUrl = ref<Record<string, string>>({});
const error = ref<string | null>(null);
const saved = ref<string | null>(null);
const saving = ref<string | null>(null);

function splitCsv(raw?: string): string[] {
  return (raw ?? '')
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean);
}

async function load(): Promise<void> {
  try {
    policies.value = await adminApi.listPolicies();
    for (const p of policies.value) {
      draft.value[p.provider] = {
        allowsStandaloneStreaming: p.allowsStandaloneStreaming,
        minBitrateKbps: p.minBitrateKbps,
        rejectionMessageKo: p.rejectionMessageKo,
      };
      allowedTiers.value[p.provider] = (p.allowedPlanTiers ?? []).join(', ');
      blockedTiers.value[p.provider] = (p.blockedPlanTiers ?? []).join(', ');
      syncOnlyTiers.value[p.provider] = (p.syncOnlyPlanTiers ?? []).join(', ');
      keywords.value[p.provider] = (p.blockedMetadataKeywords ?? []).join(', ');
      termsUrl.value[p.provider] = p.commercialTermsUrl ?? '';
    }
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  }
}

async function save(provider: string): Promise<void> {
  const d = draft.value[provider];
  if (!d) return;
  saving.value = provider;
  error.value = null;
  saved.value = null;
  try {
    await adminApi.updatePolicy(provider, {
      allowsStandaloneStreaming: d.allowsStandaloneStreaming,
      allowedPlanTiers: splitCsv(allowedTiers.value[provider]),
      blockedPlanTiers: splitCsv(blockedTiers.value[provider]),
      syncOnlyPlanTiers: splitCsv(syncOnlyTiers.value[provider]),
      minBitrateKbps: d.minBitrateKbps,
      blockedMetadataKeywords: splitCsv(keywords.value[provider]),
      rejectionMessageKo: d.rejectionMessageKo,
      commercialTermsUrl: (termsUrl.value[provider] ?? '').trim() || null,
    });
    saved.value = `${provider} 정책이 저장되었습니다.`;
    await load();
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    saving.value = null;
  }
}

onMounted(() => void load());
</script>
