<template>
  <div class="space-y-6">
    <UiPageHeader
      :eyebrow="t('admin.ds.eyebrow')"
      :title="t('admin.ds.title')"
      :description="t('admin.ds.desc')"
    />

    <UiCard padding="md">
      <form class="space-y-4" @submit.prevent="preview">
        <UiTextarea
          v-model="idsText"
          :label="t('admin.ds.fIds')"
          :placeholder="t('admin.ds.fIdsPh')"
          :rows="4"
        />
        <UiErrorBanner :message="error" :title="t('admin.ds.exportFail')" />
        <div class="flex flex-wrap gap-2">
          <UiButton variant="brand" size="sm" type="submit" :disabled="busy">{{ t('admin.ds.preview') }}</UiButton>
          <UiButton variant="secondary" size="sm" type="button" :disabled="busy || trackIds.length === 0" @click="downloadCsv">
            {{ t('admin.ds.csv') }}
          </UiButton>
        </div>
      </form>
    </UiCard>

    <UiCard v-if="kit" padding="none">
      <template #header>
        <div class="flex items-center justify-between px-5 pt-3">
          <h2 class="text-base font-medium text-white">{{ t('admin.ds.result', { ok: kit.tracks.length, skip: kit.skipped.length }) }}</h2>
          <span class="font-mono text-caption text-text-muted">{{ kit.generatedAt }}</span>
        </div>
      </template>
      <div class="overflow-x-auto">
        <table class="w-full min-w-[720px] text-xs">
          <thead>
            <tr class="bg-surface-2/60 text-left text-text-secondary">
              <th class="px-3 py-2 font-semibold">ISRC</th>
              <th class="px-3 py-2 font-semibold">{{ t('admin.ds.colTitle') }}</th>
              <th class="px-3 py-2 font-semibold">{{ t('admin.ds.colArtist') }}</th>
              <th class="px-3 py-2 font-semibold">{{ t('admin.ds.colBpm') }}</th>
              <th class="px-3 py-2 font-semibold">{{ t('admin.ds.colProvider') }}</th>
              <th class="px-3 py-2 font-semibold">{{ t('admin.ds.colMaster') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-surface-border">
            <tr v-for="trk in kit.tracks" :key="trk.trackId" class="hover:bg-surface-2/40">
              <td class="px-3 py-2 font-mono text-text-primary">{{ trk.isrc }}</td>
              <td class="px-3 py-2 font-bold text-white">{{ trk.title }}</td>
              <td class="px-3 py-2 text-text-primary">{{ trk.artistName }}</td>
              <td class="px-3 py-2 font-mono text-text-primary">{{ trk.bpm ?? '-' }} / {{ trk.musicalKey || '-' }}</td>
              <td class="px-3 py-2 text-text-primary">{{ trk.aiProvider }} {{ trk.planTierUsed }}</td>
              <td class="px-3 py-2">
                <a :href="trk.masterDownloadUrl" target="_blank" rel="noopener" class="text-brand-300 underline hover:text-brand-200">
                  {{ t('admin.ds.download') }}
                </a>
                <span class="ml-1 text-caption text-text-muted">({{ masterLeft(trk.masterUrlExpiry) }})</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div v-if="kit.skipped.length > 0" class="border-t border-surface-border px-5 py-3">
        <h3 class="mb-1 text-xs font-bold text-text-secondary">{{ t('admin.ds.skippedTitle') }}</h3>
        <ul class="space-y-1 text-caption">
          <li v-for="s in kit.skipped" :key="s.trackId" class="font-mono text-text-muted">
            {{ s.trackId }} — {{ skipReason(s.reason) }}
          </li>
        </ul>
      </div>
    </UiCard>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { computed, ref } from 'vue';
import { adminApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import type { DSPKit } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiTextarea from '@/components/ui/UiTextarea.vue';

const { t, te } = useI18n();
const idsText = ref('');
const kit = ref<DSPKit | null>(null);
const busy = ref(false);
const error = ref<string | null>(null);

const trackIds = computed(() =>
  idsText.value.split(/[\s,]+/).map((s) => s.trim()).filter((s) => s.length > 0),
);

function skipReason(reason: string): string {
  if (reason.startsWith('not_ready')) {
    return t('admin.ds.skipNotReady', { status: reason.slice('not_ready:'.length) });
  }
  const key = `admin.ds.skip.${reason}`;
  return te(key) ? String(t(key)) : reason;
}

function masterLeft(expiry: string): string {
  const ms = new Date(expiry).getTime() - Date.now();
  if (ms <= 0) return t('admin.ds.expired');
  return t('admin.ds.minutesLeft', { n: Math.ceil(ms / 60000) });
}

async function preview(): Promise<void> {
  busy.value = true;
  error.value = null;
  try {
    kit.value = await adminApi.dspKit(trackIds.value);
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    busy.value = false;
  }
}

async function downloadCsv(): Promise<void> {
  busy.value = true;
  error.value = null;
  try {
    const blob = await adminApi.dspKitCsv(trackIds.value);
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `dsp-kit-${new Date().toISOString().slice(0, 10)}.csv`;
    document.body.appendChild(a);
    a.click();
    a.remove();
    URL.revokeObjectURL(url);
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    busy.value = false;
  }
}
</script>
