<template>
  <header
    class="sticky top-0 z-40 border-b border-surface-border/80 bg-surface-0/80 backdrop-blur-xl transition-transform duration-200 lg:translate-y-0"
    :class="hideOnScroll ? '-translate-y-full' : ''"
  >
    <div class="mx-auto flex h-14 max-w-7xl items-center justify-between gap-3 px-4 sm:px-6 lg:h-16 lg:px-8">
      <div class="flex min-w-0 items-center gap-4 lg:gap-6">
        <RouterLink to="/" class="flex items-center transition-transform active:scale-95" :aria-label="`AVIBE ${t('common.nav.home')}`">
          <BrandLogo :size="36" />
        </RouterLink>

        <!-- Desktop Navigation -->
        <nav class="hidden lg:flex items-center gap-1 whitespace-nowrap text-sm font-medium text-text-secondary" :aria-label="t('common.nav.mainMenu')">
          <RouterLink
            to="/"
            class="rounded-lg px-3 py-1.5 transition-colors hover:text-white"
            :class="routeName === 'home' ? 'text-white bg-surface-2 font-semibold' : ''"
          >
            {{ t('common.nav.home') }}
          </RouterLink>
          <RouterLink
            to="/explore"
            class="rounded-lg px-3 py-1.5 transition-colors hover:text-white"
            :class="routeName === 'explore' ? 'text-white bg-surface-2 font-semibold' : ''"
          >
            {{ t('common.nav.explore') }}
          </RouterLink>
          <RouterLink
            to="/playlists"
            class="rounded-lg px-3 py-1.5 transition-colors hover:text-white"
            :class="routeName === 'playlists' || routeName === 'playlist-detail' ? 'text-white bg-surface-2 font-semibold' : ''"
          >
            {{ t('common.nav.playlists') }}
          </RouterLink>
          <RouterLink
            to="/library"
            class="rounded-lg px-3 py-1.5 transition-colors hover:text-white"
            :class="routeName === 'library' ? 'text-white bg-surface-2 font-semibold' : ''"
          >
            {{ t('common.nav.library') }}
          </RouterLink>
          <RouterLink
            to="/charts"
            class="rounded-lg px-3 py-1.5 transition-colors hover:text-white"
            :class="routeName === 'charts' ? 'text-white bg-surface-2 font-semibold' : ''"
          >
            {{ t('common.nav.charts') }}
          </RouterLink>
        </nav>
      </div>

      <!-- Desktop search -->
      <form class="hidden min-w-0 flex-1 max-w-xs lg:block" role="search" @submit.prevent="submitSearch">
        <div class="relative">
          <svg class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-text-muted" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
            <circle cx="11" cy="11" r="7" />
            <path d="m20 20-3.5-3.5" />
          </svg>
          <input
            v-model="searchQuery"
            type="search"
            :placeholder="t('common.nav.searchPh')"
            :aria-label="t('common.nav.explore')"
            class="h-9 w-full rounded-lg border border-surface-border bg-surface-1 pl-9 pr-3 text-xs text-text-primary placeholder:text-text-muted focus:border-brand-500 focus:outline-none"
          >
        </div>
      </form>

      <div class="flex shrink-0 items-center gap-2 sm:gap-3">
        <!-- Mobile search shortcut -->
        <RouterLink
          to="/explore"
          class="lg:hidden flex h-9 w-9 items-center justify-center rounded-lg border border-surface-border bg-surface-1 text-text-primary hover:text-white"
          :aria-label="t('common.nav.explore')"
        >
          <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
            <circle cx="11" cy="11" r="7" />
            <path d="m20 20-3.5-3.5" />
          </svg>
        </RouterLink>

        <RouterLink
          to="/business"
          class="hidden lg:inline-flex items-center whitespace-nowrap rounded-lg border border-brand-500/30 bg-brand-950/40 px-3 py-1.5 text-xs font-semibold text-brand-300 hover:bg-brand-950/70 hover:border-brand-500/50 transition-all"
        >
          {{ t('common.nav.business') }}
        </RouterLink>

        <LocaleSwitcher />

        <!-- User profile or Login (auth state is client-only; SSR shows login links) -->
        <ClientOnly>
          <div v-if="auth.isAuthenticated" class="relative">
            <button
              ref="userMenuButton"
              type="button"
              class="flex items-center gap-2 rounded-full border border-surface-border bg-surface-1 p-1 pr-3 text-xs text-text-primary hover:border-surface-border-hover transition-colors"
              aria-haspopup="menu"
              :aria-expanded="userMenuOpen"
              :aria-label="t('common.nav.userMenu')"
              @click="userMenuOpen = !userMenuOpen"
            >
              <UiAvatar :name="auth.displayLabel || undefined" size="sm" />
              <span class="max-w-[100px] truncate font-medium hidden sm:inline">{{ auth.displayLabel }}</span>
              <svg class="h-3 w-3 text-text-secondary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                <path d="m6 9 6 6 6-6" />
              </svg>
            </button>

            <div
              v-if="userMenuOpen"
              ref="userMenuPanel"
              role="menu"
              :aria-label="t('common.nav.userMenu')"
              class="absolute right-0 mt-2 w-48 rounded-xl border border-surface-border glass-dropdown py-1.5 shadow-2xl z-50 text-xs"
            >
              <div class="px-3 py-2 border-b border-surface-border text-text-secondary">
                <div class="font-semibold text-white truncate">{{ auth.displayLabel }}</div>
                <div class="text-caption text-text-muted truncate">{{ auth.email }}</div>
              </div>
              <RouterLink
                to="/account"
                class="block px-3 py-2 text-text-primary hover:bg-surface-3 hover:text-white"
                @click="userMenuOpen = false"
              >
                {{ t('common.nav.accountSettings') }}
              </RouterLink>
              <RouterLink
                v-if="auth.isPartner"
                to="/studio"
                class="block px-3 py-2 text-brand-300 hover:bg-surface-3 font-semibold"
                @click="userMenuOpen = false"
              >
                {{ t('common.nav.creatorStudio') }}
              </RouterLink>
              <RouterLink
                v-if="auth.isBusiness"
                to="/business/console"
                class="block px-3 py-2 text-brand-300 hover:bg-surface-3 font-semibold"
                @click="userMenuOpen = false"
              >
                {{ t('common.nav.businessConsole') }}
              </RouterLink>
              <RouterLink
                v-if="auth.isAdmin"
                to="/admin"
                class="block px-3 py-2 text-brand-300 hover:bg-surface-3 font-semibold"
                @click="userMenuOpen = false"
              >
                {{ t('common.nav.adminConsole') }}
              </RouterLink>
              <button
                class="w-full text-left px-3 py-2 text-accent-crimson hover:bg-surface-3 font-medium border-t border-surface-border"
                @click="auth.logout(); userMenuOpen = false"
              >
                {{ t('common.nav.logout') }}
              </button>
            </div>
          </div>

          <div v-else class="flex items-center gap-2">
            <RouterLink
              to="/login"
              class="rounded-lg bg-surface-2 px-3 py-1.5 text-xs font-semibold text-text-primary hover:bg-surface-3 transition-colors border border-surface-border"
            >
              {{ t('common.nav.login') }}
            </RouterLink>
            <RouterLink
              to="/signup"
              class="hidden sm:inline-flex items-center rounded-lg bg-brand-500 px-3.5 py-1.5 text-xs font-bold text-on-brand shadow-glow-brand hover:bg-brand-400 transition-all"
            >
              {{ t('common.nav.signup') }}
            </RouterLink>
          </div>
          <template #fallback>
            <div class="flex items-center gap-2">
              <RouterLink
                to="/login"
                class="rounded-lg bg-surface-2 px-3 py-1.5 text-xs font-semibold text-text-primary hover:bg-surface-3 transition-colors border border-surface-border"
              >
                {{ t('common.nav.login') }}
              </RouterLink>
              <RouterLink
                to="/signup"
                class="hidden sm:inline-flex items-center rounded-lg bg-brand-500 px-3.5 py-1.5 text-xs font-bold text-on-brand shadow-glow-brand hover:bg-brand-400 transition-all"
              >
                {{ t('common.nav.signup') }}
              </RouterLink>
            </div>
          </template>
        </ClientOnly>
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref, computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useAuthStore } from '@/stores/authStore';
import UiAvatar from './UiAvatar.vue';
import LocaleSwitcher from './LocaleSwitcher.vue';
import BrandLogo from '@/components/BrandLogo.vue';
const { t } = useI18n();

