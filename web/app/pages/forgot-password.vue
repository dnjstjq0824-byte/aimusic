<template>
  <div class="mx-auto max-w-md">
    <div class="mb-6 flex flex-col items-center gap-3 text-center">
      <BrandLogo :size="44" />
      <div>
        <h1 class="type-display text-3xl text-white sm:text-4xl">{{ t('auth.forgot.title') }}</h1>
        <p class="mt-2 text-sm text-text-secondary">{{ t('auth.forgot.subtitle') }}</p>
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
        <UiButton variant="brand" size="md" full-width type="submit" :loading="busy">
          {{ t('auth.forgot.submit') }}
        </UiButton>
        <p v-if="requested" class="text-xs leading-relaxed text-success" role="status">
          {{ t('auth.forgot.sent') }}
        </p>
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
import { ref } from 'vue';
import { authApi } from '@/api/endpoints';
import { useToast } from '@/composables/useToast';
import { useApiError } from '@/composables/useApiError';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiInput from '@/components/ui/UiInput.vue';
import BrandLogo from '@/components/BrandLogo.vue';

const { t } = useI18n();
const localePath = useLocalePath();
const toast = useToast();
const apiError = useApiError();

const email = ref('');
const busy = ref(false);
const requested = ref(false);

async function submit(): Promise<void> {
  busy.value = true;
  try {
    await authApi.requestPasswordReset(email.value);
    requested.value = true;
  } catch (err: unknown) {
    toast.error(apiError(err));
  } finally {
    busy.value = false;
  }
}
</script>
