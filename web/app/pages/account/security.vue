<template>
  <div class="space-y-6">
    <RouterLink
      :to="localePath('/account')"
      class="inline-flex min-h-11 items-center text-sm font-semibold text-text-secondary hover:text-white md:hidden"
    >
      <span aria-hidden="true" class="mr-1">‹</span>
      {{ t('account.navBack') }}
    </RouterLink>
    <div class="flex flex-col gap-6 md:flex-row">
      <AccountNav class="hidden md:block" />
      <div class="min-w-0 flex-1 space-y-6">
        <UiPageHeader :title="t('account.secTitle')" :description="t('account.secSub')" />

        <!-- S-03: admins are sent here until they enroll a second factor -->
        <div
          v-if="adminEnroll"
          class="rounded-xl border border-amber-500/40 bg-amber-950/30 p-4 text-sm text-amber-100"
          role="status"
        >
          {{ t('account.secAdminBanner') }}
          <div v-if="auth.totpEnabled === true" class="mt-3">
            <UiButton variant="brand" size="sm" @click="router.push(adminReturn)">
              {{ t('account.secAdminBack') }}
            </UiButton>
          </div>
        </div>

        <!-- 2FA card -->
        <UiCard>
          <h2 class="text-sm font-bold text-white mb-1">{{ t('account.sec2faT') }}</h2>
          <p class="text-xs text-text-secondary mb-4">
            {{ t('account.sec2faD') }}
          </p>

          <UiErrorBanner :message="totpError" />

          <!-- Enroll: show secret -->
          <div v-if="enrollment && !recoveryCodes" class="space-y-4">
            <div class="rounded-lg bg-surface-2 p-3">
              <div class="text-caption text-text-secondary mb-1">{{ t('account.secSecretLabel') }}</div>
              <code class="block break-all font-mono text-sm text-brand-300">{{ enrollment.secret }}</code>
              <button
                type="button"
                class="mt-2 text-xs text-text-secondary hover:text-white underline"
                @click="copy(enrollment.secret)"
              >
                {{ t('account.secSecretCopy') }}
              </button>
            </div>
            <p class="text-caption text-text-muted break-all">
              {{ t('account.secUriHelp') }}
              <code class="font-mono">{{ enrollment.provisioningUri }}</code>
            </p>
            <form class="space-y-3" @submit.prevent="submitConfirm">
              <UiInput
                v-model="confirmCode"
                :label="t('account.secCode')"
                placeholder="000000"
                inputmode="numeric"
                autocomplete="one-time-code"
                required
              />
              <UiButton variant="brand" size="md" full-width type="submit" :loading="busy">
                {{ t('account.secConfirm') }}
              </UiButton>
            </form>
          </div>

          <!-- Recovery codes (shown once) -->
          <div v-else-if="recoveryCodes" class="space-y-4">
            <p class="text-xs text-success font-semibold" role="status">
              {{ t('account.secRecoveryDone') }}
            </p>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
              <code
                v-for="c in recoveryCodes"
                :key="c"
                class="rounded-lg bg-surface-2 px-3 py-2 font-mono text-xs text-white"
              >
                {{ c }}
              </code>
            </div>
            <div class="flex gap-2">
              <UiButton variant="secondary" size="sm" type="button" @click="copy(recoveryCodes.join('\n'))">
                {{ t('account.secRecoveryCopyAll') }}
              </UiButton>
              <UiButton variant="brand" size="sm" type="button" @click="dismissRecovery">
                {{ t('account.secRecoveryDismiss') }}
              </UiButton>
            </div>
          </div>

          <!-- Default state: exactly one action matching the real 2FA state -->
          <div v-else class="space-y-4">
            <p class="text-xs" role="status">
              <span class="text-text-secondary">{{ t('account.secStatus') }}: </span>
              <span v-if="auth.totpEnabled === true" class="font-bold text-success">{{ t('account.sec2faOn') }}</span>
              <span v-else-if="auth.totpEnabled === false" class="font-bold text-text-primary">{{ t('account.sec2faOff') }}</span>
              <span v-else class="font-bold text-accent-amber">{{ t('account.secChecking') }}</span>
            </p>
            <div v-if="auth.totpEnabled === false">
              <UiButton variant="secondary" size="md" type="button" :loading="busy" @click="submitEnroll">
                {{ t('account.secEnroll') }}
              </UiButton>
            </div>
            <form v-else-if="auth.totpEnabled === true" class="space-y-3" @submit.prevent="submitDisable">
              <p class="text-xs text-text-secondary">{{ t('account.secDisableD') }}</p>
              <UiInput
                v-model="disablePassword"
                type="password"
                :label="t('account.secDisablePw')"
                :placeholder="t('account.secDisablePwPh')"
                autocomplete="current-password"
                required
              />
              <UiButton variant="secondary" size="md" type="button" :loading="busy" @click="submitDisable">
                {{ t('account.secDisable') }}
              </UiButton>
            </form>
            <div v-else>
              <UiButton variant="secondary" size="md" type="button" :loading="busy" @click="refreshTotpState">
                {{ t('account.secRefresh') }}
              </UiButton>
            </div>
          </div>
        </UiCard>

        <!-- Sessions card -->
        <UiCard>
          <h2 class="text-sm font-bold text-white mb-1">{{ t('account.secSessT') }}</h2>
          <p class="text-xs text-text-secondary mb-4">
            {{ t('account.secSessD') }}
          </p>
          <UiButton variant="secondary" size="md" type="button" :loading="busy" @click="submitLogoutAll">
            {{ t('account.secLogoutAll') }}
          </UiButton>
        </UiCard>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { computed, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { safeRedirect } from '@/utils/redirect';
import { useAuthStore } from '@/stores/authStore';
import { useToast } from '@/composables/useToast';
import { authApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import AccountNav from '@/components/account/AccountNav.vue';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiInput from '@/components/ui/UiInput.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';

const { t } = useI18n();
const localePath = useLocalePath();
const auth = useAuthStore();
const router = useRouter();
const route = useRoute();
const adminEnroll = computed(() => route.query.enroll === 'admin');
const adminReturn = computed(() => {
  const target = safeRedirect(route.query.redirect);
  return target.startsWith('/admin') ? target : '/admin';
});
const toast = useToast();
const busy = ref(false);

const enrollment = ref<{ secret: string; provisioningUri: string } | null>(null);
const confirmCode = ref('');
const recoveryCodes = ref<string[] | null>(null);
const disablePassword = ref('');
const totpError = ref<string | null>(null);

async function copy(text: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(text);
    toast.success(t('account.secCopied'));
  } catch {
    toast.error(t('account.secCopyFail'));
  }
}

async function submitEnroll(): Promise<void> {
  busy.value = true;
  totpError.value = null;
  try {
    enrollment.value = await authApi.enrollTOTP();
  } catch (err: unknown) {
    totpError.value = errMessageOf(err);
  } finally {
    busy.value = false;
  }
}

async function submitConfirm(): Promise<void> {
  busy.value = true;
  totpError.value = null;
  try {
    const res = await authApi.confirmTOTPEnrollment(confirmCode.value);
    recoveryCodes.value = res.recoveryCodes;
    confirmCode.value = '';
    auth.totpEnabled = true;
  } catch (err: unknown) {
    totpError.value = errMessageOf(err);
  } finally {
    busy.value = false;
  }
}

function dismissRecovery(): void {
  recoveryCodes.value = null;
  enrollment.value = null;
}

async function refreshTotpState(): Promise<void> {
  busy.value = true;
  totpError.value = null;
  try {
    await auth.hydrate();
  } catch (err: unknown) {
    totpError.value = errMessageOf(err);
  } finally {
    busy.value = false;
  }
}

async function submitDisable(): Promise<void> {
  busy.value = true;
  totpError.value = null;
  try {
    await authApi.disableTOTP(disablePassword.value);
    disablePassword.value = '';
    auth.totpEnabled = false;
    toast.success(t('account.secTotpOff'));
  } catch (err: unknown) {
    totpError.value = errMessageOf(err);
  } finally {
    busy.value = false;
  }
}

async function submitLogoutAll(): Promise<void> {
  busy.value = true;
  try {
    await authApi.logoutAll();
    auth.logout();
    toast.success(t('account.secLoggedOut'));
    await router.push('/login');
  } catch (err: unknown) {
    toast.error(errMessageOf(err), t('account.secLogoutFail'));
  } finally {
    busy.value = false;
  }
}
</script>
