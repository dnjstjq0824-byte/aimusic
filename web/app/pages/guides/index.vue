<!-- Guide hub (WI-S4): SSR index of the 3 launch guides. -->
<template>
  <div class="space-y-6">
    <div>
      <h1 class="font-display text-2xl font-medium text-white sm:text-3xl">{{ t('seo.guideHubTitle') }}</h1>
      <p class="mt-2 max-w-2xl text-sm leading-relaxed text-text-secondary">{{ t('seo.guideHubDescription') }}</p>
    </div>
    <div class="grid gap-4 md:grid-cols-3">
      <RouterLink
        v-for="g in guides"
        :key="g.slug"
        :to="localePath({ name: 'guides-slug', params: { slug: g.slug } })"
        class="group rounded-2xl border border-surface-border bg-surface-1 p-5 transition-colors hover:border-brand-500/50"
      >
        <h2 class="text-sm font-bold text-white group-hover:text-brand-300">{{ g.headline }}</h2>
        <p class="mt-2 line-clamp-3 text-xs leading-relaxed text-text-secondary">{{ g.description }}</p>
        <p class="mt-3 text-caption text-text-muted">{{ t('seo.guideUpdated', { date: g.updated }) }}</p>
      </RouterLink>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { usePublicSeo } from '@/composables/usePublicSeo';
import { breadcrumbLd, canonicalFor, needsCjkThaiFont } from '@/utils/seo';
import { GUIDE_METAS, guideBodyFor } from '@/data/guides';

const localePath = useLocalePath();
const { t, locale } = useI18n();

const guides = computed(() =>
  GUIDE_METAS.map((m) => {
    const found = guideBodyFor(m.slug, locale.value);
    return {
      slug: m.slug,
      updated: m.updated,
      headline: found?.body.headline ?? m.slug,
      description: found?.body.description ?? '',
    };
  }),
);

usePublicSeo({
  title: computed(() => `${t('seo.guideHubTitle')} · AVIBE`),
  description: computed(() => t('seo.guideHubDescription')),
  path: '/guides',
  locale,
});
useSchemaOrg([
  breadcrumbLd([
    { name: 'AVIBE', url: canonicalFor('/', locale.value) },
    { name: String(t('seo.guideHubTitle')), url: canonicalFor('/guides', locale.value) },
  ]),
]);
defineOgImageComponent('AvibeOg', {
  title: String(t('seo.guideHubTitle')),
  creator: 'AVIBE',
  verified: false,
  plain: needsCjkThaiFont(String(t('seo.guideHubTitle'))),
});
</script>
