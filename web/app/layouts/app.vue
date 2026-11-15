<template>
  <div class="min-h-screen overflow-x-clip bg-surface-0 text-text-primary flex flex-col font-sans selection:bg-brand-500 selection:text-on-brand">
    <LocaleBanner />
    <UiNavbar />

    <!-- Bottom pad = mini player 64 + mobile tabbar 56 + safe-area (H-09). -->
    <main class="mx-auto w-full max-w-7xl flex-1 px-4 py-6 pb-40 sm:px-6 sm:py-8 sm:pb-36 lg:px-8">
      <slot />
    </main>

    <UiFooter />

    <GlobalPlayerBar @toggle-queue="queueOpen = !queueOpen" @expand="fullscreenOpen = true" />

    <MobileTabBar />

    <QueueDrawer v-if="queueOpen" @close="queueOpen = false" />

    <FullscreenPlayer v-if="fullscreenOpen" @close="fullscreenOpen = false" />

    <UiToastContainer />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import LocaleBanner from '@/components/LocaleBanner.vue';
import UiNavbar from '@/components/ui/UiNavbar.vue';
import UiFooter from '@/components/ui/UiFooter.vue';
import UiToastContainer from '@/components/ui/UiToastContainer.vue';
import GlobalPlayerBar from '@/components/player/GlobalPlayerBar.vue';
import MobileTabBar from '@/components/ui/MobileTabBar.vue';
import QueueDrawer from '@/components/player/QueueDrawer.vue';
import FullscreenPlayer from '@/components/player/FullscreenPlayer.vue';

const queueOpen = ref(false);
const fullscreenOpen = ref(false);
</script>
