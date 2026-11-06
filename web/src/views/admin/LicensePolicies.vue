<template>
  <div class="space-y-6">
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
      <div>
        <div class="inline-flex items-center gap-1.5 text-xs text-accent-purple font-bold uppercase tracking-wider mb-1">
          <span>AI Policy Matrix</span>
        </div>
        <h1 class="text-2xl font-black text-white tracking-tight">AI 플랫폼별 상업 라이선스 정책 매트릭스</h1>
        <p class="text-xs text-neutral-400 mt-1">
          각 AI 서비스별 단독 스트리밍 허용 여부(`allows_standalone_streaming`), 허용 플랜, 차단 플랜을 실시간으로 관리합니다.
        </p>
      </div>
      <UiButton variant="secondary" size="sm" @click="load">새로고침</UiButton>
    </div>

    <!-- Error Alert -->
    <div v-if="error" class="rounded-xl border border-red-800 bg-red-950/40 p-4 text-xs text-red-200">
      {{ error }}
    </div>

    <!-- Policies Table -->
    <UiTable>
      <template #header>
        <th class="p-3.5">제공자 (Provider)</th>
        <th class="p-3.5">단독 스트리밍 유통</th>
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
            {{ p.allowsStandaloneStreaming ? '✓ 허용 (Standalone)' : '✕ 불허 (Sync-Only)' }}
          </UiBadge>
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
import UiInput from '@/components/ui/UiInput.vue';
import UiCheckbox from '@/components/ui/UiCheckbox.vue';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiTable from '@/components/ui/UiTable.vue';
import UiModal from '@/components/ui/UiModal.vue';

const toast = useToast();

const policies = ref<AIProviderPolicy[]>([]);
const error = ref<string | null>(null);
const editModalOpen = ref(false);
const editingPolicy = ref<AIProviderPolicy | null>(null);

const formAllowsStreaming = ref(false);
const formAllowedTiers = ref('');
const formSyncOnlyTiers = ref('');
const formBlockedTiers = ref('');
const formRejectionMsg = ref('');
const saving = ref(false);

async function load() {
  error.value = null;
  try {
    policies.value = await adminApi.listPolicies();
  } catch (err) {
    error.value = errMessageOf(err);
  }
}

function openEdit(p: AIProviderPolicy) {
  editingPolicy.value = p;
  formAllowsStreaming.value = p.allowsStandaloneStreaming;
  formAllowedTiers.value = (p.allowedPlanTiers ?? []).join(', ');
  formSyncOnlyTiers.value = (p.syncOnlyPlanTiers ?? []).join(', ');
  formBlockedTiers.value = (p.blockedPlanTiers ?? []).join(', ');
  formRejectionMsg.value = p.rejectionMessageKo || '';
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
