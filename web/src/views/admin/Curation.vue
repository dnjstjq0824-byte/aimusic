<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
      <div>
        <div class="inline-flex items-center gap-1.5 text-xs text-accent-purple font-bold uppercase tracking-wider mb-1">
          <span>Editorial &amp; Financial Operations</span>
        </div>
        <h1 class="text-2xl font-black text-white tracking-tight">공식 큐레이션 &amp; 70:30 정산 운영 콘솔</h1>
        <p class="text-xs text-neutral-400 mt-1">
          공식 테마 플레이리스트를 편성하고, 긴급 직권 임시조치 및 파트너 70:30 음원 정산 배치를 실행합니다.
        </p>
      </div>
    </div>

    <!-- Error Alert -->
    <div v-if="error" class="rounded-xl border border-red-800 bg-red-950/40 p-4 text-xs text-red-200">
      {{ error }}
    </div>

    <div class="grid gap-6 md:grid-cols-2">
      <!-- Create Official Playlist -->
      <UiCard class="space-y-4">
        <div>
          <h2 class="text-sm font-bold text-white">공식 큐레이션 플레이리스트 생성</h2>
          <p class="text-xs text-neutral-400 mt-0.5">홈 배너 및 상단에 노출되는 공식 AI 테마 앨범입니다.</p>
        </div>

        <form class="space-y-3" @submit.prevent="createCuration">
          <UiInput
            v-model="curTitle"
            label="플레이리스트 제목"
            placeholder="예: [Official] 심야 사이버펑크 드라이브"
            required
          />
          <UiInput
            v-model="curDesc"
            label="플레이리스트 설명"
            placeholder="Suno/Udio 상업 검증 일렉트로닉 앤솔로지"
          />
          <UiInput
            v-model="curTracks"
            label="포함할 트랙 UUID 목록 (쉼표로 구분)"
            placeholder="uuid1, uuid2, uuid3…"
          />
          <UiButton variant="brand" size="sm" type="submit" :loading="curCreating">
            공식 큐레이션 등록
          </UiButton>
        </form>
      </UiCard>

      <!-- Emergency Takedown Card -->
      <UiCard class="space-y-4 border-red-500/20 bg-red-950/20">
        <div>
          <h2 class="text-sm font-bold text-accent-crimson flex items-center gap-1.5">
            <span>🚨 긴급 직권 임시조치 (Emergency Takedown)</span>
          </h2>
          <p class="text-xs text-neutral-300 mt-0.5">
            사후 라이선스 취소, 실존 가수 보이스 클로닝 적발 시 즉시 스트리밍을 차단합니다.
          </p>
        </div>

        <form class="space-y-3" @submit.prevent="runTakedown">
          <UiInput
            v-model="tdTrackId"
            label="차단할 대상 트랙 UUID"
            placeholder="00000000-0000-0000-0000-000000000000"
            required
          />
          <UiInput
            v-model="tdReason"
            label="직권 차단 사유"
            placeholder="예: 실존 가수 음성 무단 도용 적발로 즉시 서비스 정지"
            required
          />
          <UiButton variant="danger" size="sm" type="submit" :loading="tdBusy">
            즉시 스트리밍 차단 실행
          </UiButton>
        </form>
      </UiCard>
    </div>

    <!-- Settlement Management Section -->
    <UiCard class="space-y-4">
      <div>
        <div class="inline-flex items-center gap-1.5 text-xs text-brand-400 font-bold uppercase tracking-wider mb-1">
          <span>Settlement Engine (70:30 Rev-Share)</span>
        </div>
        <h2 class="text-sm font-bold text-white">파트너 월별 음원 정산 배치 실행</h2>
        <p class="text-xs text-neutral-400 mt-0.5">
          기간 내 유효 스트리밍(30초 이상) 데이터를 집계하여 원천징수(3.3%)를 차감한 정산 금액을 계산합니다.
        </p>
      </div>

      <form class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4" @submit.prevent="runSettlement">
        <UiInput
          v-model="stPartnerId"
          label="파트너 UUID"
          placeholder="00000000-0000-0000-0000-000000000000"
          required
        />
        <UiInput
          v-model="stPeriodStart"
          label="정산 시작일 (YYYY-MM-DD)"
          type="date"
          required
        />
        <UiInput
          v-model="stPeriodEnd"
          label="정산 종료일 (YYYY-MM-DD)"
          type="date"
          required
        />
        <UiInput
          v-model.number="stRatePerPlay"
          label="1회 스트리밍 단가 (KRW)"
          type="number"
          placeholder="10"
        />

        <div class="sm:col-span-2 lg:col-span-4 flex justify-end">
          <UiButton variant="brand" size="sm" type="submit" :loading="calculating">
            정산 계산 실행
          </UiButton>
        </div>
      </form>

      <!-- Settlement Result Preview -->
      <div v-if="lastSettlement" class="rounded-xl bg-surface-0 p-4 border border-surface-border text-xs space-y-2">
        <div class="flex items-center justify-between">
          <span class="font-bold text-white text-sm">정산 계산 완료 (ID: {{ lastSettlement.id }})</span>
          <span class="text-accent-emerald font-bold">상태: {{ lastSettlement.status }}</span>
        </div>
        <div class="grid grid-cols-2 sm:grid-cols-4 gap-2 pt-2 border-t border-surface-border">
          <div>
            <span class="text-neutral-400 text-[10px] block">유효 스트리밍 수</span>
            <span class="font-bold text-white">{{ lastSettlement.playCount.toLocaleString() }}회</span>
          </div>
          <div>
            <span class="text-neutral-400 text-[10px] block">총 정산액 (Gross)</span>
            <span class="font-bold text-white">{{ lastSettlement.grossAmountKrw.toLocaleString() }}원</span>
          </div>
          <div>
            <span class="text-neutral-400 text-[10px] block">원천징수세 (3.3%)</span>
            <span class="font-bold text-accent-crimson">-{{ lastSettlement.withholdingKrw.toLocaleString() }}원</span>
          </div>
          <div>
            <span class="text-neutral-400 text-[10px] block">실지급 예정액 (Net)</span>
            <span class="font-black text-brand-400 text-sm">{{ lastSettlement.netAmountKrw.toLocaleString() }}원</span>
          </div>
        </div>
      </div>
    </UiCard>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { adminApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { useToast } from '@/composables/useToast';
import type { PartnerSettlement } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiInput from '@/components/ui/UiInput.vue';

const toast = useToast();

const curTitle = ref('');
const curDesc = ref('');
const curTracks = ref('');
const curCreating = ref(false);

const tdTrackId = ref('');
const tdReason = ref('');
const tdBusy = ref(false);

const stPartnerId = ref('');
const stPeriodStart = ref('2026-10-01');
const stPeriodEnd = ref('2026-10-31');
const stRatePerPlay = ref(10);
const lastSettlement = ref<PartnerSettlement | null>(null);
const calculating = ref(false);
const error = ref<string | null>(null);

async function createCuration() {
  if (!curTitle.value.trim()) return;
  curCreating.value = true;
  try {
    const trackIds = curTracks.value.split(',').map((s) => s.trim()).filter(Boolean);
    await adminApi.createCurationPlaylist(curTitle.value.trim(), curDesc.value.trim() || undefined, trackIds);
    toast.success('공식 큐레이션 플레이리스트가 등록되었습니다.');
    curTitle.value = '';
    curDesc.value = '';
    curTracks.value = '';
  } catch (err) {
    toast.error(errMessageOf(err), '등록 실패');
  } finally {
    curCreating.value = false;
  }
}

async function runTakedown() {
  if (!tdTrackId.value.trim() || !tdReason.value.trim()) return;
  tdBusy.value = true;
  try {
    await adminApi.takedown(tdTrackId.value.trim(), tdReason.value.trim());
    toast.warning('해당 트랙이 즉시 임시조치되어 스트리밍에서 차단되었습니다.');
    tdTrackId.value = '';
    tdReason.value = '';
  } catch (err) {
    toast.error(errMessageOf(err), '차단 실패');
  } finally {
    tdBusy.value = false;
  }
}

async function runSettlement() {
  if (!stPartnerId.value.trim()) {
    toast.error('파트너 UUID를 입력해주세요.');
    return;
  }
  calculating.value = true;
  try {
    lastSettlement.value = await adminApi.calculateSettlement(
      stPartnerId.value.trim(),
      stPeriodStart.value,
      stPeriodEnd.value,
      stRatePerPlay.value || 10,
    );
    toast.success('월별 정산 계산 배치가 성공적으로 완료되었습니다.');
  } catch (err) {
    toast.error(errMessageOf(err), '정산 계산 실패');
  } finally {
    calculating.value = false;
  }
}
</script>
