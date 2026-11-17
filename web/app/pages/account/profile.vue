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
      <div class="min-w-0 flex-1">
        <UiPageHeader :title="t('account.pfTitle')" :description="t('account.pfSub')" />
        <UiCard class="mt-6">
          <form class="space-y-4" @submit.prevent="submitSave">
            <UiErrorBanner :message="formError" />
            <UiInput
              v-model="nickname"
              :label="t('account.pfName')"
              :placeholder="t('account.pfNamePh')"
              autocomplete="nickname"
              required
            />
            <div>
              <div class="text-xs font-semibold text-text-secondary">{{ t('account.pfEmail') }}</div>
              <div class="mt-1 text-sm font-medium text-text-primary">{{ auth.email }}</div>
              <p class="mt-1 text-caption text-text-muted">{{ t('account.pfEmailNote') }}</p>
            </div>
            <div>
              <div class="text-xs font-semibold text-text-secondary">{{ t('account.pfRole') }}</div>
              <div class="mt-1 text-sm font-medium text-text-primary">{{ roleLabel }}</div>
            </div>
            <div>
              <UiButton variant="brand" size="md" type="submit" :loading="saving">
                {{ t('account.pfSave') }}
              </UiButton>
            </div>
          </form>
        </UiCard>
        <p class="mt-4 text-xs text-text-secondary">
          <RouterLink :to="localePath('/account/privacy')" class="underline hover:text-white">
            {{ t('account.pfDeleteLink') }}
          </RouterLink>
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { computed, ref, watch } from 'vue';
import { authApi } from '@/api/endpoints';
import { useAuthStore } from '@/stores/authStore';
import { useToast } from '@/composables/useToast';
import { useApiError } from '@/composables/useApiError';
import AccountNav from '@/components/account/AccountNav.vue';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiInput from '@/components/ui/UiInput.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';

const NICKNAME_MAX = 30;

const { t } = useI18n();
const localePath = useLocalePath();
const auth = useAuthStore();
const toast = useToast();
const apiError = useApiError();

const nickname = ref(auth.displayName ?? '');
watch(() => auth.displayName, (v) => {
  if (nickname.value === '') nickname.value = v ?? '';
});
const saving = ref(false);
const formError = ref<string | null>(null);

const roleLabel = computed(() => {
  if (auth.role === 'ADMIN') return t('account.roleAdmin');
  if (auth.role === 'PARTNER') return t('account.rolePartner');
  return t('account.roleUser');
});

async function submitSave(): Promise<void> {
  const name = nickname.value.trim();
  if (name === '') {
    formError.value = t('account.pfNameRequired');
    return;
  }
  if ([...name].length > NICKNAME_MAX) {
    formError.value = t('account.pfNameTooLong', { max: NICKNAME_MAX });
    return;
  }
  saving.value = true;
  formError.value = null;
  try {
    const res = await authApi.updateProfile(name);
    auth.displayName = res.displayName;
    toast.success(t('account.pfSaved'));
  } catch (err: unknown) {
    formError.value = apiError(err);
  } finally {
    saving.value = false;
  }
}
</script>
