<!-- Use-case landing (WI-S4): SSR, 404 when fewer than 12 READY tracks. -->
<template>
  <LandingPage
    kind="use"
    :tag-key="tagKey"
    :heading="heading"
    :description="description"
    :count="count"
    :tracks="tracks"
    :loading="loading"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useRoute } from 'vue-router';
import { fetchPublic } from '@/utils/publicApi';
import { usePublicSeo } from '@/composables/usePublicSeo';
import { tagDisplayName } from '@/utils/tags';
import {
  breadcrumbLd,
  canonicalFor,
  describeLanding,
  landingTitle,
  needsCjkThaiFont,
} from '@/utils/seo';
import type { Track, TrackPage } from '@/api/types';
import LandingPage from '@/components/landing/LandingPage.vue';

const { t, te, locale } = useI18n();
const route = useRoute();

const tagKey = computed(() => String(route.params.useCase ?? '').toLowerCase());
const { data: pageData, pending } = await useAsyncData(
  () => `landing-use-${tagKey.value}`,
  async () => {
    const gate = await fetchPublic<{ count: number; ok: boolean }>('/seo/landing', { use: tagKey.value }).catch(
      (): null => null,
    );
    if (!gate || !gate.ok) return null;
    const page = await fetchPublic<TrackPage>('/tracks', { limit: 24, mood: tagKey.value, sort: 'popular' }).catch(
      (): TrackPage => ({ items: [], total: 0, limit: 24, offset: 0 }),
    );
    return { count: gate.count, tracks: page.items ?? [] };
  },
);
if (!pageData.value) {
  throw showError({ statusCode: 404, statusMessage: 'Landing not found' });
}
const count = computed(() => pageData.value?.count ?? 0);
const tracks = computed<Track[]>(() => pageData.value?.tracks ?? []);
const loading = computed(() => pending.value);

const tagLabel = computed(() => tagDisplayName(tagKey.value, t, te) || tagKey.value);
const heading = computed(() => tagLabel.value);
const description = computed(() => describeLanding(t, 'use', tagLabel.value, count.value));
const pagePath = computed(() => `/explore/use/${tagKey.value}`);

usePublicSeo({
  title: computed(() => landingTitle(t, tagLabel.value)),
  description,
  path: pagePath,
  locale,
});
useSchemaOrg([
  breadcrumbLd([
    { name: 'AVIBE', url: canonicalFor('/', locale.value) },
    { name: String(t('common.titles.explore')), url: canonicalFor('/explore', locale.value) },
    { name: tagLabel.value, url: canonicalFor(pagePath.value, locale.value) },
  ]),
]);
// Server only: OG tags are for crawlers; keeps nuxt-og-image out of client JS.
if (import.meta.server) defineOgImageComponent('AvibeOg', {
  title: tagLabel.value,
  creator: 'AVIBE',
  verified: false,
  plain: needsCjkThaiFont(tagLabel.value),
});
</script>
