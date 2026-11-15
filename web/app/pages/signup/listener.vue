<template>
  <div class="mx-auto max-w-md">
    <div class="mb-6 flex flex-col items-center gap-3 text-center">
      <BrandLogo :size="44" />
      <div>
        <h1 class="font-display text-2xl font-black text-white tracking-tight">{{ t('auth.listener.title') }}</h1>
        <p class="text-xs text-text-secondary mt-1">{{ t('auth.listener.subtitle') }}</p>
      </div>
    </div>

    <UiCard class="shadow-2xl">
      <form class="space-y-4" @submit.prevent="submit">
        <UiInput
          v-model="email"
          type="email"
          :label="t('auth.common.email')"
          :placeholder="t('auth.common.emailPh')"
          autocomplete="email"
          required
        />
        <UiInput
          v-model="name"
          :label="t('auth.listener.name')"
          :placeholder="t('auth.listener.namePh')"
          autocomplete="nickname"
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

        <div class="pt-2 space-y-3">
          <UiCheckbox v-model="agreeTerms" required>
            <span>{{ t('auth.listener.termsPre') }}</span><RouterLink :to="localePath('/legal/TOS')" target="_blank" class="text-brand-400 underline">{{ t('auth.common.tos') }}</RouterLink><span>{{ t('auth.listener.termsMid') }}</span><RouterLink :to="localePath('/legal/PRIVACY')" target="_blank" class="text-brand-400 underline">{{ t('auth.common.privacy') }}</RouterLink><span>{{ t('auth.listener.termsPost') }}</span>
          </UiCheckbox>
          <UiCheckbox v-model="ageConfirmed" required>
            {{ t('auth.common.ageConfirm') }}
          </UiCheckbox>
        </div>

        <UiButton
          variant="brand"
          size="md"
          full-width
          type="submit"
          :loading="busy"
          :disabled="!agreeTerms || !ageConfirmed"
        >
          {{ t('auth.listener.submit') }}
        </UiButton>

        <p class="text-center text-xs text-text-secondary">
          {{ t('auth.signup.hasAccount') }}
          <RouterLink :to="localePath('/login')" class="font-medium text-brand-400 hover:underline">
            {{ t('auth.signup.loginLink') }}
          </RouterLink>
        </p>
      </form>
    </UiCard>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { ref } from 'vue';
import { useRouter } from 'vue-router';
import { useAuthStore } from '@/stores/authStore';
import { useToast } from '@/composables/useToast';
import { useApiError } from '@/composables/useApiError';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiCheckbox from '@/components/ui/UiCheckbox.vue';
import UiInput from '@/components/ui/UiInput.vue';
import BrandLogo from '@/components/BrandLogo.vue';

const { t } = useI18n();
const localePath = useLocalePath();
const auth = useAuthStore();
const router = useRouter();
const toast = useToast();
const apiError = useApiError();

const busy = ref(false);
const email = ref('');
const password = ref('');
const name = ref('');
const birthYear = ref('');
const inviteCode = ref('');
const agreeTerms = ref(false);
const ageConfirmed = ref(false);

function parseBirthYear(raw: string): number | undefined {
  const n = Number.parseInt(raw.trim(), 10);
  return Number.isFinite(n) ? n : undefined;
}

async function submit(): Promise<void> {
  busy.value = true;
  try {
    await auth.register({
      email: email.value,
      password: password.value,
      displayName: name.value,
      birthYear: parseBirthYear(birthYear.value),
      ageConfirmed: ageConfirmed.value,
      inviteCode: inviteCode.value.trim() || undefined,
    });
    toast.success(t('auth.listener.success'));
    await router.push(localePath('/login'));
  } catch (err: unknown) {
    toast.error(apiError(err));
  } finally {
    busy.value = false;
  }
}
</script>
