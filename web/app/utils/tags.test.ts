import { describe, expect, it } from 'vitest';
import { tagDisplayKey, tagDisplayName } from './tags';

const t = (key: string) => `<${key}>`;
const te = (key: string) => key === 'tags.focus' || key === 'tags.electronic';

describe('tag display (R5)', () => {
  it('normalizes keys case-insensitively', () => {
    expect(tagDisplayKey('Focus')).toBe('focus');
    expect(tagDisplayKey('  ELECTRONIC ')).toBe('electronic');
    expect(tagDisplayKey('')).toBeNull();
    expect(tagDisplayKey(null)).toBeNull();
  });

  it('translates known keys and passes the rest through', () => {
    expect(tagDisplayName('Focus', t, te)).toBe('<tags.focus>');
    expect(tagDisplayName('Polka', t, te)).toBe('Polka');
    expect(tagDisplayName('', t, te)).toBe('');
  });
});
