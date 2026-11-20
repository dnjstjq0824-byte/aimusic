<template>
  <div class="min-h-screen overflow-x-clip bg-surface-0 text-text-primary flex flex-col font-sans selection:bg-brand-500 selection:text-on-brand">
    <!-- v3 flow (phase 3): consoles share the public header language — thin
         translucent bar, wordmark, plain text links, pill actions. -->
    <header class="sticky top-0 z-40 border-b border-white/[0.06] bg-surface-0/70 backdrop-blur-xl">
      <div class="mx-auto flex h-14 max-w-7xl items-center justify-between gap-3 px-4 sm:px-6 lg:h-16 lg:px-8">
        <div class="flex min-w-0 items-center gap-3">
          <button
            type="button"
            class="lg:hidden flex h-9 w-9 shrink-0 items-center justify-center rounded-full border border-white/10 bg-white/[0.05] text-text-primary hover:text-white"
            :aria-expanded="mobileNavOpen"
            :aria-controls="navId"
            :aria-label="mobileNavOpen ? closeLabel : openLabel"
            @click="mobileNavOpen = !mobileNavOpen"
          >
            <svg v-if="!mobileNavOpen" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
              <path d="M4 7h16M4 12h16M4 17h16" />
            </svg>
            <svg v-else class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
              <path d="M18 6 6 18M6 6l12 12" />
            </svg>
          </button>
          <RouterLink :to="homeTo" class="flex min-w-0 items-center gap-3 transition-transform active:scale-95" :aria-label="homeLabel">
            <BrandLogo :size="30" tagline="" />
            <span class="h-5 w-px shrink-0 bg-white/15" aria-hidden="true" />
            <span class="truncate text-sm text-text-secondary">{{ consoleName }}</span>
          </RouterLink>
        </div>

        <div class="flex shrink-0 items-center gap-1 sm:gap-2">
          <slot name="badge" />
          <RouterLink
            to="/"
            class="hidden rounded-full px-3 py-1.5 text-sm text-text-secondary transition-colors hover:text-white sm:inline-flex"
          >
            {{ listenerHomeLabel }}
          </RouterLink>
          <button
            type="button"
            class="hidden h-9 items-center rounded-full border border-white/10 bg-white/[0.05] px-4 text-sm text-text-primary transition-colors hover:border-white/25 hover:text-white sm:inline-flex"
            @click="auth.logout()"
          >
            {{ logoutLabel }}
          </button>
        </div>
      </div>

      <nav
        v-if="mobileNavOpen"
        :id="navId"
        :aria-label="menuLabel"
        class="lg:hidden max-h-[calc(100dvh-3.5rem)] overflow-y-auto border-t border-white/[0.06] bg-surface-0/95 px-4 py-3 text-sm sm:grid sm:grid-cols-2 sm:gap-x-2"
      >
        <RouterLink
          v-for="item in menu"
          :key="item.to"
          :to="item.to"
          class="block rounded-full px-4 py-2.5 transition-colors"
          :class="isActive(item) ? 'bg-white/[0.08] text-white' : 'text-text-secondary hover:bg-white/[0.04] hover:text-white'"
          :aria-current="isActive(item) ? 'page' : undefined"
          @click="mobileNavOpen = false"
        >
          {{ item.label }}
        </RouterLink>
        <div class="col-span-2 mt-2 flex items-center gap-2 border-t border-white/[0.06] pt-3 sm:hidden">
          <RouterLink to="/" class="rounded-full px-4 py-2 text-text-secondary hover:text-white" @click="mobileNavOpen = false">
            {{ listenerHomeLabel }}
          </RouterLink>
          <button type="button" class="ml-auto rounded-full border border-white/10 bg-white/[0.05] px-4 py-2 text-text-primary" @click="auth.logout()">
            {{ logoutLabel }}
          </button>
        </div>
      </nav>
    </header>

    <div class="mx-auto flex w-full max-w-7xl flex-1 gap-8 px-4 py-6 sm:px-6 sm:py-8 lg:px-8">
      <aside class="hidden w-52 shrink-0 lg:block">
        <nav :aria-label="menuLabel" class="sticky top-24 space-y-0.5 text-sm">
          <RouterLink
            v-for="item in menu"
            :key="item.to"
            :to="item.to"
            class="block rounded-full px-4 py-2 transition-colors"
            :class="isActive(item) ? 'bg-white/[0.08] font-medium text-white' : 'text-text-secondary hover:bg-white/[0.04] hover:text-white'"
            :aria-current="isActive(item) ? 'page' : undefined"
          >
            {{ item.label }}
          </RouterLink>
        </nav>
      </aside>

      <main class="min-w-0 flex-1 pb-20">
        <slot />
      </main>
    </div>

    <UiFooter />
    <UiToastContainer />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useRoute } from 'vue-router';
import { useAuthStore } from '@/stores/authStore';
import UiFooter from '@/components/ui/UiFooter.vue';
import UiToastContainer from '@/components/ui/UiToastContainer.vue';
import BrandLogo from '@/components/BrandLogo.vue';
import { stripLocalePrefix } from '@/utils/seo';

export interface ConsoleMenuItem {
  to: string;
  /** Route name without the i18n suffix (e.g. `studio-dashboard`). */
  name: string;
  label: string;
}

const props = defineProps<{
  /** Short id used for the mobile nav element (`<id>-mobile-nav`). */
  id: string;
  homeTo: string;
  homeLabel: string;
  consoleName: string;
  menu: ConsoleMenuItem[];
  menuLabel: string;
  openLabel: string;
  closeLabel: string;
  listenerHomeLabel: string;
  logoutLabel: string;
}>();

const auth = useAuthStore();
const route = useRoute();
const routeName = computed(() => String(route.name ?? '').replace(/___.+$/, ''));
const navId = computed(() => `${props.id}-mobile-nav`);
const mobileNavOpen = ref(false);
watch(() => route.fullPath, () => (mobileNavOpen.value = false));

// Name match, plus a path fallback: folder index pages (e.g. /admin) don't
// always carry the legacy name the menu lists.
const barePath = computed(() => stripLocalePrefix(route.path).replace(/\/$/, '') || '/');
function isActive(item: ConsoleMenuItem): boolean {
  return routeName.value === item.name || barePath.value === item.to;
}
</script>
