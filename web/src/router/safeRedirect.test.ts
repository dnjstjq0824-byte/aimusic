import { describe, expect, it } from 'vitest';
import { safeRedirect } from '@/router';

describe('safeRedirect', () => {
  it('keeps in-app absolute paths', () => {
    expect(safeRedirect('/explore')).toBe('/explore');
    expect(safeRedirect('/tracks/abc?x=1')).toBe('/tracks/abc?x=1');
  });

  it('rejects protocol-relative and external URLs', () => {
    expect(safeRedirect('//evil.com/phish')).toBe('/');
    expect(safeRedirect('https://evil.com')).toBe('/');
    expect(safeRedirect('javascript:alert(1)')).toBe('/');
  });

  it('rejects backslashes and non-strings', () => {
    expect(safeRedirect('/\\evil.com')).toBe('/');
    expect(safeRedirect('')).toBe('/');
    expect(safeRedirect(undefined)).toBe('/');
    expect(safeRedirect(null)).toBe('/');
    expect(safeRedirect(42)).toBe('/');
  });
});
