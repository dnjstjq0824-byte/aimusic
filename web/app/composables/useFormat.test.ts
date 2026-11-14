import { describe, expect, it, vi, beforeEach } from 'vitest';

const localeRef = { value: 'ko' };
vi.mock('vue-i18n', () => ({
  useI18n: () => ({ locale: localeRef }),
}));

import { useFormat } from './useFormat';

describe('useFormat', () => {
  beforeEach(() => {
    localeRef.value = 'ko';
  });

  it('formats dates per locale (thai uses the buddhist year)', () => {
    const { date } = useFormat();
    const d = '2026-03-15T00:00:00Z';
    localeRef.value = 'ko';
    expect(date(d)).toContain('2026');
    localeRef.value = 'th';
    expect(date(d)).toContain('2569');
  });

  it('formats numbers and KRW per locale', () => {
    const { number, currencyKRW } = useFormat();
    localeRef.value = 'en';
    expect(number(1234567)).toBe('1,234,567');
    expect(currencyKRW(5900)).toContain('5,900');
    localeRef.value = 'ko';
    expect(currencyKRW(5900)).toContain('5,900');
  });

  it('returns empty string for invalid dates', () => {
    const { date } = useFormat();
    expect(date('not-a-date')).toBe('');
  });
});
