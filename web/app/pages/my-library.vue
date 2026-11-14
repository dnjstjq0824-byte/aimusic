<template>
  <div class="space-y-6">
    <UiPageHeader
      eyebrow="Library"
      title="내 라이브러리"
      description="내가 생성한 플레이리스트와 저장한 음악, 이용권을 관리합니다."
    >
      <template #actions>
        <UiButton variant="brand" size="sm" @click="createModalOpen = true">
          새 플레이리스트 만들기
        </UiButton>
      </template>
    </UiPageHeader>

    <!-- Subscription pass -->
    <UiCard>
      <template #header>
        <div class="flex items-center justify-between">
          <h2 class="text-xs font-bold uppercase tracking-wider text-text-primary">내 이용권</h2>
          <UiBadge v-if="subscription" variant="verified" size="sm">ACTIVE</UiBadge>
          <UiBadge v-else variant="neutral" size="sm">없음</UiBadge>
        </div>
      </template>
      <div v-if="subLoading" class="space-y-2">
        <UiSkeleton height="h-4" width="w-1/2" />
        <UiSkeleton height="h-3" width="w-1/3" />
      </div>
      <div v-else-if="subscription" class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div class="text-xs text-text-primary">
          <div class="text-sm font-bold text-white">{{ subscription.plan?.nameKo ?? '이용권' }}</div>
          <div class="mt-1">
            유효기간: {{ fmtDate(subscription.currentPeriodStart) }} ~ {{ fmtDate(subscription.currentPeriodEnd) }}
          </div>
          <div v-if="subscription.cancelRequestedAt" class="mt-1 font-semibold text-amber-300">
            해지 예정: {{ fmtDate(subscription.currentPeriodEnd) }}에 종료됩니다.
          </div>
        </div>
        <div class="flex flex-col gap-2 sm:items-end">
          <UiButton variant="outline" size="sm" @click="openCancel">이용권 해지</UiButton>
        </div>
      </div>
      <div v-else class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <p class="text-xs leading-relaxed text-text-secondary">
          이용권이 없습니다. 리스너 플러스로 HQ 고음질 스트리밍을 이용해 보세요.
        </p>
        <RouterLink to="/pricing">
          <UiButton variant="brand" size="sm">요금제 보기</UiButton>
        </RouterLink>
      </div>
    </UiCard>

    <!-- Playlists section -->
    <div>
      <h2 class="text-sm font-bold text-white mb-3 flex items-center gap-2">
        <span>내 플레이리스트</span>
        <span class="rounded-full bg-surface-2 px-2 py-0.5 text-xs text-text-secondary">{{ playlists.playlists.length }}</span>
      </h2>

      <div v-if="playlists.playlists.length === 0" class="py-8">
        <UiEmptyState
          title="아직 생성된 플레이리스트가 없습니다"
          description="나만의 분위기와 취향에 맞는 AI 음악 플레이리스트를 만들어 보세요."
        >
          <template #action>
            <UiButton variant="brand" size="sm" @click="createModalOpen = true">
              첫 플레이리스트 만들기
            </UiButton>
          </template>
        </UiEmptyState>
      </div>

      <div v-else class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        <UiCard
          v-for="p in playlists.playlists"
          :key="p.id"
          hoverable
          class="group flex flex-col justify-between"
        >
          <div class="flex items-start gap-3">
            <UiCoverArt :seed="p.id" :title="p.title" size="lg" rounded="lg" />
            <div class="min-w-0 flex-1">
              <RouterLink
                :to="{ name: 'playlist-detail', params: { id: p.id } }"
                class="font-bold text-white hover:text-brand-400 text-sm block truncate"
              >
                {{ p.title }}
              </RouterLink>
              <p class="mt-1 text-xs text-text-secondary">
                {{ p.isPublic ? '공개 플레이리스트' : '비공개 플레이리스트' }}
              </p>
              <div class="mt-2 text-caption text-text-muted font-mono">
                생성일: {{ new Date(p.createdAt).toLocaleDateString('ko-KR') }}
              </div>
            </div>
          </div>
          <div class="mt-4 flex items-center justify-between border-t border-surface-border pt-3">
            <RouterLink
              :to="{ name: 'playlist-detail', params: { id: p.id } }"
              class="text-xs font-semibold text-brand-400 hover:underline"
            >
              상세보기 &amp; 편집
            </RouterLink>
            <button
              type="button"
              class="text-xs text-text-muted hover:text-accent-crimson transition-colors"
              :aria-label="`「${p.title}」 플레이리스트 삭제`"
              @click="deleteTarget = p"
            >
              삭제
            </button>
          </div>
        </UiCard>
      </div>
    </div>

    <!-- User Account Summary & Legal Consents Card -->
    <UiCard class="space-y-4">
      <div class="flex items-center justify-between">
        <h3 class="text-sm font-bold text-white">계정 및 보안 정보</h3>
        <UiButton variant="danger" size="sm" @click="withdrawOpen = true">회원 탈퇴</UiButton>
      </div>
      <div class="grid gap-3 sm:grid-cols-3 text-xs text-text-primary">
        <div class="p-3 rounded-xl bg-surface-2 border border-surface-border">
          <div class="text-text-muted text-caption uppercase font-bold">계정 식별자</div>
          <div class="mt-1 font-mono text-text-primary truncate">{{ auth.userId }}</div>
        </div>
        <div class="p-3 rounded-xl bg-surface-2 border border-surface-border">
          <div class="text-text-muted text-caption uppercase font-bold">회원 권한</div>
          <div class="mt-1 font-semibold text-brand-400">{{ auth.role }}</div>
        </div>
        <div class="p-3 rounded-xl bg-surface-2 border border-surface-border">
          <div class="text-text-muted text-caption uppercase font-bold">인공지능 투명성 고지 동의</div>
          <div class="mt-1"><UiBadge variant="success" size="sm">인증 완료</UiBadge></div>
        </div>
      </div>
      <p class="text-caption leading-relaxed text-text-muted">
        회원 탈퇴는 즉시 처리되며, 세무·법정 보존 의무 항목을 제외한 개인식별 정보는 파기됩니다.
      </p>
    </UiCard>

    <!-- Create Playlist Modal -->
    <UiModal
      v-model="createModalOpen"
      title="새 플레이리스트 생성"
      description="원하는 타이틀과 공개 범위를 설정하세요."
      @close="createModalOpen = false"
    >
      <form class="space-y-4" @submit.prevent="submitCreate">
        <UiInput
          v-model="newTitle"
          label="플레이리스트 이름"
          placeholder="예: 심야 코딩용 Lo-Fi Synth"
          :maxlength="80"
          required
        />
        <UiCheckbox v-model="newIsPublic">
          공개 플레이리스트로 생성 (다른 사용자에게 공유 가능)
        </UiCheckbox>
        <div class="flex justify-end gap-2 pt-2">
          <UiButton variant="ghost" size="sm" @click="createModalOpen = false">취소</UiButton>
          <UiButton variant="brand" size="sm" type="submit" :loading="creating">생성하기</UiButton>
        </div>
      </form>
    </UiModal>

    <!-- Delete Playlist Confirm -->
    <UiConfirmDialog
      :model-value="deleteTarget !== null"
      title="플레이리스트 삭제"
      :description="deleteTarget ? `「${deleteTarget.title}」을(를) 삭제합니다.` : ''"
      body="삭제된 플레이리스트는 복구할 수 없습니다. 정말 삭제하시겠습니까?"
      confirm-label="삭제하기"
      danger
      :loading="deleting"
      @update:model-value="onDeleteDialogToggle"
      @confirm="confirmDeletePlaylist"
    />

    <!-- Cancel pass: the customer must choose 일반해지 or 즉시 해지 (2026-11-12 refund policy) -->
    <UiModal
      v-model="cancelSubOpen"
      title="이용권 해지"
      description="해지 방식을 선택해 주세요. 선택하기 전에는 해지되지 않습니다."
      @close="cancelSubOpen = false"
    >
      <div v-if="quoteLoading" class="space-y-2">
        <UiSkeleton height="h-14" />
        <UiSkeleton height="h-14" />
      </div>
      <div v-else-if="quote && subscription" class="space-y-3" role="radiogroup" aria-label="해지 방식">
        <label
          class="flex cursor-pointer gap-3 rounded-xl border p-4 text-sm"
          :class="cancelMode === 'END_OF_PERIOD' ? 'border-brand-400 bg-brand-500/10' : 'border-surface-border'"
        >
          <input v-model="cancelMode" type="radio" value="END_OF_PERIOD" class="mt-1" :disabled="quote.cancelRequested" />
          <span>
            <span class="block font-bold text-white">일반해지</span>
            <span class="block text-text-primary">
              {{ fmtDate(subscription.currentPeriodEnd) }}까지 이용하고 종료합니다. 환불은 없습니다.
            </span>
            <span v-if="quote.cancelRequested" class="block text-amber-300">이미 일반해지를 신청했습니다.</span>
          </span>
        </label>
        <label
          class="flex cursor-pointer gap-3 rounded-xl border p-4 text-sm"
          :class="cancelMode === 'IMMEDIATE' ? 'border-brand-400 bg-brand-500/10' : 'border-surface-border'"
        >
          <input v-model="cancelMode" type="radio" value="IMMEDIATE" class="mt-1" />
          <span>
            <span class="block font-bold text-white">즉시 해지(환불)</span>
            <span class="block text-text-primary">{{ immediateText }}</span>
            <span class="block text-text-secondary">이용권은 지금 바로 종료됩니다. 환불은 결제 수단에 따라 3~5영업일 안에 반환됩니다.</span>
          </span>
        </label>
      </div>
      <p v-else class="text-sm text-text-primary">환불 정보를 불러오지 못했습니다. 잠시 후 다시 시도해 주세요.</p>
      <template #footer>
        <div class="flex justify-end gap-2">
          <UiButton variant="secondary" size="md" @click="cancelSubOpen = false">닫기</UiButton>
          <UiButton
            variant="danger"
            size="md"
            :disabled="!cancelMode || !quote"
            :loading="cancellingSub"
            @click="confirmCancelSubscription"
          >
            {{ cancelMode === 'IMMEDIATE' ? '즉시 해지하기' : '해지 신청하기' }}
          </UiButton>
        </div>
      </template>
    </UiModal>

    <!-- Withdraw Account Confirm -->
    <UiConfirmDialog
      v-model="withdrawOpen"
      title="회원 탈퇴"
      description="탈퇴 시 계정이 즉시 비활성화되고 다시 로그인할 수 없습니다."
      body="플레이리스트와 이용권 혜택도 함께 종료됩니다. 정말 탈퇴하시겠습니까?"
      confirm-label="탈퇴하기"
      danger
      :loading="withdrawing"
      @confirm="confirmWithdraw"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { usePlaylistStore } from '@/stores/playlistStore';
