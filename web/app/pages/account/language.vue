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
        <UiPageHeader :title="t('account.langTitle')" :description="t('account.langSub')" />
        <UiCard class="mt-6">
          <ul class="divide-y divide-surface-border" role="listbox" :aria-label="t('account.langTitle')">
            <li v-for="lang in LANGS" :key="lang.code" role="option" :aria-selected="lang.code === locale">
              <button
                type="button"
                class="flex min-h-11 w-full items-center justify-between gap-3 py-2 text-left"
                @click="select(lang.code)"
              >
                <span class="flex items-center gap-2">
                  <span class="text-sm" :class="lang.code === locale ? 'font-bold text-white' : 'text-text-primary'">
                    {{ lang.name }}
                  </span>
                  <span v-if="lang.beta" class="rounded bg-surface-3 px-1.5 py-0.5 text-caption text-text-muted">
                    {{ t('common.locale.beta') }}
                  </span>
                </span>
                <span v-if="lang.code === locale" class="text-sm font-bold text-brand-400" aria-hidden="true">✓</span>
              </button>
            </li>
          </ul>
        </UiCard>
        <p class="mt-4 text-caption text-text-muted">{{ t('account.langNote') }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { useI18n } from 'vue-i18n';
import { writeStoredLocale } from '@/utils/locale';
import { authApi } from '@/api/endpoints';
import { useAuthStore } from '@/stores/authStore';
import { useApiError } from '@/composables/useApiError';
import { useToast } from '@/composables/useToast';
import AccountNav from '@/components/account/AccountNav.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';

// Same language set as LocaleSwitcher (v2 §4.5): self-names, Beta on ja/zh-cn/th.
const LANGS = [
  { code: 'ko', name: '한국어', beta: false },
  { code: 'en', name: 'English', beta: false },
  { code: 'ja', name: '日本語', beta: true },
  { code: 'zh-cn', name: '简体中文', beta: true },
  { code: 'th', name: 'ไทย', beta: true },
] as const;

type LocaleCode = (typeof LANGS)[number]['code'];

const { t, locale, setLocale } = useI18n();
const localePath = useLocalePath();
const auth = useAuthStore();
const toast = useToast();
const apiError = useApiError();

async function select(code: LocaleCode): Promise<void> {
  if (code === locale.value) return;
  writeStoredLocale(code);
  await setLocale(code);
  // A2: signed-in users persist to users.locale (detection order ①).
  if (auth.isAuthenticated) {
    try {
      await authApi.updateLocale(code);
    } catch (err: unknown) {
      toast.error(apiError(err));
    }
  }
}
</script>
