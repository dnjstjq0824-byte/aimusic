<template>
  <div
    v-if="visible"
    class="border-b border-surface-border bg-surface-1 px-4 py-2 text-center text-body-sm text-text-secondary"
  >
    {{ t('common.banner.suggest', { language: detectedName }) }}
    <button type="button" class="ml-2 font-bold text-brand-400 hover:underline" @click="go">
      {{ t('common.banner.view') }}
    </button>
    <button type="button" class="ml-2 text-text-muted hover:text-text-primary" @click="dismiss">
      {{ t('common.banner.dismiss') }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { detectBrowserLocale } from '@/utils/locale';

const { t, locale } = useI18n();
const switchLocalePath = useSwitchLocalePath();
const route = useRoute();
const router = useRouter();

// Browser-only signals (navigator, sessionStorage) are read after mount so
// the first client render matches the server HTML (no hydration mismatch).
const mounted = ref(false);
const dismissed = ref(false);
onMounted(() => {
  try {
    dismissed.value = sessionStorage.getItem('avibe.localeBannerOff') === '1';
  } catch {
    dismissed.value = false;
  }
  mounted.value = true;
});

const detected = computed(() =>
  detectBrowserLocale(typeof navigator === 'undefined' ? [] : [...navigator.languages]),
);
// Only on prefixed public pages, once per session, when detection differs.
const visible = computed(() => {
  if (!mounted.value || dismissed.value) return false;
  const prefix = `/${route.path.split('/')[1]}`;
  const prefixed = ['/en', '/ja', '/zh-cn', '/th'].includes(prefix);
  return prefixed && detected.value !== locale.value;
});

const detectedName = computed(() => {
  const key = `common.locale.${detected.value}`;
  return t(key);
});

function dismiss(): void {
  dismissed.value = true;
  try {
    sessionStorage.setItem('avibe.localeBannerOff', '1');
  } catch {
    // ignore
  }
}

function go(): void {
  dismiss();
  void router.push(switchLocalePath(detected.value));
}
</script>
