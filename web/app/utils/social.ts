// Social provider ordering + slugs (R3, UI §3.11). Engine-enabled only;
// unknown future providers append, never dropped.
export interface SocialProviderMeta {
  code: string;
  slug: string;
  labelKey: string;
  short: string;
}

const META: Record<string, Omit<SocialProviderMeta, 'code'>> = {
  GOOGLE: { slug: 'google', labelKey: 'auth.social.google', short: 'Google' },
  APPLE: { slug: 'apple', labelKey: 'auth.social.apple', short: 'Apple' },
  KAKAO: { slug: 'kakao', labelKey: 'auth.social.kakao', short: 'Kakao' },
  NAVER: { slug: 'naver', labelKey: 'auth.social.naver', short: 'Naver' },
  LINE: { slug: 'line', labelKey: 'auth.social.line', short: 'LINE' },
};

const ORDER: Record<string, string[]> = {
  ko: ['KAKAO', 'NAVER', 'GOOGLE', 'APPLE'],
  ja: ['LINE', 'GOOGLE', 'APPLE'],
  th: ['LINE', 'GOOGLE', 'APPLE'],
  'zh-cn': ['GOOGLE', 'APPLE'],
  en: ['GOOGLE', 'APPLE'],
};

export function orderSocialProviders(providerCodes: string[], locale: string): SocialProviderMeta[] {
  const enabled = new Set(providerCodes);
  const order = ORDER[locale] ?? ORDER.en ?? [];
  const ranked = order.filter((c) => enabled.has(c) && META[c]);
  for (const c of enabled) {
    if (!ranked.includes(c) && META[c]) ranked.push(c);
  }
  return ranked.flatMap((c) => {
    const meta = META[c];
    return meta ? [{ code: c, ...meta }] : [];
  });
}
