<template>
  <div class="mx-auto max-w-md">
    <h1 class="mb-4 text-xl font-bold">Login</h1>
    <form class="space-y-3" @submit.prevent="submit">
      <input v-model="email" type="email" required placeholder="email" class="w-full rounded bg-neutral-800 p-2" />
      <input v-model="password" type="password" required placeholder="password" class="w-full rounded bg-neutral-800 p-2" />
      <p v-if="error" class="text-sm text-red-400">{{ error }}</p>
      <button class="w-full rounded bg-brand-600 py-2 font-bold disabled:opacity-50" :disabled="busy">
        {{ busy ? '로그인 중…' : 'Login' }}
      </button>
    </form>
    <div class="mt-4 border-t border-neutral-800 pt-4">
      <h2 class="mb-2 font-bold">Listener sign-up</h2>
      <form class="space-y-3" @submit.prevent="signup">
        <input v-model="suEmail" type="email" required placeholder="email" class="w-full rounded bg-neutral-800 p-2" />
        <input v-model="suPassword" type="password" required minlength="8" placeholder="password (8+)" class="w-full rounded bg-neutral-800 p-2" />
        <input v-model="suName" required placeholder="display name" class="w-full rounded bg-neutral-800 p-2" />
        <p v-if="suError" class="text-sm text-red-400">{{ suError }}</p>
        <p v-if="suDone" class="text-sm text-emerald-400">가입 완료. 로그인해 주세요.</p>
        <button class="w-full rounded bg-neutral-700 py-2 font-bold">Sign up</button>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useAuthStore } from '@/stores/authStore';
import { errMessageOf } from '@/api/client';

const auth = useAuthStore();
const router = useRouter();
const route = useRoute();

const email = ref('');
const password = ref('');
const error = ref<string | null>(null);
const busy = ref(false);

const suEmail = ref('');
const suPassword = ref('');
const suName = ref('');
const suError = ref<string | null>(null);
const suDone = ref(false);

async function submit(): Promise<void> {
  busy.value = true;
  error.value = null;
  try {
    await auth.login(email.value, password.value);
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/';
    await router.push(redirect);
  } catch (err: unknown) {
    error.value = errMessageOf(err);
  } finally {
    busy.value = false;
  }
}

async function signup(): Promise<void> {
  suError.value = null;
  suDone.value = false;
  try {
    await auth.register(suEmail.value, suPassword.value, suName.value);
    suDone.value = true;
  } catch (err: unknown) {
    suError.value = errMessageOf(err);
  }
}
</script>