import { useAuthStore } from '@/stores/authStore';
import { useToast } from '@/composables/useToast';
import { errMessageOf } from '@/api/client';
import { authApi, billingApi } from '@/api/endpoints';
import UiButton from '@/components/ui/UiButton.vue';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiCheckbox from '@/components/ui/UiCheckbox.vue';
import UiCoverArt from '@/components/ui/UiCoverArt.vue';
import UiInput from '@/components/ui/UiInput.vue';
import UiModal from '@/components/ui/UiModal.vue';
import UiConfirmDialog from '@/components/ui/UiConfirmDialog.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import type { BillingSubscription, CancelMode, Playlist, RefundQuote } from '@/api/types';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';

const playlists = usePlaylistStore();
const auth = useAuthStore();
const toast = useToast();
const router = useRouter();

const createModalOpen = ref(false);
const newTitle = ref('');
const newIsPublic = ref(false);
const creating = ref(false);
const deleteTarget = ref<Playlist | null>(null);
const deleting = ref(false);

const subscription = ref<BillingSubscription | null>(null);
const subLoading = ref(true);
const cancelSubOpen = ref(false);
const cancellingSub = ref(false);
const quote = ref<RefundQuote | null>(null);
const quoteLoading = ref(false);
// No default: the customer must pick one (KFTC 2024 Melon case).
const cancelMode = ref<CancelMode | null>(null);

