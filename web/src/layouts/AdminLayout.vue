<template>
  <div class="min-h-screen bg-neutral-950 text-neutral-100">
    <header class="border-b border-neutral-800 px-4 py-3 flex items-center gap-4">
      <RouterLink to="/admin" class="font-bold text-brand-500">Admin Console</RouterLink>
      <nav class="flex gap-3 text-sm">
        <RouterLink to="/admin">Reviews</RouterLink>
        <RouterLink to="/admin/policies">License Policies</RouterLink>
        <RouterLink to="/admin/disputes">Disputes</RouterLink>
        <RouterLink to="/admin/curation">Curation</RouterLink>
        <RouterLink to="/">Listener Home</RouterLink>
      </nav>
      <div class="ml-auto text-sm">
        <button class="underline" @click="auth.logout()">Logout</button>
      </div>
    </header>
    <main class="mx-auto max-w-6xl px-4 py-6">
      <RouterView />
    </main>
  </div>
</template>

<script setup lang="ts">
import { useAuthStore } from '@/stores/authStore';

const auth = useAuthStore();
</script>
