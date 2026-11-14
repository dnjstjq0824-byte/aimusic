<template>
  <div class="space-y-6">
    <UiPageHeader
      eyebrow="Audit Console"
      title="매장 사업자 승인"
      description="사업자등록번호가 검증된 매장 신청을 승인하거나 정지합니다. 승인된 매장에서만 플레이어 등록·재생 집계가 가능합니다."
    />

    <UiErrorBanner :message="error" title="매장 목록을 불러오지 못했습니다" retry-label="다시 시도" @retry="load" />

    <UiTabs v-model="status" :tabs="statusTabs" @update:model-value="load" />

    <div v-if="loading" class="space-y-2">
      <UiSkeleton height="h-14" width="w-full" />
      <UiSkeleton height="h-14" width="w-full" />
    </div>

    <div v-else-if="stores.length === 0">
      <UiEmptyState title="해당 상태의 매장이 없습니다" description="다른 상태 탭을 선택해 보세요." />
    </div>

    <UiCard v-else padding="none">
      <ul class="divide-y divide-surface-border">
        <li v-for="s in stores" :key="s.id" class="flex flex-col gap-2 px-5 py-4 text-xs sm:flex-row sm:items-center sm:justify-between">
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <span class="text-sm font-bold text-white">{{ s.storeName }}</span>
              <UiBadge :variant="s.status === 'APPROVED' ? 'success' : s.status === 'SUSPENDED' ? 'danger' : 'warning'" size="xs">
                {{ s.status }}
              </UiBadge>
            </div>
            <p class="mt-1 font-mono text-caption text-text-muted">
              {{ s.businessNumber }} · 라이선스 {{ s.licenseCount }}대 · 신청 {{ new Date(s.createdAt).toLocaleDateString() }}
            </p>
          </div>
          <div class="flex shrink-0 gap-2">
            <UiButton v-if="s.status !== 'APPROVED'" variant="brand" size="sm" :disabled="busyId === s.id" @click="verify(s, true)">
              승인
            </UiButton>
            <UiButton v-if="s.status !== 'SUSPENDED'" variant="ghost" size="sm" :disabled="busyId === s.id" @click="verify(s, false)">
              정지
            </UiButton>
          </div>
        </li>
      </ul>
    </UiCard>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { adminApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import type { StoreBusiness } from '@/api/types';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import UiTabs from '@/components/ui/UiTabs.vue';

const statusTabs = [
  { value: 'PENDING', label: '승인 대기' },
  { value: 'APPROVED', label: '승인됨' },
  { value: 'SUSPENDED', label: '정지됨' },
];

const status = ref('PENDING');
const stores = ref<StoreBusiness[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);
const busyId = ref<string | null>(null);

async function load(): Promise<void> {
  loading.value = true;
  error.value = null;
  try {
    stores.value = (await adminApi.listStores(status.value, 50, 0)) ?? [];
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
}

async function verify(s: StoreBusiness, approve: boolean): Promise<void> {
  busyId.value = s.id;
  try {
    await adminApi.verifyStore(s.id, approve);
    await load();
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    busyId.value = null;
  }
}

onMounted(() => {
  void load();
});
</script>
