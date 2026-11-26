<!-- Help center (/help): searchable FAQ by topic + FAQPage structured data. -->
<template>
  <div class="mx-auto max-w-4xl space-y-10">
    <section class="relative pt-6 text-center sm:pt-12">
      <div class="hero-glow pointer-events-none absolute left-1/2 top-0 h-72 w-[44rem] max-w-[100vw] -translate-x-1/2 opacity-60" aria-hidden="true" />
      <div class="relative">
        <p class="text-sm font-medium text-brand-400">{{ c.help.eyebrow }}</p>
        <h1 class="type-display mt-3 text-4xl text-white sm:text-5xl">{{ c.help.title }}</h1>
        <p class="mx-auto mt-4 max-w-xl text-sm leading-relaxed text-text-secondary sm:text-base">{{ c.help.lead }}</p>
        <form class="prompt-box mx-auto mt-8 flex max-w-xl items-center gap-3 rounded-2xl px-4 py-3" role="search" @submit.prevent>
          <svg class="h-5 w-5 shrink-0 text-text-muted" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
            <circle cx="11" cy="11" r="7" /><path d="m20 20-3.5-3.5" />
          </svg>
          <input
            v-model="query"
            type="search"
            :placeholder="c.help.searchPh"
            :aria-label="c.help.searchLabel"
            class="w-full bg-transparent text-sm text-white placeholder:text-text-muted focus:outline-none"
          >
        </form>
      </div>
    </section>

    <div class="flex flex-wrap justify-center gap-2" role="tablist" :aria-label="c.help.eyebrow">
      <button
        v-for="cat in tabs"
        :key="cat.id"
        type="button"
        role="tab"
        :aria-selected="active === cat.id"
        class="rounded-full border px-4 py-1.5 text-sm transition-colors"
        :class="active === cat.id ? 'border-white bg-white text-surface-0' : 'border-white/15 text-text-secondary hover:border-white/30 hover:text-white'"
        @click="active = cat.id"
      >
        {{ cat.name }}
      </button>
    </div>

    <p class="text-center text-xs text-text-muted" aria-live="polite">{{ countLabel }}</p>

    <div v-if="visible.length > 0" class="space-y-10">
      <section v-for="cat in visible" :key="cat.id" :aria-labelledby="`help-${cat.id}`">
        <h2 :id="`help-${cat.id}`" class="mb-3 text-sm font-medium text-text-muted">{{ cat.name }}</h2>
        <div class="divide-y divide-white/[0.06] overflow-hidden rounded-2xl border border-white/[0.08] bg-white/[0.02]">
          <details v-for="(item, i) in cat.items" :key="i" class="group" :open="query.trim().length > 0">
            <summary class="flex cursor-pointer list-none items-center justify-between gap-4 px-5 py-4 text-sm font-medium text-white hover:bg-white/[0.03] sm:text-base">
              <span>{{ item.q }}</span>
              <svg class="h-4 w-4 shrink-0 text-text-muted transition-transform group-open:rotate-45" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
                <path d="M12 5v14M5 12h14" />
              </svg>
            </summary>
            <p class="px-5 pb-5 text-sm leading-relaxed text-text-secondary">{{ item.a }}</p>
          </details>
        </div>
      </section>
    </div>
    <p v-else class="rounded-2xl border border-dashed border-white/10 px-6 py-12 text-center text-sm text-text-secondary">{{ c.help.empty }}</p>

    <section class="flex flex-col items-start justify-between gap-4 rounded-2xl border border-white/[0.08] bg-white/[0.03] p-6 sm:flex-row sm:items-center">
      <div>
        <h2 class="text-lg font-medium text-white">{{ c.help.stillTitle }}</h2>
        <p class="mt-1 text-sm text-text-secondary">{{ c.help.stillSub }}</p>
      </div>
      <NuxtLink :to="localePath('/contact')" class="inline-flex min-h-11 shrink-0 items-center rounded-full bg-brand-500 px-5 text-sm font-semibold text-on-brand hover:bg-brand-400">
        {{ c.help.stillBtn }}
      </NuxtLink>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { useSiteContent } from '@/data/site';
import { usePublicSeo } from '@/composables/usePublicSeo';
import { breadcrumbLd, canonicalFor, faqLd } from '@/utils/seo';

const localePath = useLocalePath();
const { locale } = useI18n();
const route = useRoute();
const c = await useSiteContent();

const query = ref(typeof route.query.q === 'string' ? route.query.q : '');
const active = ref(typeof route.query.topic === 'string' ? route.query.topic : 'all');

const tabs = computed(() => [{ id: 'all', name: c.value.help.all }, ...c.value.help.categories.map((x) => ({ id: x.id, name: x.name }))]);

const visible = computed(() => {
  const q = query.value.trim().toLowerCase();
  return c.value.help.categories
    .filter((cat) => active.value === 'all' || cat.id === active.value)
    .map((cat) => ({
      ...cat,
      items: q ? cat.items.filter((it) => `${it.q} ${it.a}`.toLowerCase().includes(q)) : cat.items,
    }))
    .filter((cat) => cat.items.length > 0);
});

const countLabel = computed(() =>
  c.value.help.resultCount.replace('{n}', String(visible.value.reduce((n, cat) => n + cat.items.length, 0))),
);

usePublicSeo({
  title: computed(() => `${c.value.help.eyebrow} · AVIBE`),
  description: computed(() => c.value.help.lead),
  path: '/help',
  locale,
});
useSchemaOrg([
  faqLd(c.value.help.categories.flatMap((cat) => cat.items.map((it) => ({ q: it.q, a: it.a })))),
  breadcrumbLd([
    { name: 'AVIBE', url: canonicalFor('/', locale.value) },
    { name: c.value.help.eyebrow, url: canonicalFor('/help', locale.value) },
  ]),
]);
</script>
