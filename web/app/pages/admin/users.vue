<template>
  <div class="space-y-6">
    <UiPageHeader
      :eyebrow="t('admin.us.eyebrow')"
      :title="t('admin.us.title')"
      :description="t('admin.us.desc')"
    >
      <template #actions>
        <UiButton variant="secondary" size="sm" :loading="loading" @click="reload">{{ t('admin.us.refresh') }}</UiButton>
      </template>
    </UiPageHeader>

    <UiErrorBanner :message="error" :title="t('admin.us.loadFail')" :retry-label="t('common.actions.retry')" @retry="reload" />

    <UiCard class="flex flex-col gap-3 sm:flex-row sm:items-end">
      <label class="flex-[2] text-xs text-text-secondary">
        <span class="mb-1 block font-semibold">{{ t('admin.us.fQuery') }}</span>
        <input
          v-model="query"
          type="text"
          :placeholder="t('admin.us.fQueryPh')"
          class="w-full rounded-lg border border-surface-border bg-surface-1 px-3 py-2 text-sm text-white placeholder:text-text-muted"
          @keyup.enter="reload"
        >
      </label>
      <label class="flex-1 text-xs text-text-secondary">
        <span class="mb-1 block font-semibold">{{ t('admin.us.fRole') }}</span>
        <select
          v-model="roleFilter"
          class="w-full rounded-lg border border-surface-border bg-surface-1 px-3 py-2 text-sm text-white"
          @change="reload"
        >
          <option value="">{{ t('admin.us.all') }}</option>
          <option value="ADMIN">ADMIN</option>
          <option value="PARTNER">PARTNER</option>
          <option value="BUSINESS">BUSINESS</option>
          <option value="USER">USER</option>
        </select>
      </label>
      <UiButton variant="secondary" size="sm" @click="reload">{{ t('admin.us.search') }}</UiButton>
    </UiCard>

    <div v-if="loading && users.length === 0" class="space-y-3">
      <UiSkeleton height="h-16" width="w-full" />
      <UiSkeleton height="h-16" width="w-full" />
    </div>

    <div v-else-if="users.length === 0" class="py-12">
      <UiEmptyState :title="t('admin.us.empty')" :description="t('admin.us.emptyDesc')" />
    </div>

    <div v-else class="space-y-3">
      <UiCard v-for="u in users" :key="u.id" class="space-y-2">
        <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <span class="truncate text-sm font-bold text-white">{{ u.displayName }}</span>
              <UiBadge variant="info" size="xs">{{ u.role }}</UiBadge>
              <UiBadge :variant="u.isActive ? 'success' : 'danger'" size="xs">
                {{ u.isActive ? t('admin.us.active') : t('admin.us.locked') }}
              </UiBadge>
              <UiBadge v-if="businessOf(u.id)" :variant="businessOf(u.id)?.status === 'APPROVED' ? 'success' : 'warning'" size="xs">
                {{ t('admin.us.bizBadge', { status: businessOf(u.id)?.status ?? '' }) }}
              </UiBadge>
            </div>
            <div class="mt-1 truncate font-mono text-caption text-text-secondary">{{ u.email }}</div>
          </div>
          <div class="flex shrink-0 flex-wrap gap-2">
            <select
              :value="u.role"
              class="rounded-lg border border-surface-border bg-surface-1 px-2 py-1 text-xs text-white"
              :disabled="busyId === u.id"
              :aria-label="t('admin.us.changeRole')"
              @change="changeRole(u, ($event.target as HTMLSelectElement).value)"
            >
              <option value="ADMIN">ADMIN</option>
              <option value="PARTNER">PARTNER</option>
              <option value="BUSINESS">BUSINESS</option>
              <option value="USER">USER</option>
            </select>
            <UiButton
              variant="secondary"
              size="xs"
              :loading="busyId === u.id"
              @click="toggleActive(u)"
            >
              {{ u.isActive ? t('admin.us.lock') : t('admin.us.unlock') }}
            </UiButton>
            <UiButton
              v-if="businessOf(u.id)?.status === 'PENDING'"
              variant="brand"
              size="xs"
              :loading="bizBusyId === u.id"
              @click="reviewBusiness(u, true)"
            >
              {{ t('admin.us.bizApprove') }}
            </UiButton>
            <UiButton
              v-if="businessOf(u.id) && businessOf(u.id)?.status !== 'SUSPENDED'"
              variant="ghost"
              size="xs"
              :loading="bizBusyId === u.id"
              @click="reviewBusiness(u, false)"
            >
              {{ t('admin.us.bizSuspend') }}
            </UiButton>
            <UiButton
              variant="ghost"
              size="xs"
              :loading="blocksBusyId === u.id"
              @click="toggleBlocks(u.id)"
            >
              {{ expandedBlocksId === u.id ? t('admin.us.blocksHide') : t('admin.us.blocks') }}
            </UiButton>
          </div>
        </div>
        <div class="text-caption text-text-muted">
          {{ t('admin.us.joined', { at: formatTime(u.createdAt) }) }}
          <span v-if="u.ageConfirmedAt"> · {{ t('admin.us.ageYes') }}</span>
          <span v-else> · {{ t('admin.us.ageNo') }}</span>
          <span v-if="businessOf(u.id)"> · {{ businessOf(u.id)?.companyName }} ({{ businessOf(u.id)?.country }})</span>
        </div>
        <div v-if="expandedBlocksId === u.id" class="rounded-lg border border-surface-border bg-surface-0 p-3 text-xs">
          <div v-if="(blocksByUser[u.id] ?? []).length === 0" class="text-text-muted">
            {{ t('admin.us.blocksEmpty') }}
          </div>
          <ul v-else class="space-y-2">
            <li
              v-for="b in blocksByUser[u.id] ?? []"
              :key="b.id"
              class="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between"
            >
              <div>
                <UiBadge variant="danger" size="xs">{{ t('admin.us.strike', { n: b.strikeNumber }) }}</UiBadge>
                <span class="ml-2 text-text-primary">{{ b.reason }}</span>
              </div>
              <div class="font-mono text-caption text-text-muted">
                {{ formatTime(b.createdAt) }}
                <span v-if="b.expiresAt"> · {{ t('admin.us.strikeExpiry', { at: formatTime(b.expiresAt) }) }}</span>
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
          {{ t('admin.us.more') }}
        </UiButton>
        <p v-else class="text-caption text-text-muted">{{ t('admin.us.last') }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { onMounted, ref } from 'vue';
import { adminApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { useToast } from '@/composables/useToast';
import { useApiError } from '@/composables/useApiError';
import type { BusinessProfile, User, UserBlock } from '@/api/types';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';

const { t, locale } = useI18n();
const PAGE = 20;

const users = ref<User[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);
const busyId = ref<string | null>(null);
const bizBusyId = ref<string | null>(null);
const blocksBusyId = ref<string | null>(null);
const expandedBlocksId = ref<string | null>(null);
const blocksByUser = ref<Record<string, UserBlock[]>>({});
const businesses = ref<Record<string, BusinessProfile>>({});
const query = ref('');
const roleFilter = ref('');
const offset = ref(0);
const hasMore = ref(true);
const toast = useToast();
const apiError = useApiError();

function businessOf(userId: string): BusinessProfile | undefined {
  return businesses.value[userId];
}

function formatTime(iso: string | null | undefined): string {
  if (!iso) return '-';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString(locale.value === 'en' ? 'en-US' : 'ko-KR', { hour12: false });
}

async function fetchPage(reset: boolean): Promise<void> {
  loading.value = true;
  error.value = null;
  try {
    const [page, pending, approved] = await Promise.all([
      adminApi.listUsers({
        q: query.value.trim() || undefined,
        role: roleFilter.value || undefined,
        limit: PAGE,
        offset: reset ? 0 : offset.value,
      }),
      adminApi.listBusinesses('PENDING', 100).catch(() => []),
      adminApi.listBusinesses('APPROVED', 100).catch(() => []),
    ]);
    const rows = page ?? [];
    if (reset) {
      users.value = rows;
      offset.value = rows.length;
    } else {
      users.value = users.value.concat(rows);
      offset.value += rows.length;
    }
    hasMore.value = rows.length === PAGE;
    for (const p of [...(pending ?? []), ...(approved ?? [])]) {
      businesses.value[p.userId] = p;
    }
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
    toast.success(t('admin.us.roleChanged', { email: u.email, role }));
  } catch (err: unknown) {
    error.value = apiError(err);
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
    toast.success(t(u.isActive ? 'admin.us.lockedToast' : 'admin.us.unlockedToast', { email: u.email }));
  } catch (err: unknown) {
    error.value = apiError(err);
  } finally {
    busyId.value = null;
  }
}

async function reviewBusiness(u: User, approve: boolean): Promise<void> {
  bizBusyId.value = u.id;
  try {
    const p = await adminApi.reviewBusiness(u.id, approve);
    businesses.value[u.id] = p;
    toast.success(t(approve ? 'admin.us.bizApproved' : 'admin.us.bizSuspended', { email: u.email }));
  } catch (err: unknown) {
    toast.error(apiError(err), t('admin.us.bizFail'));
  } finally {
    bizBusyId.value = null;
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
    toast.error(apiError(err), t('admin.us.blocksFail'));
  } finally {
    blocksBusyId.value = null;
  }
}

onMounted(() => {
  void fetchPage(true);
});
</script>
