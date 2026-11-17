<!-- Share row (WI-S4): copy link + Kakao + LINE + X + Facebook. -->
<template>
  <div class="flex items-center gap-2">
    <span class="text-xs text-text-secondary">{{ t('seo.share.label') }}</span>
    <UiButton variant="secondary" size="sm" :aria-label="t('seo.share.copy')" @click="copy">
      {{ copied ? t('seo.share.copied') : t('seo.share.copy') }}
    </UiButton>
    <a
      v-for="ch in channels"
      :key="ch.key"
      :href="ch.href"
      target="_blank"
      rel="noopener"
      class="inline-flex h-9 items-center rounded-lg bg-surface-2 px-3 text-xs font-semibold text-text-primary transition-colors hover:bg-surface-3"
      @click="track(ch.key)"
    >
      {{ ch.label }}
    </a>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { shareLinks } from '@/utils/seo';
import { trackEvent } from '@/utils/analytics';
import UiButton from '@/components/ui/UiButton.vue';

const props = defineProps<{ url: string; title: string }>();
const { t } = useI18n();
const copied = ref(false);

const targets = computed(() => shareLinks(props.url, props.title));
const channels = computed(() => [
  { key: 'kakao', label: t('seo.share.kakao'), href: targets.value.kakao },
  { key: 'line', label: t('seo.share.line'), href: targets.value.line },
  { key: 'x', label: t('seo.share.x'), href: targets.value.x },
  { key: 'facebook', label: t('seo.share.facebook'), href: targets.value.facebook },
]);

function track(channel: string): void {
  trackEvent({ name: 'share_clicked', props: { channel } });
}

async function copy(): Promise<void> {
  try {
    await navigator.clipboard.writeText(targets.value.copy);
    copied.value = true;
    track('copy');
    setTimeout(() => {
      copied.value = false;
    }, 2000);
  } catch {
    // Clipboard unavailable (permissions); the href fallback stays usable.
  }
}
</script>
