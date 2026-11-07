<template>
  <div class="mx-auto flex min-h-[60vh] max-w-lg flex-col items-center justify-center text-center px-4">
    <BrandLogo :size="40" />
    <div class="relative my-6" aria-hidden="true">
      <div class="text-8xl font-black text-surface-2 tracking-tighter select-none">500</div>
      <div class="absolute inset-0 flex items-center justify-center text-2xl font-black text-accent-crimson">
        ERROR
      </div>
    </div>
    <h1 class="text-xl font-bold text-white mb-2">일시적인 오류가 발생했습니다</h1>
    <p class="text-xs text-neutral-400 leading-relaxed mb-6 max-w-sm">
      서버와의 통신 중 문제가 발생했습니다. 잠시 후 다시 시도해 주시기 바랍니다.
      문제가 계속되면 고객센터로 문의해 주세요.
    </p>
    <div class="flex items-center gap-3">
      <UiButton variant="brand" size="md" @click="reload">새로고침</UiButton>
      <RouterLink to="/">
        <UiButton variant="secondary" size="md">홈으로</UiButton>
      </RouterLink>
    </div>
  </div>
</template>

<script setup lang="ts">
import UiButton from '@/components/ui/UiButton.vue';
import BrandLogo from '@/components/BrandLogo.vue';

function reload() {
  window.location.reload();
}
</script>
