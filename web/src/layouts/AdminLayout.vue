<template>
  <div class="min-h-screen bg-surface-0 text-neutral-100 flex flex-col font-sans selection:bg-brand-500 selection:text-white">
    <!-- Admin Header -->
    <header class="sticky top-0 z-40 border-b border-surface-border bg-surface-1/90 backdrop-blur-xl">
      <div class="mx-auto flex max-w-7xl items-center justify-between px-4 py-3 sm:px-6">
        <div class="flex items-center gap-6">
          <RouterLink to="/admin" class="flex items-center" aria-label="관리자 콘솔 홈">
            <BrandLogo :size="32" tagline="Audit Console" />
          </RouterLink>

          <nav class="hidden md:flex items-center gap-1 text-xs font-semibold text-neutral-400" aria-label="관리자 메뉴">
            <RouterLink
              to="/admin"
              class="rounded-lg px-3 py-1.5 transition-colors hover:text-white"
              :class="route.name === 'admin-overview' ? 'bg-surface-2 text-white font-bold' : ''"
            >
              개요
            </RouterLink>
            <RouterLink
              to="/admin/reviews"
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
            type="button"
            class="hidden sm:inline text-neutral-400 hover:text-white transition-colors"
            @click="auth.logout()"
          >
            로그아웃
          </button>
          <button
            type="button"
            class="md:hidden flex h-9 w-9 items-center justify-center rounded-lg border border-surface-border bg-surface-1 text-neutral-300 hover:text-white"
            :aria-expanded="mobileNavOpen"
            aria-controls="admin-mobile-nav"
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
        id="admin-mobile-nav"
        aria-label="관리자 모바일 메뉴"
        class="md:hidden border-t border-surface-border bg-surface-1/95 px-4 py-4 space-y-1 text-sm"
      >
        <RouterLink
          to="/admin"
          class="block rounded-lg px-3 py-2 text-neutral-300 hover:bg-surface-2 hover:text-white"
          @click="mobileNavOpen = false"
        >
          개요
        </RouterLink>
        <RouterLink
          to="/admin/reviews"
          class="block rounded-lg px-3 py-2 text-neutral-300 hover:bg-surface-2 hover:text-white"
          @click="mobileNavOpen = false"
        >
          Gate 4 검수 워크벤치
        </RouterLink>
        <RouterLink
          to="/admin/policies"
          class="block rounded-lg px-3 py-2 text-neutral-300 hover:bg-surface-2 hover:text-white"
          @click="mobileNavOpen = false"
        >
          AI 정책 매트릭스
        </RouterLink>
        <RouterLink
          to="/admin/disputes"
          class="block rounded-lg px-3 py-2 text-neutral-300 hover:bg-surface-2 hover:text-white"
          @click="mobileNavOpen = false"
        >
          OSP 분쟁 &amp; 임시조치
        </RouterLink>
        <RouterLink
          to="/admin/curation"
          class="block rounded-lg px-3 py-2 text-neutral-300 hover:bg-surface-2 hover:text-white"
          @click="mobileNavOpen = false"
        >
          큐레이션 &amp; 정산배치
        </RouterLink>
        <RouterLink
          to="/"
          class="block rounded-lg px-3 py-2 text-neutral-500 hover:bg-surface-2 hover:text-neutral-300"
          @click="mobileNavOpen = false"
        >
          리스너 홈으로 ↗
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
import { ref } from 'vue';
import { useRoute } from 'vue-router';
import { useAuthStore } from '@/stores/authStore';
import UiFooter from '@/components/ui/UiFooter.vue';
import UiToastContainer from '@/components/ui/UiToastContainer.vue';
import BrandLogo from '@/components/BrandLogo.vue';

const auth = useAuthStore();
const route = useRoute();
const mobileNavOpen = ref(false);
</script>
