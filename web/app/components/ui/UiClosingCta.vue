<!-- v3 flow closing CTA: soft hero light, large light headline, 1–2 actions. -->
<template>
  <section class="relative overflow-hidden rounded-3xl border border-white/[0.06] px-6 py-14 text-center sm:py-20">
    <div class="hero-glow pointer-events-none absolute inset-0 opacity-70" aria-hidden="true" />
    <div class="relative">
      <h2 class="type-display text-3xl text-white sm:text-5xl">{{ title }}</h2>
      <p v-if="sub" class="mx-auto mt-3 max-w-xl text-sm text-text-secondary sm:text-base">{{ sub }}</p>
      <div class="mt-7 flex flex-col items-center justify-center gap-2 sm:flex-row">
        <NuxtLink :to="primaryTo" class="inline-flex min-h-11 items-center rounded-full bg-brand-500 px-6 text-sm font-semibold text-on-brand transition-colors hover:bg-brand-400">
          {{ primaryLabel }}
        </NuxtLink>
        <NuxtLink
          v-if="secondaryTo && secondaryLabel"
          :to="secondaryTo"
          class="inline-flex min-h-11 items-center rounded-full border border-white/15 px-6 text-sm font-semibold text-white transition-colors hover:bg-white/[0.06]"
        >
          {{ secondaryLabel }}
        </NuxtLink>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
defineProps<{
  title: string;
  sub?: string;
  primaryTo: string;
  primaryLabel: string;
  secondaryTo?: string;
  secondaryLabel?: string;
}>();
</script>
