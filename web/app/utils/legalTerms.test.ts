import { describe, expect, it } from 'vitest';
import { legalTermTitleKey, resolveTermTitle } from './legalTerms';

describe('legalTermTitleKey', () => {
  it('maps known term codes to locale title keys', () => {
    expect(legalTermTitleKey('TOS')).toBe('legal.docTOS');
    expect(legalTermTitleKey('PRIVACY')).toBe('legal.docPRIVACY');
    expect(legalTermTitleKey('COPYRIGHT_POLICY')).toBe('legal.docCOPYRIGHT');
    expect(legalTermTitleKey('PARTNER_CONTRACT')).toBe('legal.docPARTNER');
  });

  it('returns null for unknown codes', () => {
    expect(legalTermTitleKey('NOPE')).toBeNull();
    expect(legalTermTitleKey('')).toBeNull();
  });
});

describe('resolveTermTitle', () => {
  const t = (key: string): string => `<${key}>`;
  it('prefers the translated title when the locale ships it', () => {
    expect(resolveTermTitle('TOS', 'title-ko', () => true, t)).toBe('<legal.docTOS>');
  });
  it('falls back to the server Korean title otherwise', () => {
    expect(resolveTermTitle('TOS', 'title-ko', () => false, t)).toBe('title-ko');
    expect(resolveTermTitle('NOPE', 'title-ko', () => true, t)).toBe('title-ko');
  });
});
