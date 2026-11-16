<template>
  <div v-if="visible" role="status">
    <UiCard :border-variant="borderVariant" padding="sm">
      <div class="flex items-center gap-2">
        <UiBadge :variant="badgeVariant" size="sm" :dot="status === 'checking'">{{ title }}</UiBadge>
      </div>
      <p class="mt-2 text-xs leading-relaxed text-text-primary">{{ message }}</p>
      <p v-if="code" class="mt-1 font-mono text-caption text-text-muted">{{ code }}</p>
    </UiCard>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import UiCard from '@/components/ui/UiCard.vue';
import UiBadge from '@/components/ui/UiBadge.vue';

const props = defineProps<{ status: 'idle' | 'checking' | 'pass' | 'fail'; code?: string | null }>();
const { t, te } = useI18n();

const visible = computed(() => props.status === 'checking' || props.status === 'pass' || props.status === 'fail');
const title = computed(() =>
  props.status === 'checking'
    ? t('legal.precheck.checking')
    : props.status === 'pass'
      ? t('legal.precheck.pass')
      : t('legal.precheck.fail'),
);
const message = computed(() => {
  if (props.status === 'checking') return t('legal.precheck.checkingDesc');
  if (props.status === 'pass') return t('legal.precheck.passDesc');
  if (props.code) {
    const key = `errors.${props.code}`;
    if (te(key)) return t(key);
    return props.code;
  }
  return t('legal.precheck.failDesc');
});
const badgeVariant = computed(() =>
  props.status === 'pass' ? 'success' : props.status === 'fail' ? 'danger' : 'info',
);
const borderVariant = computed(() =>
  props.status === 'pass' ? 'success' : props.status === 'fail' ? 'danger' : 'default',
);
</script>
