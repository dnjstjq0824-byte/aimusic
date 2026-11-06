<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
      <div>
        <h1 class="text-2xl font-black text-white tracking-tight">내 라이브러리</h1>
        <p class="text-xs text-neutral-400 mt-1">내가 생성한 플레이리스트와 저장한 음악을 관리합니다.</p>
      </div>
      <UiButton variant="brand" size="sm" @click="createModalOpen = true">
        + 새 플레이리스트 만들기
      </UiButton>
    </div>

    <!-- Playlists section -->
    <div>
      <h2 class="text-sm font-bold text-white mb-3 flex items-center gap-2">
        <span>내 플레이리스트</span>
        <span class="rounded-full bg-surface-2 px-2 py-0.5 text-xs text-neutral-400">{{ playlists.playlists.length }}</span>
      </h2>

      <div v-if="playlists.playlists.length === 0" class="py-8">
        <UiEmptyState
          icon="🎶"
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
            <div class="flex h-14 w-14 shrink-0 items-center justify-center rounded-xl bg-gradient-to-tr from-surface-2 to-surface-3 border border-surface-border text-2xl">
              🎵
            </div>
            <div class="min-w-0 flex-1">
              <RouterLink
                :to="{ name: 'playlist-detail', params: { id: p.id } }"
                class="font-bold text-white hover:text-brand-400 text-sm block truncate"
              >
                {{ p.title }}
              </RouterLink>
              <p class="mt-0.5 text-xs text-neutral-400 line-clamp-1">
                {{ p.isPublic ? '공개 플레이리스트' : '비공개 플레이리스트' }}
              </p>
              <div class="mt-2 text-[11px] text-neutral-500 font-mono">
                생성일: {{ new Date(p.createdAt).toLocaleDateString() }}
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
              class="text-xs text-neutral-500 hover:text-accent-crimson transition-colors"
              @click="deletePlaylist(p.id)"
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
          <div class="mt-1 text-accent-emerald font-semibold">✓ 인증 완료</div>
        </div>
      </div>
    </UiCard>

    <!-- Create Playlist Modal -->
    <UiModal
      v-model="createModalOpen"
      title="새 플레이리스트 생성"
      description="원하는 타이틀과 소개글을 입력하세요."
      @close="createModalOpen = false"
    >
      <form class="space-y-4" @submit.prevent="submitCreate">
        <UiInput
          v-model="newTitle"
          label="플레이리스트 이름"
          placeholder="예: 심야 코딩용 Lo-Fi Synth"
          required
        />
        <UiInput
          v-model="newDesc"
          label="설명 (선택)"
          placeholder="이 플레이리스트에 담길 음악 분위기"
        />
        <div class="flex justify-end gap-2 pt-2">
          <UiButton variant="ghost" size="sm" @click="createModalOpen = false">취소</UiButton>
          <UiButton variant="brand" size="sm" type="submit" :loading="creating">생성하기</UiButton>
        </div>
      </form>
    </UiModal>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { usePlaylistStore } from '@/stores/playlistStore';
import { useAuthStore } from '@/stores/authStore';
import { useToast } from '@/composables/useToast';
import { errMessageOf } from '@/api/client';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiInput from '@/components/ui/UiInput.vue';
import UiModal from '@/components/ui/UiModal.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';

const playlists = usePlaylistStore();
const auth = useAuthStore();
const toast = useToast();

const createModalOpen = ref(false);
const newTitle = ref('');
const newDesc = ref('');
const creating = ref(false);

async function submitCreate() {
  if (!newTitle.value.trim()) return;
  creating.value = true;
  try {
    await playlists.create(newTitle.value.trim(), false);
    toast.success('플레이리스트가 생성되었습니다.');
    newTitle.value = '';
    newDesc.value = '';
    createModalOpen.value = false;
  } catch (err) {
    toast.error(errMessageOf(err), '생성 실패');
  } finally {
    creating.value = false;
  }
}

async function deletePlaylist(id: string) {
  if (!confirm('정말 이 플레이리스트를 삭제하시겠습니까?')) return;
  try {
    await playlists.remove(id);
    toast.success('플레이리스트를 삭제했습니다.');
  } catch (err) {
    toast.error(errMessageOf(err), '삭제 실패');
  }
}

onMounted(() => {
  void playlists.fetchMine();
});
</script>
