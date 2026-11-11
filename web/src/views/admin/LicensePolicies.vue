<template>
  <div class="space-y-6">
    <UiPageHeader
      eyebrow="Policy Matrix"
      title="AI 플랫폼별 상업 라이선스 정책 매트릭스"
      description="각 AI 서비스별 단독 스트리밍 허용 여부, 허용 플랜, 차단 플랜을 실시간으로 관리합니다."
    >
      <template #actions>
        <UiButton variant="secondary" size="sm" :loading="loading" @click="load">새로고침</UiButton>
      </template>
    </UiPageHeader>

    <UiErrorBanner :message="error" title="정책 매트릭스를 불러오지 못했습니다" retry-label="다시 시도" @retry="load" />

    <UiCard class="text-xs text-neutral-300 space-y-1.5">
      <div class="font-bold text-white text-sm">미등록·차단 플랫폼 사유 (소유권 원칙)</div>
      <ul class="list-disc pl-5 space-y-1 leading-relaxed">
        <li>Soundraw·Mubert: 작곡권·배포권이 플랫폼에 귀속 — 전면 차단.</li>
        <li>Boomy·Loudly·Soundful: 저작권 플랫폼 보유, 라이선스에 불과 — 불등록.</li>
        <li>Beatoven·Ecrett: Sync 전용, 단독 배포 금지 — 불등록.</li>
        <li>Splash: 양도 문구 없음 — 불등록(양도 입증 시 재심).</li>
        <li>신규 제공자는 약관 원문에서 양도·이전·보유 중 하나가 확인될 때만 등록.</li>
      </ul>
    </UiCard>

    <div v-if="loading" class="space-y-2">
      <UiSkeleton height="h-12" width="w-full" />
      <UiSkeleton height="h-12" width="w-full" />
      <UiSkeleton height="h-12" width="w-full" />
    </div>

    <!-- Policies Table -->
    <UiTable v-if="!loading">
      <template #header>
        <th class="p-3.5">제공자 (Provider)</th>
        <th class="p-3.5">단독 스트리밍 유통</th>
        <th class="p-3.5">생성 컷오프</th>
        <th class="p-3.5">허용 플랜 (Allowed)</th>
        <th class="p-3.5">영상 BGM 전용 (Sync-Only)</th>
        <th class="p-3.5">차단 플랜 (Blocked)</th>
        <th class="p-3.5 text-right">관리</th>
      </template>

      <tr
        v-for="p in policies"
        :key="p.provider"
        class="hover:bg-surface-2/40 transition-colors"
      >
        <td class="p-3.5 font-bold text-white">
          {{ p.provider }}
        </td>
        <td class="p-3.5">
          <UiBadge
            :variant="p.allowsStandaloneStreaming ? 'commercial' : 'syncOnly'"
            size="xs"
          >
            {{ p.allowsStandaloneStreaming ? '허용 (Standalone)' : '불허 (Sync-Only)' }}
          </UiBadge>
        </td>
        <td class="p-3.5 text-xs">
          <span v-if="p.maxGeneratedAt" class="text-amber-300">{{ fmtDate(p.maxGeneratedAt) }} 이후 거부</span>
          <span v-else class="text-neutral-500">없음</span>
        </td>
        <td class="p-3.5 font-mono text-neutral-300">
          {{ (p.allowedPlanTiers ?? []).join(', ') || '전체 허용' }}
        </td>
        <td class="p-3.5 font-mono text-accent-amber">
          {{ (p.syncOnlyPlanTiers ?? []).join(', ') || '–' }}
        </td>
        <td class="p-3.5 font-mono text-accent-crimson">
          {{ (p.blockedPlanTiers ?? []).join(', ') || '–' }}
        </td>
        <td class="p-3.5 text-right">
          <UiButton variant="outline" size="xs" @click="openEdit(p)">
            정책 편집
          </UiButton>
        </td>
      </tr>
    </UiTable>

    <!-- Edit Policy Modal -->
    <UiModal
      v-model="editModalOpen"
      :title="`${editingPolicy?.provider} 상업 정책 수정`"
      description="플랫폼의 상업 이용 약관 변경 시 즉시 반영합니다."
      @close="editModalOpen = false"
    >
      <form v-if="editingPolicy" class="space-y-4 text-xs" @submit.prevent="savePolicy">
        <div>
          <UiCheckbox v-model="formAllowsStreaming">
            <strong>단독 음원 스트리밍 유통 허용</strong> (체크 해제 시 해당 플랫폼의 모든 음원이 Sync-only로 분류됨)
          </UiCheckbox>
        </div>

        <UiInput
          v-model="formAllowedTiers"
          label="상업 유통 허용 요금제 (쉼표로 구분)"
          placeholder="Pro, Premier, Studio"
        />

        <UiInput
          v-model="formSyncOnlyTiers"
          label="영상 BGM 전용(Sync-Only) 요금제 (쉼표로 구분)"
          placeholder="Standard, Creator"
        />

        <UiInput
          v-model="formBlockedTiers"
          label="비상업 차단 요금제 (쉼표로 구분)"
          placeholder="Free, Basic, Trial"
        />

        <UiInput
          v-model="formRejectionMsg"
          label="업로드 반려 시 파트너에게 노출할 사유"
          placeholder="해당 플랫폼의 요금제는 단독 음원 유통이 불가합니다."
        />

        <UiInput
          v-model="formMaxGeneratedAt"
          label="생성 컷오프 (선택, RFC3339 — 이후 생성분 거부)"
          placeholder="예: 2025-10-30T00:00:00Z (비우면 미적용)"
        />

        <UiInput
          v-model="formTermsSnapshotUrl"
          label="약관 스냅샷 URL (선택)"
          placeholder="제공자 약관 PDF/캡처 보관 경로"
        />

        <div class="flex justify-end gap-2 pt-2 border-t border-surface-border">
          <UiButton variant="ghost" size="sm" @click="editModalOpen = false">취소</UiButton>
          <UiButton variant="brand" size="sm" type="submit" :loading="saving">저장하기</UiButton>
        </div>
      </form>
    </UiModal>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { adminApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { useToast } from '@/composables/useToast';
import type { AIProviderPolicy } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiInput from '@/components/ui/UiInput.vue';
import UiCheckbox from '@/components/ui/UiCheckbox.vue';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import UiTable from '@/components/ui/UiTable.vue';
import UiModal from '@/components/ui/UiModal.vue';

const toast = useToast();

const policies = ref<AIProviderPolicy[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);
const editModalOpen = ref(false);
const editingPolicy = ref<AIProviderPolicy | null>(null);

const formAllowsStreaming = ref(false);
const formAllowedTiers = ref('');
const formSyncOnlyTiers = ref('');
const formBlockedTiers = ref('');
const formRejectionMsg = ref('');
const formMaxGeneratedAt = ref('');
const formTermsSnapshotUrl = ref('');
const saving = ref(false);

function fmtDate(iso: string): string {
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleDateString('ko-KR');
}

async function load() {
  loading.value = true;
  error.value = null;
  try {
    policies.value = (await adminApi.listPolicies()) ?? [];
  } catch (err) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
}

function openEdit(p: AIProviderPolicy) {
  editingPolicy.value = p;
  formAllowsStreaming.value = p.allowsStandaloneStreaming;
  formAllowedTiers.value = (p.allowedPlanTiers ?? []).join(', ');
  formSyncOnlyTiers.value = (p.syncOnlyPlanTiers ?? []).join(', ');
  formBlockedTiers.value = (p.blockedPlanTiers ?? []).join(', ');
  formRejectionMsg.value = p.rejectionMessageKo || '';
  formMaxGeneratedAt.value = p.maxGeneratedAt ?? '';
  formTermsSnapshotUrl.value = p.termsSnapshotUrl ?? '';
  editModalOpen.value = true;
}

async function savePolicy() {
  if (!editingPolicy.value) return;
  saving.value = true;
  try {
    const parse = (s: string) => s.split(',').map((x) => x.trim()).filter(Boolean);
    await adminApi.updatePolicy(editingPolicy.value.provider, {
      allowsStandaloneStreaming: formAllowsStreaming.value,
      allowedPlanTiers: parse(formAllowedTiers.value),
      syncOnlyPlanTiers: parse(formSyncOnlyTiers.value),
      blockedPlanTiers: parse(formBlockedTiers.value),
      minBitrateKbps: editingPolicy.value.minBitrateKbps || 192,
      blockedMetadataKeywords: editingPolicy.value.blockedMetadataKeywords || [],
      rejectionMessageKo: formRejectionMsg.value || '단독 유통 불가',
      commercialTermsUrl: editingPolicy.value.commercialTermsUrl || null,
      maxGeneratedAt: formMaxGeneratedAt.value.trim() || editingPolicy.value.maxGeneratedAt || null,
      termsSnapshotUrl: formTermsSnapshotUrl.value.trim() || editingPolicy.value.termsSnapshotUrl || null,
    });
    toast.success('정책이 갱신되었습니다.');
    editModalOpen.value = false;
    await load();
  } catch (err) {
    toast.error(errMessageOf(err), '수정 실패');
  } finally {
    saving.value = false;
  }
}

onMounted(() => {
  void load();
});
</script>
