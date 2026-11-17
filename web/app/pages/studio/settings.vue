<template>
  <div class="mx-auto max-w-2xl space-y-6">
    <UiPageHeader
      :eyebrow="t('studio.settings.eyebrow')"
      :title="t('studio.settings.title')"
      :description="t('studio.settings.desc')"
    />

    <UiCard class="space-y-4">
      <div>
        <h2 class="text-sm font-bold text-white">{{ t('studio.settings.nameTitle') }}</h2>
        <p class="text-xs text-text-secondary mt-0.5">{{ t('studio.settings.nameDesc') }}</p>
      </div>
      <form class="flex flex-col sm:flex-row gap-3 sm:items-end" @submit.prevent="saveName">
        <UiInput
          v-model="displayName"
          :label="t('studio.settings.fName')"
          :placeholder="t('studio.settings.fNamePh')"
          :maxlength="30"
          required
          class="flex-1"
        />
        <UiButton variant="brand" size="sm" type="submit" :loading="nameBusy" class="shrink-0">
          {{ t('common.actions.save') }}
        </UiButton>
      </form>
      <UiErrorBanner :message="nameError" :title="t('studio.settings.nameFail')" />
    </UiCard>

    <UiCard class="space-y-4">
      <div>
        <h2 class="text-sm font-bold text-white">{{ t('studio.settings.handleTitle') }}</h2>
        <p class="text-xs text-text-secondary mt-0.5">{{ t('studio.settings.handleDesc') }}</p>
      </div>
      <form class="flex flex-col sm:flex-row gap-3 sm:items-end" @submit.prevent="claimHandle">
        <UiInput
          v-model="handle"
          :label="t('studio.settings.fHandle')"
          placeholder="creator_name"
          :maxlength="30"
          required
          class="flex-1"
        />
        <UiButton variant="brand" size="sm" type="submit" :loading="handleBusy" class="shrink-0">
          {{ t('studio.settings.claim') }}
        </UiButton>
      </form>
      <p v-if="claimedHandle" class="text-xs">
        <RouterLink :to="`/creators/${claimedHandle}`" class="font-mono text-brand-400 hover:underline">
          /creators/{{ claimedHandle }}
        </RouterLink>
      </p>
      <UiErrorBanner :message="handleError" :title="t('studio.settings.handleFail')" />
    </UiCard>

    <UiCard class="space-y-4">
      <div>
        <h2 class="text-sm font-bold text-white">{{ t('studio.settings.payoutTitle') }}</h2>
        <p class="text-xs text-text-secondary mt-0.5">{{ t('studio.settings.payoutDesc') }}</p>
      </div>
      <form class="space-y-4" @submit.prevent="savePayout">
        <div class="grid gap-3 sm:grid-cols-2">
          <UiInput
            v-model="bank"
            :label="t('studio.settings.fBank')"
            :placeholder="t('studio.settings.fBankPh')"
            required
          />
          <UiInput
            v-model="accountNumber"
            :label="t('studio.settings.fAccount')"
            :placeholder="t('studio.settings.fAccountPh')"
            inputmode="numeric"
            required
          />
        </div>
        <div class="flex justify-end">
          <UiButton variant="brand" size="sm" type="submit" :loading="payoutBusy">
            {{ t('common.actions.save') }}
          </UiButton>
        </div>
      </form>
      <UiErrorBanner :message="payoutError" :title="t('studio.settings.payoutFail')" />
    </UiCard>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { ref } from 'vue';
import { authApi, partnerApi, seoApi } from '@/api/endpoints';
import { useAuthStore } from '@/stores/authStore';
import { useToast } from '@/composables/useToast';
import { useApiError } from '@/composables/useApiError';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiInput from '@/components/ui/UiInput.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';

const { t } = useI18n();
const auth = useAuthStore();
const toast = useToast();
const apiError = useApiError();

const displayName = ref(auth.displayName ?? '');
const nameBusy = ref(false);
const nameError = ref<string | null>(null);

const handle = ref('');
const claimedHandle = ref('');
const handleBusy = ref(false);
const handleError = ref<string | null>(null);

const bank = ref('');
const accountNumber = ref('');
const payoutBusy = ref(false);
const payoutError = ref<string | null>(null);

async function saveName(): Promise<void> {
  nameBusy.value = true;
  nameError.value = null;
  try {
    const res = await authApi.updateProfile(displayName.value.trim());
    auth.displayName = res.displayName;
    toast.success(t('studio.settings.nameSaved'));
  } catch (err: unknown) {
    nameError.value = apiError(err);
  } finally {
    nameBusy.value = false;
  }
}

async function claimHandle(): Promise<void> {
  handleBusy.value = true;
  handleError.value = null;
  try {
    const res = await seoApi.claimHandle(handle.value.trim());
    claimedHandle.value = res.handle;
    toast.success(t('studio.settings.handleSaved'));
  } catch (err: unknown) {
    handleError.value = apiError(err);
  } finally {
    handleBusy.value = false;
  }
}

async function savePayout(): Promise<void> {
  payoutBusy.value = true;
  payoutError.value = null;
  try {
    await partnerApi.updateSettlementAccount(bank.value.trim(), accountNumber.value.replace(/[\s-]/g, ''));
    accountNumber.value = '';
    toast.success(t('studio.settings.payoutSaved'));
  } catch (err: unknown) {
    payoutError.value = apiError(err);
  } finally {
    payoutBusy.value = false;
  }
}
</script>
