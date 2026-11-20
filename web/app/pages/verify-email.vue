<template>
  <div class="mx-auto max-w-md">
    <div class="mb-6 flex flex-col items-center gap-3 text-center">
      <BrandLogo :size="44" />
      <div>
        <h1 class="type-display text-3xl text-white sm:text-4xl">{{ t('auth.verify.title') }}</h1>
      </div>
    </div>

    <UiCard class="shadow-2xl">
      <div v-if="!token" class="space-y-4 text-center">
        <h2 class="text-lg font-bold text-white">{{ t('auth.verify.noTokenTitle') }}</h2>
        <p class="text-xs leading-relaxed text-text-secondary">{{ t('auth.verify.noTokenBody') }}</p>
        <div>
          <RouterLink :to="localePath('/login')" class="text-xs text-text-secondary hover:text-white hover:underline">
            {{ t('auth.common.backToLogin') }}
          </RouterLink>
        </div>
      </div>
      <div v-else-if="state === 'verifying'" class="space-y-4 text-center">
        <h2 class="text-lg font-bold text-white">{{ t('auth.verify.verifyingTitle') }}</h2>
        <UiSkeleton height="h-2" width="w-2/3" class="mx-auto" />
      </div>
      <div v-else-if="state === 'success'" class="space-y-4 text-center">
        <h2 class="text-lg font-bold text-white">{{ t('auth.verify.successTitle') }}</h2>
        <p class="text-xs leading-relaxed text-text-secondary">{{ t('auth.verify.successBody', { email: verifiedEmail }) }}</p>
        <RouterLink :to="localePath('/')">
          <UiButton variant="brand" size="md" full-width>{{ t('auth.verify.continueCta') }}</UiButton>
        </RouterLink>
      </div>
      <div v-else class="space-y-4 text-center">
        <h2 class="text-lg font-bold text-white">{{ t('auth.verify.failedTitle') }}</h2>
        <p class="text-xs leading-relaxed text-text-secondary">{{ t('auth.verify.failedBody') }}</p>
        <UiButton variant="brand" size="md" full-width :loading="resending" @click="resend">
          {{ t('auth.verify.resend') }}
        </UiButton>
        <div>
          <RouterLink :to="localePath('/login')" class="text-xs text-text-secondary hover:text-white hover:underline">
            {{ t('auth.common.backToLogin') }}
          </RouterLink>
        </div>
      </div>
    </UiCard>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { computed, onMounted, ref } from 'vue';
import { useRoute } from 'vue-router';
import { authApi } from '@/api/endpoints';
import { useAuthStore } from '@/stores/authStore';
import { useApiError } from '@/composables/useApiError';
import { useToast } from '@/composables/useToast';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import BrandLogo from '@/components/BrandLogo.vue';

const { t } = useI18n();
const localePath = useLocalePath();
const route = useRoute();
const auth = useAuthStore();
const toast = useToast();
const apiError = useApiError();

const token = computed(() => {
  const tok = route.query.token;
  return typeof tok === 'string' && tok !== '' ? tok : null;
});

const state = ref<'verifying' | 'success' | 'failed'>('verifying');
const verifiedEmail = ref('');
const resending = ref(false);

onMounted(async () => {
  if (!token.value) return;
  try {
    const res = await authApi.confirmEmailVerify(token.value);
    verifiedEmail.value = res.user.email;
    // Confirm logs the user in: apply the session and continue signed in.
    await auth.loginWithTokens(res.tokens.accessToken, res.tokens.refreshToken);
    state.value = 'success';
  } catch {
    state.value = 'failed';
  }
});

async function resend(): Promise<void> {
  if (resending.value) return;
  resending.value = true;
  try {
    await authApi.requestEmailVerify();
    toast.success(t('auth.verify.resent'));
  } catch (err: unknown) {
    toast.error(apiError(err));
  } finally {
    resending.value = false;
  }
}
</script>
