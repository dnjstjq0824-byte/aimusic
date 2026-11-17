<template>
  <form v-if="!done" class="space-y-4" @submit.prevent="submit">
    <div class="grid gap-4 sm:grid-cols-2">
      <UiInput v-model="form.company" :label="t('business.fCompany')" :placeholder="t('business.fCompanyPh')" required />
      <UiInput v-model="form.country" :label="t('business.fCountry')" :placeholder="t('business.fCountryPh')" />
      <UiInput v-model="stores" type="number" :label="t('business.fStores')" :placeholder="t('business.fStoresPh')" inputmode="numeric" min="0" />
      <UiInput v-model="form.useCase" :label="t('business.fUse')" :placeholder="t('business.fUsePh')" />
      <UiInput v-model="form.contactName" :label="t('business.fName')" :placeholder="t('business.fNamePh')" autocomplete="name" />
      <UiInput v-model="form.contactEmail" type="email" :label="t('business.fEmail')" :placeholder="t('business.fEmailPh')" autocomplete="email" required />
      <UiInput v-model="form.contactPhone" :label="t('business.fPhone')" :placeholder="t('business.fPhonePh')" autocomplete="tel" class="sm:col-span-2" />
    </div>
    <div>
      <label for="inq-msg" class="mb-1.5 block text-xs font-semibold text-text-secondary">
        {{ t('business.fMsg') }} <span class="text-accent-crimson" aria-hidden="true">*</span>
      </label>
      <textarea
        id="inq-msg"
        v-model="form.message"
        rows="4"
        required
        :placeholder="t('business.fMsgPh')"
        class="w-full rounded-xl border border-surface-border bg-surface-1 px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:border-brand-500 focus:outline-none"
      />
    </div>
    <!-- Honeypot: invisible to humans, bots fill it. -->
    <div class="hidden" aria-hidden="true">
      <label>Website<input v-model="form.website" type="text" tabindex="-1" autocomplete="off" /></label>
    </div>
    <UiErrorBanner :message="error" />
    <UiButton variant="brand" size="md" full-width type="submit" :loading="busy" :disabled="!valid">
      {{ t('business.submit') }}
    </UiButton>
  </form>
  <div v-else class="space-y-2 py-6 text-center">
    <h3 class="text-lg font-semibold text-white">{{ t('business.successT') }}</h3>
    <p class="text-xs leading-relaxed text-text-secondary">{{ t('business.successD') }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { businessApi, type InquiryInput } from '@/api/endpoints';
import { useApiError } from '@/composables/useApiError';
import { trackEvent } from '@/utils/analytics';
import UiButton from '@/components/ui/UiButton.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiInput from '@/components/ui/UiInput.vue';

const { t } = useI18n();
const apiError = useApiError();

const form = ref<InquiryInput>({ company: '', contactEmail: '', message: '' });
const stores = ref('');
const busy = ref(false);
const done = ref(false);
const error = ref<string | null>(null);

const valid = computed(
  () => form.value.company.trim() !== '' &&
    /.+@.+\..+/.test(form.value.contactEmail.trim()) &&
    form.value.message.trim() !== '',
);

onMounted(() => {
  trackEvent({ name: 'inquiry_opened' });
});

async function submit(): Promise<void> {
  if (!valid.value || busy.value) return;
  busy.value = true;
  error.value = null;
  try {
    const n = Number.parseInt(stores.value.trim(), 10);
    await businessApi.submitInquiry({
      ...form.value,
      company: form.value.company.trim(),
      contactEmail: form.value.contactEmail.trim(),
      message: form.value.message.trim(),
      storeCount: Number.isFinite(n) && n >= 0 ? n : undefined,
    });
    trackEvent({ name: 'inquiry_submitted', props: { useCase: form.value.useCase || 'unknown' } });
    done.value = true;
  } catch (err: unknown) {
    error.value = apiError(err);
  } finally {
    busy.value = false;
  }
}
</script>
