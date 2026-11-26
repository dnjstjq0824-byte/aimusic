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
        <UiPageHeader :title="t('account.privTitle')" :description="t('account.privExportD')" />

        <UiCard class="space-y-2">
          <h3 class="text-sm font-bold text-white">{{ t('account.privExportT') }}</h3>
          <p class="text-xs leading-relaxed text-text-secondary">{{ t('account.privExportD') }}</p>
          <div>
            <UiButton variant="secondary" size="sm" :loading="exporting" @click="submitExport">
              {{ t('account.privExportBtn') }}
            </UiButton>
          </div>
        </UiCard>

        <UiCard class="space-y-2">
          <h3 class="text-sm font-bold text-white">{{ t('account.privConsentsT') }}</h3>
          <p v-if="consents.length === 0" class="text-xs text-text-muted">
            {{ t('account.privConsentsEmpty') }}
          </p>
          <ul v-else class="divide-y divide-surface-border">
            <li
              v-for="c in consents"
              :key="c.id"
              class="flex items-center justify-between gap-3 py-2.5 text-xs"
            >
              <div class="min-w-0">
                <div class="truncate font-semibold text-text-primary">{{ c.title }}</div>
                <div class="text-caption text-text-muted">
                  {{ t('account.privAgreed') }}: {{ fmtDate(c.consentedAt) }}
                </div>
              </div>
              <UiBadge v-if="c.required" variant="neutral" size="xs">
                {{ t('account.privRequired') }}
              </UiBadge>
            </li>
          </ul>
        </UiCard>

        <UiCard class="space-y-2 border-accent-crimson/30">
          <h3 class="text-sm font-bold text-white">{{ t('account.withdrawTitle') }}</h3>
          <p class="text-xs leading-relaxed text-text-secondary">{{ t('account.withdrawDesc') }}</p>
          <div>
            <UiButton variant="danger" size="sm" @click="withdrawOpen = true">
              {{ t('account.withdrawConfirm') }}
            </UiButton>
          </div>
        </UiCard>

        <UiConfirmDialog
          v-model="withdrawOpen"
          :title="t('account.withdrawTitle')"
          :body="t('account.withdrawBody')"
          :confirm-label="t('account.withdrawConfirm')"
          danger
          :loading="withdrawLoading"
          @confirm="confirmWithdraw"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { authApi, legalApi } from '@/api/endpoints';
import { useAuthStore } from '@/stores/authStore';
import { useToast } from '@/composables/useToast';
import { useApiError } from '@/composables/useApiError';
import { resolveTermTitle } from '@/utils/legalTerms';
import AccountNav from '@/components/account/AccountNav.vue';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiConfirmDialog from '@/components/ui/UiConfirmDialog.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';

interface ConsentRow {
  id: string;
  title: string;
  consentedAt: string;
  required: boolean;
}

const { t, te } = useI18n();
const localePath = useLocalePath();
const auth = useAuthStore();
const router = useRouter();
const toast = useToast();
const apiError = useApiError();

const exporting = ref(false);
const consents = ref<ConsentRow[]>([]);
const withdrawOpen = ref(false);
const withdrawLoading = ref(false);

function fmtDate(v?: string | null): string {
  if (!v) return '';
  const d = new Date(v);
  return Number.isNaN(d.getTime()) ? '' : d.toLocaleDateString();
}

async function loadConsents(): Promise<void> {
  if (!auth.isAuthenticated) return;
  try {
    const [mine, terms] = await Promise.all([legalApi.myConsents(), legalApi.listTerms()]);
    consents.value = (mine ?? []).map((c) => {
      const term = (terms ?? []).find((x) => x.id === c.termsId);
      return {
        id: c.id,
        title: term ? resolveTermTitle(term.code, term.titleKo, te, t) : t('account.privConsentUnknown'),
        consentedAt: c.consentedAt,
        required: term?.isRequired ?? false,
      };
    });
  } catch (err: unknown) {
    toast.error(apiError(err));
  }
}

async function submitExport(): Promise<void> {
  exporting.value = true;
  try {
    const bundle = await authApi.exportData();
    const blob = new Blob([JSON.stringify(bundle, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `avibe-my-data-${new Date().toISOString().slice(0, 10)}.json`;
    document.body.appendChild(a);
    a.click();
    a.remove();
    URL.revokeObjectURL(url);
    toast.success(t('account.privExportDone'));
  } catch (err: unknown) {
    toast.error(apiError(err), t('account.privExportFail'));
  } finally {
    exporting.value = false;
  }
}

async function confirmWithdraw(): Promise<void> {
  withdrawLoading.value = true;
  try {
    await authApi.withdraw();
    auth.logout();
    withdrawOpen.value = false;
    toast.success(t('account.privWithdrawOk'));
    await router.push(localePath('/'));
  } catch (err: unknown) {
    toast.error(apiError(err));
  } finally {
    withdrawLoading.value = false;
  }
}

onMounted(() => {
  void loadConsents();
});
</script>
