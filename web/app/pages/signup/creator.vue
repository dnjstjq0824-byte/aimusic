<template>
  <div class="mx-auto max-w-md">
    <div class="mb-6 flex flex-col items-center gap-3 text-center">
      <BrandLogo :size="44" />
      <div>
        <h1 class="font-display text-2xl font-black text-white tracking-tight">{{ t('auth.creator.title') }}</h1>
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
            v-model="stageName"
            :label="t('auth.creator.stageName')"
            :placeholder="t('auth.creator.stageNamePh')"
            required
          />
          <UiInput
            v-model="country"
            :label="t('auth.creator.country')"
            :placeholder="t('auth.creator.countryPh')"
          />
          <UiInput
            v-model="aiPlatform"
            :label="t('auth.creator.aiPlatform')"
            :placeholder="t('auth.creator.aiPlatformPh')"
          />
          <p class="text-xs leading-relaxed text-text-muted">{{ t('auth.creator.profileNote') }}</p>
        </template>

        <template v-else>
          <div class="space-y-3">
            <UiCheckbox v-model="agreeContract" required>
              <span>{{ t('auth.creator.termsPre') }}</span><RouterLink :to="localePath('/legal/PARTNER_CONTRACT')" target="_blank" class="text-brand-400 underline">{{ t('auth.common.contract') }}</RouterLink><span>{{ t('auth.creator.termsMid') }}</span><RouterLink :to="localePath('/legal/COPYRIGHT_POLICY')" target="_blank" class="text-brand-400 underline">{{ t('auth.common.copyright') }}</RouterLink><span>{{ t('auth.creator.termsPost') }}</span>
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
            {{ step === 2 ? t('auth.creator.submit') : t('auth.common.next') }}
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

const stepLabels = computed(() => [t('auth.creator.step1'), t('auth.creator.step2'), t('auth.creator.step3')]);

const busy = ref(false);
const step = ref(0);
const email = ref('');
const password = ref('');
const birthYear = ref('');
const inviteCode = ref('');
const stageName = ref('');
const country = ref('');
const aiPlatform = ref('');
const agreeContract = ref(false);
const ageConfirmed = ref(false);

const canProceed = computed(() => {
  if (step.value === 0) return email.value.trim() !== '' && password.value.length >= 8;
  if (step.value === 1) return stageName.value.trim() !== '';
  return agreeContract.value && ageConfirmed.value;
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
    await auth.registerPartner({
      email: email.value,
      password: password.value,
      displayName: stageName.value,
      company: stageName.value,
      birthYear: parseBirthYear(birthYear.value),
      ageConfirmed: ageConfirmed.value,
      inviteCode: inviteCode.value.trim() || undefined,
    });
    toast.success(t('auth.creator.success'));
    await router.push(localePath('/login'));
  } catch (err: unknown) {
    toast.error(apiError(err));
  } finally {
    busy.value = false;
  }
}
</script>
