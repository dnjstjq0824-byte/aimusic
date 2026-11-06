<template>
  <div class="min-h-screen bg-neutral-950 text-neutral-100">
    <header class="border-b border-neutral-800 px-4 py-3 flex items-center gap-3 sm:gap-4">
      <RouterLink to="/" class="shrink-0 text-neutral-100"><BrandLogo /></RouterLink>
      <nav class="flex gap-3 overflow-x-auto whitespace-nowrap text-sm">
        <RouterLink to="/">Home</RouterLink>
        <RouterLink to="/disputes">Dispute Center</RouterLink>
        <RouterLink v-if="auth.isPartner" to="/partner">Partner</RouterLink>
        <RouterLink v-if="auth.isAdmin" to="/admin">Admin</RouterLink>
      </nav>
      <div class="ml-auto flex shrink-0 items-center gap-3 text-sm">
        <span v-if="auth.userId" class="hidden sm:inline">{{ auth.role }}</span>
        <button v-if="auth.isAuthenticated" class="underline" @click="auth.logout()">Logout</button>
        <RouterLink v-else to="/login">Login</RouterLink>
      </div>
    </header>
    <main class="mx-auto max-w-5xl px-4 py-6 pb-32">
      <RouterView />
    </main>
    <GlobalPlayerBar />
    <QueueDrawer v-if="queueOpen" @close="queueOpen = false" />
    <button
      class="fixed bottom-24 right-4 rounded-full bg-brand-600 px-4 py-2 text-sm shadow"
      @click="queueOpen = !queueOpen"
    >
      Queue
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { useAuthStore } from '@/stores/authStore';
import BrandLogo from '@/components/BrandLogo.vue';
import GlobalPlayerBar from '@/components/player/GlobalPlayerBar.vue';
import QueueDrawer from '@/components/player/QueueDrawer.vue';

const auth = useAuthStore();
const queueOpen = ref(false);
</script>
