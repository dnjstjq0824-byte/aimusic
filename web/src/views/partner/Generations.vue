<template>
  <div class="space-y-6">
    <UiPageHeader
      eyebrow="Studio Console"
      title="AI 생성 콘솔"
      description="ACE-Step 생성 워커에 음악 생성을 요청하고 작업 상태를 확인합니다. 일일 할당량과 금지 프롬프트 필터가 적용됩니다."
    >
      <template #actions>
        <div class="flex gap-2">
          <UiButton variant="secondary" size="sm" :loading="loading" @click="load">
            새로고침
          </UiButton>
          <UiButton variant="brand" size="sm" @click="createModalOpen = true">
            새 생성 요청
          </UiButton>
        </div>
      </template>
    </UiPageHeader>

    <UiErrorBanner :message="error" title="생성 작업을 불러오지 못했습니다" retry-label="다시 시도" @retry="load" />

    <UiCard padding="none">
      <template #header>
        <div class="px-5 pt-3 flex items-center justify-between">
          <h2 class="text-xs font-bold text-neutral-400 uppercase tracking-wider">내 생성 작업</h2>
          <span class="text-xs text-neutral-500 font-mono">{{ jobs.length }}개</span>
        </div>
      </template>

      <div v-if="loading" class="space-y-2 p-4">
        <UiSkeleton height="h-12" width="w-full" />
        <UiSkeleton height="h-12" width="w-full" />
      </div>

      <div v-else-if="jobs.length === 0" class="py-12">
        <UiEmptyState
          title="생성 작업이 없습니다"
          description="프롬프트를 입력해 첫 AI 음악 생성을 요청해 보세요."
        >
          <template #action>
            <UiButton variant="brand" size="sm" @click="createModalOpen = true">
              생성 요청하기
            </UiButton>
          </template>
        </UiEmptyState>
      </div>

      <ul v-else class="divide-y divide-surface-border">
        <li
          v-for="j in jobs"
          :key="j.id"
          class="px-5 py-4 text-xs"
        >
          <button
            type="button"
            class="flex w-full flex-col gap-2 text-left sm:flex-row sm:items-center sm:justify-between"
            @click="toggleDetail(j.id)"
          >
            <div class="min-w-0 flex-1 space-y-1">
              <div class="flex flex-wrap items-center gap-2">
                <UiBadge :variant="statusVariant(j.status)" size="xs">
                  {{ statusKo(j.status) }}
                </UiBadge>
                <span class="font-mono text-[11px] text-neutral-500">{{ j.model }}</span>
              </div>
              <p class="truncate text-sm font-semibold text-white">{{ j.prompt }}</p>
              <div class="font-mono text-[11px] text-neutral-500">
                {{ new Date(j.createdAt).toLocaleString() }} · 시도 {{ j.attempts }}회
                <span v-if="j.durationMs"> · {{ Math.round(j.durationMs / 1000) }}초</span>
              </div>
            </div>
            <span class="shrink-0 text-[11px] font-semibold text-brand-400">
              {{ expandedId === j.id ? '접기 ▲' : '상세 ▼' }}
            </span>
          </button>

          <div v-if="expandedId === j.id" class="mt-3 space-y-2 rounded-lg border border-surface-border bg-surface-0 p-3">
            <div v-if="detailLoading" class="py-2">
              <UiSkeleton height="h-4" width="w-2/3" />
            </div>
            <template v-else-if="detail">
              <div v-if="detail.lyrics" class="whitespace-pre-wrap text-xs text-neutral-300">{{ detail.lyrics }}</div>
              <div class="font-mono text-[11px] text-neutral-500">작업 ID: {{ detail.id }}</div>
              <div v-if="detail.outputObjectKey" class="font-mono text-[11px] text-accent-emerald">
                출력: {{ detail.outputObjectKey }}
              </div>
              <div v-if="detail.error" class="text-[11px] font-semibold text-accent-crimson">
                실패 사유: {{ failureKo(detail.error) }}
              </div>
              <div v-if="detail.licenseRef" class="text-[11px] text-neutral-400">
                라이선스 참조: {{ detail.licenseRef }}
              </div>
            </template>
          </div>
        </li>
      </ul>
    </UiCard>

    <UiModal
      v-model="createModalOpen"
      title="새 AI 생성 요청"
      description="프롬프트와 가사(선택)를 입력하면 생성 큐에 등록됩니다."
      @close="createModalOpen = false"
    >
      <form class="space-y-4" @submit.prevent="create">
        <UiTextarea
          v-model="prompt"
          label="프롬프트"
          placeholder="예: 잔잔한 피아노와 스트링의 시네마틱 인트로, 90 BPM"
          required
        />

        <UiTextarea
          v-model="lyrics"
          label="가사 (선택)"
          placeholder="가사가 있는 곡이면 입력하세요"
        />

        <UiInput
          v-model="model"
          label="모델"
          placeholder="ace-step-v1"
        />

        <UiInput
          v-model="licenseRef"
          label="라이선스 참조 (선택)"
          placeholder="예: 구독/계약 참조 번호"
        />

        <div class="flex justify-end gap-2 pt-2 border-t border-surface-border">
          <UiButton variant="ghost" size="sm" @click="createModalOpen = false">취소</UiButton>
          <UiButton variant="brand" size="sm" type="submit" :loading="busy">생성 요청</UiButton>
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
import type { GenerationJob } from '@/api/types';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiInput from '@/components/ui/UiInput.vue';
import UiModal from '@/components/ui/UiModal.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import UiTextarea from '@/components/ui/UiTextarea.vue';

