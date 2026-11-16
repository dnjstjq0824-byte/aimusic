<template>
  <div class="space-y-6">
    <UiPageHeader
      :eyebrow="t('studio.proofs.eyebrow')"
      :title="t('studio.proofs.title')"
      :description="t('studio.proofs.desc')"
    >
      <template #actions>
        <UiButton variant="brand" size="sm" @click="createModalOpen = true">
          {{ t('studio.proofs.new') }}
        </UiButton>
      </template>
    </UiPageHeader>

    <UiErrorBanner :message="error" :title="t('studio.proofs.loadFail')" :retry-label="t('common.actions.retry')" @retry="load" />

    <UiCard padding="none">
      <template #header>
        <div class="px-5 pt-3 flex items-center justify-between">
          <h2 class="text-xs font-bold text-text-secondary uppercase tracking-wider">{{ t('studio.proofs.listTitle') }}</h2>
          <span class="text-xs text-text-muted font-mono">{{ t('studio.proofs.count', { n: subs.length }) }}</span>
        </div>
      </template>

      <div v-if="loading" class="space-y-2 p-4">
        <UiSkeleton height="h-12" width="w-full" />
        <UiSkeleton height="h-12" width="w-full" />
      </div>

      <div v-else-if="subs.length === 0" class="py-12">
        <UiEmptyState
          :title="t('studio.proofs.empty')"
          :description="t('studio.proofs.emptyDesc')"
        >
          <template #action>
            <UiButton variant="brand" size="sm" @click="createModalOpen = true">
              {{ t('studio.proofs.new') }}
            </UiButton>
          </template>
        </UiEmptyState>
      </div>

      <ul v-else class="divide-y divide-surface-border">
        <li
          v-for="s in subs"
          :key="s.id"
          class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 px-5 py-4 hover:bg-surface-2/50 transition-colors text-xs"
        >
          <div class="min-w-0 flex-1 space-y-1">
            <div class="flex items-center gap-2">
              <span class="font-bold text-white text-sm">{{ s.provider }}</span>
              <span class="font-semibold text-brand-400">{{ s.planName }}</span>
              <UiBadge
                :variant="s.status === 'VERIFIED' ? 'success' : s.status === 'REJECTED' ? 'danger' : 'warning'"
                size="xs"
              >
                {{ subStatus(s.status) }}
              </UiBadge>
            </div>
            <div class="font-mono text-caption text-text-muted">{{ t('studio.proofs.proofId', { id: s.id }) }}</div>
            <div v-if="s.expiresAt" class="text-caption text-text-secondary">
              {{ t('studio.proofs.expires', { at: fmtDate(s.expiresAt) }) }}
            </div>
          </div>

          <div class="text-caption shrink-0">
            <span v-if="s.status === 'VERIFIED'" class="text-verified font-semibold">{{ t('studio.proofs.usable') }}</span>
            <span v-else-if="s.status === 'PENDING'" class="text-accent-amber font-semibold">{{ t('studio.proofs.pending') }}</span>
            <span v-else class="text-accent-crimson font-semibold">{{ subStatus(s.status) }}</span>
          </div>
        </li>
      </ul>
    </UiCard>

    <UiModal
      v-model="createModalOpen"
      :title="t('studio.proofs.modalTitle')"
      :description="t('studio.proofs.modalDesc')"
      @close="createModalOpen = false"
    >
      <form class="space-y-4" @submit.prevent="create">
        <UiSelect v-model="provider" :label="t('studio.proofs.fProvider')" required>
          <option value="">{{ t('studio.proofs.fProviderPh') }}</option>
          <option v-for="p in providers" :key="p" :value="p">{{ p }}</option>
        </UiSelect>

        <UiInput
          v-model="planName"
          :label="t('studio.proofs.fPlan')"
          :placeholder="t('studio.proofs.fPlanPh')"
          required
        />

        <UiInput
          v-model="proofKey"
          :label="t('studio.proofs.fProof')"
          :placeholder="t('studio.proofs.fProofPh')"
        />

        <div class="flex justify-end gap-2 pt-2 border-t border-surface-border">
          <UiButton variant="ghost" size="sm" @click="createModalOpen = false">{{ t('common.actions.cancel') }}</UiButton>
          <UiButton variant="brand" size="sm" type="submit" :loading="busy">{{ t('studio.proofs.submit') }}</UiButton>
        </div>
      </form>
    </UiModal>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { onMounted, ref } from 'vue';
import { partnerApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { useToast } from '@/composables/useToast';
import { useApiError } from '@/composables/useApiError';
import type { PartnerAISubscription } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiInput from '@/components/ui/UiInput.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSelect from '@/components/ui/UiSelect.vue';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiModal from '@/components/ui/UiModal.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import UiEmptyState from '@/components/ui/UiEmptyState.vue';

const { t, te, locale } = useI18n();
const toast = useToast();
const apiError = useApiError();

const providers = ['SUNO', 'UDIO', 'AIVA', 'STABLE_AUDIO', 'SOUNDRAW', 'MUBERT', 'CUSTOM_OPEN_MODEL'];

function subStatus(status: string): string {
  const key = `studio.proofs.subStatus.${status}`;
  return te(key) ? String(t(key)) : status;
}

function fmtDate(iso: string): string {
  try {
    return new Date(iso).toLocaleDateString(locale.value === 'zh-cn' ? 'zh-CN' : locale.value);
  } catch {
    return iso;
  }
}

const subs = ref<PartnerAISubscription[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);
const busy = ref(false);
const createModalOpen = ref(false);

const provider = ref('');
const planName = ref('');
const proofKey = ref('');

async function load(): Promise<void> {
  loading.value = true;
  error.value = null;
  try {
    subs.value = (await partnerApi.listSubscriptions()) ?? [];
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
}

async function create(): Promise<void> {
  busy.value = true;
  try {
    await partnerApi.createSubscription({
      provider: provider.value,
      planName: planName.value,
      proofObjectKey: proofKey.value || undefined,
    });
    toast.success(t('studio.proofs.created'));
    provider.value = '';
    planName.value = '';
    proofKey.value = '';
    createModalOpen.value = false;
    await load();
  } catch (err: unknown) {
    toast.error(apiError(err), t('studio.proofs.createFail'));
  } finally {
    busy.value = false;
  }
}

onMounted(() => void load());
</script>
