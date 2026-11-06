<template>
  <div class="min-h-screen bg-neutral-950 text-neutral-100">
    <header class="border-b border-neutral-800 px-4 py-3 flex items-center gap-4">
      <RouterLink to="/partner" class="font-bold text-brand-500">Partner Console</RouterLink>
      <nav class="flex gap-3 text-sm">
        <RouterLink to="/partner">Dashboard</RouterLink>
        <RouterLink to="/partner/upload">Upload</RouterLink>
        <RouterLink to="/partner/subscriptions">Subscriptions</RouterLink>
        <RouterLink to="/">Listener Home</RouterLink>
      </nav>
      <div class="ml-auto text-sm">
        <button class="underline" @click="auth.logout()">Logout</button>
      </div>
    </header>
    <main class="mx-auto max-w-5xl px-4 py-6">
      <RouterView />
    </main>
  </div>
</template>

<script setup lang="ts">
import { useAuthStore } from '@/stores/authStore';

const auth = useAuthStore();
</script>
