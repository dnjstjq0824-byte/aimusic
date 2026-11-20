<!-- Guide article (WI-S4): SSR, Article + FAQPage JSON-LD, share row. -->
<template>
  <div class="mx-auto max-w-3xl space-y-6">
    <nav :aria-label="t('seo.guideHubTitle')" class="text-xs text-text-secondary">
      <RouterLink :to="localePath({ name: 'guides' })" class="hover:text-brand-300 hover:underline">
        {{ t('seo.guideHubTitle') }}
      </RouterLink>
    </nav>

    <div>
      <h1 class="font-display text-2xl font-medium text-white sm:text-3xl">{{ body.headline }}</h1>
      <p class="mt-2 text-sm leading-relaxed text-text-secondary">{{ body.description }}</p>
      <p class="mt-1 text-caption text-text-muted">{{ t('seo.guideUpdated', { date: body.updated }) }}</p>
      <div
        v-if="draftNotice"
        class="mt-3 rounded-xl border border-accent-amber/40 bg-amber-950/30 p-4 text-xs text-amber-100"
        role="status"
      >
        {{ t('seo.guideDraftNotice') }}
      </div>
    </div>

    <article class="space-y-6">
      <section v-for="(s, i) in body.sections" :key="i" class="space-y-2">
        <h2 class="text-base font-bold text-white">{{ s.h }}</h2>
        <p v-for="(p, j) in s.ps" :key="j" class="text-sm leading-relaxed text-text-primary">{{ p }}</p>
      </section>

      <section :aria-label="t('seo.guideFaqTitle')" class="space-y-3">
        <h2 class="text-base font-bold text-white">{{ t('seo.guideFaqTitle') }}</h2>
        <div v-for="(f, i) in body.faqs" :key="i" class="rounded-xl border border-surface-border bg-surface-1 p-4">
          <h3 class="text-sm font-bold text-white">{{ f.q }}</h3>
          <p class="mt-1 text-sm leading-relaxed text-text-secondary">{{ f.a }}</p>
        </div>
      </section>

      <p class="border-t border-surface-border pt-4 text-caption leading-relaxed text-text-muted">
        {{ t('seo.guideSources') }}: {{ body.sourcesNote }}
      </p>
    </article>

    <ShareButtons :url="shareUrl" :title="body.headline" />

    <section :aria-label="t('seo.guideRelated')" class="space-y-2">
      <h2 class="text-sm font-bold text-white">{{ t('seo.guideRelated') }}</h2>
      <div class="grid gap-2 sm:grid-cols-2">
        <RouterLink
          v-for="g in related"
          :key="g.slug"
          :to="localePath({ name: 'guides-slug', params: { slug: g.slug } })"
          class="truncate rounded-xl border border-surface-border bg-surface-1 px-3 py-2 text-xs font-semibold text-text-primary transition-colors hover:border-brand-500/50 hover:text-brand-300"
        >
          {{ g.headline }}
        </RouterLink>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useRoute } from 'vue-router';
import { usePublicSeo } from '@/composables/usePublicSeo';
import {
  articleLd,
  breadcrumbLd,
  canonicalFor,
  faqLd,
  guideTitle,
  needsCjkThaiFont,
} from '@/utils/seo';
import { GUIDE_METAS, guideBodyFor } from '@/data/guides';
import ShareButtons from '@/components/seo/ShareButtons.vue';

const localePath = useLocalePath();
const { t, locale } = useI18n();
const route = useRoute();

const slug = computed(() => String(route.params.slug ?? ''));
const found = computed(() => guideBodyFor(slug.value, locale.value));
if (!found.value) {
  throw showError({ statusCode: 404, statusMessage: 'Guide not found' });
}
const body = computed(() => found.value!.body);
const draftNotice = computed(() => found.value!.draftNotice);
const pagePath = computed(() => `/guides/${slug.value}`);
const shareUrl = computed(() => canonicalFor(pagePath.value, locale.value));

const related = computed(() =>
  GUIDE_METAS.filter((m) => m.slug !== slug.value).map((m) => ({
    slug: m.slug,
    headline: guideBodyFor(m.slug, locale.value)?.body.headline ?? m.slug,
  })),
);

usePublicSeo({
  title: computed(() => guideTitle(t, body.value.headline)),
  description: computed(() => body.value.description),
  path: pagePath,
  locale,
  ogType: 'article',
  image: null,
  publishedTime: body.value.updated,
  modifiedTime: body.value.updated,
});
useSchemaOrg([
  articleLd(body.value.headline, shareUrl.value, body.value.updated),
  faqLd(body.value.faqs),
  breadcrumbLd([
    { name: 'AVIBE', url: canonicalFor('/', locale.value) },
    { name: String(t('seo.guideHubTitle')), url: canonicalFor('/guides', locale.value) },
    { name: body.value.headline, url: shareUrl.value },
  ]),
]);
// Server only: OG tags are for crawlers; keeps nuxt-og-image out of client JS.
if (import.meta.server) defineOgImageComponent('AvibeOg', {
  title: body.value.headline,
  creator: 'AVIBE Guide',
  verified: false,
  plain: needsCjkThaiFont(body.value.headline),
});
</script>
