<template>
  <div class="space-y-6">
    <UiPageHeader
      :eyebrow="t('admin.cu.eyebrow')"
      :title="t('admin.cu.title')"
      :description="t('admin.cu.desc')"
    />

    <UiErrorBanner :message="error" :title="t('admin.cu.workFail')" />

    <div class="grid gap-6 md:grid-cols-2">
      <UiCard class="space-y-4">
        <div>
          <h2 class="text-sm font-bold text-white">{{ t('admin.cu.playlistTitle') }}</h2>
          <p class="text-xs text-text-secondary mt-0.5">{{ t('admin.cu.playlistDesc') }}</p>
        </div>

        <form class="space-y-3" @submit.prevent="createCuration">
          <UiInput
            v-model="curTitle"
            :label="t('admin.cu.fTitle')"
            :placeholder="t('admin.cu.fTitlePh')"
            required
          />
          <UiInput
            v-model="curDesc"
            :label="t('admin.cu.fDesc')"
            :placeholder="t('admin.cu.fDescPh')"
          />
          <UiInput
            v-model="curTracks"
            :label="t('admin.cu.fTracks')"
            placeholder="uuid1, uuid2, uuid3…"
          />
          <UiButton variant="brand" size="sm" type="submit" :loading="curCreating">
            {{ t('admin.cu.create') }}
          </UiButton>
        </form>
      </UiCard>

      <UiCard class="space-y-4" border-variant="danger">
        <div>
          <h2 class="text-sm font-bold text-accent-crimson flex items-center gap-1.5">
            <span>{{ t('admin.cu.tdTitle') }}</span>
          </h2>
          <p class="text-xs text-text-primary mt-0.5">
            {{ t('admin.cu.tdDesc') }}
          </p>
        </div>

        <form class="space-y-3" @submit.prevent="takedownConfirmOpen = true">
          <UiInput
            v-model="tdTrackId"
            :label="t('admin.cu.tdTrack')"
            placeholder="00000000-0000-0000-0000-000000000000"
            required
          />
          <UiInput
            v-model="tdReason"
            :label="t('admin.cu.tdReason')"
            :placeholder="t('admin.cu.tdReasonPh')"
            required
          />
          <UiButton variant="danger" size="sm" type="submit" :loading="tdBusy">
            {{ t('admin.cu.tdGo') }}
          </UiButton>
        </form>
      </UiCard>
    </div>

    <UiCard class="space-y-4">
      <div>
        <div class="inline-flex items-center gap-1.5 text-xs text-brand-400 font-bold uppercase tracking-wider mb-1">
          <span>{{ t('admin.cu.stEyebrow') }}</span>
        </div>
        <h2 class="text-sm font-bold text-white">{{ t('admin.cu.stTitle') }}</h2>
        <p class="text-xs text-text-secondary mt-0.5">
          {{ t('admin.cu.stDesc') }}
        </p>
      </div>

      <form class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4" @submit.prevent="runSettlement">
        <UiInput
          v-model="stPartnerId"
          :label="t('admin.cu.stPartner')"
          placeholder="00000000-0000-0000-0000-000000000000"
          required
        />
        <UiInput
          v-model="stPeriodStart"
          :label="t('admin.cu.stStart')"
          type="date"
          required
        />
        <UiInput
          v-model="stPeriodEnd"
          :label="t('admin.cu.stEnd')"
          type="date"
          required
        />
        <UiInput
          v-model.number="stStorePool"
          :label="t('admin.cu.stPool')"
          type="number"
          :min="0"
          placeholder="0"
        />
        <UiInput
          v-model.number="stDirect"
          :label="t('admin.cu.stDirect')"
          type="number"
          :min="0"
          placeholder="0"
        />
        <UiSelect
          v-model="stIncomeType"
          :label="t('admin.cu.stIncome')"
        >
          <option value="BUSINESS_33">{{ t('admin.cu.income33') }}</option>
          <option value="OTHER_88">{{ t('admin.cu.income88') }}</option>
        </UiSelect>

        <div class="sm:col-span-2 lg:col-span-4 flex justify-end">
          <UiButton variant="brand" size="sm" type="submit" :loading="calculating">
            {{ t('admin.cu.stRun') }}
          </UiButton>
        </div>
      </form>

      <div v-if="lastSettlement" class="rounded-xl bg-surface-0 p-4 border border-surface-border text-xs space-y-2">
        <div class="flex items-center justify-between">
          <span class="font-bold text-white text-sm">{{ t('admin.cu.stDone', { id: lastSettlement.id }) }}</span>
          <UiBadge variant="success" size="sm">{{ settleStatus(lastSettlement.status) }}</UiBadge>
        </div>
        <div class="grid grid-cols-2 sm:grid-cols-4 gap-2 pt-2 border-t border-surface-border">
          <div>
            <span class="text-text-secondary text-caption block">{{ t('admin.cu.stSecs') }}</span>
            <span class="font-bold text-white">{{ t('admin.cu.stSecsValue', { a: lastSettlement.storePlaySecs.toLocaleString(), b: lastSettlement.totalStorePlaySecs.toLocaleString() }) }}</span>
          </div>
          <div>
            <span class="text-text-secondary text-caption block">{{ t('admin.cu.stPlays') }}</span>
            <span class="font-bold text-white">{{ t('admin.cu.stPlaysValue', { n: lastSettlement.playCount.toLocaleString() }) }}</span>
          </div>
          <div>
            <span class="text-text-secondary text-caption block">{{ t('admin.cu.stGross') }}</span>
            <span class="font-bold text-white">{{ t('admin.cu.krw', { n: lastSettlement.grossAmountKrw.toLocaleString() }) }}</span>
          </div>
          <div>
            <span class="text-text-secondary text-caption block">{{ t('admin.cu.stTax', { rate: lastSettlement.incomeType === 'OTHER_88' ? '8.8%' : '3.3%' }) }}</span>
            <span class="font-bold text-accent-crimson">{{ t('admin.cu.krwNeg', { n: lastSettlement.withholdingKrw.toLocaleString() }) }}</span>
          </div>
          <div>
            <span class="text-text-secondary text-caption block">{{ t('admin.cu.stNet') }}</span>
            <span class="font-black text-brand-400 text-sm">{{ t('admin.cu.krw', { n: lastSettlement.netAmountKrw.toLocaleString() }) }}</span>
          </div>
        </div>
        <div class="flex justify-end pt-2 border-t border-surface-border">
          <UiButton
            variant="brand"
            size="sm"
            :disabled="lastSettlement.status !== 'PENDING'"
            :loading="approving"
            @click="approveLastSettlement"
          >
            {{ t('admin.cu.stApprove') }}
          </UiButton>
        </div>
      </div>
    </UiCard>

    <UiConfirmDialog
      v-model="takedownConfirmOpen"
      :title="t('admin.cu.tdConfirmTitle')"
      :description="tdTrackId ? t('admin.cu.tdConfirmDesc', { id: tdTrackId }) : t('admin.cu.tdConfirmDescNoId')"
      :body="t('admin.cu.tdConfirmBody')"
      :confirm-label="t('admin.cu.tdConfirmGo')"
      danger
      :loading="tdBusy"
      @confirm="runTakedown"
    />
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { ref } from 'vue';
import { adminApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { useToast } from '@/composables/useToast';
import { useApiError } from '@/composables/useApiError';
import type { PartnerSettlement } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiConfirmDialog from '@/components/ui/UiConfirmDialog.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiInput from '@/components/ui/UiInput.vue';
import UiSelect from '@/components/ui/UiSelect.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';

const { t, te } = useI18n();
const toast = useToast();
const apiError = useApiError();

function settleStatus(status: string): string {
  const key = `admin.cu.settleStatus.${status}`;
  return te(key) ? String(t(key)) : status;
}

const curTitle = ref('');
const curDesc = ref('');
const curTracks = ref('');
const curCreating = ref(false);

const tdTrackId = ref('');
const tdReason = ref('');
const tdBusy = ref(false);

const stPartnerId = ref('');
const stPeriodStart = ref(prevMonthStart());
const stPeriodEnd = ref(prevMonthEnd());
const stStorePool = ref(0);
const stDirect = ref(0);
const stIncomeType = ref('BUSINESS_33');
const lastSettlement = ref<PartnerSettlement | null>(null);
const calculating = ref(false);
const approving = ref(false);
const error = ref<string | null>(null);
const takedownConfirmOpen = ref(false);

function prevMonthStart(): string {
  const now = new Date();
  const first = new Date(now.getFullYear(), now.getMonth() - 1, 1);
  return `${first.getFullYear()}-${String(first.getMonth() + 1).padStart(2, '0')}-01`;
}

function prevMonthEnd(): string {
  const now = new Date();
  const last = new Date(now.getFullYear(), now.getMonth(), 0);
  return `${last.getFullYear()}-${String(last.getMonth() + 1).padStart(2, '0')}-${String(last.getDate()).padStart(2, '0')}`;
}

async function createCuration() {
  if (!curTitle.value.trim()) return;
  curCreating.value = true;
  try {
    const trackIds = curTracks.value.split(',').map((s) => s.trim()).filter(Boolean);
    await adminApi.createCurationPlaylist(curTitle.value.trim(), curDesc.value.trim() || undefined, trackIds);
    toast.success(t('admin.cu.playlistCreated'));
    curTitle.value = '';
    curDesc.value = '';
    curTracks.value = '';
  } catch (err) {
    toast.error(apiError(err), t('admin.cu.createFail'));
  } finally {
    curCreating.value = false;
  }
}

async function runTakedown() {
  if (!tdTrackId.value.trim() || !tdReason.value.trim()) return;
  tdBusy.value = true;
  try {
    await adminApi.takedown(tdTrackId.value.trim(), tdReason.value.trim());
    toast.warning(t('admin.cu.tdDone'));
    tdTrackId.value = '';
    tdReason.value = '';
    takedownConfirmOpen.value = false;
  } catch (err) {
    toast.error(apiError(err), t('admin.cu.tdFail'));
  } finally {
    tdBusy.value = false;
  }
}

async function runSettlement() {
  if (!stPartnerId.value.trim()) {
    toast.error(t('admin.cu.stNeedPartner'));
    return;
  }
  calculating.value = true;
  try {
    lastSettlement.value = await adminApi.calculateSettlement(
      stPartnerId.value.trim(),
      stPeriodStart.value,
      stPeriodEnd.value,
      stStorePool.value || 0,
      stDirect.value || 0,
      stIncomeType.value,
    );
    toast.success(t('admin.cu.stCalculated'));
  } catch (err) {
    toast.error(apiError(err), t('admin.cu.stCalcFail'));
  } finally {
    calculating.value = false;
  }
}

async function approveLastSettlement() {
  if (!lastSettlement.value || lastSettlement.value.status !== 'PENDING') return;
  approving.value = true;
  try {
    lastSettlement.value = await adminApi.approveSettlement(
      stPartnerId.value.trim(),
      stPeriodStart.value,
      stPeriodEnd.value,
    );
    toast.success(t('admin.cu.stApproved'));
  } catch (err) {
    toast.error(apiError(err), t('admin.cu.stApproveFail'));
  } finally {
    approving.value = false;
  }
}
</script>
