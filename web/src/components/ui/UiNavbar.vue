<template>
  <header class="sticky top-0 z-40 border-b border-surface-border/80 bg-surface-0/80 backdrop-blur-xl transition-all duration-200">
    <div class="mx-auto flex max-w-7xl items-center justify-between px-4 py-3 sm:px-6">
      <!-- Left: Logo & Core Nav -->
      <div class="flex items-center gap-6 sm:gap-8">
        <RouterLink to="/" class="flex items-center gap-2.5 transition-transform active:scale-95">
          <div class="flex h-9 w-9 items-center justify-center rounded-xl bg-gradient-to-tr from-brand-600 via-brand-500 to-amber-400 shadow-glow-brand text-white font-black text-base">
            AI
          </div>
          <div class="flex flex-col">
            <span class="font-extrabold tracking-tight text-white text-base leading-none">AIVIBE</span>
            <span class="text-[9px] font-semibold text-brand-400 tracking-wider uppercase mt-0.5">Commercial Gate</span>
          </div>
        </RouterLink>

        <!-- Desktop Navigation -->
        <nav class="hidden md:flex items-center gap-1 text-sm font-medium text-neutral-400">
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
        </nav>
      </div>

      <!-- Right: Role portals & Auth -->
      <div class="flex items-center gap-2.5 sm:gap-3">
        <!-- Partner Portal button (for partners or listeners who want to upload) -->
        <RouterLink
          v-if="auth.isPartner || auth.isAdmin"
          to="/partner"
          class="hidden sm:inline-flex items-center gap-1.5 rounded-lg border border-brand-500/30 bg-brand-950/40 px-3 py-1.5 text-xs font-semibold text-brand-300 hover:bg-brand-950/70 hover:border-brand-500/50 transition-all"
        >
          <span>스튜디오</span>
        </RouterLink>

        <!-- Admin Portal button -->
        <RouterLink
          v-if="auth.isAdmin"
          to="/admin"
          class="hidden sm:inline-flex items-center gap-1.5 rounded-lg border border-accent-purple/30 bg-purple-950/40 px-3 py-1.5 text-xs font-semibold text-accent-purple hover:bg-purple-950/70 transition-all"
        >
          <span>어드민 콘솔</span>
        </RouterLink>

        <!-- User profile or Login -->
        <div v-if="auth.isAuthenticated" class="relative">
          <button
            type="button"
            class="flex items-center gap-2 rounded-full border border-surface-border bg-surface-1 p-1 pr-3 text-xs text-neutral-200 hover:border-surface-border-hover transition-colors"
            @click="userMenuOpen = !userMenuOpen"
          >
            <UiAvatar :name="auth.userId ?? undefined" size="sm" />
            <span class="max-w-[100px] truncate font-medium hidden sm:inline">{{ auth.role }}</span>
            <span class="text-[10px] text-neutral-400">▾</span>
          </button>

          <!-- Dropdown menu -->
          <div
            v-if="userMenuOpen"
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
            class="hidden sm:inline-flex items-center rounded-lg bg-brand-500 px-3.5 py-1.5 text-xs font-bold text-white shadow-glow-brand hover:bg-brand-600 transition-all"
          >
            회원가입
          </RouterLink>
        </div>

        <!-- Mobile hamburger -->
        <button
          type="button"
          class="md:hidden flex h-9 w-9 items-center justify-center rounded-lg border border-surface-border bg-surface-1 text-neutral-300 hover:text-white"
          @click="mobileNavOpen = !mobileNavOpen"
        >
          ☰
        </button>
      </div>
    </div>

    <!-- Mobile Drawer -->
    <div
      v-if="mobileNavOpen"
      class="md:hidden border-t border-surface-border bg-surface-1/95 backdrop-blur-2xl px-4 py-4 space-y-2 text-sm"
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
      <div v-if="auth.isPartner || auth.isAdmin" class="pt-2 border-t border-surface-border">
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
      <div v-if="auth.isAdmin" class="pt-2 border-t border-surface-border">
        <RouterLink
          to="/admin"
          class="block rounded-lg px-3 py-2 font-semibold text-accent-purple hover:bg-surface-2"
          @click="mobileNavOpen = false"
        >
          관리자 검수 워크벤치
        </RouterLink>
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { useRoute } from 'vue-router';
import { useAuthStore } from '@/stores/authStore';
import UiAvatar from './UiAvatar.vue';

const auth = useAuthStore();
const route = useRoute();

const userMenuOpen = ref(false);
const mobileNavOpen = ref(false);
</script>
