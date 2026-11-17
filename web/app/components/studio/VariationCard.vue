<template>
  <UiCard class="space-y-2">
    <div class="flex items-center justify-between gap-2">
      <span class="text-xs font-bold text-text-secondary">{{ t('studio.create.v2.variationTitle', { n: index }) }}</span>
      <UiBadge :variant="badgeVariant" size="xs">{{ statusText }}</UiBadge>
    </div>
    <p class="text-caption leading-relaxed text-text-muted">
      {{ note || t('studio.create.v2.variationSoon') }}
    </p>
  </UiCard>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import UiBadge from '@/components/ui/UiBadge.vue';
import UiCard from '@/components/ui/UiCard.vue';

const props = withDefaults(
  defineProps<{ index: number; status: 'soon' | 'queued' | 'running' | 'done' | 'failed'; note?: string }>(),
  { note: '' },
);

const { t } = useI18n();

const badgeVariant = computed<'success' | 'warning' | 'danger' | 'info'>(() => {
  if (props.status === 'done') return 'success';
  if (props.status === 'failed') return 'danger';
  if (props.status === 'running') return 'info';
  return 'warning';
});

const statusText = computed(() => {
  if (props.status === 'soon') return String(t('studio.create.v2.variationSoonBadge'));
  if (props.status === 'queued') return String(t('studio.genStatus.QUEUED'));
  if (props.status === 'running') return String(t('studio.genStatus.RUNNING'));
  if (props.status === 'done') return String(t('studio.genStatus.SUCCEEDED'));
  return String(t('studio.genStatus.FAILED'));
});
</script>
