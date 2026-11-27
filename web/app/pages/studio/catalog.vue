<template>
  <div class="space-y-6">
    <UiPageHeader
      :eyebrow="t('studio.catalog.eyebrow')"
      :title="t('studio.catalog.title')"
      :description="t('studio.catalog.desc')"
    >
      <template #actions>
        <RouterLink to="/studio/upload">
          <UiButton variant="brand" size="sm">
            {{ t('studio.catalog.uploadCta') }}
          </UiButton>
        </RouterLink>
      </template>
    </UiPageHeader>

    <UiErrorBanner :message="error" :title="t('studio.catalog.loadFail')" :retry-label="t('common.actions.retry')" @retry="load" />

    <UiCard padding="none">
      <template #header>
        <div class="px-5 pt-3 flex flex-wrap items-center justify-between gap-2">
          <h2 class="text-base font-medium text-white">{{ t('studio.catalog.listTitle') }}</h2>
          <div class="flex items-center gap-2">
            <UiSelect v-model="statusFilter" :label="t('studio.catalog.filter')" class="w-44">
              <option value="">{{ t('studio.catalog.filterAll') }}</option>
              <option v-for="s in STATUS_OPTIONS" :key="s" :value="s">{{ statusLabel(s) }}</option>
            </UiSelect>
            <span class="text-xs text-text-muted font-mono">{{ filtered.length }}</span>
          </div>
        </div>
      </template>

      <div v-if="loading" class="space-y-2 p-4">
        <UiSkeleton height="h-10" width="w-full" />
        <UiSkeleton height="h-10" width="w-full" />
      </div>

      <div v-else-if="filtered.length === 0" class="py-12">
        <UiEmptyState
          :title="t('studio.catalog.empty')"
          :description="t('studio.catalog.emptyDesc')"
        >
          <template #action>
            <RouterLink to="/studio/upload">
              <UiButton variant="brand" size="sm">{{ t('studio.catalog.uploadCta') }}</UiButton>
            </RouterLink>
          </template>
        </UiEmptyState>
      </div>

      <div v-else class="overflow-x-auto">
        <table class="w-full min-w-[720px] text-xs">
          <thead>
            <tr class="bg-surface-2/60 text-left text-text-secondary">
              <th class="px-4 py-2 font-semibold">{{ t('studio.catalog.colTitle') }}</th>
              <th class="px-4 py-2 font-semibold">{{ t('studio.catalog.colStatus') }}</th>
              <th class="px-4 py-2 font-semibold">{{ t('studio.catalog.colCert') }}</th>
              <th class="px-4 py-2 font-semibold text-right">{{ t('studio.catalog.colPlays') }}</th>
              <th class="px-4 py-2 font-semibold text-right">{{ t('studio.catalog.colUpdated') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-surface-border">
            <tr
              v-for="trk in filtered"
              :key="trk.id"
              class="cursor-pointer hover:bg-surface-2/40"
              tabindex="0"
              @click="openDrawer(trk)"
              @keydown.enter="openDrawer(trk)"
            >
              <td class="px-4 py-3 font-bold text-white">{{ trk.title }}</td>
              <td class="px-4 py-3">
                <UiBadge :variant="statusVariant(trk.status)" size="xs">{{ statusLabel(trk.status) }}</UiBadge>
              </td>
              <td class="px-4 py-3">
                <span v-if="certNumbers[trk.id]" class="font-mono text-verified">{{ certNumbers[trk.id] }}</span>
                <span v-else class="text-text-muted">—</span>
              </td>
              <td class="px-4 py-3 text-right font-mono text-text-primary">{{ (trk.playCount || 0).toLocaleString() }}</td>
              <td class="px-4 py-3 text-right text-text-secondary">{{ fmtDate(trk.updatedAt) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </UiCard>

    <UiModal
      v-model="drawerOpen"
      :title="selected?.title ?? ''"
      :description="t('studio.catalog.drawerDesc')"
      @close="drawerOpen = false"
    >
      <div v-if="selected" class="space-y-4">
        <div class="flex flex-wrap items-center gap-2">
          <UiBadge :variant="statusVariant(selected.status)" size="sm">{{ statusLabel(selected.status) }}</UiBadge>
          <span class="text-caption text-text-secondary">{{ selected.aiProvider }}</span>
        </div>
        <ReviewTimeline :items="drawerItems" />
        <div v-if="certLoading" class="py-2">
          <UiSkeleton height="h-4" width="w-2/3" />
        </div>
        <div v-else-if="drawerCert" class="rounded-xl border border-verified/30 bg-surface-0 p-3 text-xs">
          <RouterLink :to="`/certificate/${drawerCert.certificate.certNumber}`" class="font-mono font-bold text-verified hover:underline">
            {{ drawerCert.certificate.certNumber }}
          </RouterLink>
          <p class="mt-1 text-caption text-text-secondary">
            {{ t('studio.catalog.certIssued', { at: fmtDate(drawerCert.certificate.issuedAt) }) }}
          </p>
        </div>
        <p v-if="selected.rejectionReasonKo" class="rounded-xl border border-red-600/50 bg-red-950/40 p-3 text-xs text-red-200">
          {{ selected.rejectionReasonKo }}
        </p>
        <p class="text-xs text-text-secondary">
          {{ t('studio.catalog.playStat', { count: (selected.playCount || 0).toLocaleString() }) }}
        </p>

        <form v-if="selected.status !== 'TAKEN_DOWN'" class="space-y-3 border-t border-surface-border pt-4" @submit.prevent="saveEdit">
          <h4 class="text-xs font-bold text-white">{{ t('studio.catalog.editTitle') }}</h4>
          <p class="text-caption text-text-muted">{{ t('studio.catalog.editHint') }}</p>
          <UiInput v-model="editGenre" :label="t('studio.catalog.editGenre')" :maxlength="40" />
          <UiInput v-model="editMoods" :label="t('studio.catalog.editMoods')" :hint="t('studio.catalog.editMoodsHint')" />
          <UiTextarea v-model="editLyrics" :label="t('studio.catalog.editLyrics')" :rows="4" :maxlength="10000" />
          <UiCheckbox v-model="editPromptPublic">{{ t('studio.catalog.editPromptPublic') }}</UiCheckbox>
          <div class="flex flex-wrap justify-between gap-2">
            <UiButton variant="danger" size="sm" type="button" @click="deleteOpen = true">
              {{ t('studio.catalog.delete') }}
            </UiButton>
            <div class="flex gap-2">
              <RouterLink v-if="selected.status === 'READY'" :to="localePath({ name: 'track-detail', params: { id: selected.id } })">
                <UiButton variant="secondary" size="sm" type="button">{{ t('studio.catalog.viewTrack') }}</UiButton>
              </RouterLink>
              <UiButton variant="brand" size="sm" type="submit" :loading="saving">{{ t('studio.catalog.save') }}</UiButton>
            </div>
          </div>
        </form>
      </div>
    </UiModal>

    <UiConfirmDialog
      v-model="deleteOpen"
      :title="t('studio.catalog.delTitle')"
      :body="t('studio.catalog.delBody', { title: selected?.title ?? '' })"
      :confirm-label="t('studio.catalog.delConfirm')"
      danger
      :loading="deleting"
      @confirm="confirmDelete"
    />
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { computed, onMounted, reactive, ref } from 'vue';
import { partnerApi, trackApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import type { CertificateView, Track } from '@/api/types';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiCheckbox from '@/components/ui/UiCheckbox.vue';
import UiConfirmDialog from '@/components/ui/UiConfirmDialog.vue';
import UiInput from '@/components/ui/UiInput.vue';
import UiTextarea from '@/components/ui/UiTextarea.vue';
import { useToast } from '@/composables/useToast';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiModal from '@/components/ui/UiModal.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSelect from '@/components/ui/UiSelect.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import ReviewTimeline, { type TimelineItem } from '@/components/studio/ReviewTimeline.vue';

const { t, te, locale } = useI18n();
const localePath = useLocalePath();

const STATUS_OPTIONS = ['UPLOADED', 'TRANSCODING', 'LICENSE_GATE_PENDING', 'LICENSE_GATE_REJECTED', 'READY', 'TAKEN_DOWN'];

function statusLabel(status: string): string {
  const key = `studio.status.${status}`;
  return te(key) ? String(t(key)) : status;
}

function statusVariant(status: string): 'success' | 'danger' | 'warning' | 'neutral' {
  if (status === 'READY') return 'success';
  if (status === 'LICENSE_GATE_REJECTED' || status === 'TAKEN_DOWN') return 'danger';
  if (status === 'LICENSE_GATE_PENDING' || status === 'TRANSCODING' || status === 'UPLOADED') return 'warning';
  return 'neutral';
}

function fmtDate(iso: string): string {
  try {
    return new Date(iso).toLocaleDateString(locale.value === 'zh-cn' ? 'zh-CN' : locale.value);
  } catch {
    return iso;
  }
}

const tracks = ref<Track[]>([]);
const certNumbers = reactive<Record<string, string>>({});
const loading = ref(true);
const error = ref<string | null>(null);
const statusFilter = ref('');

const filtered = computed(() =>
  statusFilter.value === '' ? tracks.value : tracks.value.filter((x) => x.status === statusFilter.value),
);

const toast = useToast();
const editGenre = ref('');
const editMoods = ref('');
const editLyrics = ref('');
const editPromptPublic = ref(false);
const saving = ref(false);
const deleteOpen = ref(false);
const deleting = ref(false);

function fillEdit(trk: Track): void {
  editGenre.value = trk.genre ?? '';
  editMoods.value = (trk.moodTags ?? []).join(', ');
  editLyrics.value = trk.lyrics ?? '';
  editPromptPublic.value = trk.isPromptPublic !== false;
}

function replaceTrack(next: Track): void {
  const i = tracks.value.findIndex((x) => x.id === next.id);
  if (i >= 0) tracks.value[i] = next;
  selected.value = next;
}

async function saveEdit(): Promise<void> {
  if (!selected.value) return;
  saving.value = true;
  try {
    const next = await partnerApi.updateMyTrack(selected.value.id, {
      genre: editGenre.value.trim(),
      moodTags: editMoods.value.split(',').map((m) => m.trim()).filter(Boolean),
      lyrics: editLyrics.value,
      isPromptPublic: editPromptPublic.value,
    });
    replaceTrack(next);
    fillEdit(next);
    toast.success(t('studio.catalog.saved'));
  } catch (err: unknown) {
    toast.error(errMessageOf(err));
  } finally {
    saving.value = false;
  }
}

async function confirmDelete(): Promise<void> {
  if (!selected.value) return;
  deleting.value = true;
  try {
    const res = await partnerApi.deleteMyTrack(selected.value.id);
    replaceTrack({ ...selected.value, status: res.status });
    deleteOpen.value = false;
    drawerOpen.value = false;
    toast.success(t('studio.catalog.deleted'));
  } catch (err: unknown) {
    toast.error(errMessageOf(err));
  } finally {
    deleting.value = false;
  }
}

const drawerOpen = ref(false);
const selected = ref<Track | null>(null);
const drawerCert = ref<CertificateView | null>(null);
const certLoading = ref(false);

const drawerItems = computed<TimelineItem[]>(() => {
  const x = selected.value;
  if (!x) return [];
  const items: TimelineItem[] = [
    { label: t('studio.catalog.tlSubmitted'), at: fmtDate(x.createdAt), state: 'done' },
  ];
  if (x.status === 'READY') {
    items.push({ label: t('studio.catalog.tlApproved'), at: fmtDate(x.updatedAt), state: 'done' });
    items.push({
      label: drawerCert.value ? t('studio.catalog.tlCertified') : t('studio.catalog.tlCertPending'),
      at: drawerCert.value ? fmtDate(drawerCert.value.certificate.issuedAt) : undefined,
      state: drawerCert.value ? 'done' : 'now',
    });
  } else if (x.status === 'LICENSE_GATE_REJECTED' || x.status === 'TAKEN_DOWN') {
    items.push({ label: statusLabel(x.status), at: fmtDate(x.updatedAt), state: 'now' });
  } else {
    items.push({ label: t('studio.catalog.tlReviewing'), at: fmtDate(x.updatedAt), state: 'now' });
    items.push({ label: t('studio.catalog.tlCertPending'), state: 'todo' });
  }
  return items;
});

async function openDrawer(trk: Track): Promise<void> {
  selected.value = trk;
  fillEdit(trk);
  drawerCert.value = null;
  drawerOpen.value = true;
  if (trk.status !== 'READY') return;
  certLoading.value = true;
  try {
    const cert = await trackApi.getCertificate(trk.id);
    drawerCert.value = cert;
    if (cert) certNumbers[trk.id] = cert.certificate.certNumber;
  } catch {
    drawerCert.value = null;
  } finally {
    certLoading.value = false;
  }
}

async function load(): Promise<void> {
  loading.value = true;
  error.value = null;
  try {
    tracks.value = (await partnerApi.listMyTracks(50, 0)) ?? [];
    // Certificate numbers ride on the list rows (no per-track request).
    for (const x of tracks.value) {
      if (x.certNumber) certNumbers[x.id] = x.certNumber;
    }
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
}

onMounted(() => void load());
</script>
