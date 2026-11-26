<!-- AI model directory (/models): own model + per-platform acceptance conditions (license gate mirror). -->
<template>
  <div class="space-y-14">
    <UiPageHeader :eyebrow="c.models.eyebrow" :title="c.models.title" :description="c.models.lead" />

    <!-- Own model: feature card -->
    <section v-if="own" :aria-label="c.models.ownTitle">
      <NuxtLink
        :to="localePath(`/models/${own.slug}`)"
        class="group relative grid overflow-hidden rounded-3xl border border-white/[0.08] bg-white/[0.03] md:grid-cols-[1.2fr_1fr]"
      >
        <div class="relative p-7 sm:p-10">
          <div class="flex items-center gap-2">
            <span class="rounded-full border border-verified/40 px-2.5 py-0.5 text-xs font-medium text-verified">{{ c.models.statusOwn }}</span>
            <span class="text-xs text-text-muted">Born Verified</span>
          </div>
          <h2 class="type-display mt-4 text-3xl text-white sm:text-4xl">{{ own.name }}</h2>
          <p class="mt-3 max-w-lg text-sm leading-relaxed text-text-secondary sm:text-base">{{ c.models.items.avibe.summary }}</p>
          <ul class="mt-6 space-y-2">
            <li v-for="line in c.models.items.avibe.accepted" :key="line" class="flex gap-2 text-sm text-text-primary">
              <span class="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full bg-verified" aria-hidden="true" />{{ line }}
            </li>
          </ul>
          <span class="mt-8 inline-flex items-center gap-1 text-sm font-medium text-white">{{ c.models.viewDetail }} <span class="transition-transform group-hover:translate-x-0.5" aria-hidden="true">→</span></span>
        </div>
        <div class="relative hidden min-h-72 md:block">
          <div class="hero-glow absolute inset-0 opacity-90" aria-hidden="true" />
          <div class="absolute inset-0 flex items-center justify-center" aria-hidden="true">
            <div class="flex h-40 items-end gap-1.5">
              <span v-for="(h, i) in bars" :key="i" class="w-2 rounded-full bg-white/70" :style="{ height: `${h}%` }" />
            </div>
          </div>
        </div>
      </NuxtLink>
    </section>

    <section :aria-label="c.models.partnerTitle">
      <div class="mb-5 flex flex-wrap items-end justify-between gap-2">
        <h2 class="type-display text-2xl text-white sm:text-3xl">{{ c.models.partnerTitle }}</h2>
        <p class="text-xs text-text-muted">{{ c.models.basis }}</p>
      </div>
      <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        <NuxtLink
          v-for="m in partners"
          :key="m.slug"
          :to="localePath(`/models/${m.slug}`)"
          class="group flex flex-col rounded-2xl border border-white/[0.08] bg-white/[0.02] p-5 transition-colors hover:border-white/20 hover:bg-white/[0.04]"
        >
          <div class="flex items-center justify-between gap-2">
            <h3 class="text-base font-medium text-white">{{ m.name }}</h3>
            <span
              class="shrink-0 rounded-full px-2 py-0.5 text-caption font-medium"
              :class="m.status === 'blocked' ? 'bg-accent-crimson/10 text-accent-crimson' : 'bg-verified/10 text-verified'"
            >
              {{ m.status === 'blocked' ? c.models.statusBlocked : c.models.statusAccepted }}
            </span>
          </div>
          <p class="mt-2 line-clamp-3 flex-1 text-sm leading-relaxed text-text-secondary">{{ itemFor(m.slug)?.summary }}</p>
          <span class="mt-4 text-sm text-text-muted transition-colors group-hover:text-white">{{ c.models.viewDetail }} →</span>
        </NuxtLink>
      </div>
    </section>

    <section class="rounded-2xl border border-white/[0.08] p-6 sm:p-8">
      <h2 class="text-lg font-medium text-white">{{ c.models.commonTitle }}</h2>
      <ul class="mt-4 grid gap-3 sm:grid-cols-2">
        <li v-for="line in c.models.common" :key="line" class="flex gap-3 text-sm text-text-secondary">
          <svg class="mt-0.5 h-4 w-4 shrink-0 text-verified" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m5 12 5 5L20 7" /></svg>
          {{ line }}
        </li>
      </ul>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useSiteContent } from '@/data/site';
import { MODEL_METAS } from '@/data/models';
import { usePublicSeo } from '@/composables/usePublicSeo';
import { breadcrumbLd, canonicalFor } from '@/utils/seo';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';

const localePath = useLocalePath();
const { locale } = useI18n();
const c = await useSiteContent();

const own = MODEL_METAS.find((m) => m.status === 'own');
const partners = MODEL_METAS.filter((m) => m.status !== 'own');
const bars = [30, 52, 74, 46, 88, 64, 96, 58, 80, 42, 70, 50, 36, 62, 84, 48];

type ModelItems = typeof c.value.models.items;
function itemFor(slug: string): ModelItems[keyof ModelItems] | undefined {
  return (c.value.models.items as Record<string, ModelItems[keyof ModelItems]>)[slug];
}

usePublicSeo({
  title: computed(() => `${c.value.models.eyebrow} · AVIBE`),
  description: computed(() => c.value.models.lead),
  path: '/models',
  locale,
});
useSchemaOrg([
  breadcrumbLd([
    { name: 'AVIBE', url: canonicalFor('/', locale.value) },
    { name: c.value.models.eyebrow, url: canonicalFor('/models', locale.value) },
  ]),
]);
</script>
