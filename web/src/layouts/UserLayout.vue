<template>
  <div class="min-h-screen bg-surface-0 text-neutral-100 flex flex-col font-sans selection:bg-brand-500 selection:text-white">
    <!-- Top Navigation -->
    <UiNavbar />

    <!-- Main Content Area with padding for bottom player -->
    <main class="mx-auto w-full max-w-7xl flex-1 px-4 py-8 pb-36 sm:px-6">
      <RouterView />
    </main>

    <!-- Global Footer -->
    <UiFooter />

    <!-- Studio Audio Player Bar -->
    <GlobalPlayerBar @toggle-queue="queueOpen = !queueOpen" />

    <!-- Queue Drawer -->
    <QueueDrawer v-if="queueOpen" @close="queueOpen = false" />

    <!-- Global Toast Container -->
    <UiToastContainer />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import UiNavbar from '@/components/ui/UiNavbar.vue';
import UiFooter from '@/components/ui/UiFooter.vue';
import UiToastContainer from '@/components/ui/UiToastContainer.vue';
import GlobalPlayerBar from '@/components/player/GlobalPlayerBar.vue';
import QueueDrawer from '@/components/player/QueueDrawer.vue';

const queueOpen = ref(false);
</script>
