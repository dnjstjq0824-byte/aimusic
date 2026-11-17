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
        <UiPageHeader :title="t('account.connTitle')" :description="t('account.connSub')" />

        <UiCard>
          <div class="flex items-center justify-between gap-3">
            <div class="min-w-0">
              <h3 class="text-sm font-bold text-white">{{ t('account.connEmail') }}</h3>
              <p class="mt-1 truncate text-xs text-text-secondary">{{ auth.email }}</p>
              <p class="mt-0.5 text-caption text-text-muted">{{ t('account.connEmailDesc') }}</p>
            </div>
            <UiBadge variant="success" size="xs">{{ t('account.connCurrent') }}</UiBadge>
          </div>
        </UiCard>

        <UiCard class="space-y-3">
          <h3 class="text-sm font-bold text-white">{{ t('account.connSocialT') }}</h3>
          <div v-if="loading" class="space-y-2">
            <UiSkeleton height="h-12" width="w-full" />
            <UiSkeleton height="h-12" width="w-full" />
          </div>
          <div v-else class="space-y-2">
            <div
              v-for="row in rows"
              :key="row.code"
              class="flex items-center justify-between gap-3 rounded-lg border border-surface-border px-3 py-2"
            >
              <div class="min-w-0">
                <p class="text-sm font-bold text-white">{{ row.short }}</p>
                <p class="truncate text-caption text-text-secondary">{{ row.email || t('account.connConnected') }}</p>
              </div>
              <UiButton
                v-if="!row.linked"
                variant="secondary"
                size="sm"
                :loading="busy === row.code"
                @click="connect(row)"
              >
                {{ t('account.connConnect') }}
              </UiButton>
              <UiButton
                v-else-if="confirming === row.code"
                variant="danger"
                size="sm"
                :loading="busy === row.code"
                @click="unlink(row.code)"
              >
                {{ t('account.connConfirm') }}
              </UiButton>
              <UiButton v-else variant="ghost" size="sm" @click="confirming = row.code">
                {{ t('account.connDisconnect') }}
              </UiButton>
            </div>
          </div>
          <UiErrorBanner v-if="pageError" :message="pageError" />
          <p class="text-caption text-text-muted">{{ t('account.connLastNote') }}</p>
        </UiCard>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { computed, onMounted, ref } from 'vue';
import { useRoute } from 'vue-router';
import { API_BASE_URL } from '@/api/client';
import { authApi } from '@/api/endpoints';
import type { OAuthConnection, OAuthProviderInfo } from '@/api/types';
import { orderSocialProviders } from '@/utils/social';
import { useAuthStore } from '@/stores/authStore';
import { useApiError } from '@/composables/useApiError';
import { useToast } from '@/composables/useToast';
import AccountNav from '@/components/account/AccountNav.vue';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';

const { t, locale } = useI18n();
const localePath = useLocalePath();
const route = useRoute();
const auth = useAuthStore();
const apiError = useApiError();
const toast = useToast();

const providers = ref<OAuthProviderInfo[]>([]);
const connections = ref<OAuthConnection[]>([]);
const loading = ref(true);
const busy = ref<string | null>(null);
const confirming = ref<string | null>(null);
const pageError = ref<string | null>(null);

interface ConnRow {
  code: string;
  slug: string;
  short: string;
  linked: boolean;
  email: string;
}

const rows = computed<ConnRow[]>(() => {
  const linked = new Map(connections.value.map((c) => [c.provider, c]));
  return orderSocialProviders(
    providers.value.filter((p) => p.enabled).map((p) => p.provider),
    locale.value,
  ).map((m) => ({
    code: m.code,
    slug: m.slug,
    short: m.short,
    linked: linked.has(m.code),
    email: linked.get(m.code)?.email ?? '',
  }));
});

async function connect(row: ConnRow): Promise<void> {
  busy.value = row.code;
  pageError.value = null;
  try {
    const { linkToken } = await authApi.oauthLinkToken();
    window.location.href =
      `${API_BASE_URL}/auth/oauth/${row.slug}/start?intent=link&linkToken=${encodeURIComponent(linkToken)}`;
  } catch (err) {
    pageError.value = apiError(err);
    busy.value = null;
  }
}

async function unlink(code: string): Promise<void> {
  busy.value = code;
  pageError.value = null;
  try {
    await authApi.oauthUnlink(code);
    connections.value = connections.value.filter((c) => c.provider !== code);
    confirming.value = null;
    const short = rows.value.find((r) => r.code === code)?.short ?? code;
    toast.success(t('account.connUnlinked', { provider: short }));
  } catch (err) {
    pageError.value = apiError(err);
  } finally {
    busy.value = null;
  }
}

onMounted(async () => {
  try {
    const [p, c] = await Promise.all([
      authApi.oauthProviders(locale.value),
      authApi.oauthConnections(),
    ]);
    providers.value = p ?? [];
    connections.value = c ?? [];
  } catch (err) {
    pageError.value = apiError(err);
  } finally {
    loading.value = false;
  }
  const linked = route.query.linked;
  if (typeof linked === 'string' && linked !== '') {
    const meta = orderSocialProviders([linked.toUpperCase()], locale.value)[0];
    toast.success(t('account.connLinked', { provider: meta?.short ?? linked }));
  }
});
</script>
