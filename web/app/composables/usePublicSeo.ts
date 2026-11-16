// One-call public-page head (WI-S2): title, description, canonical,
// hreflang + x-default, OG/music tags, Twitter card, robots, and oEmbed
// discovery. SSR-safe (useHead/useSeoMeta render server-side).
// Every field accepts a ref so layout-level fallbacks stay correct across
// client-side navigations and locale switches.
import { computed, toValue, unref, type MaybeRef } from 'vue';
import { canonicalFor, hreflangLinks, SITE_URL } from '@/utils/seo';

export type OgType = 'website' | 'music.song' | 'music.playlist' | 'profile' | 'article';

export interface PublicSeoInput {
  title: MaybeRef<string>;
  description: MaybeRef<string>;
  /** Unprefixed page path (query stripped by callers). */
  path: MaybeRef<string>;
  locale: MaybeRef<string>;
  ogType?: OgType;
  /**
   * OG/Twitter image. `null` skips image tags (nuxt-og-image owns them via
   * defineOgImageComponent); undefined falls back to the brand mark.
   */
  image?: string | null;
  imageAlt?: string;
  /** music:* tags for song/playlist pages. */
  musicDuration?: number | null;
  musicMusician?: string | null;
  noindex?: MaybeRef<boolean>;
  publishedTime?: string | null;
  modifiedTime?: string | null;
  oembedUrl?: string | null;
}

const OG_LOCALE: Record<string, string> = {
  ko: 'ko_KR',
  en: 'en_US',
  ja: 'ja_JP',
  'zh-cn': 'zh_CN',
  th: 'th_TH',
};

export function usePublicSeo(input: PublicSeoInput): void {
  const localeNorm = computed(() => (toValue(input.locale) || 'ko').toLowerCase());
  const canonical = computed(() => canonicalFor(toValue(input.path), localeNorm.value));
  const alternates = computed(() => hreflangLinks(toValue(input.path)));
  const image = computed(() => (input.image === null ? null : input.image || `${SITE_URL}/favicon.svg`));
  const robots = computed(() => (unref(input.noindex) ? 'noindex, follow' : 'index, follow, max-image-preview:large'));

  useSeoMeta({
    title: computed(() => toValue(input.title)),
    description: computed(() => toValue(input.description)),
    ogTitle: computed(() => toValue(input.title)),
    ogDescription: computed(() => toValue(input.description)),
    ogType: input.ogType || 'website',
    ogUrl: canonical,
    ogImage: computed(() => image.value ?? undefined),
    ogImageAlt: computed(() => (image.value ? input.imageAlt || toValue(input.title) : undefined)),
    ogSiteName: 'AVIBE',
    ogLocale: computed(() => OG_LOCALE[localeNorm.value] || 'ko_KR'),
    ogLocaleAlternate: computed(() =>
      Object.values(OG_LOCALE).filter((l) => l !== (OG_LOCALE[localeNorm.value] || 'ko_KR')),
    ),
    twitterCard: 'summary_large_image',
    twitterTitle: computed(() => toValue(input.title)),
    twitterDescription: computed(() => toValue(input.description)),
    twitterImage: computed(() => image.value ?? undefined),
    robots,
    ...(input.musicDuration ? { musicDuration: String(input.musicDuration) } : {}),
    ...(input.musicMusician ? { musicMusician: input.musicMusician } : {}),
    ...(input.publishedTime ? { articlePublishedTime: input.publishedTime } : {}),
    ...(input.modifiedTime ? { articleModifiedTime: input.modifiedTime } : {}),
  });
  type HeadLink =
    | { rel: 'canonical'; href: string }
    | { rel: 'alternate'; hreflang: string; href: string }
    | { rel: 'alternate'; type: string; href: string; title: string };
  const links = computed<HeadLink[]>(() => {
    const list: HeadLink[] = [{ rel: 'canonical', href: canonical.value }];
    for (const a of alternates.value) list.push({ rel: 'alternate', hreflang: a.hreflang, href: a.href });
    if (input.oembedUrl) {
      list.push({ rel: 'alternate', type: 'application/json+oembed', href: input.oembedUrl, title: toValue(input.title) });
    }
    return list;
  });
  useHead({ link: links });
}
