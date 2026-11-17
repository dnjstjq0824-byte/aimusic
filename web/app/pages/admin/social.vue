<template>
  <div class="space-y-6">
    <UiPageHeader
      :eyebrow="t('admin.so.eyebrow')"
      :title="t('admin.so.title')"
      :description="t('admin.so.desc')"
    >
      <template #actions>
        <UiButton variant="secondary" size="sm" :loading="loading" @click="load">{{ t('admin.so.refresh') }}</UiButton>
      </template>
    </UiPageHeader>

    <UiErrorBanner :message="error" :title="t('admin.so.loadFail')" :retry-label="t('common.actions.retry')" @retry="load" />

    <div v-if="loading" class="grid gap-4 sm:grid-cols-3">
      <UiCard v-for="i in 3" :key="i" class="space-y-2">
        <UiSkeleton height="h-3" width="w-1/2" />
        <UiSkeleton height="h-8" width="w-2/3" />
      </UiCard>
    </div>

    <template v-else-if="stats">
      <div class="grid gap-4 sm:grid-cols-3">
        <UiCard class="space-y-1">
          <div class="text-caption font-bold text-text-muted uppercase tracking-wider">{{ t('admin.so.total') }}</div>
          <div class="text-2xl font-semibold text-white">{{ stats.totalLinks.toLocaleString() }}</div>
        </UiCard>
        <UiCard class="space-y-1">
          <div class="text-caption font-bold text-text-muted uppercase tracking-wider">{{ t('admin.so.users') }}</div>
          <div class="text-2xl font-semibold text-brand-400">{{ stats.linkedUsers.toLocaleString() }}</div>
        </UiCard>
        <UiCard class="space-y-1">
          <div class="text-caption font-bold text-text-muted uppercase tracking-wider">{{ t('admin.so.providers') }}</div>
          <div class="text-2xl font-semibold text-success">{{ stats.byProvider.length }}</div>
        </UiCard>
      </div>

      <UiCard padding="none">
        <template #header>
          <div class="px-5 pt-3">
            <h2 class="text-xs font-bold text-text-secondary uppercase tracking-wider">{{ t('admin.so.byProvider') }}</h2>
          </div>
        </template>
        <p v-if="stats.byProvider.length === 0" class="px-5 py-6 text-xs text-text-muted">
          {{ t('admin.so.empty') }}
        </p>
        <ul v-else class="divide-y divide-surface-border">
          <li
            v-for="p in stats.byProvider"
            :key="p.provider"
            class="flex items-center justify-between gap-2 px-5 py-3 text-xs"
          >
            <span class="font-bold text-white">{{ p.provider }}</span>
            <span class="font-mono text-text-primary">{{ t('admin.so.links', { n: p.links.toLocaleString() }) }}</span>
          </li>
        </ul>
      </UiCard>
    </template>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { onMounted, ref } from 'vue';
import { adminApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import type { OAuthStats } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';

const { t } = useI18n();
const stats = ref<OAuthStats | null>(null);
const loading = ref(true);
const error = ref<string | null>(null);

async function load(): Promise<void> {
  loading.value = true;
  error.value = null;
  try {
    stats.value = await adminApi.oauthStats();
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
