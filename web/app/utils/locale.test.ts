import { describe, expect, it } from 'vitest';
import { browserToLocale, detectBrowserLocale, isPrefixedPath, shouldApplyServerLocale } from './locale';

describe('shouldApplyServerLocale (A2 detection order 1)', () => {
  it('applies the server locale on unprefixed non-public screens', () => {
    expect(shouldApplyServerLocale('en', 'ko', '/account/language', 'account-language')).toBe('en');
    expect(shouldApplyServerLocale('zh-cn', 'ko', '/library', 'library')).toBe('zh-cn');
  });

  it('keeps the current locale when already matching or unsupported', () => {
    expect(shouldApplyServerLocale('ko', 'ko', '/library', 'library')).toBeNull();
    expect(shouldApplyServerLocale(null, 'ko', '/library', 'library')).toBeNull();
    expect(shouldApplyServerLocale('xx', 'ko', '/library', 'library')).toBeNull();
  });

  it('never overrides public pages or prefixed paths (URL wins)', () => {
    expect(shouldApplyServerLocale('en', 'ko', '/pricing', 'pricing')).toBeNull();
    expect(shouldApplyServerLocale('ko', 'en', '/en/pricing', 'pricing')).toBeNull();
    expect(shouldApplyServerLocale('ja', 'ko', '/ja', 'home')).toBeNull();
  });
});

describe('isPrefixedPath', () => {
  it('detects locale prefixes', () => {
    expect(isPrefixedPath('/en/pricing')).toBe(true);
    expect(isPrefixedPath('/zh-cn/explore')).toBe(true);
    expect(isPrefixedPath('/pricing')).toBe(false);
    expect(isPrefixedPath('/account/language')).toBe(false);
  });
});

describe('browserToLocale (R-2 detection order)', () => {
  it('maps direct language tags', () => {
    expect(browserToLocale('ko-KR')).toBe('ko');
    expect(browserToLocale('ja')).toBe('ja');
    expect(browserToLocale('th-TH')).toBe('th');
    expect(browserToLocale('en-US')).toBe('en');
  });

  it('maps simplified chinese to zh-cn, traditional to en', () => {
    expect(browserToLocale('zh-CN')).toBe('zh-cn');
    expect(browserToLocale('zh-Hans')).toBe('zh-cn');
    expect(browserToLocale('zh-TW')).toBe('en');
    expect(browserToLocale('zh-Hant-HK')).toBe('en');
  });

  it('falls back to en for unsupported or missing tags', () => {
    expect(browserToLocale('de-DE')).toBe('en');
    expect(browserToLocale('')).toBe('en');
    expect(browserToLocale(null)).toBe('en');
  });

  it('detectBrowserLocale picks the first supported tag', () => {
    expect(detectBrowserLocale(['de-DE', 'ja-JP', 'en'])).toBe('ja');
    expect(detectBrowserLocale(['zh-TW', 'ko-KR'])).toBe('en');
    expect(detectBrowserLocale([])).toBe('en');
    expect(detectBrowserLocale(undefined)).toBe('en');
  });
});
