<template>
  <div class="mx-auto max-w-md">
    <div class="mb-6 flex flex-col items-center gap-3 text-center">
      <BrandLogo :size="44" />
      <div>
        <h1 class="type-display text-3xl text-white sm:text-4xl">{{ t('auth.reset.title') }}</h1>
        <p class="mt-2 text-sm text-text-secondary">{{ t('auth.reset.subtitle') }}</p>
      </div>
    </div>

    <UiCard class="shadow-2xl">
      <form class="space-y-4" @submit.prevent="submit">
        <UiInput
          v-model="newPassword"
          type="password"
          :label="t('auth.reset.newPassword')"
          :placeholder="t('auth.reset.newPasswordPh')"
          :minlength="8"
          autocomplete="new-password"
          required
        />
        <UiInput
          v-model="confirmPassword"
          type="password"
          :label="t('auth.reset.confirm')"
          :placeholder="t('auth.reset.confirmPh')"
          autocomplete="new-password"
          required
        />
        <UiErrorBanner :message="mismatch" />
        <UiButton variant="brand" size="md" full-width type="submit" :loading="busy" :disabled="mismatch !== null">
          {{ t('auth.reset.submit') }}
        </UiButton>
        <div class="text-center">
          <RouterLink :to="localePath('/login')" class="text-xs text-text-secondary hover:text-white hover:underline">
            {{ t('auth.common.backToLogin') }}
          </RouterLink>
        </div>
      </form>
    </UiCard>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { computed, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { authApi } from '@/api/endpoints';
import { useToast } from '@/composables/useToast';
import { useApiError } from '@/composables/useApiError';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiInput from '@/components/ui/UiInput.vue';
import BrandLogo from '@/components/BrandLogo.vue';

const { t } = useI18n();
const localePath = useLocalePath();
const route = useRoute();
const router = useRouter();
const toast = useToast();
const apiError = useApiError();

const token = computed(() => {
  const tok = route.query.token;
  return typeof tok === 'string' && tok !== '' ? tok : null;
});

const newPassword = ref('');
const confirmPassword = ref('');
const busy = ref(false);

const mismatch = computed(() => {
  if (confirmPassword.value === '') return null;
  return newPassword.value === confirmPassword.value ? null : t('auth.reset.mismatch');
});

async function submit(): Promise<void> {
  if (!token.value || mismatch.value) return;
  busy.value = true;
  try {
    await authApi.confirmPasswordReset(token.value, newPassword.value);
    toast.success(t('auth.reset.success'));
    await router.push(localePath('/login'));
  } catch (err: unknown) {
    toast.error(apiError(err));
  } finally {
    busy.value = false;
  }
}
</script>
