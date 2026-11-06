<template>
  <div class="min-h-screen bg-surface-0 text-neutral-100 flex flex-col font-sans selection:bg-brand-500 selection:text-white">
    <!-- Partner Header -->
    <header class="sticky top-0 z-40 border-b border-surface-border bg-surface-1/90 backdrop-blur-xl">
      <div class="mx-auto flex max-w-7xl items-center justify-between px-4 py-3 sm:px-6">
        <div class="flex items-center gap-6">
          <RouterLink to="/partner" class="flex items-center gap-2">
            <div class="flex h-8 w-8 items-center justify-center rounded-xl bg-brand-500 text-white font-black text-sm shadow-glow-brand">
              P
            </div>
            <div class="flex flex-col">
              <span class="font-bold text-white text-sm">AIVIBE Studio</span>
              <span class="text-[9px] font-semibold text-brand-400 uppercase">Partner Console</span>
            </div>
          </RouterLink>

          <nav class="hidden md:flex items-center gap-1 text-xs font-semibold text-neutral-400">
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

        <div class="flex items-center gap-3 text-xs">
          <span class="rounded-full bg-brand-950 border border-brand-500/30 px-2.5 py-1 text-brand-300 font-semibold hidden sm:inline">
            파트너 프로필
          </span>
          <button
            class="text-neutral-400 hover:text-white transition-colors"
            @click="auth.logout()"
          >
            로그아웃
          </button>
        </div>
      </div>
    </header>

    <!-- Content -->
    <main class="mx-auto w-full max-w-7xl flex-1 px-4 py-8 pb-20 sm:px-6">
      <RouterView />
    </main>

    <!-- Footer -->
    <UiFooter />

    <!-- Global Toast Container -->
    <UiToastContainer />
  </div>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router';
import { useAuthStore } from '@/stores/authStore';
import UiFooter from '@/components/ui/UiFooter.vue';
import UiToastContainer from '@/components/ui/UiToastContainer.vue';

const auth = useAuthStore();
const route = useRoute();
</script>
