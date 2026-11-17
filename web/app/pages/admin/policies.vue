<template>
  <div class="space-y-6">
    <UiPageHeader
      :eyebrow="t('admin.po.eyebrow')"
      :title="t('admin.po.title')"
      :description="t('admin.po.desc')"
    >
      <template #actions>
        <UiButton variant="secondary" size="sm" :loading="loading" @click="load">{{ t('admin.po.refresh') }}</UiButton>
      </template>
    </UiPageHeader>

    <UiErrorBanner :message="error" :title="t('admin.po.loadFail')" :retry-label="t('common.actions.retry')" @retry="load" />

    <UiCard class="text-xs text-text-primary space-y-1.5">
      <div class="font-bold text-white text-sm">{{ t('admin.po.guideTitle') }}</div>
      <ul class="list-disc pl-5 space-y-1 leading-relaxed">
        <li>{{ t('admin.po.guide1') }}</li>
        <li>{{ t('admin.po.guide2') }}</li>
        <li>{{ t('admin.po.guide3') }}</li>
        <li>{{ t('admin.po.guide4') }}</li>
        <li>{{ t('admin.po.guide5') }}</li>
      </ul>
    </UiCard>

    <div v-if="loading" class="space-y-2">
      <UiSkeleton height="h-12" width="w-full" />
      <UiSkeleton height="h-12" width="w-full" />
      <UiSkeleton height="h-12" width="w-full" />
    </div>

    <UiTable v-if="!loading">
      <template #header>
        <th class="p-3.5">{{ t('admin.po.colProvider') }}</th>
        <th class="p-3.5">{{ t('admin.po.colStandalone') }}</th>
        <th class="p-3.5">{{ t('admin.po.colCutoff') }}</th>
        <th class="p-3.5">{{ t('admin.po.colAllowed') }}</th>
        <th class="p-3.5">{{ t('admin.po.colSync') }}</th>
        <th class="p-3.5">{{ t('admin.po.colBlocked') }}</th>
        <th class="p-3.5 text-right">{{ t('admin.po.colManage') }}</th>
      </template>

      <tr
        v-for="p in policies"
        :key="p.provider"
        class="hover:bg-surface-2/40 transition-colors"
      >
        <td class="p-3.5 font-bold text-white">
          {{ p.provider }}
        </td>
        <td class="p-3.5">
          <UiBadge
            :variant="p.allowsStandaloneStreaming ? 'commercial' : 'syncOnly'"
            size="xs"
          >
            {{ p.allowsStandaloneStreaming ? t('admin.po.allowYes') : t('admin.po.allowNo') }}
          </UiBadge>
        </td>
        <td class="p-3.5 text-xs">
          <span v-if="p.maxGeneratedAt" class="text-accent-amber">{{ t('admin.po.cutoffAfter', { at: fmtDate(p.maxGeneratedAt) }) }}</span>
          <span v-else class="text-text-muted">{{ t('admin.po.cutoffNone') }}</span>
        </td>
        <td class="p-3.5 font-mono text-text-primary">
          {{ (p.allowedPlanTiers ?? []).join(', ') || t('admin.po.allAllowed') }}
        </td>
        <td class="p-3.5 font-mono text-accent-amber">
          {{ (p.syncOnlyPlanTiers ?? []).join(', ') || '–' }}
        </td>
        <td class="p-3.5 font-mono text-accent-crimson">
          {{ (p.blockedPlanTiers ?? []).join(', ') || '–' }}
        </td>
        <td class="p-3.5 text-right">
          <UiButton variant="outline" size="xs" @click="openEdit(p)">
            {{ t('admin.po.edit') }}
          </UiButton>
        </td>
      </tr>
    </UiTable>

    <UiModal
      v-model="editModalOpen"
      :title="t('admin.po.modalTitle', { provider: editingPolicy?.provider ?? '' })"
      :description="t('admin.po.modalDesc')"
      @close="editModalOpen = false"
    >
      <form v-if="editingPolicy" class="space-y-4 text-xs" @submit.prevent="savePolicy">
        <div>
          <UiCheckbox v-model="formAllowsStreaming">
            <strong>{{ t('admin.po.fAllow') }}</strong> {{ t('admin.po.fAllowHint') }}
          </UiCheckbox>
        </div>

        <UiInput
          v-model="formAllowedTiers"
          :label="t('admin.po.fAllowed')"
          placeholder="Pro, Premier, Studio"
        />

        <UiInput
          v-model="formSyncOnlyTiers"
          :label="t('admin.po.fSync')"
          placeholder="Standard, Creator"
        />

        <UiInput
          v-model="formBlockedTiers"
          :label="t('admin.po.fBlocked')"
          placeholder="Free, Basic, Trial"
        />

        <UiInput
          v-model="formRejectionMsg"
          :label="t('admin.po.fReason')"
          :placeholder="t('admin.po.fReasonPh')"
        />

        <UiInput
          v-model="formMaxGeneratedAt"
          :label="t('admin.po.fCutoff')"
          :placeholder="t('admin.po.fCutoffPh')"
        />

        <UiInput
          v-model="formTermsSnapshotUrl"
          :label="t('admin.po.fSnapshot')"
          :placeholder="t('admin.po.fSnapshotPh')"
        />

        <div class="flex justify-end gap-2 pt-2 border-t border-surface-border">
          <UiButton variant="ghost" size="sm" @click="editModalOpen = false">{{ t('common.actions.cancel') }}</UiButton>
          <UiButton variant="brand" size="sm" type="submit" :loading="saving">{{ t('admin.po.save') }}</UiButton>
        </div>
      </form>
    </UiModal>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { onMounted, ref } from 'vue';
import { adminApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { useToast } from '@/composables/useToast';
import { useApiError } from '@/composables/useApiError';
import type { AIProviderPolicy } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiInput from '@/components/ui/UiInput.vue';
import UiCheckbox from '@/components/ui/UiCheckbox.vue';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import UiTable from '@/components/ui/UiTable.vue';
import UiModal from '@/components/ui/UiModal.vue';

const { t, locale } = useI18n();
const toast = useToast();
const apiError = useApiError();

const policies = ref<AIProviderPolicy[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);
const editModalOpen = ref(false);
const editingPolicy = ref<AIProviderPolicy | null>(null);

const formAllowsStreaming = ref(false);
const formAllowedTiers = ref('');
const formSyncOnlyTiers = ref('');
const formBlockedTiers = ref('');
const formRejectionMsg = ref('');
const formMaxGeneratedAt = ref('');
const formTermsSnapshotUrl = ref('');
const saving = ref(false);

function fmtDate(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleDateString(locale.value === 'en' ? 'en-US' : 'ko-KR');
}

async function load() {
  loading.value = true;
  error.value = null;
  try {
    policies.value = (await adminApi.listPolicies()) ?? [];
  } catch (err) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
}

function openEdit(p: AIProviderPolicy) {
  editingPolicy.value = p;
  formAllowsStreaming.value = p.allowsStandaloneStreaming;
  formAllowedTiers.value = (p.allowedPlanTiers ?? []).join(', ');
  formSyncOnlyTiers.value = (p.syncOnlyPlanTiers ?? []).join(', ');
  formBlockedTiers.value = (p.blockedPlanTiers ?? []).join(', ');
  formRejectionMsg.value = p.rejectionMessageKo || '';
  formMaxGeneratedAt.value = p.maxGeneratedAt ?? '';
  formTermsSnapshotUrl.value = p.termsSnapshotUrl ?? '';
  editModalOpen.value = true;
}

async function savePolicy() {
  if (!editingPolicy.value) return;
  saving.value = true;
  try {
    const parse = (s: string) => s.split(',').map((x) => x.trim()).filter(Boolean);
    await adminApi.updatePolicy(editingPolicy.value.provider, {
      allowsStandaloneStreaming: formAllowsStreaming.value,
      allowedPlanTiers: parse(formAllowedTiers.value),
      syncOnlyPlanTiers: parse(formSyncOnlyTiers.value),
      blockedPlanTiers: parse(formBlockedTiers.value),
      minBitrateKbps: editingPolicy.value.minBitrateKbps || 192,
      blockedMetadataKeywords: editingPolicy.value.blockedMetadataKeywords || [],
      rejectionMessageKo: formRejectionMsg.value || t('admin.po.fallbackReject'),
      commercialTermsUrl: editingPolicy.value.commercialTermsUrl || null,
      maxGeneratedAt: formMaxGeneratedAt.value.trim() || editingPolicy.value.maxGeneratedAt || null,
      termsSnapshotUrl: formTermsSnapshotUrl.value.trim() || editingPolicy.value.termsSnapshotUrl || null,
    });
    toast.success(t('admin.po.saved'));
    editModalOpen.value = false;
    await load();
  } catch (err) {
    toast.error(apiError(err), t('admin.po.saveFail'));
  } finally {
    saving.value = false;
  }
}

onMounted(() => {
  void load();
});
</script>
