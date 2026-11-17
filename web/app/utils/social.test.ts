import { describe, expect, it } from 'vitest';
import { orderSocialProviders } from './social';

describe('orderSocialProviders (UI §3.11)', () => {
  const ALL = ['GOOGLE', 'APPLE', 'KAKAO', 'NAVER', 'LINE'];

  it('orders ko as kakao·naver·google·apple', () => {
    expect(orderSocialProviders(ALL, 'ko').map((p) => p.code)).toEqual(['KAKAO', 'NAVER', 'GOOGLE', 'APPLE', 'LINE']);
  });

  it('orders ja/th as line·google·apple', () => {
    expect(orderSocialProviders(ALL, 'ja').map((p) => p.code)).toEqual(['LINE', 'GOOGLE', 'APPLE', 'KAKAO', 'NAVER']);
    expect(orderSocialProviders(ALL, 'th').map((p) => p.code)).toEqual(['LINE', 'GOOGLE', 'APPLE', 'KAKAO', 'NAVER']);
  });

  it('orders zh-cn/en as google·apple', () => {
    expect(orderSocialProviders(ALL, 'zh-cn').map((p) => p.code)).toEqual(['GOOGLE', 'APPLE', 'KAKAO', 'NAVER', 'LINE']);
    expect(orderSocialProviders(ALL, 'en').map((p) => p.code)).toEqual(['GOOGLE', 'APPLE', 'KAKAO', 'NAVER', 'LINE']);
  });

  it('keeps only enabled providers, preserving locale order', () => {
    expect(orderSocialProviders(['GOOGLE', 'KAKAO'], 'ko').map((p) => p.code)).toEqual(['KAKAO', 'GOOGLE']);
    expect(orderSocialProviders(['APPLE', 'LINE'], 'ja').map((p) => p.code)).toEqual(['LINE', 'APPLE']);
  });

  it('appends off-list providers instead of dropping them', () => {
    expect(orderSocialProviders(['GOOGLE', 'KAKAO'], 'en').map((p) => p.code)).toEqual(['GOOGLE', 'KAKAO']);
  });

  it('ignores unknown provider codes', () => {
    expect(orderSocialProviders(['GOOGLE', 'WECHAT'], 'en').map((p) => p.code)).toEqual(['GOOGLE']);
  });

  it('falls back to en order for unknown locales', () => {
    expect(orderSocialProviders(ALL, 'xx').map((p) => p.code)).toEqual(['GOOGLE', 'APPLE', 'KAKAO', 'NAVER', 'LINE']);
  });

  it('exposes slug + labelKey + short name', () => {
    const [kakao] = orderSocialProviders(['KAKAO'], 'ko');
    expect(kakao).toEqual({ code: 'KAKAO', slug: 'kakao', labelKey: 'auth.social.kakao', short: 'Kakao' });
  });
});

describe('socialBrand (WI-A1)', () => {
  it('covers every supported provider with opaque colors', async () => {
    const { socialBrand } = await import('./social');
    for (const code of ['GOOGLE', 'APPLE', 'KAKAO', 'NAVER', 'LINE']) {
      const b = socialBrand(code);
      expect(b.bg).toMatch(/^#[0-9A-F]{6}$/);
      expect(b.fg).toMatch(/^#[0-9A-F]{6}$/);
    }
  });

  it('falls back for unknown codes', async () => {
    const { socialBrand } = await import('./social');
    expect(socialBrand('WECHAT').bg).toBe('#2A2F3A');
  });
});
