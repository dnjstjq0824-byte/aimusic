<template>
  <div class="mx-auto max-w-md">
    <div class="mb-6 flex flex-col items-center gap-3 text-center">
      <BrandLogo :size="44" />
      <div>
        <h1 class="type-display text-3xl text-white sm:text-4xl">{{ t('auth.login.title') }}</h1>
        <p class="mt-2 text-sm text-text-secondary">{{ t('auth.login.subtitle') }}</p>
      </div>
    </div>

    <UiErrorBanner :message="oauthError" :title="t('auth.login.oauthFailTitle')" class="mb-4" />

    <div v-if="registered" class="mb-4 rounded-xl border border-brand-500/40 bg-brand-500/10 px-4 py-3 text-center">
      <p class="text-sm font-bold text-white">{{ t('auth.login.registeredTitle') }}</p>
      <p class="mt-1 text-xs leading-relaxed text-text-secondary">{{ t('auth.login.registeredBody') }}</p>
    </div>

    <UiCard class="shadow-2xl">
      <div v-if="socialLoading" class="space-y-2">
        <UiSkeleton height="h-10" width="w-full" />
      </div>
      <div v-else-if="hasSocial" class="space-y-2">
        <SocialButtons :providers="providers" />
        <div class="flex items-center gap-3 py-2 text-caption text-text-muted">
          <span class="h-px flex-1 bg-surface-border" aria-hidden="true" />
          <span>{{ t('auth.login.orEmail') }}</span>
          <span class="h-px flex-1 bg-surface-border" aria-hidden="true" />
        </div>
      </div>

      <form class="space-y-4" @submit.prevent="submitLogin">
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
          :label="t('auth.login.password')"
          :placeholder="t('auth.login.passwordPh')"
          autocomplete="current-password"
          required
        >
          <template #labelRight>
            <RouterLink :to="localePath('/forgot-password')" class="font-medium text-brand-400 hover:underline">
              {{ t('auth.login.forgot') }}
            </RouterLink>
          </template>
        </UiInput>
        <UiInput
          v-if="totpStep"
          v-model="totpCode"
          :label="t('auth.login.totpLabel')"
          :placeholder="t('auth.login.totpPh')"
          autocomplete="one-time-code"
          inputmode="text"
          required
        />
        <UiButton
          variant="brand"
          size="md"
          full-width
          type="submit"
          :loading="busy"
        >
          {{ totpStep ? t('auth.login.totpSubmit') : t('auth.login.submit') }}
        </UiButton>

        <p class="text-center text-xs text-text-secondary">
          {{ t('auth.login.noAccount') }}
          <RouterLink :to="localePath('/signup')" class="font-medium text-brand-400 hover:underline">
            {{ t('auth.login.signupLink') }}
          </RouterLink>
        </p>

        <div v-if="isDev" class="pt-4 border-t border-surface-border text-center">
          <div class="text-caption text-text-secondary mb-2 font-medium">{{ t('auth.login.devTitle') }}</div>
          <div class="flex flex-wrap justify-center gap-2">
            <button
              type="button"
              class="rounded-lg bg-brand-500/10 px-3 py-1.5 text-xs text-brand-300 hover:bg-brand-500/20 border border-brand-500/40 transition-colors font-semibold"
              @click="fillCreds('partner@example.com', 'Admin1234!', true)"
            >
              🎵 {{ t('auth.login.devPartner') }}
            </button>
            <button
              type="button"
              class="rounded-lg bg-surface-2 px-3 py-1.5 text-xs text-text-primary hover:text-white hover:bg-surface-3 border border-surface-border transition-colors font-medium"
              @click="fillCreds('user@example.com', 'Admin1234!', true)"
            >
              🎧 {{ t('auth.login.devListener') }}
            </button>
            <button
              type="button"
              class="rounded-lg bg-surface-2 px-3 py-1.5 text-xs text-text-secondary hover:text-white hover:bg-surface-3 border border-surface-border transition-colors font-medium"
              @click="fillCreds('admin@example.com', 'Admin1234!', true)"
            >
              🛡️ {{ t('auth.login.devAdmin') }}
            </button>
          </div>
        </div>
      </form>
    </UiCard>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { computed, onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { safeRedirect } from '@/utils/redirect';
import { useAuthStore } from '@/stores/authStore';
import { useToast } from '@/composables/useToast';
import { useApiError } from '@/composables/useApiError';
import { authApi } from '@/api/endpoints';
import { errCodeOf } from '@/api/client';
import type { OAuthProviderInfo } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiInput from '@/components/ui/UiInput.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';
import BrandLogo from '@/components/BrandLogo.vue';
import SocialButtons from '@/components/auth/SocialButtons.vue';

const { t, locale } = useI18n();
const localePath = useLocalePath();
const auth = useAuthStore();
const router = useRouter();
const route = useRoute();
const toast = useToast();
const apiError = useApiError();

const busy = ref(false);
const isDev = import.meta.env.DEV;

const email = ref('');
const password = ref('');
const totpCode = ref('');
const totpStep = ref(false);

const providers = ref<OAuthProviderInfo[]>([]);
const socialLoading = ref(true);
const hasSocial = computed(() => providers.value.some((p) => p.enabled));

const oauthError = computed(() => {
  const code = route.query.oauthError;
  if (typeof code !== 'string' || code === '') return null;
  return apiError({ response: { data: { error: { code, message: code } } } });
});

// A2: fresh signups land here with ?registered=1 and a verification guide.
const registered = computed(() => route.query.registered === '1');

function fillCreds(e: string, p: string, autoSubmit = false) {
  email.value = e;
  password.value = p;
  if (autoSubmit) {
    void submitLogin();
  }
}

async function submitLogin(): Promise<void> {
  busy.value = true;
  try {
    await auth.login(email.value, password.value, totpStep.value ? totpCode.value : undefined);
    toast.success(t('auth.login.success'));
    await router.push(safeRedirect(route.query.redirect));
  } catch (err: unknown) {
    if (errCodeOf(err) === 'ERR_TOTP_REQUIRED' && !totpStep.value) {
      totpStep.value = true;
      toast.info(t('auth.login.totpHint'));
    } else {
      toast.error(apiError(err));
      if (errCodeOf(err) === 'ERR_TOTP_INVALID') totpCode.value = '';
    }
  } finally {
    busy.value = false;
  }
}

onMounted(async () => {
  try {
    providers.value = (await authApi.oauthProviders(locale.value)) ?? [];
  } catch {
    providers.value = [];
  } finally {
    socialLoading.value = false;
  }
});
</script>
