<template>
  <NuxtLayout>
    <NuxtPage />
  </NuxtLayout>
  <AgeConfirmGate />
  <TermsConsentGate />
</template>

<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { SESSION_EXPIRED_EVENT } from '@/api/client';
import { useToast } from '@/composables/useToast';
import AgeConfirmGate from '@/components/AgeConfirmGate.vue';
import TermsConsentGate from '@/components/TermsConsentGate.vue';
const localePath = useLocalePath();

const route = useRoute();
const router = useRouter();
const toast = useToast();
const { t } = useI18n();

let lastExpiredAt = 0;

function onSessionExpired(): void {
  // Concurrent requests fail rotation together; notify only once per burst.
  const now = Date.now();
  if (now - lastExpiredAt < 3000) return;
  lastExpiredAt = now;
  toast.warning(t('common.auth.sessionExpired'), t('common.auth.sessionExpiredTitle'));
  if (route.meta.requiresAuth && route.name !== 'login') {
    void router.replace(localePath({ name: 'login', query: { redirect: route.fullPath } }));
  }
}

onMounted(() => {
  window.addEventListener(SESSION_EXPIRED_EVENT, onSessionExpired);
});

onUnmounted(() => {
  window.removeEventListener(SESSION_EXPIRED_EVENT, onSessionExpired);
});
</script>
