<template>
  <div class="space-y-6">
    <UiPageHeader
      :eyebrow="t('admin.au.eyebrow')"
      :title="t('admin.au.title')"
      :description="t('admin.au.desc')"
    >
      <template #actions>
        <UiButton variant="secondary" size="sm" :loading="loading" @click="reload">{{ t('admin.au.refresh') }}</UiButton>
      </template>
    </UiPageHeader>

    <UiErrorBanner :message="error" :title="t('admin.au.loadFail')" :retry-label="t('common.actions.retry')" @retry="reload" />

    <UiCard class="flex flex-col gap-3 sm:flex-row sm:items-end">
      <label class="flex-1 text-xs text-text-secondary">
        <span class="mb-1 block font-semibold">{{ t('admin.au.fAction') }}</span>
        <select
          v-model="actionFilter"
          class="w-full rounded-lg border border-surface-border bg-surface-1 px-3 py-2 text-sm text-white"
          @change="reload"
        >
          <option value="">{{ t('admin.au.all') }}</option>
          <option v-for="a in knownActions" :key="a" :value="a">{{ a }}</option>
        </select>
      </label>
      <label class="flex-1 text-xs text-text-secondary">
        <span class="mb-1 block font-semibold">{{ t('admin.au.fEntity') }}</span>
        <input
          v-model="entityFilter"
          type="text"
          :placeholder="t('admin.au.all')"
          class="w-full rounded-lg border border-surface-border bg-surface-1 px-3 py-2 text-sm text-white placeholder:text-text-muted"
          @keyup.enter="reload"
        >
      </label>
      <UiButton variant="secondary" size="sm" @click="reload">{{ t('admin.au.search') }}</UiButton>
    </UiCard>

    <div v-if="loading && events.length === 0" class="space-y-3">
      <UiSkeleton height="h-20" width="w-full" />
      <UiSkeleton height="h-20" width="w-full" />
    </div>

    <div v-else-if="events.length === 0" class="py-12">
      <UiEmptyState
        :title="t('admin.au.empty')"
        :description="t('admin.au.emptyDesc')"
      />
    </div>

    <div v-else class="space-y-3">
      <UiCard v-for="e in events" :key="e.id" class="space-y-2">
        <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <div class="flex flex-wrap items-center gap-2">
            <UiBadge variant="info" size="xs">{{ e.action }}</UiBadge>
            <span v-if="e.entityType" class="text-xs text-text-primary">
              {{ e.entityType }}<span v-if="e.entityId" class="font-mono text-text-secondary">:{{ shortId(e.entityId) }}</span>
            </span>
          </div>
          <time class="text-caption text-text-muted" :datetime="e.createdAt">{{ formatTime(e.createdAt) }}</time>
        </div>
        <div class="flex flex-wrap gap-x-4 gap-y-1 text-caption text-text-muted">
          <span>
            {{ t('admin.au.actor') }}
            <strong class="text-text-primary">{{ e.actorUserId ? shortId(e.actorUserId) : t('admin.au.system') }}</strong>
          </span>
          <span v-if="e.ip">
            IP: <span class="font-mono">{{ e.ip }}</span>
          </span>
        </div>
        <details v-if="e.metadata && e.metadata !== '{}'" class="text-caption">
          <summary class="cursor-pointer text-text-secondary hover:text-white">{{ t('admin.au.detail') }}</summary>
          <pre class="mt-1 overflow-x-auto rounded-lg bg-surface-0 p-2 font-mono text-text-primary">{{ prettyMeta(e.metadata) }}</pre>
        </details>
      </UiCard>

      <div class="flex justify-center pt-2">
        <UiButton
          v-if="hasMore"
          variant="secondary"
          size="sm"
          :loading="loading"
          @click="loadMore"
        >
          {{ t('admin.au.more') }}
        </UiButton>
        <p v-else class="text-caption text-text-muted">{{ t('admin.au.last') }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { onMounted, ref } from 'vue';
import { adminApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import type { AuditEvent } from '@/api/types';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';

const { t, locale } = useI18n();
const PAGE = 50;

const knownActions = [
  'TRACK_REVIEW',
  'PARTNER_REVIEW',
  'SUBSCRIPTION_REVIEW',
  'POLICY_UPDATE',
  'DISPUTE_RESOLVE',
  'TRACK_TAKEDOWN',
  'CURATION_CREATE',
  'SETTLEMENT_CALCULATE',
  'SETTLEMENT_APPROVE',
  'SETTLEMENT_MARK_PAID',
  'ADMIN_CREATE',
  'USER_SET_ROLE',
  'USER_SET_ACTIVE',
  'BUSINESS_REVIEW',
  'STORE_VERIFY',
];

const events = ref<AuditEvent[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);
const actionFilter = ref('');
const entityFilter = ref('');
const offset = ref(0);
const hasMore = ref(true);

function shortId(id: string): string {
  return id.length > 13 ? `${id.slice(0, 8)}…` : id;
}

function formatTime(iso: string | null | undefined): string {
  if (!iso) return '-';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString(locale.value === 'en' ? 'en-US' : 'ko-KR', { hour12: false });
}

function prettyMeta(raw: string): string {
  try {
    return JSON.stringify(JSON.parse(raw), null, 2);
  } catch {
    return raw;
  }
}

async function fetchPage(reset: boolean): Promise<void> {
  loading.value = true;
  error.value = null;
  try {
    const page = (await adminApi.auditEvents({
      action: actionFilter.value || undefined,
      entityType: entityFilter.value.trim() || undefined,
      limit: PAGE,
      offset: reset ? 0 : offset.value,
    })) ?? [];
    if (reset) {
      events.value = page;
      offset.value = page.length;
    } else {
      events.value = events.value.concat(page);
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

onMounted(() => {
  void fetchPage(true);
});
</script>
