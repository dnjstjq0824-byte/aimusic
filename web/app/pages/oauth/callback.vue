<template>
  <div class="mx-auto max-w-md py-8 text-center">
    <BrandLogo :size="48" />
    <div class="mt-6 space-y-3">
      <div v-if="failed" class="space-y-4">
        <h1 class="text-xl font-black text-white">{{ t('auth.oauth.failedTitle') }}</h1>
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
        <h1 class="text-xl font-black text-white">{{ t('auth.oauth.processingTitle') }}</h1>
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
  const access = route.query.accessToken;
  const refresh = route.query.refreshToken;
  if (typeof access !== 'string' || typeof refresh !== 'string' || access === '' || refresh === '') {
    failed.value = true;
    error.value = t('auth.oauth.invalidResponse');
    return;
  }
  try {
    await auth.loginWithTokens(access, refresh);
    toast.success(t('auth.oauth.success'));
    // Strip tokens from history so they never linger in the URL bar.
    await router.replace(localePath('/'));
  } catch {
    failed.value = true;
    error.value = t('auth.oauth.sessionFailed');
  }
});
</script>