const won = (n: number): string => n.toLocaleString('ko-KR') + '원';
const immediateText = computed(() => {
  const q = quote.value;
  if (!q) return '';
  if (q.type === 'WITHDRAWAL') return '결제 후 7일 이내 미이용으로 전액 ' + won(q.refundKrw) + '을 환불합니다.';
  if (q.refundKrw === 0) return '이용 ' + q.usedDays + '일분을 빼면 환불 금액이 없습니다. 이용권만 즉시 종료됩니다.';
  const deduct = q.amountKrw - q.refundKrw;
  return '이용 ' + q.usedDays + '/' + q.totalDays + '일분 ' + won(deduct) + '을 뺀 ' + won(q.refundKrw) + '을 환불합니다.';
});

async function openCancel(): Promise<void> {
  cancelMode.value = null;
  quote.value = null;
  cancelSubOpen.value = true;
  quoteLoading.value = true;
  try {
    quote.value = await billingApi.refundQuote();
  } catch (err) {
    toast.error(errMessageOf(err), '환불 정보 조회 실패');
  } finally {
    quoteLoading.value = false;
  }
}
const withdrawOpen = ref(false);
const withdrawing = ref(false);

function onDeleteDialogToggle(open: boolean): void {
  if (!open) deleteTarget.value = null;
}

