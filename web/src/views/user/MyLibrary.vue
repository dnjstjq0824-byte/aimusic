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
          <h2 class="text-xs font-bold uppercase tracking-wider text-neutral-300">내 이용권</h2>
          <UiBadge v-if="subscription" variant="verified" size="sm">ACTIVE</UiBadge>
          <UiBadge v-else variant="neutral" size="sm">없음</UiBadge>
        </div>
      </template>
      <div v-if="subLoading" class="space-y-2">
        <UiSkeleton height="h-4" width="w-1/2" />
        <UiSkeleton height="h-3" width="w-1/3" />
      </div>
      <div v-else-if="subscription" class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div class="text-xs text-neutral-300">
          <div class="text-sm font-bold text-white">{{ subscription.plan?.nameKo ?? '이용권' }}</div>
          <div class="mt-1">
            유효기간: {{ fmtDate(subscription.currentPeriodStart) }} ~ {{ fmtDate(subscription.currentPeriodEnd) }}
          </div>
        </div>
        <UiButton variant="outline" size="sm" @click="cancelSubOpen = true">이용권 해지</UiButton>
      </div>
      <div v-else class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <p class="text-xs leading-relaxed text-neutral-400">
          이용권이 없습니다. 리스너 플러스로 고음질 스트리밍과 오프라인 큐를 이용해 보세요.
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
        <span class="rounded-full bg-surface-2 px-2 py-0.5 text-xs text-neutral-400">{{ playlists.playlists.length }}</span>
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
              <p class="mt-1 text-xs text-neutral-400">
                {{ p.isPublic ? '공개 플레이리스트' : '비공개 플레이리스트' }}
              </p>
              <div class="mt-2 text-[11px] text-neutral-500 font-mono">
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
              class="text-xs text-neutral-500 hover:text-accent-crimson transition-colors"
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
      <h3 class="text-sm font-bold text-white">계정 및 보안 정보</h3>
      <div class="grid gap-3 sm:grid-cols-3 text-xs text-neutral-300">
        <div class="p-3 rounded-xl bg-surface-2 border border-surface-border">
          <div class="text-neutral-500 text-[10px] uppercase font-bold">계정 식별자</div>
          <div class="mt-1 font-mono text-neutral-200 truncate">{{ auth.userId }}</div>
        </div>
        <div class="p-3 rounded-xl bg-surface-2 border border-surface-border">
          <div class="text-neutral-500 text-[10px] uppercase font-bold">회원 권한</div>
          <div class="mt-1 font-semibold text-brand-400">{{ auth.role }}</div>
        </div>
        <div class="p-3 rounded-xl bg-surface-2 border border-surface-border">
          <div class="text-neutral-500 text-[10px] uppercase font-bold">인공지능 투명성 고지 동의</div>
          <div class="mt-1"><UiBadge variant="success" size="sm">인증 완료</UiBadge></div>
        </div>
      </div>
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

    <!-- Cancel Subscription Confirm -->
    <UiConfirmDialog
      v-model="cancelSubOpen"
      title="이용권 해지"
      description="이용권을 지금 해지하면 남은 기간 혜택이 즉시 종료됩니다."
      confirm-label="해지하기"
      danger
      :loading="cancellingSub"
      @confirm="confirmCancelSubscription"
    />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { usePlaylistStore } from '@/stores/playlistStore';
import { useAuthStore } from '@/stores/authStore';
import { useToast } from '@/composables/useToast';
import { errMessageOf } from '@/api/client';
import { billingApi } from '@/api/endpoints';
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
import type { BillingSubscription, Playlist } from '@/api/types';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';

const playlists = usePlaylistStore();
const auth = useAuthStore();
const toast = useToast();

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
  cancellingSub.value = true;
  try {
    await billingApi.cancelSubscription();
    subscription.value = null;
    cancelSubOpen.value = false;
    toast.success('이용권을 해지했습니다.');
  } catch (err) {
    toast.error(errMessageOf(err), '해지 실패');
  } finally {
    cancellingSub.value = false;
  }
}

onMounted(() => {
  void playlists.fetchMine();
  void loadSubscription();
});
</script>
