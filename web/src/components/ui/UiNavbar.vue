<template>
  <header class="sticky top-0 z-40 border-b border-surface-border/80 bg-surface-0/80 backdrop-blur-xl transition-all duration-200">
    <div class="mx-auto flex max-w-7xl items-center justify-between gap-3 px-4 py-3 sm:px-6 lg:px-8">
      <!-- Left: Logo & Core Nav -->
      <div class="flex min-w-0 items-center gap-4 lg:gap-8">
        <RouterLink to="/" class="flex items-center transition-transform active:scale-95" aria-label="AVIBE 홈">
          <BrandLogo :size="36" />
        </RouterLink>

        <!-- Desktop Navigation -->
        <nav class="hidden lg:flex items-center gap-1 whitespace-nowrap text-sm font-medium text-neutral-400" aria-label="주 메뉴">
          <RouterLink
            to="/"
            class="rounded-lg px-3 py-1.5 transition-colors hover:text-white"
            :class="route.name === 'home' ? 'text-white bg-surface-2 font-semibold' : ''"
          >
            홈
          </RouterLink>
          <RouterLink
            to="/explore"
            class="rounded-lg px-3 py-1.5 transition-colors hover:text-white"
            :class="route.name === 'explore' ? 'text-white bg-surface-2 font-semibold' : ''"
          >
            음원 탐색
          </RouterLink>
          <RouterLink
            to="/playlists"
            class="rounded-lg px-3 py-1.5 transition-colors hover:text-white"
            :class="route.name === 'playlists' || route.name === 'playlist-detail' ? 'text-white bg-surface-2 font-semibold' : ''"
          >
            플레이리스트
          </RouterLink>
          <RouterLink
            v-if="auth.isAuthenticated"
            to="/my-library"
            class="rounded-lg px-3 py-1.5 transition-colors hover:text-white"
            :class="route.name === 'my-library' ? 'text-white bg-surface-2 font-semibold' : ''"
          >
            내 라이브러리
          </RouterLink>
          <RouterLink
            to="/disputes"
            class="rounded-lg px-3 py-1.5 transition-colors hover:text-white"
            :class="route.name === 'dispute-center' ? 'text-white bg-surface-2 font-semibold' : ''"
          >
            권리보호센터
          </RouterLink>
          <RouterLink
            to="/pricing"
            class="rounded-lg px-3 py-1.5 transition-colors hover:text-white"
            :class="route.name === 'pricing' ? 'text-white bg-surface-2 font-semibold' : ''"
          >
            요금제
          </RouterLink>
        </nav>
      </div>

      <!-- Right: Role portals & Auth -->
      <div class="flex shrink-0 items-center gap-2 sm:gap-3">
        <!-- Partner Portal button (for partners or listeners who want to upload) -->
        <RouterLink
          v-if="auth.isPartner || auth.isAdmin"
          to="/partner"
          class="hidden lg:inline-flex items-center gap-1.5 whitespace-nowrap rounded-lg border border-brand-500/30 bg-brand-950/40 px-3 py-1.5 text-xs font-semibold text-brand-300 hover:bg-brand-950/70 hover:border-brand-500/50 transition-all"
        >
          <span>스튜디오</span>
        </RouterLink>

        <!-- Admin Portal button -->
        <RouterLink
          v-if="auth.isAdmin"
          to="/admin"
          class="hidden lg:inline-flex items-center gap-1.5 whitespace-nowrap rounded-lg border border-brand-500/30 bg-brand-950/40 px-3 py-1.5 text-xs font-semibold text-brand-300 hover:bg-brand-950/70 transition-all"
        >
          <span>어드민 콘솔</span>
        </RouterLink>

        <!-- User profile or Login -->
        <div v-if="auth.isAuthenticated" class="relative">
          <button
            ref="userMenuButton"
            type="button"
            class="flex items-center gap-2 rounded-full border border-surface-border bg-surface-1 p-1 pr-3 text-xs text-neutral-200 hover:border-surface-border-hover transition-colors"
            aria-haspopup="menu"
            :aria-expanded="userMenuOpen"
            aria-label="사용자 메뉴"
            @click="userMenuOpen = !userMenuOpen"
          >
            <UiAvatar :name="auth.userId ?? undefined" size="sm" />
            <span class="max-w-[100px] truncate font-medium hidden sm:inline">{{ auth.role }}</span>
            <svg class="h-3 w-3 text-neutral-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <path d="m6 9 6 6 6-6" />
            </svg>
          </button>

          <!-- Dropdown menu -->
          <div
            v-if="userMenuOpen"
            ref="userMenuPanel"
            role="menu"
            aria-label="사용자 메뉴"
            class="absolute right-0 mt-2 w-48 rounded-xl border border-surface-border glass-dropdown py-1.5 shadow-2xl z-50 text-xs"
          >
            <div class="px-3 py-2 border-b border-surface-border text-neutral-400">
              <div class="text-[10px] uppercase font-bold text-neutral-500">역할</div>
              <div class="font-semibold text-white">{{ auth.role }}</div>
            </div>
            <RouterLink
              to="/my-library"
              class="block px-3 py-2 text-neutral-300 hover:bg-surface-3 hover:text-white"
              @click="userMenuOpen = false"
            >
              내 플레이리스트 & 라이브러리
            </RouterLink>
            <RouterLink
              v-if="auth.isPartner || auth.isAdmin"
              to="/partner"
              class="block px-3 py-2 text-brand-300 hover:bg-surface-3 font-semibold lg:hidden"
              @click="userMenuOpen = false"
            >
              파트너 스튜디오
            </RouterLink>
            <RouterLink
              v-if="auth.isAdmin"
              to="/admin"
              class="block px-3 py-2 text-brand-300 hover:bg-surface-3 font-semibold lg:hidden"
              @click="userMenuOpen = false"
            >
              어드민 콘솔
            </RouterLink>
            <RouterLink
              v-if="auth.isPartner || auth.isAdmin"
              to="/partner/upload"
              class="block px-3 py-2 text-brand-400 hover:bg-surface-3 hover:text-brand-300 font-medium"
              @click="userMenuOpen = false"
            >
              음원 업로드 (Gatekeeper)
            </RouterLink>
            <RouterLink
              to="/legal"
              class="block px-3 py-2 text-neutral-300 hover:bg-surface-3 hover:text-white"
              @click="userMenuOpen = false"
            >
              이용약관 및 법적 정책
            </RouterLink>
            <RouterLink
              to="/account/security"
              class="block px-3 py-2 text-neutral-300 hover:bg-surface-3 hover:text-white"
              @click="userMenuOpen = false"
            >
              계정 보안 (2단계 인증)
            </RouterLink>
            <button
              class="w-full text-left px-3 py-2 text-accent-crimson hover:bg-surface-3 font-medium border-t border-surface-border"
              @click="auth.logout(); userMenuOpen = false"
            >
              로그아웃
            </button>
          </div>
        </div>

        <div v-else class="flex items-center gap-2">
          <RouterLink
            to="/login"
            class="rounded-lg bg-surface-2 px-3 py-1.5 text-xs font-semibold text-neutral-200 hover:bg-surface-3 transition-colors border border-surface-border"
          >
            로그인
          </RouterLink>
          <RouterLink
            to="/login?tab=register"
            class="hidden sm:inline-flex items-center rounded-lg bg-brand-500 px-3.5 py-1.5 text-xs font-bold text-[#04121a] shadow-glow-brand hover:bg-brand-400 transition-all"
          >
            회원가입
          </RouterLink>
        </div>

        <!-- Mobile hamburger -->
        <button
          type="button"
          class="lg:hidden flex h-9 w-9 items-center justify-center rounded-lg border border-surface-border bg-surface-1 text-neutral-300 hover:text-white"
          :aria-expanded="mobileNavOpen"
          aria-controls="mobile-nav"
          :aria-label="mobileNavOpen ? '모바일 메뉴 닫기' : '모바일 메뉴 열기'"
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
      id="mobile-nav"
      aria-label="모바일 메뉴"
      class="lg:hidden border-t border-surface-border bg-surface-1/95 backdrop-blur-2xl px-4 py-4 space-y-1 text-sm sm:grid sm:grid-cols-2 sm:gap-1 sm:space-y-0 sm:px-6"
    >
      <RouterLink
        to="/"
        class="block rounded-lg px-3 py-2 text-neutral-300 hover:bg-surface-2 hover:text-white"
        @click="mobileNavOpen = false"
      >
        홈
      </RouterLink>
      <RouterLink
        to="/explore"
        class="block rounded-lg px-3 py-2 text-neutral-300 hover:bg-surface-2 hover:text-white"
        @click="mobileNavOpen = false"
      >
        음원 탐색 &amp; 필터
      </RouterLink>
      <RouterLink
        to="/playlists"
        class="block rounded-lg px-3 py-2 text-neutral-300 hover:bg-surface-2 hover:text-white"
        @click="mobileNavOpen = false"
      >
        플레이리스트
      </RouterLink>
      <RouterLink
        v-if="auth.isAuthenticated"
        to="/my-library"
        class="block rounded-lg px-3 py-2 text-neutral-300 hover:bg-surface-2 hover:text-white"
        @click="mobileNavOpen = false"
      >
        내 라이브러리
      </RouterLink>
      <RouterLink
        to="/disputes"
        class="block rounded-lg px-3 py-2 text-neutral-300 hover:bg-surface-2 hover:text-white"
        @click="mobileNavOpen = false"
      >
        권리보호센터 (OSP 분쟁)
      </RouterLink>
      <RouterLink
        to="/pricing"
        class="block rounded-lg px-3 py-2 text-neutral-300 hover:bg-surface-2 hover:text-white"
        @click="mobileNavOpen = false"
      >
        요금제
      </RouterLink>
      <RouterLink
        v-if="auth.isAuthenticated"
        to="/account/security"
        class="block rounded-lg px-3 py-2 text-neutral-300 hover:bg-surface-2 hover:text-white"
        @click="mobileNavOpen = false"
      >
        계정 보안 (2단계 인증)
      </RouterLink>
      <div v-if="auth.isPartner || auth.isAdmin" class="pt-2 border-t border-surface-border sm:col-span-2 sm:grid sm:grid-cols-2 sm:gap-1">
        <RouterLink
          to="/partner"
          class="block rounded-lg px-3 py-2 font-semibold text-brand-400 hover:bg-surface-2"
          @click="mobileNavOpen = false"
        >
          파트너 스튜디오 대시보드
        </RouterLink>
        <RouterLink
          to="/partner/upload"
          class="block rounded-lg px-3 py-2 text-neutral-300 hover:bg-surface-2 hover:text-white"
          @click="mobileNavOpen = false"
        >
          음원 업로드 위저드
        </RouterLink>
      </div>
      <div v-if="auth.isAdmin" class="pt-2 border-t border-surface-border sm:col-span-2">
        <RouterLink
          to="/admin"
          class="block rounded-lg px-3 py-2 font-semibold text-brand-300 hover:bg-surface-2"
          @click="mobileNavOpen = false"
        >
          관리자 검수 워크벤치
        </RouterLink>
      </div>
    </nav>
  </header>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue';
