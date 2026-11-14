<template>
  <div class="min-h-screen overflow-x-clip bg-surface-0 text-text-primary flex flex-col font-sans selection:bg-brand-500 -on-brand">
    <!-- Admin Header -->
    <header class="sticky top-0 z-40 border-b border-surface-border bg-surface-1/90 backdrop-blur-xl">
      <div class="mx-auto flex max-w-7xl items-center justify-between gap-3 px-4 py-3 sm:px-6 lg:px-8">
        <div class="flex min-w-0 items-center gap-6">
          <RouterLink to="/admin" class="flex items-center" aria-label="관리자 콘솔 홈">
            <BrandLogo :size="32" tagline="Audit Console" />
          </RouterLink>

          <nav class="hidden lg:flex items-center gap-1 whitespace-nowrap text-xs font-semibold text-text-secondary" aria-label="관리자 메뉴">
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
              to="/admin/audit"
              class="rounded-lg px-3 py-1.5 transition-colors hover:text-white"
              :class="route.name === 'admin-audit' ? 'bg-surface-2 text-white font-bold' : ''"
            >
              감사 추적
            </RouterLink>
            <RouterLink
              to="/admin/users"
              class="rounded-lg px-3 py-1.5 transition-colors hover:text-white"
              :class="route.name === 'admin-users' ? 'bg-surface-2 text-white font-bold' : ''"
            >
              회원 관리
            </RouterLink>
            <RouterLink
              to="/admin/stores"
              class="rounded-lg px-3 py-1.5 transition-colors hover:text-white"
              :class="route.name === 'admin-stores' ? 'bg-surface-2 text-white font-bold' : ''"
            >
              매장 승인
            </RouterLink>
            <RouterLink
              to="/admin/distribution"
              class="rounded-lg px-3 py-1.5 transition-colors hover:text-white"
              :class="route.name === 'admin-distribution' ? 'bg-surface-2 text-white font-bold' : ''"
            >
              DSP 키트
            </RouterLink>
            <RouterLink
              to="/admin/settlements"
              class="rounded-lg px-3 py-1.5 transition-colors hover:text-white"
              :class="route.name === 'admin-settlements' ? 'bg-surface-2 text-white font-bold' : ''"
            >
              정산 관리
            </RouterLink>
            <RouterLink
              to="/"
              class="rounded-lg px-3 py-1.5 text-text-muted hover:text-text-primary ml-2"
            >
              리스너 홈으로 ↗
            </RouterLink>
          </nav>
        </div>

        <div class="flex shrink-0 items-center gap-3 text-xs">
          <span class="rounded-full bg-brand-950 border border-brand-500/30 px-2.5 py-1 text-brand-300 font-semibold hidden sm:inline">
            최고 관리자 (Admin)
          </span>
          <button
            type="button"
            class="hidden sm:inline text-text-secondary hover:text-white transition-colors"
            @click="auth.logout()"
          >
            로그아웃
          </button>
          <button
            type="button"
            class="lg:hidden flex h-9 w-9 items-center justify-center rounded-lg border border-surface-border bg-surface-1 text-text-primary hover:text-white"
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
        class="lg:hidden border-t border-surface-border bg-surface-1/95 px-4 py-4 space-y-1 text-sm sm:grid sm:grid-cols-2 sm:gap-1 sm:space-y-0"
      >
        <RouterLink
          to="/admin"
          class="block rounded-lg px-3 py-2 text-text-primary hover:bg-surface-2 hover:text-white"
          @click="mobileNavOpen = false"
        >
          개요
        </RouterLink>
        <RouterLink
          to="/admin/reviews"
          class="block rounded-lg px-3 py-2 text-text-primary hover:bg-surface-2 hover:text-white"
          @click="mobileNavOpen = false"
        >
          Gate 4 검수 워크벤치
        </RouterLink>
        <RouterLink
          to="/admin/policies"
          class="block rounded-lg px-3 py-2 text-text-primary hover:bg-surface-2 hover:text-white"
          @click="mobileNavOpen = false"
        >
          AI 정책 매트릭스
        </RouterLink>
        <RouterLink
          to="/admin/disputes"
          class="block rounded-lg px-3 py-2 text-text-primary hover:bg-surface-2 hover:text-white"
          @click="mobileNavOpen = false"
        >
          OSP 분쟁 &amp; 임시조치
        </RouterLink>
        <RouterLink
          to="/admin/curation"
          class="block rounded-lg px-3 py-2 text-text-primary hover:bg-surface-2 hover:text-white"
          @click="mobileNavOpen = false"
        >
          큐레이션 &amp; 정산배치
        </RouterLink>
        <RouterLink
          to="/admin/audit"
          class="block rounded-lg px-3 py-2 text-text-primary hover:bg-surface-2 hover:text-white"
          @click="mobileNavOpen = false"
        >
          감사 추적
        </RouterLink>
        <RouterLink
          to="/admin/users"
          class="block rounded-lg px-3 py-2 text-text-primary hover:bg-surface-2 hover:text-white"
          @click="mobileNavOpen = false"
        >
          회원 관리
        </RouterLink>
        <RouterLink
          to="/admin/stores"
          class="block rounded-lg px-3 py-2 text-text-primary hover:bg-surface-2 hover:text-white"
          @click="mobileNavOpen = false"
        >
          매장 승인
        </RouterLink>
        <RouterLink
          to="/admin/distribution"
          class="block rounded-lg px-3 py-2 text-text-primary hover:bg-surface-2 hover:text-white"
          @click="mobileNavOpen = false"
        >
          DSP 키트
        </RouterLink>
        <RouterLink
          to="/admin/settlements"
          class="block rounded-lg px-3 py-2 text-text-primary hover:bg-surface-2 hover:text-white"
          @click="mobileNavOpen = false"
        >
          정산 관리
        </RouterLink>
        <RouterLink
          to="/"
          class="block rounded-lg px-3 py-2 text-text-muted hover:bg-surface-2 hover:text-text-primary"
          @click="mobileNavOpen = false"
        >
          리스너 홈으로 ↗
        </RouterLink>
        <button
          type="button"
          class="block w-full rounded-lg px-3 py-2 text-left text-text-secondary hover:bg-surface-2 hover:text-white"
          @click="auth.logout()"
        >
          로그아웃
        </button>
      </nav>
    </header>

    <!-- Main Content -->
    <main class="mx-auto w-full max-w-7xl flex-1 px-4 py-6 pb-20 sm:px-6 sm:py-8 lg:px-8">
      <slot />
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
