<template>
  <div class="mx-auto max-w-md">
    <div class="mb-6 flex flex-col items-center gap-3 text-center">
      <BrandLogo :size="44" />
      <div>
        <h1 class="font-display text-2xl font-black text-white tracking-tight">{{ t('auth.completeEmail.title') }}</h1>
        <p class="text-xs text-text-secondary mt-1">{{ t('auth.completeEmail.subtitle') }}</p>
      </div>
    </div>

    <UiCard class="shadow-2xl">
      <div v-if="noToken" class="space-y-4 text-center">
        <h2 class="text-base font-bold text-white">{{ t('auth.completeEmail.noTokenTitle') }}</h2>
        <p class="text-xs leading-relaxed text-text-secondary">{{ t('auth.completeEmail.noTokenBody') }}</p>
        <RouterLink :to="localePath('/login')">
          <UiButton variant="brand" size="md" full-width>{{ t('auth.common.backToLogin') }}</UiButton>
        </RouterLink>
      </div>
      <div v-else-if="submittedEmail" class="space-y-4 text-center">
        <h2 class="text-base font-bold text-white">{{ t('auth.completeEmail.pendingTitle') }}</h2>
        <p class="text-xs leading-relaxed text-text-secondary">{{ t('auth.completeEmail.pendingBody', { email: submittedEmail }) }}</p>
        <RouterLink :to="localePath('/login')">
          <UiButton variant="brand" size="md" full-width>{{ t('auth.common.backToLogin') }}</UiButton>
        </RouterLink>
      </div>
      <form v-else class="space-y-4" @submit.prevent="submit">
        <UiInput
          v-model="email"
          type="email"
          :label="t('auth.common.email')"
          :placeholder="t('auth.common.emailPh')"
          autocomplete="email"
          required
        />
        <UiErrorBanner v-if="formError" :message="formError" />
        <UiButton variant="brand" size="md" full-width type="submit" :loading="busy">
          {{ t('auth.completeEmail.submit') }}
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
import { ref } from 'vue';
import { useRoute } from 'vue-router';
import { authApi } from '@/api/endpoints';
import { useApiError } from '@/composables/useApiError';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiInput from '@/components/ui/UiInput.vue';
import BrandLogo from '@/components/BrandLogo.vue';

const { t, locale } = useI18n();
const localePath = useLocalePath();
const route = useRoute();
const apiError = useApiError();

const tokenQuery = route.query.token;
const noToken = typeof tokenQuery !== 'string' || tokenQuery === '';

const email = ref('');
const busy = ref(false);
const formError = ref<string | null>(null);
const submittedEmail = ref<string | null>(null);

async function submit(): Promise<void> {
  if (noToken || busy.value) return;
  busy.value = true;
  formError.value = null;
  try {
    const res = await authApi.oauthCompleteEmail(tokenQuery as string, email.value.trim(), locale.value);
    submittedEmail.value = res.email;
  } catch (err) {
    formError.value = apiError(err);
  } finally {
    busy.value = false;
  }
}
</script>