function fmtDate(iso: string): string {
  return new Date(iso).toLocaleDateString('ko-KR');
}

async function submitCreate() {
  if (!newTitle.value.trim()) return;
  creating.value = true;
  try {
    await playlists.create(newTitle.value.trim(), newIsPublic.value);
    toast.success('플레이리스트가 생성되었습니다.');
    newTitle.value = '';
    newIsPublic.value = false;
    createModalOpen.value = false;
  } catch (err) {
    toast.error(errMessageOf(err), '생성 실패');
  } finally {
    creating.value = false;
  }
}

async function confirmDeletePlaylist() {
  if (!deleteTarget.value) return;
  deleting.value = true;
  try {
    await playlists.remove(deleteTarget.value.id);
    toast.success('플레이리스트를 삭제했습니다.');
    deleteTarget.value = null;
  } catch (err) {
    toast.error(errMessageOf(err), '삭제 실패');
  } finally {
    deleting.value = false;
  }
}

async function loadSubscription(): Promise<void> {
  subLoading.value = true;
  try {
    const res = await billingApi.mySubscription();
    subscription.value = res.subscription;
  } catch (err) {
    toast.error(errMessageOf(err), '이용권 조회 실패');
  } finally {
    subLoading.value = false;
  }
}

async function confirmCancelSubscription(): Promise<void> {
  if (!cancelMode.value) return;
  cancellingSub.value = true;
  try {
    const res = await billingApi.cancelSubscription(cancelMode.value, '이용자 해지 요청');
    cancelSubOpen.value = false;
    if (res.mode === 'IMMEDIATE') {
      subscription.value = null;
      const refunded = res.quote?.refundKrw ?? 0;
      toast.success(refunded > 0 ? '즉시 해지했습니다. ' + won(refunded) + '이 환불됩니다.' : '즉시 해지했습니다.');
    } else if (subscription.value && res.subscription) {
      subscription.value = { ...subscription.value, cancelRequestedAt: res.subscription.cancelRequestedAt };
      toast.success('일반해지를 신청했습니다. 만료일까지 이용할 수 있습니다.');
    }
  } catch (err) {
    toast.error(errMessageOf(err), '해지 실패');
  } finally {
    cancellingSub.value = false;
  }
}

async function confirmWithdraw(): Promise<void> {
  withdrawing.value = true;
  try {
    await authApi.withdraw();
    auth.logout();
    withdrawOpen.value = false;
    toast.success('회원 탈퇴가 완료되었습니다.');
    await router.push('/');
  } catch (err) {
    toast.error(errMessageOf(err), '탈퇴 실패');
  } finally {
    withdrawing.value = false;
  }
}

onMounted(() => {
  void playlists.fetchMine();
  void loadSubscription();
});
</script>
