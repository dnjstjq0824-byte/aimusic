// Guide hub loader (WI-S4). Bodies live in web/content/*.json (outside the
// i18n hardcoding scan: article content, not UI strings). Facts grounded in
// docs/handoff/20261111/음악생성AI_상업이용_플랫폼조사_재설계_20261111.md
// (survey 2026-11-11). ko + en ship; ja/zh-cn/th render ko + draft notice.
import koData from '../../content/guides.ko.json';
import enData from '../../content/guides.en.json';

export interface GuideSection {
  h: string;
  ps: string[];
}

export interface GuideFaq {
  q: string;
  a: string;
}

export interface GuideBody {
  headline: string;
  description: string;
  updated: string;
  sections: GuideSection[];
  faqs: GuideFaq[];
  sourcesNote: string;
}

export interface GuideMeta {
  slug: string;
  updated: string;
}

interface GuideFile {
  metas: GuideMeta[];
  bodies: Record<string, GuideBody>;
}

const KO = koData as unknown as GuideFile;
const EN = enData as unknown as GuideFile;

export const GUIDE_METAS: GuideMeta[] = KO.metas;

export function guideSlugs(): string[] {
  return GUIDE_METAS.map((m) => m.slug);
}

export function guideBodyFor(slug: string, locale: string): { body: GuideBody; draftNotice: boolean } | null {
  if (locale === 'en') {
    const body = EN.bodies[slug];
    return body ? { body, draftNotice: false } : null;
  }
  if (locale === 'ko') {
    const body = KO.bodies[slug];
    return body ? { body, draftNotice: false } : null;
  }
  // ja/zh-cn/th: ko original + draft notice until native review (G-L).
  const body = KO.bodies[slug];
  return body ? { body, draftNotice: true } : null;
}
