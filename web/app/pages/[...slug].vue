<template>
  <div class="mx-auto flex min-h-[60vh] max-w-lg flex-col items-center justify-center text-center px-4">
    <BrandLogo :size="40" />
    <div class="relative my-6" aria-hidden="true">
      <div class="text-8xl font-black text-surface-2 tracking-tighter select-none">404</div>
      <div class="absolute inset-0 flex items-center justify-center text-2xl font-black text-brand-500">
        NOT FOUND
      </div>
    </div>
    <h1 class="text-xl font-bold text-white mb-2">페이지를 찾을 수 없습니다</h1>
    <p class="text-xs text-text-secondary leading-relaxed mb-2 max-w-sm">
      요청하신 페이지가 삭제되었거나 주소가 변경되었습니다.
    </p>
    <p class="text-caption font-mono text-text-muted mb-6 max-w-sm break-all">{{ attemptedPath }}</p>
    <div class="flex items-center gap-3">
      <RouterLink to="/">
        <UiButton variant="brand" size="md">홈으로 돌아가기</UiButton>
      </RouterLink>
      <RouterLink to="/explore">
        <UiButton variant="secondary" size="md">음원 탐색하기</UiButton>
      </RouterLink>
    </div>
  </div>
</template>

<script setup lang="ts">
defineI18nRoute(false);
import { computed } from 'vue';
import { useRoute } from 'vue-router';
import UiButton from '@/components/ui/UiButton.vue';
import BrandLogo from '@/components/BrandLogo.vue';

const route = useRoute();
const attemptedPath = computed(() => route.fullPath);
</script>
