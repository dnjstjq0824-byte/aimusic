<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
      <div>
        <h1 class="text-2xl font-black text-white tracking-tight">권리보호센터 (OSP 분쟁 관리)</h1>
        <p class="text-xs text-neutral-400 mt-1">
          저작권법 제103조(온라인서비스제공자의 책임제한) 및 부정경쟁방지법(보이스 클로닝 방지) 전용 창구입니다.
        </p>
      </div>
      <UiButton variant="brand" size="sm" @click="reportModalOpen = true">
        + 권리침해 신고서 작성
      </UiButton>
    </div>

    <!-- Notice & Takedown Blueprint Banner -->
    <UiCard class="border-brand-500/20 bg-brand-950/20">
      <div class="flex items-start gap-3">
        <span class="text-xl">⚖️</span>
        <div class="text-xs text-neutral-300 space-y-1">
          <div class="font-bold text-white text-sm">OSP 책임제한(Notice &amp; Takedown) 즉시 조치 원칙</div>
          <p class="leading-relaxed">
            신고가 접수되면 저작권법 제103조에 따라 해당 AI 음원은 <strong>즉시 임시조치(TAKEDOWN_TEMP)</strong>되어 스트리밍 및 검색에서 차단됩니다. 파트너에게 30일 이내 소명 기회가 주어지며, 허위 신고 또는 허위 등록 시 법적 손해배상 책임이 부과될 수 있습니다.
          </p>
        </div>
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
          icon="🛡️"
          title="차단된 아티스트가 없습니다"
          description="원치 않는 파트너 아티스트를 차단하면 해당 아티스트의 모든 음원이 재생 큐 및 추천에서 자동으로 제외됩니다."
        />
      </div>

      <ul v-else class="divide-y divide-surface-border">
        <li
          v-for="b in blocks"
          :key="b.id"
          class="flex items-center justify-between py-3 text-xs"
        >
          <div>
            <div class="font-bold text-white">파트너 ID: {{ b.blockedPartnerId }}</div>
            <div class="text-[11px] text-neutral-400 mt-0.5">사유: {{ b.reason }}</div>
          </div>
          <UiButton
            variant="secondary"
            size="xs"
            @click="unblock(b.blockedPartnerId)"
          >
            차단 해제
          </UiButton>
        </li>
      </ul>
    </UiCard>

    <!-- Report Modal -->
    <DisputeReportModal
      v-if="reportModalOpen"
      track-id=""
      track-title=""
      @close="reportModalOpen = false"
      @reported="onReported"
    />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { userBlocksApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { useToast } from '@/composables/useToast';
import type { UserArtistBlock } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';
import DisputeReportModal from '@/components/legal/DisputeReportModal.vue';

const toast = useToast();

const blocks = ref<UserArtistBlock[]>([]);
const loading = ref(true);
const reportModalOpen = ref(false);

async function loadBlocks() {
  loading.value = true;
  try {
    blocks.value = await userBlocksApi.listArtistBlocks();
  } catch (err) {
    toast.error(errMessageOf(err), '차단 목록 불러오기 실패');
  } finally {
    loading.value = false;
  }
}

async function unblock(partnerId: string) {
  try {
    await userBlocksApi.unblockArtist(partnerId);
    blocks.value = blocks.value.filter((b) => b.blockedPartnerId !== partnerId);
    toast.success('아티스트 차단이 해제되었습니다.');
  } catch (err) {
    toast.error(errMessageOf(err), '해제 실패');
  }
}

function onReported(id: string) {
  reportModalOpen.value = false;
  toast.success(`신고가 접수되었습니다. (ID: ${id})`);
}

onMounted(() => {
  void loadBlocks();
});
</script>
