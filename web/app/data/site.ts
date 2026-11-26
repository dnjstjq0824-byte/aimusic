// Company/support page content (about, help, contact, models, creators
// directory, certificate lookup). Bodies live in web/content/site.*.json
// (article content outside the i18n hardcoding scan, like guides). Each
// locale is its own chunk, so a page only downloads the language it shows.
import type ko from '../../content/site.ko.json';

export type SiteContent = typeof ko;

const LOADERS: Record<string, () => Promise<{ default: SiteContent }>> = {
  ko: () => import('../../content/site.ko.json') as Promise<{ default: SiteContent }>,
  en: () => import('../../content/site.en.json') as Promise<{ default: SiteContent }>,
  ja: () => import('../../content/site.ja.json') as Promise<{ default: SiteContent }>,
  'zh-cn': () => import('../../content/site.zh-cn.json') as Promise<{ default: SiteContent }>,
  th: () => import('../../content/site.th.json') as Promise<{ default: SiteContent }>,
};

export async function loadSiteContent(locale: string): Promise<SiteContent> {
  const load = LOADERS[locale.toLowerCase()] ?? LOADERS.ko!;
  try {
    return (await load()).default;
  } catch {
    return (await LOADERS.ko!()).default;
  }
}

/** Reactive content for the current locale (SSR-safe, keyed per locale). */
export async function useSiteContent() {
  const { locale } = useI18n();
  const { data } = await useAsyncData(
    () => `site-content-${locale.value}`,
    () => loadSiteContent(locale.value),
    { watch: [locale] },
  );
  const fallback = data.value ?? (await loadSiteContent('ko'));
  return computed<SiteContent>(() => data.value ?? fallback);
}
