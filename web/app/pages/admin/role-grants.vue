<template>
  <div class="space-y-6">
    <UiPageHeader
      :eyebrow="t('admin.rg.eyebrow')"
      :title="t('admin.rg.title')"
      :description="t('admin.rg.desc')"
    >
      <template #actions>
        <UiButton variant="secondary" size="sm" :loading="loading" @click="load">{{ t('admin.rg.refresh') }}</UiButton>
      </template>
    </UiPageHeader>

    <UiErrorBanner :message="error" :title="t('admin.rg.loadFail')" :retry-label="t('common.actions.retry')" @retry="load" />

    <div v-if="loading" class="space-y-2">
      <UiSkeleton height="h-12" width="w-full" />
      <UiSkeleton height="h-12" width="w-full" />
    </div>

    <div v-else-if="grants.length === 0">
      <UiEmptyState :title="t('admin.rg.empty')" :description="t('admin.rg.emptyDesc')" />
    </div>

    <UiCard v-else padding="none">
      <div class="overflow-x-auto">
        <table class="w-full min-w-[760px] text-xs">
          <thead>
            <tr class="bg-surface-2/60 text-left text-text-secondary">
              <th class="px-4 py-2 font-semibold">{{ t('admin.rg.colAccount') }}</th>
              <th class="px-4 py-2 font-semibold">{{ t('admin.rg.colPrimary') }}</th>
              <th class="px-4 py-2 font-semibold">{{ t('admin.rg.colGrant') }}</th>
              <th class="px-4 py-2 font-semibold">{{ t('admin.rg.colSource') }}</th>
              <th class="px-4 py-2 font-semibold text-right">{{ t('admin.rg.colAt') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-surface-border">
            <tr v-for="g in grants" :key="`${g.userId}-${g.role}`" class="hover:bg-surface-2/40">
              <td class="px-4 py-3">
                <div class="font-bold text-white">{{ g.displayName }}</div>
                <div class="font-mono text-caption text-text-muted">{{ g.email }}</div>
              </td>
              <td class="px-4 py-3">
                <UiBadge variant="neutral" size="xs">{{ g.primaryRole }}</UiBadge>
              </td>
              <td class="px-4 py-3">
                <UiBadge :variant="g.role === 'BUSINESS' ? 'info' : g.role === 'ADMIN' ? 'danger' : 'success'" size="xs">
                  {{ g.role }}
                </UiBadge>
              </td>
              <td class="px-4 py-3 text-text-primary">{{ grantSource(g.source) }}</td>
              <td class="px-4 py-3 text-right font-mono text-text-secondary">{{ fmtDateTime(g.grantedAt) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </UiCard>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { onMounted, ref } from 'vue';
import { adminApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import type { RoleGrant } from '@/api/types';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';

const { t, te, locale } = useI18n();

function grantSource(source: string): string {
  const key = `admin.rg.sources.${source}`;
  return te(key) ? String(t(key)) : source;
}

function fmtDateTime(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString(locale.value === 'en' ? 'en-US' : 'ko-KR', { hour12: false });
}

const grants = ref<RoleGrant[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);

async function load(): Promise<void> {
  loading.value = true;
  error.value = null;
  try {
    grants.value = (await adminApi.roleGrants(100)) ?? [];
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
}

onMounted(() => {
  void load();
});
</script>
