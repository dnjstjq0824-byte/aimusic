<template>
  <UiModal
    v-if="auth.needsTermsConsent && !auth.needsAgeConfirm"
    :model-value="true"
    :title="t('legal.consentGate.title')"
    @update:model-value="noop"
  >
    <div class="space-y-4">
      <p class="text-xs leading-relaxed text-text-primary">
        {{ t('legal.consentGate.desc') }}
      </p>
      <ul class="space-y-2">
        <li
          v-for="term in auth.pendingConsents"
          :key="term.id"
          class="flex items-center justify-between gap-3 rounded-lg bg-surface-2 px-3 py-2"
        >
          <div class="min-w-0">
            <div class="truncate text-sm font-semibold text-white">{{ resolveTermTitle(term.code, term.titleKo, te, t) }}</div>
            <div class="text-caption text-text-secondary">{{ term.code }} · {{ term.version }}</div>
          </div>
          <div class="flex shrink-0 gap-2">
            <RouterLink :to="`/legal/${term.code}`" target="_blank">
              <UiButton variant="secondary" size="xs" type="button">{{ t('legal.consentGate.viewDoc') }}</UiButton>
            </RouterLink>
            <UiButton
              variant="brand"
              size="xs"
              type="button"
              :loading="busyId === term.id"
              @click="consentOne(term.id)"
            >
              {{ t('legal.consentGate.consent') }}
            </UiButton>
          </div>
        </li>
      </ul>
      <UiErrorBanner :message="error" />
      <UiButton variant="secondary" size="md" type="button" full-width @click="auth.logout()">
        {{ t('common.actions.logout') }}
      </UiButton>
    </div>
  </UiModal>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useAuthStore } from '@/stores/authStore';
import { useToast } from '@/composables/useToast';
import { resolveTermTitle } from '@/utils/legalTerms';
import { errMessageOf } from '@/api/client';
import UiButton from '@/components/ui/UiButton.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiModal from '@/components/ui/UiModal.vue';

const auth = useAuthStore();
const toast = useToast();
const { t, te } = useI18n();

const busyId = ref<string | null>(null);
const error = ref<string | null>(null);

// The gate cannot be dismissed except by consenting or signing out.
function noop(): void {}

async function consentOne(termsId: string): Promise<void> {
  busyId.value = termsId;
  error.value = null;
  try {
    await auth.consentTerms(termsId);
    if (auth.pendingConsents.length === 0) {
      toast.success(t('legal.consentGate.done'));
    }
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    busyId.value = null;
  }
}
</script>
