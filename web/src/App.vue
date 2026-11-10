<template>
  <RouterView />
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

const route = useRoute();
const router = useRouter();
const toast = useToast();

let lastExpiredAt = 0;

function onSessionExpired(): void {
  // Concurrent requests fail rotation together; notify only once per burst.
  const now = Date.now();
  if (now - lastExpiredAt < 3000) return;
  lastExpiredAt = now;
  toast.warning('세션이 만료되었습니다. 다시 로그인해 주세요.', '로그인 필요');
  if (route.meta.requiresAuth && route.name !== 'login') {
    void router.replace({ name: 'login', query: { redirect: route.fullPath } });
  }
}

onMounted(() => {
  window.addEventListener(SESSION_EXPIRED_EVENT, onSessionExpired);
});

onUnmounted(() => {
  window.removeEventListener(SESSION_EXPIRED_EVENT, onSessionExpired);
});
</script>
