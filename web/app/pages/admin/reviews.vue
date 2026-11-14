<template>
  <div class="space-y-6">
    <UiPageHeader
      eyebrow="Gate 4 Workbench"
      title="AI 음원 &amp; 상업 라이선스 최종 심사"
      description="Gate 1~3을 통과한 대기 음원과 파트너 증빙을 대조 심사하여 승인(READY) 또는 반려 처리합니다."
    >
      <template #actions>
        <UiButton variant="secondary" size="sm" :loading="loading" @click="loadAll">
          새로고침
        </UiButton>
      </template>
    </UiPageHeader>

    <UiErrorBanner :message="error" title="심사 대기열을 불러오지 못했습니다" retry-label="다시 시도" @retry="loadAll" />

    <UiCard class="text-xs text-neutral-300 space-y-1.5">
      <div class="font-bold text-white text-sm">Gate 4 심사 기준 (소유권 원칙)</div>
      <ul class="list-disc pl-5 space-y-1 leading-relaxed">
        <li>구독자인 생성자 본인이 소유권(양도·이전·보유)을 가진 음원만 승인. 라이선스만 부여되는 플랫폼은 반려.</li>
        <li>Stable Audio 직접 가중치 구동분은 해당 라이선스(Community/Enterprise) 확인 후 승인.</li>
        <li>Udio는 2025-10-30 이전 유료 생성분(기득권)만 승인. 이후분은 자동 거부됨.</li>
      </ul>
    </UiCard>

    <!-- Tabs -->
    <UiTabs v-model="activeTab" :tabs="tabOptions" />

    <div v-if="loading" class="grid gap-4 lg:grid-cols-2">
      <UiCard v-for="i in 4" :key="i" class="space-y-3">
        <UiSkeleton height="h-5" width="w-2/3" />
        <UiSkeleton height="h-3" width="w-1/2" />
        <UiSkeleton height="h-10" width="w-full" />
      </UiCard>
    </div>

    <!-- Tab 1: Pending Tracks Review -->
    <div v-if="!loading && activeTab === 'tracks'" class="space-y-4">
      <div v-if="pendingTracks.length === 0" class="py-12">
        <UiEmptyState
          title="검수 대기 중인 트랙이 없습니다"
          description="현재 심사 대기열에 등록된 음원이 모두 처리되었습니다."
        />
      </div>

      <div v-else class="grid gap-4 lg:grid-cols-2">
        <UiCard
          v-for="t in pendingTracks"
          :key="t.id"
          class="space-y-3.5"
        >
          <div class="flex items-start justify-between gap-3">
            <div>
              <h3 class="font-bold text-white text-base">{{ t.title }}</h3>
              <div class="text-[11px] font-mono text-neutral-500 mt-0.5">ID: {{ t.id }}</div>
            </div>
            <UiButton
              variant="brand"
              size="xs"
              @click="previewTrack(t)"
            >
              미리듣기
            </UiButton>
          </div>

          <!-- Metadata & Specs -->
          <div class="grid grid-cols-2 gap-2 text-xs">
            <div class="p-2.5 rounded-lg bg-surface-0 border border-surface-border">
              <span class="text-neutral-500 text-[10px] uppercase font-bold block">AI 플랫폼</span>
              <span class="font-semibold text-white">{{ t.aiProvider }}</span>
            </div>
            <div class="p-2.5 rounded-lg bg-surface-0 border border-surface-border">
              <span class="text-neutral-500 text-[10px] uppercase font-bold block">비트레이트</span>
              <span class="font-semibold text-accent-emerald">{{ t.bitrateKbps ? `${t.bitrateKbps}kbps` : '192k+' }}</span>
            </div>
          </div>

          <!-- Forensic & Gate 1-3 Summary -->
          <div v-if="rightsOf(t.id)" class="rounded-xl bg-surface-0 p-3 text-xs space-y-1.5 border border-surface-border">
            <div class="flex justify-between text-neutral-300">
              <span>요금제: <strong class="text-white">{{ rightsOf(t.id)?.planTierUsed || '구독 기본' }}</strong></span>
              <span>생성일: {{ fmtDate(rightsOf(t.id)?.generatedAt) }}</span>
            </div>
            <div class="font-mono text-[11px] text-accent-emerald flex items-center gap-2">
              <span>Gate1(Policy): {{ rightsOf(t.id)?.gate1PolicyPassed ? '✓ PASS' : '✕ FAIL' }}</span>
              <span>·</span>
              <span>Gate2(Date): {{ rightsOf(t.id)?.gate2DatePassed ? '✓ PASS' : '✕ FAIL' }}</span>
              <span>·</span>
              <span>Gate3(FFprobe): {{ rightsOf(t.id)?.gate3FfprobePassed ? '✓ PASS' : ' 대기/통과' }}</span>
            </div>
            <div v-if="rightsOf(t.id)?.providerTrackUrl" class="truncate text-[11px]">
              <span class="text-neutral-400">원본 링크: </span>
              <a :href="rightsOf(t.id)?.providerTrackUrl ?? '#'" target="_blank" rel="noreferrer" class="text-brand-400 underline">
                {{ rightsOf(t.id)?.providerTrackUrl }}
              </a>
            </div>
          </div>

          <!-- ISRC inline assign -->
          <div class="flex flex-col sm:flex-row items-stretch sm:items-center gap-2 pt-2 border-t border-surface-border">
            <UiInput
              v-model="trackIsrc[t.id]"
              :placeholder="t.isrc ? `ISRC: ${t.isrc}` : 'ISRC 입력 (예: KRA242600001)'"
            />
            <UiButton
              variant="secondary"
              size="sm"
              class="shrink-0"
              :loading="isrcBusyId === t.id"
              @click="saveIsrc(t)"
            >
              ISRC 저장
            </UiButton>
          </div>

          <!-- Review Action Controls -->
          <div class="flex flex-col sm:flex-row items-stretch sm:items-center gap-2 pt-2 border-t border-surface-border">
            <UiInput
              v-model="trackReason[t.id]"
              placeholder="반려 사유 (거절 시 작성)"
            />
            <div class="flex gap-2 shrink-0">
              <UiButton
                variant="success"
                size="sm"
                @click="reviewTrack(t.id, true)"
              >
                승인 (READY)
              </UiButton>
              <UiButton
                variant="danger"
                size="sm"
                @click="reviewTrack(t.id, false)"
              >
                반려
              </UiButton>
            </div>
          </div>
        </UiCard>
      </div>
    </div>

    <!-- Tab 2: Partner Profile Reviews -->
    <div v-else-if="!loading && activeTab === 'partners'" class="space-y-4">
      <div v-if="(queues.partners?.length ?? 0) === 0" class="py-12">
        <UiEmptyState
          title="대기 중인 파트너 신청이 없습니다"
        />
      </div>
      <div v-else class="grid gap-3 sm:grid-cols-2">
        <UiCard v-for="p in queues.partners ?? []" :key="p.id" class="space-y-3">
          <div class="flex items-center justify-between">
            <div class="font-bold text-white text-sm">{{ p.companyName }}</div>
            <UiBadge variant="warning" size="xs">{{ partnerStatusKo(p.status) }}</UiBadge>
          </div>
          <div class="font-mono text-[11px] text-neutral-500">파트너 ID: {{ p.id }}</div>
          <div class="flex gap-2 pt-2 border-t border-surface-border">
            <UiButton variant="success" size="xs" @click="reviewPartner(p.id, true)">승인</UiButton>
            <UiButton variant="danger" size="xs" @click="reviewPartner(p.id, false)">거절 / 정지</UiButton>
          </div>
        </UiCard>
      </div>
    </div>

    <!-- Tab 3: Subscriptions Review -->
    <div v-else-if="!loading" class="space-y-4">
      <div v-if="(queues.subscriptions?.length ?? 0) === 0" class="py-12">
        <UiEmptyState
          title="대기 중인 구독 증빙이 없습니다"
        />
      </div>
      <div v-else class="grid gap-3 sm:grid-cols-2">
        <UiCard v-for="s in queues.subscriptions ?? []" :key="s.id" class="space-y-3">
          <div class="flex items-center justify-between">
            <div class="font-bold text-white text-sm">{{ s.provider }} — {{ s.planName }}</div>
            <UiBadge variant="neutral" size="xs">{{ subStatusKo(s.status) }}</UiBadge>
          </div>
          <div class="font-mono text-[11px] text-neutral-500">증빙 ID: {{ s.id }}</div>
          <div v-if="s.proofObjectKey" class="text-xs text-neutral-300">
            인보이스 파일: <span class="font-mono text-brand-400">{{ s.proofObjectKey }}</span>
          </div>
          <div class="flex items-center gap-2 pt-2 border-t border-surface-border">
            <UiInput
              v-model="subExpiry[s.id]"
              type="date"
            />
            <UiButton variant="success" size="sm" @click="reviewSub(s.id, true)">검증 승인</UiButton>
            <UiButton variant="danger" size="sm" @click="reviewSub(s.id, false)">반려</UiButton>
          </div>
        </UiCard>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { adminApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { usePlayerStore } from '@/stores/playerStore';
import { useToast } from '@/composables/useToast';
import type { ReviewQueues, Track, TrackRightsDeclaration, PartnerAISubscription } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiInput from '@/components/ui/UiInput.vue';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import UiTabs, { type TabOption } from '@/components/ui/UiTabs.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';

const player = usePlayerStore();
const toast = useToast();

const tabOptions: TabOption[] = [
  { label: '대기 음원 심사 (Tracks)', value: 'tracks' },
  { label: '파트너 신청 심사 (Partners)', value: 'partners' },
  { label: '구독 증빙 검증 (Subscriptions)', value: 'subscriptions' },
];

const activeTab = ref('tracks');
const queues = ref<ReviewQueues>({ tracks: [], partners: [], subscriptions: [] });
const loading = ref(true);
const error = ref<string | null>(null);

const PARTNER_STATUS_KO: Record<string, string> = {
  PENDING: '승인 대기',
  APPROVED: '승인됨',
  SUSPENDED: '정지됨',
  TERMINATED: '해지됨',
};

const SUB_STATUS_KO: Record<string, string> = {
  PENDING: '검증 대기',
  VERIFIED: '검증 완료',
  REJECTED: '반려됨',
  EXPIRED: '만료됨',
};

function partnerStatusKo(status: string): string {
  return PARTNER_STATUS_KO[status] ?? status;
}

function subStatusKo(status: string): string {
  return SUB_STATUS_KO[status] ?? status;
}

const trackReason = ref<Record<string, string>>({});
const subExpiry = ref<Record<string, string>>({});
const trackIsrc = ref<Record<string, string>>({});
const isrcBusyId = ref<string | null>(null);

const pendingTracks = computed(() => queues.value.tracks ?? []);

function rightsOf(trackId: string): TrackRightsDeclaration | undefined {
  return queues.value.rightsByTrack?.[trackId];
}

function fmtDate(d?: string): string {
  if (!d) return '-';
  return new Date(d).toLocaleDateString();
}

async function loadAll() {
  loading.value = true;
  error.value = null;
  try {
    const q = await adminApi.pending();
    queues.value = q;
  } catch (err) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
}

function previewTrack(t: Track) {
  player.setQueue([t], t.id);
}

async function reviewTrack(trackId: string, approved: boolean) {
  try {
    await adminApi.reviewTrack(trackId, approved, trackReason.value[trackId]);
    toast.success(approved ? '음원이 승인되어 READY 상태로 전환되었습니다.' : '음원이 반려되었습니다.');
    await loadAll();
  } catch (err) {
    toast.error(errMessageOf(err), '처리 실패');
  }
}

async function saveIsrc(t: Track) {
  const isrc = (trackIsrc.value[t.id] ?? '').trim();
  if (!isrc) {
    toast.error('ISRC를 입력하세요.', 'ISRC 저장 실패');
    return;
  }
  isrcBusyId.value = t.id;
  try {
    await adminApi.setTrackISRC(t.id, isrc);
    toast.success(`ISRC가 저장되었습니다: ${isrc}`);
    trackIsrc.value[t.id] = '';
    await loadAll();
  } catch (err) {
    toast.error(errMessageOf(err), 'ISRC 저장 실패');
  } finally {
    isrcBusyId.value = null;
  }
}

async function reviewPartner(partnerId: string, approved: boolean) {
  try {
    await adminApi.reviewPartner(partnerId, approved);
    toast.success(approved ? '파트너가 승인되었습니다.' : '파트너가 거절되었습니다.');
    await loadAll();
  } catch (err) {
    toast.error(errMessageOf(err), '처리 실패');
  }
}

async function reviewSub(subId: string, verified: boolean) {
  try {
    const exp = subExpiry.value[subId] ? new Date(subExpiry.value[subId]).toISOString() : undefined;
    await adminApi.reviewSubscription(subId, verified, exp);
    toast.success(verified ? '구독 증빙이 검증 승인되었습니다.' : '구독 증빙이 반려되었습니다.');
    await loadAll();
  } catch (err) {
    toast.error(errMessageOf(err), '처리 실패');
  }
}

onMounted(() => {
  void loadAll();
});
</script>
