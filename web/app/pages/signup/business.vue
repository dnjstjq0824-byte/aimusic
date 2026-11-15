<template>
  <div class="mx-auto max-w-md">
    <div class="mb-6 flex flex-col items-center gap-3 text-center">
      <BrandLogo :size="44" />
      <div>
        <h1 class="font-display text-2xl font-black text-white tracking-tight">{{ t('auth.business.title') }}</h1>
      </div>
    </div>

    <UiCard class="shadow-2xl">
      <Stepper :steps="stepLabels" :current="step" class="mb-6" />

      <form class="space-y-4" @submit.prevent="nextOrSubmit">
        <template v-if="step === 0">
          <UiInput
            v-model="email"
            type="email"
            :label="t('auth.common.email')"
            :placeholder="t('auth.common.emailPh')"
            autocomplete="email"
            required
          />
          <UiInput
            v-model="password"
            type="password"
            :label="t('auth.common.password8')"
            :placeholder="t('auth.common.passwordPh')"
            :minlength="8"
            autocomplete="new-password"
            required
          />
          <UiInput
            v-model="birthYear"
            type="number"
            :label="t('auth.common.birthYear')"
            :placeholder="t('auth.common.birthYearPh')"
          />
          <UiInput
            v-model="inviteCode"
            :label="t('auth.common.invite')"
            :placeholder="t('auth.common.invitePh')"
            autocomplete="off"
          />
        </template>

        <template v-else-if="step === 1">
          <UiInput
            v-model="company"
            :label="t('auth.business.company')"
            :placeholder="t('auth.business.companyPh')"
            required
          />
          <UiInput
            v-model="bizNo"
            :label="t('auth.business.bizNo')"
            :placeholder="t('auth.business.bizNoPh')"
            inputmode="numeric"
          />
          <UiInput
            v-model="industry"
            :label="t('auth.business.industry')"
            :placeholder="t('auth.business.industryPh')"
          />
        </template>

        <template v-else>
          <div class="rounded-xl border border-surface-border bg-surface-2 p-4 text-xs leading-relaxed text-text-secondary" role="note">
            {{ t('auth.business.notice') }}
          </div>
          <div class="space-y-3">
            <UiCheckbox v-model="agreeTerms" required>
              <span>{{ t('auth.business.termsPre') }}</span><RouterLink :to="localePath('/legal/TOS')" target="_blank" class="text-brand-400 underline">{{ t('auth.common.tos') }}</RouterLink><span>{{ t('auth.business.termsMid') }}</span><span>{{ t('auth.business.storeConditions') }}</span><span>{{ t('auth.business.termsPost') }}</span>
            </UiCheckbox>
            <UiCheckbox v-model="ageConfirmed" required>
              {{ t('auth.common.ageConfirm') }}
            </UiCheckbox>
          </div>
        </template>

        <div class="flex gap-3 pt-2">
          <UiButton
            v-if="step > 0"
            variant="secondary"
            size="md"
            type="button"
            class="flex-1"
            @click="step--"
          >
            {{ t('auth.common.prev') }}
          </UiButton>
          <UiButton
            variant="brand"
            size="md"
            type="submit"
            class="flex-1"
            :loading="busy"
            :disabled="!canProceed"
          >
            {{ step === 2 ? t('auth.business.submit') : t('auth.common.next') }}
          </UiButton>
        </div>
      </form>
    </UiCard>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { computed, ref } from 'vue';
import { useRouter } from 'vue-router';
import { useAuthStore } from '@/stores/authStore';
import { useToast } from '@/composables/useToast';
import { useApiError } from '@/composables/useApiError';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiCheckbox from '@/components/ui/UiCheckbox.vue';
import UiInput from '@/components/ui/UiInput.vue';
import Stepper from '@/components/ui/Stepper.vue';
import BrandLogo from '@/components/BrandLogo.vue';

const { t } = useI18n();
const localePath = useLocalePath();
const auth = useAuthStore();
const router = useRouter();
const toast = useToast();
const apiError = useApiError();

const stepLabels = computed(() => [t('auth.business.step1'), t('auth.business.step2'), t('auth.business.step3')]);

const busy = ref(false);
const step = ref(0);
const email = ref('');
const password = ref('');
const birthYear = ref('');
const inviteCode = ref('');
const company = ref('');
const bizNo = ref('');
const industry = ref('');
const agreeTerms = ref(false);
const ageConfirmed = ref(false);

const canProceed = computed(() => {
  if (step.value === 0) return email.value.trim() !== '' && password.value.length >= 8;
  if (step.value === 1) return company.value.trim() !== '';
  return agreeTerms.value && ageConfirmed.value;
});

function parseBirthYear(raw: string): number | undefined {
  const n = Number.parseInt(raw.trim(), 10);
  return Number.isFinite(n) ? n : undefined;
}

async function nextOrSubmit(): Promise<void> {
  if (step.value < 2) {
    if (canProceed.value) step.value++;
    return;
  }
  busy.value = true;
  try {
    // Pre-R10: no business endpoint yet — register as partner so the account
    // works today. Extended profile (bizNo/industry) wires up in R10.
    await auth.registerPartner({
      email: email.value,
      password: password.value,
      displayName: company.value,
      company: company.value,
      birthYear: parseBirthYear(birthYear.value),
      ageConfirmed: ageConfirmed.value,
      inviteCode: inviteCode.value.trim() || undefined,
    });
    toast.success(t('auth.business.success'));
    await router.push(localePath('/login'));
  } catch (err: unknown) {
    toast.error(apiError(err));
  } finally {
    busy.value = false;
  }
}
</script>
