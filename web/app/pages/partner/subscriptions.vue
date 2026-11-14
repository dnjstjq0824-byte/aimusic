<template>
  <div class="space-y-6">
    <UiPageHeader
      eyebrow="Studio Console"
      title="AI 유료 구독 증빙 관리"
      description="Suno Pro, AIVA Pro 등 유료 상업 플랜 결제 영수증을 등록하면 관리자 확인 후 음원 업로드가 가능합니다."
    >
      <template #actions>
        <UiButton variant="brand" size="sm" @click="createModalOpen = true">
          새 유료 구독 증빙 등록
        </UiButton>
      </template>
    </UiPageHeader>

    <UiErrorBanner :message="error" title="구독 증빙을 불러오지 못했습니다" retry-label="다시 시도" @retry="load" />

    <!-- Subscriptions List -->
    <UiCard padding="none">
      <template #header>
        <div class="px-5 pt-3 flex items-center justify-between">
          <h2 class="text-xs font-bold text-text-secondary uppercase tracking-wider">등록된 AI 플랫폼 구독 증빙</h2>
          <span class="text-xs text-text-muted font-mono">{{ subs.length }}개</span>
        </div>
      </template>

      <div v-if="loading" class="space-y-2 p-4">
        <UiSkeleton height="h-12" width="w-full" />
        <UiSkeleton height="h-12" width="w-full" />
      </div>

      <div v-else-if="subs.length === 0" class="py-12">
        <UiEmptyState
          title="등록된 구독 증빙이 없습니다"
          description="Suno Pro/Premier 또는 AIVA Pro 등 유료 상업용 구독 인보이스를 등록해 주세요."
        >
          <template #action>
            <UiButton variant="brand" size="sm" @click="createModalOpen = true">
              구독 증빙 등록하기
            </UiButton>
          </template>
        </UiEmptyState>
      </div>

      <ul v-else class="divide-y divide-surface-border">
        <li
          v-for="s in subs"
          :key="s.id"
          class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 px-5 py-4 hover:bg-surface-2/50 transition-colors text-xs"
        >
          <div class="min-w-0 flex-1 space-y-1">
            <div class="flex items-center gap-2">
              <span class="font-bold text-white text-sm">{{ s.provider }}</span>
              <span class="font-semibold text-brand-400">{{ s.planName }}</span>
              <UiBadge
                :variant="s.status === 'VERIFIED' ? 'success' : s.status === 'REJECTED' ? 'danger' : 'warning'"
                size="xs"
              >
                {{ subStatusKo(s.status) }}
              </UiBadge>
            </div>
            <div class="font-mono text-caption text-text-muted">증빙 ID: {{ s.id }}</div>
            <div v-if="s.expiresAt" class="text-caption text-text-secondary">
              구독 유효기간: <strong class="text-text-primary">{{ new Date(s.expiresAt).toLocaleDateString() }}</strong>까지
            </div>
          </div>

          <div class="text-caption shrink-0">
            <span v-if="s.status === 'VERIFIED'" class="text-verified font-semibold">Gate 1-2 사용 가능</span>
            <span v-else-if="s.status === 'PENDING'" class="text-accent-amber font-semibold">관리자 검증 대기중</span>
            <span v-else class="text-accent-crimson font-semibold">반려됨</span>
          </div>
        </li>
      </ul>
    </UiCard>

    <!-- Create Subscription Modal -->
    <UiModal
      v-model="createModalOpen"
      title="AI 유료 구독 증빙 등록"
      description="Suno, AIVA, Udio 등 상업용 요금제 인보이스/영수증 정보를 입력하세요."
      @close="createModalOpen = false"
    >
      <form class="space-y-4" @submit.prevent="create">
        <UiSelect v-model="provider" label="AI 플랫폼" required>
          <option value="">플랫폼을 선택하세요…</option>
          <option v-for="p in providers" :key="p" :value="p">{{ p }}</option>
        </UiSelect>

        <UiInput
          v-model="planName"
          label="구독 플랜명"
          placeholder="예: Pro, Premier, Studio"
          required
        />

        <UiInput
          v-model="proofKey"
          label="인보이스 파일 키 / 영수증 문서 번호 (선택)"
          placeholder="예: invoices/suno_pro_202611.pdf"
        />

        <div class="flex justify-end gap-2 pt-2 border-t border-surface-border">
          <UiButton variant="ghost" size="sm" @click="createModalOpen = false">취소</UiButton>
          <UiButton variant="brand" size="sm" type="submit" :loading="busy">등록 완료</UiButton>
        </div>
      </form>
    </UiModal>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { partnerApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { useToast } from '@/composables/useToast';
import type { PartnerAISubscription } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiInput from '@/components/ui/UiInput.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSelect from '@/components/ui/UiSelect.vue';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiModal from '@/components/ui/UiModal.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';

const toast = useToast();

const providers = ['SUNO', 'UDIO', 'AIVA', 'STABLE_AUDIO', 'SOUNDRAW', 'MUBERT', 'CUSTOM_OPEN_MODEL'];

const SUB_STATUS_KO: Record<string, string> = {
  PENDING: '검증 대기',
  VERIFIED: '검증 완료',
  REJECTED: '반려됨',
  EXPIRED: '만료됨',
};

function subStatusKo(status: string): string {
  return SUB_STATUS_KO[status] ?? status;
}
const subs = ref<PartnerAISubscription[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);
const busy = ref(false);
const createModalOpen = ref(false);

const provider = ref('');
const planName = ref('');
const proofKey = ref('');

async function load(): Promise<void> {
  loading.value = true;
  error.value = null;
  try {
    subs.value = (await partnerApi.listSubscriptions()) ?? [];
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
}

async function create(): Promise<void> {
  busy.value = true;
  try {
    await partnerApi.createSubscription({
      provider: provider.value,
      planName: planName.value,
      proofObjectKey: proofKey.value || undefined,
    });
    toast.success('구독 증빙이 등록되었습니다. 관리자가 검증 후 활성화됩니다.');
    provider.value = '';
    planName.value = '';
    proofKey.value = '';
    createModalOpen.value = false;
    await load();
  } catch (err: unknown) {
    toast.error(errMessageOf(err), '등록 실패');
  } finally {
    busy.value = false;
  }
}

onMounted(() => void load());
</script>
