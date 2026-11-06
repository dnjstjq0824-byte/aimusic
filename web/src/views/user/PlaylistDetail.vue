<template>
  <div class="space-y-6">
    <RouterLink to="/my-library" class="inline-flex items-center gap-1.5 text-xs text-neutral-400 hover:text-white transition-colors">
      <span>← 내 라이브러리로 돌아가기</span>
    </RouterLink>

    <div v-if="error" class="rounded-xl border border-red-800 bg-red-950/40 p-4 text-xs text-red-200">
      {{ error }}
    </div>

    <div v-else-if="loading" class="space-y-4">
      <UiSkeleton height="h-32" width="w-full" rounded="xl" />
      <UiSkeleton height="h-64" width="w-full" rounded="xl" />
    </div>

    <div v-else-if="playlist" class="space-y-6">
      <!-- Playlist Header Hero -->
      <UiCard class="relative overflow-hidden">
        <div class="flex flex-col sm:flex-row sm:items-center gap-5">
          <div class="flex h-24 w-24 shrink-0 items-center justify-center rounded-2xl bg-gradient-to-tr from-brand-600 via-brand-500 to-amber-500 text-white text-4xl shadow-glow-brand">
            🎵
          </div>
          <div class="min-w-0 flex-1">
            <span class="text-[10px] font-bold uppercase tracking-wider text-brand-400">플레이리스트</span>
            <h1 class="text-2xl sm:text-3xl font-black text-white tracking-tight truncate mt-0.5">
              {{ playlist.title }}
            </h1>
            <p class="text-xs text-neutral-400 mt-1">
              {{ playlist.isPublic ? '공개 플레이리스트' : '비공개 플레이리스트' }}
            </p>
            <div class="mt-2 flex items-center gap-3 text-xs text-neutral-400">
              <span>총 <strong>{{ tracks.length }}곡</strong></span>
              <span>·</span>
              <span>재생 큐 연동 완료</span>
            </div>
          </div>
        </div>

        <div class="mt-6 flex flex-wrap items-center gap-2 pt-4 border-t border-surface-border">
          <UiButton
            variant="brand"
            size="sm"
            :disabled="tracks.length === 0"
            @click="playAll(false)"
          >
            ▶ 전체 재생
          </UiButton>
          <UiButton
            variant="secondary"
            size="sm"
            :disabled="tracks.length === 0"
            @click="playAll(true)"
          >
            🔀 셔플 재생
          </UiButton>
          <UiButton
            variant="outline"
            size="sm"
            @click="editModalOpen = true"
          >
            정보 수정
          </UiButton>
        </div>
      </UiCard>

      <!-- Track List Table -->
      <UiCard padding="none">
        <template #header>
          <div class="px-5 pt-3 flex items-center justify-between text-xs font-bold text-neutral-400 uppercase tracking-wider">
            <span>트랙 목록 (순서 및 재생)</span>
            <span>순서 조정</span>
          </div>
        </template>

        <div v-if="tracks.length === 0" class="py-12">
          <UiEmptyState
            icon="🎶"
            title="플레이리스트가 비어 있습니다"
            description="음원 탐색 화면에서 좋아하는 AI 음악을 담아보세요."
          >
            <template #action>
              <RouterLink to="/explore">
                <UiButton variant="brand" size="sm">음원 탐색하러 가기</UiButton>
              </RouterLink>
            </template>
          </UiEmptyState>
        </div>

        <ul v-else class="divide-y divide-surface-border">
          <li
            v-for="(t, idx) in tracks"
            :key="t.id"
            class="flex items-center gap-3 px-4 py-3 hover:bg-surface-2/60 transition-colors"
            :class="t.status !== 'READY' ? 'opacity-50' : ''"
          >
            <!-- Track index / play -->
            <button
              class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-surface-2 text-xs font-bold text-neutral-300 hover:bg-brand-500 hover:text-white transition-colors"
              :disabled="t.status !== 'READY'"
              @click="playTrack(t)"
            >
              {{ idx + 1 }}
            </button>

            <!-- Title & Provider -->
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2">
                <RouterLink
                  :to="{ name: 'track-detail', params: { id: t.id } }"
                  class="font-bold text-white hover:text-brand-400 text-xs sm:text-sm truncate"
                >
                  {{ t.title }}
                </RouterLink>
                <AiBadge :track="t" />
                <span v-if="t.status !== 'READY'" class="text-[10px] text-accent-crimson font-semibold">
                  (임시조치/검수중)
                </span>
              </div>
              <div class="text-[11px] text-neutral-500">
                {{ t.aiProvider }} · {{ fmtDuration(t.durationMs ?? 0) }}
              </div>
            </div>

            <!-- Position controls & delete -->
            <div class="flex items-center gap-1 text-xs">
              <button
                class="rounded p-1 text-neutral-400 hover:text-white hover:bg-surface-3 disabled:opacity-30"
                :disabled="idx === 0"
                title="위로 이동"
                @click="moveTrack(idx, -1)"
              >
                ▲
              </button>
              <button
                class="rounded p-1 text-neutral-400 hover:text-white hover:bg-surface-3 disabled:opacity-30"
                :disabled="idx === tracks.length - 1"
                title="아래로 이동"
                @click="moveTrack(idx, 1)"
              >
                ▼
              </button>
              <button
                class="rounded p-1 text-neutral-500 hover:text-accent-crimson hover:bg-surface-3 ml-2"
                title="플레이리스트에서 삭제"
                @click="removeTrack(t.id)"
              >
                ✕
              </button>
            </div>
          </li>
        </ul>
      </UiCard>
    </div>

    <!-- Edit Info Modal -->
    <UiModal
      v-model="editModalOpen"
      title="플레이리스트 정보 수정"
      @close="editModalOpen = false"
    >
      <form v-if="playlist" class="space-y-4" @submit.prevent="saveEdit">
        <UiInput
          v-model="editTitle"
          label="제목"
          required
        />
        <UiCheckbox v-model="editIsPublic">
          공개 플레이리스트로 설정 (다른 사용자에게 공유 가능)
        </UiCheckbox>
        <div class="flex justify-end gap-2 pt-2">
          <UiButton variant="ghost" size="sm" @click="editModalOpen = false">취소</UiButton>
          <UiButton variant="brand" size="sm" type="submit" :loading="saving">저장하기</UiButton>
        </div>
      </form>
    </UiModal>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useRoute } from 'vue-router';
import { playlistApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { usePlayerStore } from '@/stores/playerStore';
import { usePlaylistStore } from '@/stores/playlistStore';
import { useToast } from '@/composables/useToast';
import type { Playlist, Track } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiInput from '@/components/ui/UiInput.vue';
import UiCheckbox from '@/components/ui/UiCheckbox.vue';
import UiModal from '@/components/ui/UiModal.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';
import AiBadge from '@/components/legal/AiBadge.vue';

const route = useRoute();
const player = usePlayerStore();
const playlists = usePlaylistStore();
const toast = useToast();

const playlist = ref<Playlist | null>(null);
const tracks = ref<Track[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);

const editModalOpen = ref(false);
const editTitle = ref('');
const editIsPublic = ref(false);
const saving = ref(false);

function fmtDuration(ms?: number | null) {
  if (!ms) return '0:00';
  const totalSec = Math.floor(ms / 1000);
  const min = Math.floor(totalSec / 60);
  const sec = totalSec % 60;
  return `${min}:${sec < 10 ? '0' : ''}${sec}`;
}

async function loadData() {
  const id = route.params.id as string;
  if (!id) return;
  loading.value = true;
  error.value = null;
  try {
    playlist.value = await playlistApi.get(id);
    await playlists.fetchTracks(id);
    tracks.value = playlists.entryTracks[id] ?? [];
    editTitle.value = playlist.value.title;
    editIsPublic.value = playlist.value.isPublic;
  } catch (err) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
}

function playTrack(t: Track) {
  const ready = tracks.value.filter((tr) => tr.status === 'READY');
  player.setQueue(ready, t.id);
}

function playAll(shuffle: boolean) {
  const ready = tracks.value.filter((tr) => tr.status === 'READY');
  if (ready.length === 0) return;
  let q = [...ready];
  if (shuffle) {
    q = q.sort(() => Math.random() - 0.5);
  }
  player.setQueue(q, q[0]?.id);
  toast.success(shuffle ? '셔플 재생을 시작합니다.' : '순차 재생을 시작합니다.');
}

async function removeTrack(trackId: string) {
  if (!playlist.value) return;
  try {
    await playlistApi.removeTrack(playlist.value.id, trackId);
    tracks.value = tracks.value.filter((t) => t.id !== trackId);
    toast.success('트랙이 제거되었습니다.');
  } catch (err) {
    toast.error(errMessageOf(err), '제거 실패');
  }
}

async function moveTrack(idx: number, delta: number) {
  if (!playlist.value) return;
  const targetIdx = idx + delta;
  if (targetIdx < 0 || targetIdx >= tracks.value.length) return;

  const temp = tracks.value[idx];
  tracks.value[idx] = tracks.value[targetIdx];
  tracks.value[targetIdx] = temp;

  try {
    await playlistApi.reorder(playlist.value.id, tracks.value.map((t) => t.id));
    toast.info('순서가 저장되었습니다.');
  } catch (err) {
    toast.error(errMessageOf(err), '순서 저장 실패');
  }
}

async function saveEdit() {
  if (!playlist.value || !editTitle.value.trim()) return;
  saving.value = true;
  try {
    const updated = await playlistApi.update(playlist.value.id, editTitle.value.trim(), editIsPublic.value);
    playlist.value = updated;
    editModalOpen.value = false;
    toast.success('플레이리스트 정보가 수정되었습니다.');
  } catch (err) {
    toast.error(errMessageOf(err), '수정 실패');
  } finally {
    saving.value = false;
  }
}

onMounted(() => {
  void loadData();
});
</script>
