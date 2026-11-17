<template>
  <span class="inline-flex flex-wrap items-center gap-1">
    <UiBadge v-if="track.aiGenerated" variant="brand" size="xs" :title="t('legal.aiBadge.aiTitle')">AI</UiBadge>
    <UiBadge variant="neutral" size="xs" :title="track.aiDisclosureText ?? ''">
      {{ disclosureLabel }}
    </UiBadge>
    <UiBadge
      v-if="isCertified"
      variant="commercial"
      size="xs"
      :title="t('legal.aiBadge.certTitle')"
    >
      {{ t('legal.aiBadge.certLabel') }}
    </UiBadge>
    <UiBadge v-else :variant="statusVariant" size="xs">
      {{ statusLabel }}
    </UiBadge>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import UiBadge from '@/components/ui/UiBadge.vue';
import type { Track } from '@/api/types';

const props = defineProps<{ track: Track }>();
const { t, te } = useI18n();

const disclosureLabel = computed(() => {
  const genType = props.track.aiGenerationType ?? '';
  const key = `studio.genType.${genType}`;
  if (genType !== '' && te(key)) return t(key);
  return props.track.aiProvider;
});

const statusLabel = computed(() => {
  const key = `studio.status.${props.track.status}`;
  return te(key) ? t(key) : t('studio.status.PROCESSING');
});
const statusVariant = computed(() => {
  const st = props.track.status;
  if (st === 'READY') return 'success';
  if (st === 'TAKEN_DOWN' || st === 'REJECTED' || st === 'LICENSE_GATE_REJECTED') return 'danger';
  return 'warning';
});

// Demo/launch tracks carry "(Launch Demo)" in the title and must never show
// the certification badge. The track API exposes no per-track cert field, so
// the title marker is the only frontend-distinguishable signal (a backend
// cert/demo field is the structural fix, engine scope).
const isDemo = computed(() => props.track.title.includes('(Launch Demo)'));
const isCertified = computed(() => props.track.status === 'READY' && !isDemo.value);
</script>
