<template>
  <div class="space-y-6">
    <UiPageHeader
      :eyebrow="t('admin.rv.eyebrow')"
      :title="t('admin.rv.title')"
      :description="t('admin.rv.desc')"
    >
      <template #actions>
        <UiButton variant="secondary" size="sm" :loading="loading" @click="loadAll">
          {{ t('admin.rv.refresh') }}
        </UiButton>
      </template>
    </UiPageHeader>

    <UiErrorBanner :message="error" :title="t('admin.rv.loadFail')" :retry-label="t('common.actions.retry')" @retry="loadAll" />

    <UiCard class="text-xs text-text-primary space-y-1.5">
      <div class="font-bold text-white text-sm">{{ t('admin.rv.guideTitle') }}</div>
      <ul class="list-disc pl-5 space-y-1 leading-relaxed">
        <li>{{ t('admin.rv.guide1') }}</li>
        <li>{{ t('admin.rv.guide2') }}</li>
        <li>{{ t('admin.rv.guide3') }}</li>
      </ul>
    </UiCard>

    <UiTabs v-model="activeTab" :tabs="tabOptions" />

    <div v-if="loading" class="grid gap-4 lg:grid-cols-2">
      <UiCard v-for="i in 4" :key="i" class="space-y-3">
        <UiSkeleton height="h-5" width="w-2/3" />
        <UiSkeleton height="h-3" width="w-1/2" />
        <UiSkeleton height="h-10" width="w-full" />
      </UiCard>
    </div>

    <div v-if="!loading && activeTab === 'tracks'" class="space-y-4">
      <div v-if="pendingTracks.length === 0" class="py-12">
        <UiEmptyState
          :title="t('admin.rv.tracksEmpty')"
          :description="t('admin.rv.tracksEmptyDesc')"
        />
      </div>

      <div v-else class="grid gap-4 lg:grid-cols-2">
        <UiCard
          v-for="trk in pendingTracks"
          :key="trk.id"
          class="space-y-3.5"
        >
          <div class="flex items-start justify-between gap-3">
            <div>
              <h3 class="font-bold text-white text-base">{{ trk.title }}</h3>
              <div class="text-caption font-mono text-text-muted mt-0.5">ID: {{ trk.id }}</div>
            </div>
            <UiButton
              variant="brand"
              size="xs"
              @click="previewTrack(trk)"
            >
              {{ t('admin.rv.preview') }}
            </UiButton>
          </div>

          <div class="grid grid-cols-2 gap-2 text-xs">
            <div class="p-2.5 rounded-lg bg-surface-0 border border-surface-border">
              <span class="text-text-muted text-caption uppercase font-bold block">{{ t('admin.rv.provider') }}</span>
              <span class="font-semibold text-white">{{ trk.aiProvider }}</span>
            </div>
            <div class="p-2.5 rounded-lg bg-surface-0 border border-surface-border">
              <span class="text-text-muted text-caption uppercase font-bold block">{{ t('admin.rv.bitrate') }}</span>
              <span class="font-semibold text-success">{{ trk.bitrateKbps ? `${trk.bitrateKbps}kbps` : '192k+' }}</span>
            </div>
          </div>

          <div v-if="rightsOf(trk.id)" class="rounded-xl bg-surface-0 p-3 text-xs space-y-1.5 border border-surface-border">
            <div class="flex justify-between text-text-primary">
              <span>{{ t('admin.rv.tier', { tier: rightsOf(trk.id)?.planTierUsed || t('admin.rv.tierDefault') }) }}</span>
              <span>{{ t('admin.rv.generated', { at: fmtDate(rightsOf(trk.id)?.generatedAt) }) }}</span>
            </div>
            <div class="font-mono text-caption text-success flex items-center gap-2">
              <span>Gate1(Policy): {{ rightsOf(trk.id)?.gate1PolicyPassed ? '✓ PASS' : '✕ FAIL' }}</span>
              <span>·</span>
              <span>Gate2(Date): {{ rightsOf(trk.id)?.gate2DatePassed ? '✓ PASS' : '✕ FAIL' }}</span>
              <span>·</span>
              <span>Gate3(FFprobe): {{ rightsOf(trk.id)?.gate3FfprobePassed ? '✓ PASS' : t('admin.rv.gate3Wait') }}</span>
            </div>
            <div v-if="rightsOf(trk.id)?.providerTrackUrl" class="truncate text-caption">
              <span class="text-text-secondary">{{ t('admin.rv.originLink') }} </span>
              <a :href="rightsOf(trk.id)?.providerTrackUrl ?? '#'" target="_blank" rel="noreferrer" class="text-brand-400 underline">
                {{ rightsOf(trk.id)?.providerTrackUrl }}
              </a>
            </div>
          </div>

          <div class="flex flex-col sm:flex-row items-stretch sm:items-center gap-2 pt-2 border-t border-surface-border">
            <UiInput
              v-model="trackIsrc[trk.id]"
              :placeholder="trk.isrc ? `ISRC: ${trk.isrc}` : t('admin.rv.isrcPh')"
            />
            <UiButton
              variant="secondary"
              size="sm"
              class="shrink-0"
              :loading="isrcBusyId === trk.id"
              @click="saveIsrc(trk)"
            >
              {{ t('admin.rv.isrcSave') }}
            </UiButton>
          </div>

          <div class="flex flex-col sm:flex-row items-stretch sm:items-center gap-2 pt-2 border-t border-surface-border">
            <UiInput
              v-model="trackReason[trk.id]"
              :placeholder="t('admin.rv.reasonPh')"
            />
            <div class="flex gap-2 shrink-0">
              <UiButton
                variant="success"
                size="sm"
                @click="reviewTrack(trk.id, true)"
              >
                {{ t('admin.rv.approveTrack') }}
              </UiButton>
              <UiButton
                variant="danger"
                size="sm"
                @click="reviewTrack(trk.id, false)"
              >
                {{ t('admin.rv.reject') }}
              </UiButton>
            </div>
          </div>
        </UiCard>
      </div>
    </div>

    <div v-else-if="!loading && activeTab === 'partners'" class="space-y-4">
      <div v-if="(queues.partners?.length ?? 0) === 0" class="py-12">
        <UiEmptyState
          :title="t('admin.rv.partnersEmpty')"
        />
      </div>
      <div v-else class="grid gap-3 sm:grid-cols-2">
        <UiCard v-for="p in queues.partners ?? []" :key="p.id" class="space-y-3">
          <div class="flex items-center justify-between">
            <div class="font-bold text-white text-sm">{{ p.companyName }}</div>
            <UiBadge variant="warning" size="xs">{{ partnerStatus(p.status) }}</UiBadge>
          </div>
          <div class="font-mono text-caption text-text-muted">{{ t('admin.rv.partnerId', { id: p.id }) }}</div>
          <div class="flex gap-2 pt-2 border-t border-surface-border">
            <UiButton variant="success" size="xs" @click="reviewPartner(p.id, true)">{{ t('admin.rv.approve') }}</UiButton>
            <UiButton variant="danger" size="xs" @click="reviewPartner(p.id, false)">{{ t('admin.rv.rejectSuspend') }}</UiButton>
          </div>
        </UiCard>
      </div>
    </div>

    <div v-else-if="!loading" class="space-y-4">
      <div v-if="(queues.subscriptions?.length ?? 0) === 0" class="py-12">
        <UiEmptyState
          :title="t('admin.rv.subsEmpty')"
        />
      </div>
      <div v-else class="grid gap-3 sm:grid-cols-2">
        <UiCard v-for="s in queues.subscriptions ?? []" :key="s.id" class="space-y-3">
          <div class="flex items-center justify-between">
            <div class="font-bold text-white text-sm">{{ s.provider }} — {{ s.planName }}</div>
            <UiBadge variant="neutral" size="xs">{{ subStatus(s.status) }}</UiBadge>
          </div>
          <div class="font-mono text-caption text-text-muted">{{ t('admin.rv.proofId', { id: s.id }) }}</div>
          <div v-if="s.proofObjectKey" class="text-xs text-text-primary">
            {{ t('admin.rv.invoice') }} <span class="font-mono text-brand-400">{{ s.proofObjectKey }}</span>
          </div>
          <div class="flex items-center gap-2 pt-2 border-t border-surface-border">
            <UiInput
              v-model="subExpiry[s.id]"
              type="date"
            />
            <UiButton variant="success" size="sm" @click="reviewSub(s.id, true)">{{ t('admin.rv.verifyApprove') }}</UiButton>
            <UiButton variant="danger" size="sm" @click="reviewSub(s.id, false)">{{ t('admin.rv.reject') }}</UiButton>
          </div>
        </UiCard>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { computed, onMounted, ref } from 'vue';
import { adminApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { usePlayerStore } from '@/stores/playerStore';
import { useToast } from '@/composables/useToast';
import { useApiError } from '@/composables/useApiError';
import type { ReviewQueues, Track, TrackRightsDeclaration } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiInput from '@/components/ui/UiInput.vue';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import UiTabs, { type TabOption } from '@/components/ui/UiTabs.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';

const { t, te, locale } = useI18n();
const player = usePlayerStore();
const toast = useToast();
const apiError = useApiError();

const tabOptions = computed<TabOption[]>(() => [
  { label: t('admin.rv.tabTracks'), value: 'tracks' },
  { label: t('admin.rv.tabPartners'), value: 'partners' },
  { label: t('admin.rv.tabSubs'), value: 'subscriptions' },
]);

function partnerStatus(status: string): string {
  const key = `admin.rv.partnerStatus.${status}`;
  return te(key) ? String(t(key)) : status;
}

function subStatus(status: string): string {
  const key = `admin.rv.subStatus.${status}`;
  return te(key) ? String(t(key)) : status;
}

const activeTab = ref('tracks');
const queues = ref<ReviewQueues>({ tracks: [], partners: [], subscriptions: [] });
const loading = ref(true);
const error = ref<string | null>(null);

const trackReason = ref<Record<string, string>>({});
const subExpiry = ref<Record<string, string>>({});
const trackIsrc = ref<Record<string, string>>({});
const isrcBusyId = ref<string | null>(null);

const pendingTracks = computed(() => queues.value.tracks ?? []);

function rightsOf(trackId: string): TrackRightsDeclaration | undefined {
  return queues.value.rightsByTrack?.[trackId];
}

function fmtDate(d?: string): string {
  if (!d) return '-';
  const dt = new Date(d);
  if (Number.isNaN(dt.getTime())) return d;
  return dt.toLocaleDateString(locale.value === 'en' ? 'en-US' : 'ko-KR');
}

async function loadAll() {
  loading.value = true;
  error.value = null;
  try {
    const q = await adminApi.pending();
    queues.value = q;
  } catch (err) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
}

function previewTrack(trk: Track) {
  player.setQueue([trk], trk.id);
}

async function reviewTrack(trackId: string, approved: boolean) {
  try {
    await adminApi.reviewTrack(trackId, approved, trackReason.value[trackId]);
    toast.success(t(approved ? 'admin.rv.trackApproved' : 'admin.rv.trackRejected'));
    await loadAll();
  } catch (err) {
    toast.error(apiError(err), t('admin.rv.processFail'));
  }
}

async function saveIsrc(trk: Track) {
  const isrc = (trackIsrc.value[trk.id] ?? '').trim();
  if (!isrc) {
    toast.error(t('admin.rv.isrcRequired'), t('admin.rv.isrcFail'));
    return;
  }
  isrcBusyId.value = trk.id;
  try {
    await adminApi.setTrackISRC(trk.id, isrc);
    toast.success(t('admin.rv.isrcSaved', { isrc }));
    trackIsrc.value[trk.id] = '';
    await loadAll();
  } catch (err) {
    toast.error(apiError(err), t('admin.rv.isrcFail'));
  } finally {
    isrcBusyId.value = null;
  }
}

async function reviewPartner(partnerId: string, approved: boolean) {
  try {
    await adminApi.reviewPartner(partnerId, approved);
    toast.success(t(approved ? 'admin.rv.partnerApproved' : 'admin.rv.partnerRejected'));
    await loadAll();
  } catch (err) {
    toast.error(apiError(err), t('admin.rv.processFail'));
  }
}

async function reviewSub(subId: string, verified: boolean) {
  try {
    const exp = subExpiry.value[subId] ? new Date(subExpiry.value[subId]).toISOString() : undefined;
    await adminApi.reviewSubscription(subId, verified, exp);
    toast.success(t(verified ? 'admin.rv.subApproved' : 'admin.rv.subRejected'));
    await loadAll();
  } catch (err) {
    toast.error(apiError(err), t('admin.rv.processFail'));
  }
}

onMounted(() => {
  void loadAll();
});
</script>
