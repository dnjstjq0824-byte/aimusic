<template>
  <div class="min-h-screen overflow-x-clip bg-surface-0 text-text-primary flex flex-col font-sans selection:bg-brand-500 selection:text-on-brand">
    <LocaleBanner />
    <EmailVerifyBanner />
    <UiNavbar />

    <!-- Bottom pad = mini player 64 + mobile tabbar 56 + safe-area (H-09). -->
    <main class="mx-auto w-full max-w-7xl flex-1 px-4 py-6 pb-40 sm:px-6 sm:py-8 sm:pb-36 lg:px-8">
      <slot />
    </main>

    <UiFooter />

    <!-- v3 flow: the bar appears once something is loaded (no idle bar on
         landing pages). Client-only: the queue restores from storage. -->
    <ClientOnly>
      <GlobalPlayerBar v-if="player.currentTrack" @toggle-queue="queueOpen = !queueOpen" @expand="fullscreenOpen = true" />
    </ClientOnly>

    <MobileTabBar />

    <QueueDrawer v-if="queueOpen" @close="queueOpen = false" />

    <FullscreenPlayer v-if="fullscreenOpen" @close="fullscreenOpen = false" />

    <UiToastContainer />
  </div>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, ref } from 'vue';
import LocaleBanner from '@/components/LocaleBanner.vue';
import EmailVerifyBanner from '@/components/EmailVerifyBanner.vue';
import UiNavbar from '@/components/ui/UiNavbar.vue';
import UiFooter from '@/components/ui/UiFooter.vue';
import UiToastContainer from '@/components/ui/UiToastContainer.vue';
import MobileTabBar from '@/components/ui/MobileTabBar.vue';
import { usePublicSeo } from '@/composables/usePublicSeo';
import { usePlayerStore } from '@/stores/playerStore';
import { stripLocalePrefix } from '@/utils/seo';

// Player UI loads on first use (bundle diet): all three are v-if gated.
const GlobalPlayerBar = defineAsyncComponent(() => import('@/components/player/GlobalPlayerBar.vue'));
const QueueDrawer = defineAsyncComponent(() => import('@/components/player/QueueDrawer.vue'));
const FullscreenPlayer = defineAsyncComponent(() => import('@/components/player/FullscreenPlayer.vue'));

// WI-S2 fallback head for public pages without page-level SEO (static
// landings etc.). Page-level usePublicSeo calls override title/description;
// canonical/hreflang are path-identical and dedupe.
const route = useRoute();
const { t, locale } = useI18n();
usePublicSeo({
  title: 'AVIBE — Beyond Your Music',
  description: computed(() => t('seo.siteDescription')),
  path: computed(() => stripLocalePrefix(route.path)),
  locale,
});

const player = usePlayerStore();
const queueOpen = ref(false);
const fullscreenOpen = ref(false);
</script>
