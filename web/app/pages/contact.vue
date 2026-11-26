<!-- Contact hub (/contact): routes each kind of question to its channel; business inquiries inline. -->
<template>
  <div class="space-y-14">
    <UiPageHeader :eyebrow="c.contact.eyebrow" :title="c.contact.title" :description="c.contact.lead" />

    <section class="grid gap-4 sm:grid-cols-2">
      <NuxtLink
        v-for="card in cards"
        :key="card.id"
        :to="card.to"
        class="group flex flex-col items-start rounded-2xl border border-white/[0.08] bg-white/[0.03] p-6 text-left transition-colors hover:border-white/20 hover:bg-white/[0.05]"
      >
        <span class="flex h-10 w-10 items-center justify-center rounded-full border border-white/10 text-brand-300" aria-hidden="true">
          <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
            <path :d="card.icon" />
          </svg>
        </span>
        <h2 class="mt-4 text-lg font-medium text-white">{{ card.t }}</h2>
        <p class="mt-1 text-sm leading-relaxed text-text-secondary">{{ card.d }}</p>
        <span class="mt-5 inline-flex items-center gap-1 text-sm font-medium text-white">
          {{ card.cta }} <span class="transition-transform group-hover:translate-x-0.5" aria-hidden="true">→</span>
        </span>
      </NuxtLink>
    </section>

    <section v-if="emails.length > 0" class="rounded-2xl border border-white/[0.08] p-6">
      <h2 class="text-sm font-medium text-text-muted">{{ c.contact.emailTitle }}</h2>
      <ul class="mt-3 space-y-2 text-sm">
        <li v-for="e in emails" :key="e.addr" class="flex flex-wrap items-center gap-2">
          <a :href="`mailto:${e.addr}`" class="font-medium text-white hover:text-brand-300">{{ e.addr }}</a>
          <span v-if="e.note" class="text-xs text-text-muted">· {{ e.note }}</span>
        </li>
      </ul>
    </section>

    <section id="inquiry" :aria-label="c.contact.formTitle" class="scroll-mt-24">
      <UiCard class="mx-auto max-w-2xl p-6">
        <h2 class="text-xl font-medium text-white">{{ c.contact.formTitle }}</h2>
        <p class="mt-1 text-xs text-text-secondary">{{ c.contact.formSub }}</p>
        <div class="mt-4">
          <ClientOnly>
            <InquiryForm />
          </ClientOnly>
        </div>
      </UiCard>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent } from 'vue';
import { useSiteContent } from '@/data/site';
import { usePublicSeo } from '@/composables/usePublicSeo';
import { breadcrumbLd, canonicalFor } from '@/utils/seo';
import UiPageHeader from '@/components/ui/UiPageHeader.vue';
import UiCard from '@/components/ui/UiCard.vue';

const InquiryForm = defineAsyncComponent(() => import('@/components/business/InquiryForm.vue'));

const localePath = useLocalePath();
const { locale } = useI18n();
const c = await useSiteContent();

const ICONS: Record<string, string> = {
  business: 'M3 21h18M5 21V7l7-4 7 4v14M9 9h1m4 0h1M9 13h1m4 0h1M9 17h1m4 0h1',
  rights: 'M12 3 4 6v6c0 5 3.5 8 8 9 4.5-1 8-4 8-9V6l-8-3zM9 12l2 2 4-4',
  creator: 'M9 18V5l12-2v13M9 18a3 3 0 1 1-6 0 3 3 0 0 1 6 0zm12-2a3 3 0 1 1-6 0 3 3 0 0 1 6 0z',
  help: 'M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18zM9.1 9a3 3 0 0 1 5.8 1c0 2-3 2.5-3 4M12 17.5h.01',
};
const TARGETS: Record<string, string> = { rights: '/rights-center', creator: '/for-creators', help: '/help' };

const cards = computed(() =>
  c.value.contact.cards.map((card) => ({
    ...card,
    icon: ICONS[card.id] ?? ICONS.help,
    to: card.id === 'business' ? { hash: '#inquiry' } : (localePath(TARGETS[card.id] ?? '/help') || (TARGETS[card.id] ?? '/help')),
  })),
);

const env = import.meta.env as Record<string, string | undefined>;
const emails = computed(() => {
  const out: { addr: string; note: string }[] = [];
  const support = (env.VITE_BUSINESS_EMAIL ?? '').trim();
  const osp = (env.VITE_OSP_CONTACT_EMAIL ?? '').trim();
  if (support) out.push({ addr: support, note: '' });
  if (osp && osp !== support) out.push({ addr: osp, note: c.value.contact.emailOsp });
  return out;
});

usePublicSeo({
  title: computed(() => `${c.value.contact.eyebrow} · AVIBE`),
  description: computed(() => c.value.contact.lead),
  path: '/contact',
  locale,
});
useSchemaOrg([
  breadcrumbLd([
    { name: 'AVIBE', url: canonicalFor('/', locale.value) },
    { name: c.value.contact.eyebrow, url: canonicalFor('/contact', locale.value) },
  ]),
]);
</script>
