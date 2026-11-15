<template>
  <div class="space-y-12">
    <section class="mx-auto max-w-3xl px-4 pt-4 text-center sm:pt-8">
      <UiBadge variant="brand" size="sm" dot>
        {{ t('business.crSoon') }}
      </UiBadge>
      <h1 class="font-display mt-4 text-4xl font-extrabold leading-tight tracking-tight text-white sm:text-5xl">
        {{ t('business.crHeroTitle') }}
      </h1>
      <p class="mx-auto mt-4 max-w-xl text-sm leading-relaxed text-text-secondary sm:text-base">
        {{ t('business.crHeroSub') }}
      </p>
    </section>

    <section class="mx-auto max-w-md">
      <UiCard class="p-6">
        <form v-if="!done" class="space-y-4" @submit.prevent="submit">
          <UiInput
            v-model="email"
            type="email"
            :label="t('business.crEmail')"
            :placeholder="t('business.crEmailPh')"
            autocomplete="email"
            required
          />
          <UiInput
            v-model="interest"
            :label="t('business.crInterest')"
            :placeholder="t('business.crInterestPh')"
          />
          <UiErrorBanner :message="error" />
          <UiButton variant="brand" size="md" full-width type="submit" :loading="busy" :disabled="!valid">
            {{ t('business.crSubmit') }}
          </UiButton>
        </form>
        <div v-else class="space-y-2 py-4 text-center">
          <h2 class="text-lg font-black text-white">{{ t('business.crDoneT') }}</h2>
          <p class="text-xs leading-relaxed text-text-secondary">{{ t('business.crDoneD') }}</p>
        </div>
      </UiCard>
    </section>
  </div>
</template>

<script setup lang="ts">
// G5 shares this waitlist: the form stays, generation UI lands on top.
import { computed, ref } from 'vue';
import { businessApi } from '@/api/endpoints';
import { useApiError } from '@/composables/useApiError';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiInput from '@/components/ui/UiInput.vue';

const { t, locale } = useI18n();
const apiError = useApiError();

const email = ref('');
const interest = ref('');
const busy = ref(false);
const done = ref(false);
const error = ref<string | null>(null);

const valid = computed(() => /.+@.+\..+/.test(email.value.trim()));

async function submit(): Promise<void> {
  if (!valid.value || busy.value) return;
  busy.value = true;
  error.value = null;
  try {
    await businessApi.joinWaitlist({
      email: email.value.trim(),
      interest: interest.value.trim() || undefined,
      locale: locale.value,
    });
    done.value = true;
  } catch (err: unknown) {
    error.value = apiError(err);
  } finally {
    busy.value = false;
  }
}
</script>
