<template>
  <div class="min-h-screen bg-surface-0 text-neutral-100 flex flex-col font-sans selection:bg-brand-500 selection:text-white">
    <!-- Admin Header -->
    <header class="sticky top-0 z-40 border-b border-surface-border bg-surface-1/90 backdrop-blur-xl">
      <div class="mx-auto flex max-w-7xl items-center justify-between px-4 py-3 sm:px-6">
        <div class="flex items-center gap-6">
          <RouterLink to="/admin" class="flex items-center gap-2">
            <div class="flex h-8 w-8 items-center justify-center rounded-xl bg-accent-purple text-white font-black text-sm shadow-md">
              A
            </div>
            <div class="flex flex-col">
              <span class="font-bold text-white text-sm">AIVIBE Admin</span>
              <span class="text-[9px] font-semibold text-accent-purple uppercase">Audit Console</span>
            </div>
          </RouterLink>

          <nav class="hidden md:flex items-center gap-1 text-xs font-semibold text-neutral-400">
            <RouterLink
              to="/admin"
              class="rounded-lg px-3 py-1.5 transition-colors hover:text-white"
              :class="route.name === 'admin-reviews' ? 'bg-surface-2 text-white font-bold' : ''"
            >
              Gate 4 검수 워크벤치
            </RouterLink>
            <RouterLink
              to="/admin/policies"
              class="rounded-lg px-3 py-1.5 transition-colors hover:text-white"
              :class="route.name === 'admin-policies' ? 'bg-surface-2 text-white font-bold' : ''"
            >
              AI 정책 매트릭스
            </RouterLink>
            <RouterLink
              to="/admin/disputes"
              class="rounded-lg px-3 py-1.5 transition-colors hover:text-white"
              :class="route.name === 'admin-disputes' ? 'bg-surface-2 text-white font-bold' : ''"
            >
              OSP 분쟁 &amp; 임시조치
            </RouterLink>
            <RouterLink
              to="/admin/curation"
              class="rounded-lg px-3 py-1.5 transition-colors hover:text-white"
              :class="route.name === 'admin-curation' ? 'bg-surface-2 text-white font-bold' : ''"
            >
              큐레이션 &amp; 정산배치
            </RouterLink>
            <RouterLink
              to="/"
              class="rounded-lg px-3 py-1.5 text-neutral-500 hover:text-neutral-300 ml-2"
            >
              리스너 홈으로 ↗
            </RouterLink>
          </nav>
        </div>

        <div class="flex items-center gap-3 text-xs">
          <span class="rounded-full bg-purple-950 border border-purple-500/30 px-2.5 py-1 text-purple-300 font-semibold hidden sm:inline">
            최고 관리자 (Admin)
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

    <!-- Main Content -->
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
