<template>
  <div class="space-y-6">
    <UiPageHeader
      eyebrow="Rights Protection"
      title="권리보호센터"
      description="저작권법 제103조(복제·전송의 중단) 및 부정경쟁방지법(보이스 클로닝 방지) 전용 창구입니다."
    />

    <!-- Notice & Takedown Blueprint Banner -->
    <UiCard border-variant="brand">
      <div class="flex items-start gap-3">
        <svg class="h-6 w-6 shrink-0 text-brand-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <path d="M12 3v18M5 7l7-4 7 4M5 7v4a3.5 3.5 0 0 0 7 0V7M19 7v4a3.5 3.5 0 0 1-7 0" />
          <path d="M8 21h8" />
        </svg>
        <div class="text-xs text-neutral-300 space-y-1">
          <div class="font-bold text-white text-sm">OSP 책임제한(Notice &amp; Takedown) 즉시 조치 원칙</div>
          <p class="leading-relaxed">
            신고가 접수되면 저작권법 제103조에 따라 해당 AI 음원은 <strong>즉시 복제·전송이 중단</strong>되어 스트리밍 및 검색에서 차단됩니다.
            파트너는 재개 요구(소명)를 제출할 수 있으며 접수 후 7영업일 내 검토를 원칙으로 합니다.
            허위 신고 또는 허위 등록 시 법적 손해배상 책임이 부과될 수 있습니다.
          </p>
        </div>
      </div>
    </UiCard>

    <!-- Report a track -->
    <UiCard>
      <template #header>
        <h2 class="text-xs font-bold text-neutral-300 uppercase tracking-wider">
          권리침해 신고서 작성
        </h2>
      </template>
      <div class="space-y-3">
        <form class="flex flex-col gap-2 sm:flex-row" @submit.prevent="searchTracks">
          <UiInput
            v-model="searchQuery"
            placeholder="신고할 음원 제목으로 검색…"
            class="flex-1"
          />
          <UiButton variant="brand" size="sm" type="submit" :loading="searching">검색</UiButton>
        </form>
        <div v-if="searching" class="space-y-2">
          <UiSkeleton height="h-10" width="w-full" />
          <UiSkeleton height="h-10" width="w-full" />
        </div>
        <ul v-else-if="searchResults.length > 0" class="divide-y divide-surface-border rounded-xl border border-surface-border">
          <li
            v-for="t in searchResults"
            :key="t.id"
            class="flex items-center gap-3 px-3 py-2.5"
          >
            <UiCoverArt :seed="t.id" :title="t.title" size="xs" rounded="md" />
            <div class="min-w-0 flex-1">
              <div class="truncate text-xs font-bold text-white">{{ t.title }}</div>
              <div class="text-[11px] text-neutral-500">{{ t.aiProvider }}</div>
            </div>
            <UiButton variant="outline" size="xs" @click="reportTarget = t">신고하기</UiButton>
          </li>
        </ul>
        <p v-else-if="searched" class="text-xs text-neutral-500">
          검색 결과가 없습니다. 음원 상세 페이지에서도 신고할 수 있습니다.
        </p>
        <p v-else class="text-xs text-neutral-500">
          신고할 음원을 먼저 검색하세요. 음원 상세 페이지에서도 바로 신고할 수 있습니다.
        </p>
      </div>
    </UiCard>

    <!-- Blocked Artists Section (UGC App Store Guideline 1.2 Compliance) -->
    <UiCard>
      <template #header>
        <div class="flex items-center justify-between">
          <h2 class="text-xs font-bold text-neutral-300 uppercase tracking-wider">
            내가 차단한 파트너(아티스트) 목록
          </h2>
          <span class="rounded-full bg-surface-2 px-2 py-0.5 text-[11px] font-mono text-neutral-400">
            {{ blocks.length }}명
          </span>
        </div>
      </template>

      <div v-if="loading" class="space-y-2">
        <UiSkeleton height="h-10" width="w-full" />
        <UiSkeleton height="h-10" width="w-full" />
      </div>

      <div v-else-if="blocks.length === 0" class="py-8">
        <UiEmptyState
          title="차단된 아티스트가 없습니다"
          description="원치 않는 파트너 아티스트를 차단하면 해당 아티스트의 모든 음원이 재생 큐 및 추천에서 자동으로 제외됩니다."
        />
      </div>

      <ul v-else class="divide-y divide-surface-border">
        <li
          v-for="b in blocks"
          :key="b.id"
          class="flex items-center justify-between gap-3 py-3 text-xs"
        >
          <div class="min-w-0">
            <div class="font-bold text-white">
              차단 파트너 <span class="font-mono font-medium text-neutral-400">{{ shortId(b.blockedPartnerId) }}</span>
            </div>
            <div class="text-[11px] text-neutral-400 mt-0.5 truncate">사유: {{ b.reason || '사유 미기재' }}</div>
          </div>
          <UiButton
            variant="secondary"
            size="xs"
            :loading="unblockingId === b.blockedPartnerId"
            @click="unblock(b.blockedPartnerId)"
          >
            차단 해제
          </UiButton>
        </li>
      </ul>
    </UiCard>

    <!-- Report Modal -->
    <DisputeReportModal
      v-if="reportTarget"
      :track-id="reportTarget.id"
      :track-title="reportTarget.title"
      @close="reportTarget = null"
      @reported="onReported"
    />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { trackApi, userBlocksApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { useToast } from '@/composables/useToast';
import type { Track, UserArtistBlock } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiCoverArt from '@/components/ui/UiCoverArt.vue';
import UiInput from '@/components/ui/UiInput.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';
import DisputeReportModal from '@/components/legal/DisputeReportModal.vue';

const toast = useToast();

const blocks = ref<UserArtistBlock[]>([]);
const loading = ref(true);
const unblockingId = ref<string | null>(null);

const searchQuery = ref('');
const searchResults = ref<Track[]>([]);
const searching = ref(false);
const searched = ref(false);
const reportTarget = ref<Track | null>(null);

function shortId(id: string): string {
  return id.length > 13 ? `${id.slice(0, 8)}…` : id;
}

async function loadBlocks() {
  loading.value = true;
  try {
    blocks.value = (await userBlocksApi.listArtistBlocks()) ?? [];
  } catch (err) {
    toast.error(errMessageOf(err), '차단 목록 불러오기 실패');
  } finally {
    loading.value = false;
  }
}

async function searchTracks(): Promise<void> {
  const q = searchQuery.value.trim();
  if (q === '') return;
  searching.value = true;
  searched.value = true;
  try {
    const page = await trackApi.list(10, 0, { q });
    searchResults.value = page.items ?? [];
  } catch (err) {
    toast.error(errMessageOf(err), '음원 검색 실패');
  } finally {
    searching.value = false;
  }
}

async function unblock(partnerId: string) {
  unblockingId.value = partnerId;
  try {
    await userBlocksApi.unblockArtist(partnerId);
    blocks.value = blocks.value.filter((b) => b.blockedPartnerId !== partnerId);
    toast.success('아티스트 차단이 해제되었습니다.');
  } catch (err) {
    toast.error(errMessageOf(err), '해제 실패');
  } finally {
    unblockingId.value = null;
  }
}

function onReported(id: string) {
  reportTarget.value = null;
  toast.success(`신고가 접수되었습니다. (ID: ${id})`);
}

onMounted(() => {
  void loadBlocks();
});
</script>
