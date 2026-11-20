<template>
  <div class="mx-auto max-w-2xl space-y-6 py-8">
    <UiPageHeader
      eyebrow="Certificate Verification"
      :title="t('legal.aiBadge.certLabel')"
      :description="t('legal.cert.desc')"
    >
      <template #actions>
        <UiButton v-if="view" variant="secondary" size="sm" class="print:hidden" @click="printPage">
          {{ t('legal.cert.print') }}
        </UiButton>
      </template>
    </UiPageHeader>

    <UiErrorBanner :message="error" :title="t('legal.cert.loadFail')" />

    <div v-if="loading" class="space-y-2">
      <UiSkeleton height="h-10" width="w-full" />
      <UiSkeleton height="h-40" width="w-full" />
    </div>

    <UiCard v-else-if="view" class="space-y-4">
      <div class="flex flex-wrap items-center justify-between gap-2 border-b border-surface-border pb-3">
        <div class="font-mono text-lg font-semibold text-white">{{ view.certificate.certNumber }}</div>
        <UiBadge variant="success" size="sm">{{ t('legal.cert.verified') }}</UiBadge>
      </div>
      <dl class="grid gap-3 text-xs sm:grid-cols-2">
        <div>
          <dt class="font-bold uppercase tracking-wider text-text-muted text-caption">{{ t('legal.cert.fTitle') }}</dt>
          <dd class="mt-0.5 font-semibold text-white">{{ view.title }}</dd>
        </div>
        <div>
          <dt class="font-bold uppercase tracking-wider text-text-muted text-caption">{{ t('legal.cert.fProvider') }}</dt>
          <dd class="mt-0.5 text-text-primary">
            {{ view.aiProvider }}<span v-if="view.aiModel"> ({{ view.aiModel }})</span>
          </dd>
        </div>
        <div>
          <dt class="font-bold uppercase tracking-wider text-text-muted text-caption">{{ t('legal.cert.fTier') }}</dt>
          <dd class="mt-0.5 text-text-primary">{{ view.planTier ?? t('legal.cert.na') }}</dd>
        </div>
        <div>
          <dt class="font-bold uppercase tracking-wider text-text-muted text-caption">{{ t('legal.cert.fGenerated') }}</dt>
          <dd class="mt-0.5 text-text-primary">{{ view.generatedAt ?? t('legal.cert.na') }}</dd>
        </div>
        <div>
          <dt class="font-bold uppercase tracking-wider text-text-muted text-caption">{{ t('legal.cert.fGates') }}</dt>
          <dd class="mt-0.5 font-semibold" :class="view.gatesPassed ? 'text-verified' : 'text-accent-amber'">
            {{ view.gatesPassed ? t('legal.cert.gatesPass') : t('legal.cert.gatesPartial') }}
          </dd>
        </div>
        <div>
          <dt class="font-bold uppercase tracking-wider text-text-muted text-caption">{{ t('legal.cert.fIssued') }}</dt>
          <dd class="mt-0.5 text-text-primary">{{ fmt.dateISO(view.certificate.issuedAt) }}</dd>
        </div>
        <div v-if="view.isrc">
          <dt class="font-bold uppercase tracking-wider text-text-muted text-caption">ISRC</dt>
          <dd class="mt-0.5 font-mono text-text-primary">{{ view.isrc }}</dd>
        </div>
      </dl>
      <p class="border-t border-surface-border pt-3 text-caption leading-relaxed text-text-muted">
        {{ t('legal.cert.notice') }}
      </p>
    </UiCard>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute } from 'vue-router';
import { trackApi } from '@/api/endpoints';
import { errMessageOf } from '@/api/client';
import { useFormat } from '@/composables/useFormat';
import type { CertificateView } from '@/api/types';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiButton from '@/components/ui/UiButton.vue';
import UiCard from '@/components/ui/UiCard.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiSkeleton from '@/components/ui/UiSkeleton.vue';

const route = useRoute();
const { t } = useI18n();
const fmt = useFormat();
const view = ref<CertificateView | null>(null);
const loading = ref(true);
const error = ref<string | null>(null);

function printPage(): void {
  window.print();
}

onMounted(async () => {
  const number = route.params.number as string;
  if (!number) {
    loading.value = false;
    error.value = t('legal.cert.errNoNumber');
    return;
  }
  try {
    view.value = await trackApi.verifyCertificate(number);
    if (!view.value) {
      error.value = t('legal.cert.errNotFound');
    }
  } catch (err) {
    error.value = errMessageOf(err);
  } finally {
    loading.value = false;
  }
});
</script>
