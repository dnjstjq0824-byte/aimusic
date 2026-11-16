<template>
  <div class="mx-auto max-w-4xl py-6">
    <UiPageHeader
      eyebrow="Legal"
      :title="currentTerm?.titleKo || t('legal.title')"
      :description="currentTerm
        ? t('legal.meta', { date: fmtDate(currentTerm.effectiveAt), version: currentTerm.version })
        : t('legal.sub')"
    >
      <template #actions>
        <UiButton
          v-if="auth.isAuthenticated && !consented"
          variant="brand"
          size="sm"
          :loading="consenting"
          @click="consent"
        >
          {{ t('legal.consent') }}
        </UiButton>
        <UiBadge v-else-if="consented" variant="success" size="sm">{{ t('legal.consented') }}</UiBadge>
        <RouterLink v-else to="/login">
          <UiButton variant="secondary" size="sm">{{ t('legal.loginToConsent') }}</UiButton>
        </RouterLink>
      </template>
    </UiPageHeader>

    <!-- Non-Korean readers: the Korean original is binding until translated. -->
    <div
      v-if="locale !== 'ko'"
      class="mt-6 rounded-xl border border-accent-amber/40 bg-amber-950/30 p-4 text-xs text-amber-100"
      role="status"
    >
      <span class="font-bold">{{ t('legal.transPendingT') }}</span>
      <span class="ml-2">{{ t('legal.transPendingD') }}</span>
    </div>

    <!-- Navigation Pills -->
    <div class="mt-6 mb-6 flex flex-wrap gap-2" role="tablist" :aria-label="t('legal.pickDoc')">
      <button
        v-for="code in termCodes"
        :key="code.id"
        type="button"
        role="tab"
        :aria-selected="activeCode === code.id"
        class="rounded-xl px-3 py-1.5 text-xs font-semibold transition-all duration-150"
        :class="activeCode === code.id ? 'bg-brand-500 text-on-brand shadow-glow-brand' : 'bg-surface-2 text-text-primary hover:bg-surface-3 hover:text-white border border-surface-border'"
        @click="selectCode(code.id)"
      >
        {{ code.name }}
      </button>
    </div>

    <!-- Loading & Error -->
    <div v-if="loading" class="space-y-4">
      <UiSkeleton height="h-8" width="w-2/3" />
      <UiSkeleton height="h-4" width="w-full" />
      <UiSkeleton height="h-4" width="w-5/6" />
      <UiSkeleton height="h-24" width="w-full" />
    </div>

    <UiErrorBanner v-else-if="error" :message="error" :title="t('legal.loadFail')" :retry-label="t('legal.retry')" @retry="loadTerms" />

    <UiCard v-else-if="notFound" class="text-sm text-text-primary">
      <p class="font-semibold text-white">{{ t('legal.notFoundT') }}</p>
      <p class="mt-1">{{ t('legal.notFoundD') }}</p>
    </UiCard>

    <!-- Markdown Content Viewer -->
    <UiCard v-else class="text-xs leading-relaxed text-text-primary">
      <div class="whitespace-pre-wrap space-y-3" v-html="renderedContent" />
    </UiCard>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { legalApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { renderLegalBody } from '@/utils/legalBody';
import { useAuthStore } from '@/stores/authStore';
import { useToast } from '@/composables/useToast';
import type { LegalTerm } from '@/api/types';
import UiButton from '@/components/ui/UiButton.vue';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';

const { t, locale } = useI18n();
const route = useRoute();
const router = useRouter();
const auth = useAuthStore();
const toast = useToast();

const termCodes = computed(() => [
  { id: 'TOS', name: t('legal.docTOS') },
  { id: 'PRIVACY', name: t('legal.docPRIVACY') },
  { id: 'COPYRIGHT_POLICY', name: t('legal.docCOPYRIGHT') },
  { id: 'PARTNER_CONTRACT', name: t('legal.docPARTNER') },
]);

const activeCode = ref<string>((route.params.code as string) || 'TOS');
const terms = ref<LegalTerm[]>([]);
const currentTerm = ref<LegalTerm | null>(null);
const loading = ref(true);
const error = ref<string | null>(null);
const consenting = ref(false);
const consented = ref(false);
const notFound = ref(false);

const renderedContent = computed(() =>
  renderLegalBody(currentTerm.value?.bodyMd, t('legal.emptyBody')),
);

function fmtDate(iso?: string): string {
  if (!iso) return '2026. 11. 01';
  const tag = locale.value === 'zh-cn' ? 'zh-CN' : locale.value;
  return new Date(iso).toLocaleDateString(tag);
}

async function loadTerms() {
  loading.value = true;
  error.value = null;
  try {
    const list = await legalApi.listTerms();
    terms.value = list;
    syncCurrentTerm();
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
}

function syncCurrentTerm() {
  const match = terms.value.find((x) => x.code === activeCode.value || x.id === activeCode.value);
  currentTerm.value = match ?? null;
  notFound.value = !match && terms.value.length > 0;
}

function selectCode(code: string) {
  activeCode.value = code;
  syncCurrentTerm();
  void router.replace(`/legal/${code}`);
}

async function consent() {
  if (!currentTerm.value) return;
  consenting.value = true;
  try {
    await legalApi.consent(currentTerm.value.id);
    consented.value = true;
    toast.success(t('legal.consentOk'));
  } catch (err: unknown) {
    toast.error(errMessageOf(err), t('legal.consentFail'));
  } finally {
    consenting.value = false;
  }
}

watch(() => route.params.code, (newCode) => {
  if (newCode && typeof newCode === 'string') {
    activeCode.value = newCode;
    syncCurrentTerm();
  }
});

onMounted(() => {
  void loadTerms();
});
</script>
