<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
      <div>
        <div class="inline-flex items-center gap-1.5 text-xs text-accent-purple font-bold uppercase tracking-wider mb-1">
          <span>Gate 4 Admin Workbench</span>
        </div>
        <h1 class="text-2xl font-black text-white tracking-tight">AI 음원 &amp; 상업 라이선스 최종 심사</h1>
        <p class="text-xs text-neutral-400 mt-1">
          Gate 1~3을 통과한 대기 음원과 파트너 증빙을 대조 심사하여 승인(READY) 또는 반려 처리합니다.
        </p>
      </div>
      <UiButton variant="secondary" size="sm" @click="loadAll">
        새로고침
      </UiButton>
    </div>

    <!-- Error Alert -->
    <div v-if="error" class="rounded-xl border border-red-800 bg-red-950/40 p-4 text-xs text-red-200">
      {{ error }}
    </div>

    <!-- Tabs -->
    <UiTabs v-model="activeTab" :tabs="tabOptions" />

    <!-- Tab 1: Pending Tracks Review -->
    <div v-if="activeTab === 'tracks'" class="space-y-4">
      <div v-if="pendingTracks.length === 0" class="py-12">
        <UiEmptyState
          icon="✓"
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
              ▶ 미리듣기
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
    <div v-else-if="activeTab === 'partners'" class="space-y-4">
      <div v-if="(queues.partners?.length ?? 0) === 0" class="py-12">
        <UiEmptyState
          icon="✓"
          title="대기 중인 파트너 신청이 없습니다"
        />
      </div>
      <div v-else class="grid gap-3 sm:grid-cols-2">
        <UiCard v-for="p in queues.partners ?? []" :key="p.id" class="space-y-3">
          <div class="flex items-center justify-between">
            <div class="font-bold text-white text-sm">{{ p.companyName }}</div>
            <UiBadge variant="warning" size="xs">{{ p.status }}</UiBadge>
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
    <div v-else class="space-y-4">
      <div v-if="(queues.subscriptions?.length ?? 0) === 0" class="py-12">
        <UiEmptyState
          icon="✓"
          title="대기 중인 구독 증빙이 없습니다"
        />
      </div>
      <div v-else class="grid gap-3 sm:grid-cols-2">
        <UiCard v-for="s in queues.subscriptions ?? []" :key="s.id" class="space-y-3">
          <div class="flex items-center justify-between">
            <div class="font-bold text-white text-sm">{{ s.provider }} — {{ s.planName }}</div>
            <UiBadge variant="neutral" size="xs">{{ s.status }}</UiBadge>
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
import UiInput from '@/components/ui/UiInput.vue';
import UiBadge from '@/components/ui/UiBadge.vue';
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
const error = ref<string | null>(null);

const trackReason = ref<Record<string, string>>({});
const subExpiry = ref<Record<string, string>>({});

const pendingTracks = computed(() => queues.value.tracks ?? []);

function rightsOf(trackId: string): TrackRightsDeclaration | undefined {
  return queues.value.rightsByTrack?.[trackId];
}

function fmtDate(d?: string): string {
  if (!d) return '-';
  return new Date(d).toLocaleDateString();
}

async function loadAll() {
  error.value = null;
  try {
    const q = await adminApi.pending();
    queues.value = q;
  } catch (err) {
    error.value = errMessageOf(err);
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
