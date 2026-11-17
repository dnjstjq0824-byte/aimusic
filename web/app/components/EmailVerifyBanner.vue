<template>
  <div
    v-if="visible"
    class="border-b border-surface-border bg-surface-1 px-4 py-2 text-center text-body-sm text-text-secondary"
  >
    {{ t('auth.verify.bannerText') }}
    <button
      type="button"
      class="ml-2 font-bold text-brand-400 hover:underline disabled:opacity-50"
      :disabled="busy"
      @click="resend"
    >
      {{ t('auth.verify.resend') }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { authApi } from '@/api/endpoints';
import { useAuthStore } from '@/stores/authStore';
import { useApiError } from '@/composables/useApiError';
import { useToast } from '@/composables/useToast';

// Persistent banner for signed-in but unverified accounts (A2): listening
// stays open, commerce waits for the mailed link. No dismiss — it clears on
// verify (auth.emailVerified flips after the next /me refresh).
const { t } = useI18n();
const auth = useAuthStore();
const toast = useToast();
const apiError = useApiError();

const busy = ref(false);
const visible = computed(() => auth.isAuthenticated && auth.emailVerified === false);

async function resend(): Promise<void> {
  if (busy.value) return;
  busy.value = true;
  try {
    await authApi.requestEmailVerify();
    toast.success(t('auth.verify.resent'));
  } catch (err: unknown) {
    toast.error(apiError(err));
  } finally {
    busy.value = false;
  }
}
</script>
