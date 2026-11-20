<template>
  <div class="mx-auto max-w-md py-8 text-center">
    <BrandLogo :size="48" />
    <div class="mt-6 space-y-3">
      <div v-if="failed" class="space-y-4">
        <h1 class="text-xl font-semibold text-white">{{ t('auth.oauth.failedTitle') }}</h1>
        <UiErrorBanner :message="error" />
        <RouterLink :to="localePath('/login')">
          <UiButton variant="brand" size="md">{{ t('auth.oauth.back') }}</UiButton>
        </RouterLink>
      </div>
      <div v-else class="space-y-3">
        <div class="flex justify-center" aria-hidden="true">
          <svg class="h-8 w-8 animate-spin text-brand-500" viewBox="0 0 24 24" fill="none">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z" />
          </svg>
        </div>
        <h1 class="text-xl font-semibold text-white">{{ t('auth.oauth.processingTitle') }}</h1>
        <p class="text-xs text-text-secondary">{{ t('auth.oauth.processingBody') }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useAuthStore } from '@/stores/authStore';
import { useToast } from '@/composables/useToast';
import { authApi } from '@/api/endpoints';
import UiButton from '@/components/ui/UiButton.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import BrandLogo from '@/components/BrandLogo.vue';

const { t } = useI18n();
const localePath = useLocalePath();
const route = useRoute();
const router = useRouter();
const auth = useAuthStore();
const toast = useToast();

const failed = ref(false);
const error = ref<string | null>(null);

onMounted(async () => {
  // C-16: the callback carries a one-time code, never tokens.
  const code = route.query.code;
  if (typeof code !== 'string' || code === '') {
    failed.value = true;
    error.value = t('auth.oauth.invalidResponse');
    return;
  }
  try {
    const { tokens } = await authApi.oauthExchange(code);
    await auth.loginWithTokens(tokens.accessToken, tokens.refreshToken);
    toast.success(t('auth.oauth.success'));
    // Strip the code from history so it can never be replayed from the URL bar.
    await router.replace(localePath('/'));
  } catch {
    failed.value = true;
    error.value = t('auth.oauth.sessionFailed');
  }
});
</script>
