<template>
  <div class="space-y-6">
    <UiPageHeader
      eyebrow="Admin Governance"
      title="회원 관리"
      description="계정 검색, 역할 변경, 잠금/해제를 처리합니다. 본인 계정 변경과 마지막 관리자 강등은 거부됩니다."
    >
      <template #actions>
        <UiButton variant="secondary" size="sm" :loading="loading" @click="reload">새로고침</UiButton>
      </template>
    </UiPageHeader>

    <UiErrorBanner :message="error" title="회원 목록을 불러오지 못했습니다" retry-label="다시 시도" @retry="reload" />

    <!-- Filters -->
    <UiCard class="flex flex-col gap-3 sm:flex-row sm:items-end">
      <label class="flex-[2] text-xs text-neutral-400">
        <span class="mb-1 block font-semibold">이메일/표시 이름 검색</span>
        <input
          v-model="query"
          type="text"
          placeholder="검색어"
          class="w-full rounded-lg border border-surface-border bg-surface-1 px-3 py-2 text-sm text-white placeholder:text-neutral-600"
          @keyup.enter="reload"
        />
      </label>
      <label class="flex-1 text-xs text-neutral-400">
        <span class="mb-1 block font-semibold">역할</span>
        <select
          v-model="roleFilter"
          class="w-full rounded-lg border border-surface-border bg-surface-1 px-3 py-2 text-sm text-white"
          @change="reload"
        >
          <option value="">전체</option>
          <option value="ADMIN">ADMIN</option>
          <option value="PARTNER">PARTNER</option>
          <option value="USER">USER</option>
        </select>
      </label>
      <UiButton variant="secondary" size="sm" @click="reload">조회</UiButton>
    </UiCard>

    <div v-if="loading && users.length === 0" class="space-y-3">
      <UiSkeleton height="h-16" width="w-full" />
      <UiSkeleton height="h-16" width="w-full" />
    </div>

    <div v-else-if="users.length === 0" class="py-12">
      <UiEmptyState title="회원이 없습니다" description="검색 조건에 맞는 계정이 없습니다." />
    </div>

    <div v-else class="space-y-3">
      <UiCard v-for="u in users" :key="u.id" class="space-y-2">
        <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <span class="truncate text-sm font-bold text-white">{{ u.displayName }}</span>
              <UiBadge variant="info" size="xs">{{ u.role }}</UiBadge>
              <UiBadge :variant="u.isActive ? 'success' : 'danger'" size="xs">
                {{ u.isActive ? '활성' : '잠금' }}
              </UiBadge>
            </div>
            <div class="mt-1 truncate font-mono text-[11px] text-neutral-400">{{ u.email }}</div>
          </div>
          <div class="flex shrink-0 flex-wrap gap-2">
            <select
              :value="u.role"
              class="rounded-lg border border-surface-border bg-surface-1 px-2 py-1 text-xs text-white"
              :disabled="busyId === u.id"
              aria-label="역할 변경"
              @change="changeRole(u, ($event.target as HTMLSelectElement).value)"
            >
              <option value="ADMIN">ADMIN</option>
              <option value="PARTNER">PARTNER</option>
              <option value="USER">USER</option>
            </select>
            <UiButton
              variant="secondary"
              size="xs"
              :loading="busyId === u.id"
              @click="toggleActive(u)"
            >
              {{ u.isActive ? '잠그기' : '해제하기' }}
            </UiButton>
            <UiButton
              variant="ghost"
              size="xs"
              :loading="blocksBusyId === u.id"
              @click="toggleBlocks(u.id)"
            >
              {{ expandedBlocksId === u.id ? '제재 내역 접기' : '제재 내역' }}
            </UiButton>
          </div>
        </div>
        <div class="text-[11px] text-neutral-500">
          가입: {{ formatTime(u.createdAt) }}
          <span v-if="u.ageConfirmedAt"> · 14+ 확인됨</span>
          <span v-else> · 14+ 미확인</span>
        </div>
        <div v-if="expandedBlocksId === u.id" class="rounded-lg border border-surface-border bg-surface-0 p-3 text-xs">
          <div v-if="(blocksByUser[u.id] ?? []).length === 0" class="text-neutral-500">
            제재 기록이 없습니다.
          </div>
          <ul v-else class="space-y-2">
            <li
              v-for="b in blocksByUser[u.id] ?? []"
              :key="b.id"
              class="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between"
            >
              <div>
                <UiBadge variant="danger" size="xs">스트라이크 {{ b.strikeNumber }}</UiBadge>
                <span class="ml-2 text-neutral-200">{{ b.reason }}</span>
              </div>
              <div class="font-mono text-[11px] text-neutral-500">
                {{ formatTime(b.createdAt) }}
                <span v-if="b.expiresAt"> · 만료 {{ formatTime(b.expiresAt) }}</span>
              </div>
            </li>
          </ul>
        </div>
      </UiCard>

      <div class="flex justify-center pt-2">
        <UiButton
          v-if="hasMore"
          variant="secondary"
          size="sm"
          :loading="loading"
          @click="loadMore"
        >
          더 불러오기
        </UiButton>
        <p v-else class="text-[11px] text-neutral-600">마지막 회원입니다.</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { adminApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { useToast } from '@/composables/useToast';
import type { User, UserBlock } from '@/api/types';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';

const PAGE = 20;

const users = ref<User[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);
const busyId = ref<string | null>(null);
const blocksBusyId = ref<string | null>(null);
const expandedBlocksId = ref<string | null>(null);
const blocksByUser = ref<Record<string, UserBlock[]>>({});
const query = ref('');
const roleFilter = ref('');
const offset = ref(0);
const hasMore = ref(true);
const toast = useToast();

function formatTime(iso: string | null | undefined): string {
  if (!iso) return '-';
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleString('ko-KR', { hour12: false });
}

async function fetchPage(reset: boolean): Promise<void> {
  loading.value = true;
  error.value = null;
  try {
    const page = (await adminApi.listUsers({
      q: query.value.trim() || undefined,
      role: roleFilter.value || undefined,
      limit: PAGE,
      offset: reset ? 0 : offset.value,
    })) ?? [];
    if (reset) {
      users.value = page;
      offset.value = page.length;
    } else {
      users.value = users.value.concat(page);
      offset.value += page.length;
    }
    hasMore.value = page.length === PAGE;
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
}

function reload(): void {
  void fetchPage(true);
}

function loadMore(): void {
  void fetchPage(false);
}

async function changeRole(u: User, role: string): Promise<void> {
  if (role === u.role) return;
  busyId.value = u.id;
  error.value = null;
  try {
    const updated = await adminApi.setUserRole(u.id, role);
    const i = users.value.findIndex((x) => x.id === u.id);
    if (i >= 0) users.value[i] = updated;
    toast.success(`${u.email} 역할을 ${role}(으)로 변경했습니다.`);
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    busyId.value = null;
  }
}

async function toggleActive(u: User): Promise<void> {
  busyId.value = u.id;
  error.value = null;
  try {
    const updated = await adminApi.setUserActive(u.id, !u.isActive);
    const i = users.value.findIndex((x) => x.id === u.id);
    if (i >= 0) users.value[i] = updated;
    toast.success(u.isActive ? `${u.email} 계정을 잠갔습니다.` : `${u.email} 계정을 해제했습니다.`);
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    busyId.value = null;
  }
}

async function toggleBlocks(userId: string): Promise<void> {
  if (expandedBlocksId.value === userId) {
    expandedBlocksId.value = null;
    return;
  }
  blocksBusyId.value = userId;
  try {
    blocksByUser.value[userId] = await adminApi.listUserBlocks(userId);
    expandedBlocksId.value = userId;
  } catch (err: unknown) {
    toast.error(errMessageOf(err), '제재 내역 조회 실패');
  } finally {
    blocksBusyId.value = null;
  }
}

onMounted(() => {
  void fetchPage(true);
});
</script>
