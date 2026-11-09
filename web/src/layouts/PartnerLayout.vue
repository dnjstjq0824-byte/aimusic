<template>
  <div class="min-h-screen overflow-x-clip bg-surface-0 text-neutral-100 flex flex-col font-sans selection:bg-brand-500 selection:text-[#04121a]">
    <!-- Partner Header -->
    <header class="sticky top-0 z-40 border-b border-surface-border bg-surface-1/90 backdrop-blur-xl">
      <div class="mx-auto flex max-w-7xl items-center justify-between gap-3 px-4 py-3 sm:px-6 lg:px-8">
        <div class="flex min-w-0 items-center gap-6">
          <RouterLink to="/partner" class="flex items-center" aria-label="파트너 스튜디오 홈">
            <BrandLogo :size="32" tagline="Studio Console" />
          </RouterLink>

          <nav class="hidden lg:flex items-center gap-1 whitespace-nowrap text-xs font-semibold text-neutral-400" aria-label="파트너 메뉴">
            <RouterLink
              to="/partner"
              class="rounded-lg px-3 py-1.5 transition-colors hover:text-white"
              :class="route.name === 'partner-dashboard' ? 'bg-surface-2 text-white font-bold' : ''"
            >
              대시보드
            </RouterLink>
            <RouterLink
              to="/partner/upload"
              class="rounded-lg px-3 py-1.5 transition-colors hover:text-white"
              :class="route.name === 'partner-upload' ? 'bg-surface-2 text-white font-bold' : ''"
            >
              음원 업로드 (Gatekeeper)
            </RouterLink>
            <RouterLink
              to="/partner/subscriptions"
              class="rounded-lg px-3 py-1.5 transition-colors hover:text-white"
              :class="route.name === 'partner-subscriptions' ? 'bg-surface-2 text-white font-bold' : ''"
            >
              구독 증빙 관리
            </RouterLink>
            <RouterLink
              to="/"
              class="rounded-lg px-3 py-1.5 text-neutral-500 hover:text-neutral-300 ml-2"
            >
              리스너 홈으로 이동 ↗
            </RouterLink>
          </nav>
        </div>

        <div class="flex shrink-0 items-center gap-3 text-xs">
          <span class="rounded-full bg-brand-950 border border-brand-500/30 px-2.5 py-1 text-brand-300 font-semibold hidden sm:inline">
            파트너 프로필
          </span>
          <button
            type="button"
            class="hidden sm:inline text-neutral-400 hover:text-white transition-colors"
            @click="auth.logout()"
          >
            로그아웃
          </button>
          <button
            type="button"
            class="lg:hidden flex h-9 w-9 items-center justify-center rounded-lg border border-surface-border bg-surface-1 text-neutral-300 hover:text-white"
            :aria-expanded="mobileNavOpen"
            aria-controls="partner-mobile-nav"
            :aria-label="mobileNavOpen ? '메뉴 닫기' : '메뉴 열기'"
            @click="mobileNavOpen = !mobileNavOpen"
          >
            <svg v-if="!mobileNavOpen" class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
              <path d="M4 7h16M4 12h16M4 17h16" />
            </svg>
            <svg v-else class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
              <path d="M18 6 6 18M6 6l12 12" />
            </svg>
          </button>
        </div>
      </div>

      <!-- Mobile Drawer -->
      <nav
        v-if="mobileNavOpen"
        id="partner-mobile-nav"
        aria-label="파트너 모바일 메뉴"
        class="lg:hidden border-t border-surface-border bg-surface-1/95 px-4 py-4 space-y-1 text-sm sm:grid sm:grid-cols-2 sm:gap-1 sm:space-y-0"
      >
        <RouterLink
          to="/partner"
          class="block rounded-lg px-3 py-2 text-neutral-300 hover:bg-surface-2 hover:text-white"
          @click="mobileNavOpen = false"
        >
          대시보드
        </RouterLink>
        <RouterLink
          to="/partner/upload"
          class="block rounded-lg px-3 py-2 text-neutral-300 hover:bg-surface-2 hover:text-white"
          @click="mobileNavOpen = false"
        >
          음원 업로드 (Gatekeeper)
        </RouterLink>
        <RouterLink
          to="/partner/subscriptions"
          class="block rounded-lg px-3 py-2 text-neutral-300 hover:bg-surface-2 hover:text-white"
          @click="mobileNavOpen = false"
        >
          구독 증빙 관리
        </RouterLink>
        <RouterLink
          to="/"
          class="block rounded-lg px-3 py-2 text-neutral-500 hover:bg-surface-2 hover:text-neutral-300"
          @click="mobileNavOpen = false"
        >
          리스너 홈으로 이동 ↗
        </RouterLink>
        <button
          type="button"
          class="block w-full rounded-lg px-3 py-2 text-left text-neutral-400 hover:bg-surface-2 hover:text-white"
          @click="auth.logout()"
        >
          로그아웃
        </button>
      </nav>
    </header>

    <!-- Content -->
    <main class="mx-auto w-full max-w-7xl flex-1 px-4 py-6 pb-20 sm:px-6 sm:py-8 lg:px-8">
      <RouterView />
    </main>

    <!-- Footer -->
    <UiFooter />

    <!-- Global Toast Container -->
    <UiToastContainer />
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';
import { useRoute } from 'vue-router';
import { useAuthStore } from '@/stores/authStore';
import UiFooter from '@/components/ui/UiFooter.vue';
import UiToastContainer from '@/components/ui/UiToastContainer.vue';
import BrandLogo from '@/components/BrandLogo.vue';

const auth = useAuthStore();
const route = useRoute();
const mobileNavOpen = ref(false);
watch(() => route.fullPath, () => (mobileNavOpen.value = false));
</script>