const toast = useToast();

const STATUS_KO: Record<string, string> = {
  QUEUED: '대기 중',
  RUNNING: '생성 중',
  SUCCEEDED: '완료',
  FAILED: '실패',
};

// EV-P08: never expose internal variable names; map known backend failures
// to user-facing guidance. Backend connection itself is engine scope.
function failureKo(raw: string): string {
  if (raw.includes('GENERATION_BACKEND_URL')) {
    return '음악 생성 서버가 연결되지 않았습니다. 관리자에게 생성 서버 연결을 요청하세요.';
  }
  return raw;
}

function statusKo(status: string): string {
  return STATUS_KO[status] ?? status;
}

function statusVariant(status: string): 'success' | 'warning' | 'danger' | 'info' {
  if (status === 'SUCCEEDED') return 'success';
  if (status === 'FAILED') return 'danger';
  if (status === 'RUNNING') return 'info';
  return 'warning';
}

const jobs = ref<GenerationJob[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);
const busy = ref(false);
const createModalOpen = ref(false);
const expandedId = ref<string | null>(null);
const detail = ref<GenerationJob | null>(null);
const detailLoading = ref(false);

const prompt = ref('');
const lyrics = ref('');
const model = ref('ace-step-v1');
const licenseRef = ref('');

async function load(): Promise<void> {
  loading.value = true;
  error.value = null;
  try {
    jobs.value = (await partnerApi.listGenerations()) ?? [];
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
}

async function toggleDetail(id: string): Promise<void> {
  if (expandedId.value === id) {
    expandedId.value = null;
    detail.value = null;
    return;
  }
  expandedId.value = id;
  detail.value = null;
  detailLoading.value = true;
  try {
    detail.value = await partnerApi.getGeneration(id);
  } catch (err: unknown) {
    toast.error(errMessageOf(err), '상세 조회 실패');
    expandedId.value = null;
  } finally {
    detailLoading.value = false;
  }
}

async function create(): Promise<void> {
  busy.value = true;
  try {
    await partnerApi.submitGeneration({
      prompt: prompt.value,
      lyrics: lyrics.value || undefined,
      model: model.value || undefined,
      licenseRef: licenseRef.value || undefined,
    });
    toast.success('생성 작업이 큐에 등록되었습니다.');
    prompt.value = '';
    lyrics.value = '';
    licenseRef.value = '';
    createModalOpen.value = false;
    await load();
  } catch (err: unknown) {
    toast.error(errMessageOf(err), '생성 요청 실패');
  } finally {
    busy.value = false;
  }
}

onMounted(() => void load());
</script>
