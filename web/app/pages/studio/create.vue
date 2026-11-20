<template>
  <div class="space-y-6">
    <UiPageHeader
      :eyebrow="t('studio.create.eyebrow')"
      :title="t('studio.create.title')"
      :description="t('studio.create.desc')"
    >
      <template #actions>
        <div class="flex gap-2">
          <UiButton variant="secondary" size="sm" :loading="loading" @click="load">
            {{ t('studio.create.refresh') }}
          </UiButton>
          <UiButton variant="brand" size="sm" @click="createModalOpen = true">
            {{ t('studio.create.new') }}
          </UiButton>
        </div>
      </template>
    </UiPageHeader>

    <!-- WI-G5 v2 panel: flag-gated seat (NUXT_PUBLIC_GENERATE_UI=1). Off by
         default, so the operational screen below stays byte-identical. -->
    <UiCard v-if="genV2" class="space-y-4 p-5">
      <div>
        <h2 class="font-display text-lg font-medium tracking-[-0.03em] text-white">{{ t('studio.create.v2.panelTitle') }}</h2>
        <p class="mt-1 text-xs leading-relaxed text-text-secondary">{{ t('studio.create.v2.panelDesc') }}</p>
      </div>
      <PromptBox v-model="v2Prompt" />
      <StyleTags v-model="v2Tags" />
      <LyricsEditor v-model="v2Lyrics" />
      <div class="grid gap-4 sm:grid-cols-2">
        <UiSelect v-model="v2Length" :label="t('studio.create.v2.lengthLabel')">
          <option v-for="s in [15, 30, 60]" :key="s" :value="String(s)">
            {{ t('studio.create.seconds', { n: s }) }}
          </option>
        </UiSelect>
        <UiInput
          v-model="v2Seed"
          type="number"
          min="0"
          step="1"
          :label="t('studio.create.v2.seedLabel')"
          :placeholder="t('studio.create.v2.seedPh')"
        />
      </div>
      <div class="flex flex-wrap gap-2">
        <UiButton variant="brand" size="sm" :loading="busy" :disabled="!v2Prompt.trim()" @click="createV2">
          {{ t('studio.create.v2.generateCta') }}
        </UiButton>
        <UiButton variant="secondary" size="sm" disabled :title="t('studio.create.v2.variationSoon')">
          {{ t('studio.create.v2.continueCta') }}
        </UiButton>
        <UiButton variant="secondary" size="sm" disabled :title="t('studio.create.v2.variationSoon')">
          {{ t('studio.create.v2.stemsCta') }}
        </UiButton>
        <RouterLink v-if="lastJob?.status === 'SUCCEEDED'" :to="localePath('/studio/upload')">
          <UiButton variant="ghost" size="sm">{{ t('studio.create.v2.certCta') }}</UiButton>
        </RouterLink>
      </div>
      <div class="space-y-2">
        <h3 class="text-base font-medium text-white">{{ t('studio.create.v2.variationsTitle') }}</h3>
        <div class="grid gap-4 sm:grid-cols-2">
          <VariationCard :index="1" :status="slot1Status" :note="lastJob ? '' : t('studio.create.v2.variationEmpty')" />
          <VariationCard :index="2" status="soon" />
        </div>
      </div>
    </UiCard>

    <UiErrorBanner :message="error" :title="t('studio.create.loadFail')" :retry-label="t('common.actions.retry')" @retry="load" />

    <UiCard padding="none">
      <template #header>
        <div class="px-5 pt-3 flex items-center justify-between">
          <h2 class="text-base font-medium text-white">{{ t('studio.create.listTitle') }}</h2>
          <span class="text-xs text-text-muted font-mono">{{ t('studio.create.count', { n: jobs.length }) }}</span>
        </div>
      </template>

      <div v-if="loading" class="space-y-2 p-4">
        <UiSkeleton height="h-12" width="w-full" />
        <UiSkeleton height="h-12" width="w-full" />
      </div>

      <div v-else-if="jobs.length === 0" class="py-12">
        <UiEmptyState
          :title="t('studio.create.empty')"
          :description="t('studio.create.emptyDesc')"
        >
          <template #action>
            <UiButton variant="brand" size="sm" @click="createModalOpen = true">
              {{ t('studio.create.new') }}
            </UiButton>
          </template>
        </UiEmptyState>
      </div>

      <ul v-else class="divide-y divide-surface-border">
        <li
          v-for="j in jobs"
          :key="j.id"
          class="px-5 py-4 text-xs"
        >
          <button
            type="button"
            class="flex w-full flex-col gap-2 text-left sm:flex-row sm:items-center sm:justify-between"
            @click="toggleDetail(j.id)"
          >
            <div class="min-w-0 flex-1 space-y-1">
              <div class="flex flex-wrap items-center gap-2">
                <UiBadge :variant="statusVariant(j.status)" size="xs">
                  {{ genStatus(j.status) }}
                </UiBadge>
                <span class="font-mono text-caption text-text-muted">{{ j.model }}</span>
              </div>
              <p class="truncate text-sm font-semibold text-white">{{ j.prompt }}</p>
              <div class="font-mono text-caption text-text-muted">
                {{ fmtDateTime(j.createdAt) }} · {{ t('studio.create.attempts', { n: j.attempts }) }}
                <span v-if="j.durationMs"> · {{ t('studio.create.seconds', { n: Math.round(j.durationMs / 1000) }) }}</span>
              </div>
            </div>
            <span class="shrink-0 text-caption font-semibold text-brand-400">
              {{ expandedId === j.id ? t('studio.create.collapse') : t('studio.create.expand') }}
            </span>
          </button>

          <div v-if="expandedId === j.id" class="mt-3 space-y-2 rounded-lg border border-surface-border bg-surface-0 p-3">
            <div v-if="detailLoading" class="py-2">
              <UiSkeleton height="h-4" width="w-2/3" />
            </div>
            <template v-else-if="detail">
              <div v-if="detail.lyrics" class="whitespace-pre-wrap text-xs text-text-primary">{{ detail.lyrics }}</div>
              <div class="font-mono text-caption text-text-muted">{{ t('studio.create.jobId', { id: detail.id }) }}</div>
              <div v-if="detail.outputObjectKey" class="font-mono text-caption text-success">
                {{ t('studio.create.output', { key: detail.outputObjectKey }) }}
              </div>
              <div v-if="detail.error" class="text-caption font-semibold text-accent-crimson">
                {{ t('studio.create.failReason', { reason: failureText(detail.error) }) }}
              </div>
              <div v-if="detail.licenseRef" class="text-caption text-text-secondary">
                {{ t('studio.create.licenseRef', { ref: detail.licenseRef }) }}
              </div>
            </template>
          </div>
        </li>
      </ul>
    </UiCard>

    <UiModal
      v-model="createModalOpen"
      :title="t('studio.create.modalTitle')"
      :description="t('studio.create.modalDesc')"
      @close="createModalOpen = false"
    >
      <form class="space-y-4" @submit.prevent="create">
        <UiTextarea
          v-model="prompt"
          :label="t('studio.create.fPrompt')"
          :placeholder="t('studio.create.fPromptPh')"
          required
        />

        <UiTextarea
          v-model="lyrics"
          :label="t('studio.create.fLyrics')"
          :placeholder="t('studio.create.fLyricsPh')"
        />

        <UiInput
          v-model="model"
          :label="t('studio.create.fModel')"
          placeholder="ace-step-v1"
        />

        <UiInput
          v-model="licenseRef"
          :label="t('studio.create.fLicense')"
          :placeholder="t('studio.create.fLicensePh')"
        />

        <div class="flex justify-end gap-2 pt-2 border-t border-surface-border">
          <UiButton variant="ghost" size="sm" @click="createModalOpen = false">{{ t('common.actions.cancel') }}</UiButton>
          <UiButton variant="brand" size="sm" type="submit" :loading="busy">{{ t('studio.create.submit') }}</UiButton>
        </div>
      </form>
    </UiModal>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { computed, onMounted, ref } from 'vue';
import { partnerApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { useToast } from '@/composables/useToast';
import { useApiError } from '@/composables/useApiError';
import type { GenerationJob } from '@/api/types';
import LyricsEditor from '@/components/studio/LyricsEditor.vue';
import PromptBox from '@/components/studio/PromptBox.vue';
import StyleTags from '@/components/studio/StyleTags.vue';
import VariationCard from '@/components/studio/VariationCard.vue';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiInput from '@/components/ui/UiInput.vue';
import UiModal from '@/components/ui/UiModal.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSelect from '@/components/ui/UiSelect.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import UiTextarea from '@/components/ui/UiTextarea.vue';

const { t, te, locale } = useI18n();
const localePath = useLocalePath();
const toast = useToast();
const apiError = useApiError();

function genStatus(status: string): string {
  const key = `studio.genStatus.${status}`;
  return te(key) ? String(t(key)) : status;
}

function statusVariant(status: string): 'success' | 'warning' | 'danger' | 'info' {
  if (status === 'SUCCEEDED') return 'success';
  if (status === 'FAILED') return 'danger';
  if (status === 'RUNNING') return 'info';
  return 'warning';
}

// EV-P08: never expose internal variable names; map known backend failures
// to user-facing guidance. Backend connection itself is engine scope.
function failureText(raw: string): string {
  if (raw.includes('GENERATION_BACKEND_URL')) {
    return t('studio.create.backendDown');
  }
  return raw;
}

function fmtDateTime(iso: string): string {
  try {
    return new Date(iso).toLocaleString(locale.value === 'zh-cn' ? 'zh-CN' : locale.value);
  } catch {
    return iso;
  }
}

const jobs = ref<GenerationJob[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);
const busy = ref(false);
const createModalOpen = ref(false);
const expandedId = ref<string | null>(null);
const detail = ref<GenerationJob | null>(null);
const detailLoading = ref(false);

const prompt = ref('');
const lyrics = ref('');
const model = ref('ace-step-v1');
const licenseRef = ref('');

async function load(): Promise<void> {
  loading.value = true;
  error.value = null;
  try {
    jobs.value = (await partnerApi.listGenerations()) ?? [];
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
}

async function toggleDetail(id: string): Promise<void> {
  if (expandedId.value === id) {
    expandedId.value = null;
    detail.value = null;
    return;
  }
  expandedId.value = id;
  detail.value = null;
  detailLoading.value = true;
  try {
    detail.value = await partnerApi.getGeneration(id);
  } catch (err: unknown) {
    toast.error(apiError(err), t('studio.create.detailFail'));
    expandedId.value = null;
  } finally {
    detailLoading.value = false;
  }
}

async function create(): Promise<void> {
  busy.value = true;
  try {
    await partnerApi.submitGeneration({
      prompt: prompt.value,
      lyrics: lyrics.value || undefined,
      model: model.value || undefined,
      licenseRef: licenseRef.value || undefined,
    });
    toast.success(t('studio.create.queued'));
    prompt.value = '';
    lyrics.value = '';
    licenseRef.value = '';
    createModalOpen.value = false;
    await load();
  } catch (err: unknown) {
    toast.error(apiError(err), t('studio.create.submitFail'));
  } finally {
    busy.value = false;
  }
}

// WI-G5 v2 panel state (flag-gated; inert when the flag is off).
// Nitro env overrides can deliver the flag as number 1, not only boolean
// true, so accept every truthy spelling.
const runtimeConfig = useRuntimeConfig();
const genV2 = computed(() => {
  const v = runtimeConfig.public.generateUi as unknown;
  return v === true || v === 1 || v === '1';
});
const v2Prompt = ref('');
const v2Tags = ref<string[]>([]);
const v2Lyrics = ref('');
const v2Length = ref('30');
const v2Seed = ref('');
const lastJob = ref<GenerationJob | null>(null);

const slot1Status = computed<'soon' | 'queued' | 'running' | 'done' | 'failed'>(() => {
  if (!lastJob.value) return 'soon';
  if (lastJob.value.status === 'SUCCEEDED') return 'done';
  if (lastJob.value.status === 'FAILED') return 'failed';
  if (lastJob.value.status === 'RUNNING') return 'running';
  return 'queued';
});

async function createV2(): Promise<void> {
  if (!v2Prompt.value.trim() || busy.value) return;
  busy.value = true;
  try {
    const params: Record<string, unknown> = { durationSec: Number.parseInt(v2Length.value, 10) || 30 };
    if (v2Tags.value.length > 0) params.styleTags = [...v2Tags.value];
    const seedNum = Number.parseInt(v2Seed.value, 10);
    if (v2Seed.value.trim() !== '' && Number.isInteger(seedNum) && seedNum >= 0) params.seed = seedNum;
    const job = await partnerApi.submitGeneration({
      prompt: v2Prompt.value.trim(),
      lyrics: v2Lyrics.value.trim() || undefined,
      params,
    });
    lastJob.value = job;
    toast.success(t('studio.create.queued'));
    await load();
  } catch (err: unknown) {
    toast.error(apiError(err), t('studio.create.submitFail'));
  } finally {
    busy.value = false;
  }
}

onMounted(() => void load());
</script>
