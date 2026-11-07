<template>
  <div class="mx-auto max-w-md py-16 text-center">
    <BrandLogo :size="48" />
    <div class="mt-6 space-y-3">
      <div v-if="failed" class="space-y-4">
        <h1 class="text-xl font-black text-white">소셜 로그인에 실패했습니다</h1>
        <UiErrorBanner :message="error" />
        <RouterLink to="/login">
          <UiButton variant="brand" size="md">로그인으로 돌아가기</UiButton>
        </RouterLink>
      </div>
      <div v-else class="space-y-3">
        <div class="flex justify-center" aria-hidden="true">
          <svg class="h-8 w-8 animate-spin text-brand-500" viewBox="0 0 24 24" fill="none">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z" />
          </svg>
        </div>
        <h1 class="text-xl font-black text-white">소셜 로그인 처리 중…</h1>
        <p class="text-xs text-neutral-400">잠시만 기다려 주세요. 창을 닫지 마세요.</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useAuthStore } from '@/stores/authStore';
import { useToast } from '@/composables/useToast';
import UiButton from '@/components/ui/UiButton.vue';
import UiErrorBanner from '@/components/ui/UiErrorBanner.vue';
import BrandLogo from '@/components/BrandLogo.vue';

const route = useRoute();
const router = useRouter();
const auth = useAuthStore();
const toast = useToast();

const failed = ref(false);
const error = ref<string | null>(null);

onMounted(async () => {
  const access = route.query.accessToken;
  const refresh = route.query.refreshToken;
  if (typeof access !== 'string' || typeof refresh !== 'string' || access === '' || refresh === '') {
    failed.value = true;
    error.value = '소셜 로그인 응답이 올바르지 않습니다. 다시 시도해 주세요.';
    return;
  }
  try {
    await auth.loginWithTokens(access, refresh);
    toast.success('소셜 로그인되었습니다.');
    // Strip tokens from history so they never linger in the URL bar.
    await router.replace('/');
  } catch (err: unknown) {
    failed.value = true;
    error.value = err instanceof Error ? err.message : '소셜 로그인 세션을 확인할 수 없습니다.';
  }
});
</script>