const auth = useAuthStore();
const route = useRoute();
const router = useRouter();
const routeName = computed(() => String(route.name ?? '').replace(/___.+$/, ''));

const userMenuOpen = ref(false);
const userMenuButton = ref<HTMLElement | null>(null);
const userMenuPanel = ref<HTMLElement | null>(null);

const searchQuery = ref('');

async function submitSearch(): Promise<void> {
  const q = searchQuery.value.trim();
  await router.push(q ? { path: '/explore', query: { q } } : '/explore');
}

// Mobile top bar hides on scroll down (UI §2.2); desktop bar stays.
const hideOnScroll = ref(false);
let lastY = 0;

function onScroll(): void {
  const y = window.scrollY;
  hideOnScroll.value = window.innerWidth < 1024 && y > 120 && y > lastY;
  lastY = y;
}

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
  if (e.key !== 'Escape' || !userMenuOpen.value) return;
  userMenuOpen.value = false;
  userMenuButton.value?.focus();
}

onMounted(() => {
  lastY = window.scrollY;
  document.addEventListener('pointerdown', onPointerDown);
  window.addEventListener('keydown', onKeydown);
  window.addEventListener('scroll', onScroll, { passive: true });
});

onUnmounted(() => {
  document.removeEventListener('pointerdown', onPointerDown);
  window.removeEventListener('keydown', onKeydown);
  window.removeEventListener('scroll', onScroll);
});
</script>
