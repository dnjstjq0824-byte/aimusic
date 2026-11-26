<!-- Certificate lookup (/certificate): number entry -> /certificate/:number (path fixed). -->
<template>
  <div class="mx-auto max-w-2xl space-y-10 pt-4 sm:pt-10">
    <section class="relative text-center">
      <div class="hero-glow pointer-events-none absolute left-1/2 top-0 h-64 w-[40rem] max-w-[100vw] -translate-x-1/2 opacity-60" aria-hidden="true" />
      <div class="relative">
        <span class="mx-auto flex h-14 w-14 items-center justify-center rounded-full border border-verified/40 bg-verified/10 text-verified" aria-hidden="true">
          <svg class="h-7 w-7" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M12 3 4 6v6c0 5 3.5 8 8 9 4.5-1 8-4 8-9V6l-8-3z" /><path d="m9 12 2 2 4-4" /></svg>
        </span>
        <p class="mt-5 text-sm font-medium text-brand-400">{{ c.cert.eyebrow }}</p>
        <h1 class="type-display mt-2 text-4xl text-white sm:text-5xl">{{ c.cert.title }}</h1>
        <p class="mx-auto mt-4 max-w-lg text-sm leading-relaxed text-text-secondary sm:text-base">{{ c.cert.lead }}</p>
      </div>
    </section>

    <form class="space-y-2" novalidate @submit.prevent="submit">
      <label for="cert-number" class="block text-xs font-medium text-text-secondary">{{ c.cert.label }}</label>
      <div class="prompt-box flex items-center gap-2 rounded-2xl p-2 pl-4">
        <input
          id="cert-number"
          v-model="number"
          type="text"
          inputmode="text"
          autocomplete="off"
          spellcheck="false"
          :placeholder="c.cert.ph"
          :aria-invalid="invalid"
          aria-describedby="cert-hint"
          class="min-w-0 flex-1 bg-transparent font-mono text-base uppercase tracking-wider text-white placeholder:text-text-muted focus:outline-none"
          @input="invalid = false"
        >
        <button type="submit" class="min-h-11 shrink-0 rounded-xl bg-brand-500 px-5 text-sm font-semibold text-on-brand hover:bg-brand-400">
          {{ c.cert.submit }}
        </button>
      </div>
      <p id="cert-hint" class="text-xs" :class="invalid ? 'text-accent-crimson' : 'text-text-muted'" :role="invalid ? 'alert' : undefined">
        {{ c.cert.invalid }}
      </p>
    </form>

    <section class="rounded-2xl border border-white/[0.08] bg-white/[0.02] p-6">
      <h2 class="text-sm font-medium text-white">{{ c.cert.whereTitle }}</h2>
      <ol class="mt-3 space-y-2">
        <li v-for="(w, i) in c.cert.where" :key="i" class="flex gap-3 text-sm text-text-secondary">
          <span class="font-mono text-xs text-text-muted">{{ i + 1 }}</span>{{ w }}
        </li>
      </ol>
      <NuxtLink :to="localePath('/trust')" class="mt-5 inline-flex text-sm font-medium text-white hover:text-brand-300">{{ c.cert.trustLink }} →</NuxtLink>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { useSiteContent } from '@/data/site';
import { usePublicSeo } from '@/composables/usePublicSeo';

const CERT_RE = /^AVB-[A-Z2-9]{10}$/;

const localePath = useLocalePath();
const { locale } = useI18n();
const c = await useSiteContent();

const number = ref('');
const invalid = ref(false);

async function submit(): Promise<void> {
  const raw = number.value.trim().toUpperCase().replace(/\s+/g, '');
  const normalized = raw.startsWith('AVB-') ? raw : `AVB-${raw.replace(/^AVB/, '')}`;
  if (!CERT_RE.test(normalized)) {
    invalid.value = true;
    return;
  }
  await navigateTo(localePath(`/certificate/${normalized}`));
}

usePublicSeo({
  title: computed(() => `${c.value.cert.title} · AVIBE`),
  description: computed(() => c.value.cert.lead),
  path: '/certificate',
  locale,
});
</script>
