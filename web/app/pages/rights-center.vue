<template>
  <div class="space-y-6">
    <UiPageHeader
      :eyebrow="t('legal.rcEyebrow')"
      :title="t('legal.rcTitle')"
      :description="t('legal.rcDesc')"
    />

    <!-- Notice & Takedown Blueprint Banner -->
    <UiCard border-variant="brand">
      <div class="flex items-start gap-3">
        <svg class="h-6 w-6 shrink-0 text-brand-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <path d="M12 3v18M5 7l7-4 7 4M5 7v4a3.5 3.5 0 0 0 7 0V7M19 7v4a3.5 3.5 0 0 1-7 0" />
          <path d="M8 21h8" />
        </svg>
        <div class="text-xs text-text-primary space-y-1">
          <div class="font-bold text-white text-sm">{{ t('legal.rcNoticeT') }}</div>
          <p class="leading-relaxed">
            <strong>{{ t('legal.rcNoticeA') }}</strong>
            {{ t('legal.rcNoticeB') }}
            {{ t('legal.rcNoticeC') }}
          </p>
        </div>
      </div>
    </UiCard>

    <!-- Report a track -->
    <UiCard>
      <template #header>
        <h2 class="text-xs font-bold text-text-primary uppercase tracking-wider">
          {{ t('legal.rcReportT') }}
        </h2>
      </template>
      <div class="space-y-3">
        <form class="flex flex-col gap-2 sm:flex-row" @submit.prevent="searchTracks">
          <UiInput
            v-model="searchQuery"
            :placeholder="t('legal.rcSearchPh')"
            class="flex-1"
          />
          <UiButton variant="brand" size="sm" type="submit" :loading="searching">
            {{ t('legal.rcSearch') }}
          </UiButton>
        </form>
        <div v-if="searching" class="space-y-2">
          <UiSkeleton height="h-10" width="w-full" />
          <UiSkeleton height="h-10" width="w-full" />
        </div>
        <ul v-else-if="searchResults.length > 0" class="divide-y divide-surface-border rounded-xl border border-surface-border">
          <li
            v-for="tr in searchResults"
            :key="tr.id"
            class="flex items-center gap-3 px-3 py-2.5"
          >
            <UiCoverArt :seed="tr.id" :title="tr.title" size="xs" rounded="md" />
            <div class="min-w-0 flex-1">
              <div class="truncate text-xs font-bold text-white">{{ tr.title }}</div>
              <div class="text-caption text-text-muted">{{ tr.aiProvider }}</div>
            </div>
            <UiButton variant="outline" size="xs" @click="reportTarget = tr">
              {{ t('legal.rcReportBtn') }}
            </UiButton>
          </li>
        </ul>
        <p v-else-if="searched" class="text-xs text-text-muted">
          {{ t('legal.rcNoResult') }}
        </p>
        <p v-else class="text-xs text-text-muted">
          {{ t('legal.rcSearchHint') }}
        </p>
      </div>
    </UiCard>

    <!-- Blocked Artists Section (UGC App Store Guideline 1.2 Compliance) -->
    <UiCard>
      <template #header>
        <div class="flex items-center justify-between">
          <h2 class="text-xs font-bold text-text-primary uppercase tracking-wider">
            {{ t('legal.rcBlocksT') }}
          </h2>
          <span class="rounded-full bg-surface-2 px-2 py-0.5 text-caption font-mono text-text-secondary">
            {{ t('legal.rcBlocksN', { n: blocks.length }) }}
          </span>
        </div>
      </template>

      <div v-if="loading" class="space-y-2">
        <UiSkeleton height="h-10" width="w-full" />
        <UiSkeleton height="h-10" width="w-full" />
      </div>

      <div v-else-if="blocks.length === 0" class="py-8">
        <UiEmptyState
          :title="t('legal.rcBlocksEmptyT')"
          :description="t('legal.rcBlocksEmptyD')"
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
              {{ t('legal.rcBlockedPartner') }} <span class="font-mono font-medium text-text-secondary">{{ shortId(b.blockedPartnerId) }}</span>
            </div>
            <div class="text-caption text-text-secondary mt-0.5 truncate">
              {{ t('legal.rcReason', { r: b.reason || t('legal.rcNoReason') }) }}
            </div>
          </div>
          <UiButton
            variant="secondary"
            size="xs"
            :loading="unblockingId === b.blockedPartnerId"
            @click="unblock(b.blockedPartnerId)"
          >
            {{ t('legal.rcUnblock') }}
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
defineI18nRoute(false);
import { onMounted, ref } from 'vue';
import { trackApi, userBlocksApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { useAuthStore } from '@/stores/authStore';
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

const { t } = useI18n();
const auth = useAuthStore();
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
  // The center is public; the per-user block list needs a session.
  if (!auth.isAuthenticated) {
    loading.value = false;
    return;
  }
  loading.value = true;
  try {
    blocks.value = (await userBlocksApi.listArtistBlocks()) ?? [];
  } catch (err) {
    toast.error(errMessageOf(err), t('legal.rcToastBlocksFail'));
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
    toast.error(errMessageOf(err), t('legal.rcToastSearchFail'));
  } finally {
    searching.value = false;
  }
}

async function unblock(partnerId: string) {
  unblockingId.value = partnerId;
  try {
    await userBlocksApi.unblockArtist(partnerId);
    blocks.value = blocks.value.filter((b) => b.blockedPartnerId !== partnerId);
    toast.success(t('legal.rcToastUnblocked'));
  } catch (err) {
    toast.error(errMessageOf(err), t('legal.rcToastUnblockFail'));
  } finally {
    unblockingId.value = null;
  }
}

function onReported(id: string) {
  reportTarget.value = null;
  toast.success(t('legal.rcReported', { id }));
}

onMounted(() => {
  void loadBlocks();
});
</script>