import { useRoute } from 'vue-router';
import { useAuthStore } from '@/stores/authStore';
import UiAvatar from './UiAvatar.vue';
import BrandLogo from '@/components/BrandLogo.vue';

const auth = useAuthStore();
const route = useRoute();

const userMenuOpen = ref(false);
const mobileNavOpen = ref(false);
const userMenuButton = ref<HTMLElement | null>(null);
const userMenuPanel = ref<HTMLElement | null>(null);

// Collapse menus whenever navigation happens (back button, programmatic push).
watch(
  () => route.fullPath,
  () => {
    mobileNavOpen.value = false;
    userMenuOpen.value = false;
  },
);

function onPointerDown(e: PointerEvent): void {
  if (!userMenuOpen.value) return;
  const target = e.target as Node | null;
  if (
    target &&
    !userMenuButton.value?.contains(target) &&
    !userMenuPanel.value?.contains(target)
  ) {
    userMenuOpen.value = false;
  }
}

function onKeydown(e: KeyboardEvent): void {
  if (e.key !== 'Escape') return;
  if (userMenuOpen.value) {
    userMenuOpen.value = false;
    userMenuButton.value?.focus();
  }
  if (mobileNavOpen.value) {
    mobileNavOpen.value = false;
  }
}

onMounted(() => {
  document.addEventListener('pointerdown', onPointerDown);
  window.addEventListener('keydown', onKeydown);
});

onUnmounted(() => {
  document.removeEventListener('pointerdown', onPointerDown);
  window.removeEventListener('keydown', onKeydown);
});
</script>
