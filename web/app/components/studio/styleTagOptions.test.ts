import { describe, expect, it } from 'vitest';
import { MAX_STYLE_TAGS, STYLE_TAG_GROUPS, toggleTag } from './styleTagOptions';

describe('toggleTag (WI-G5)', () => {
  it('adds a tag', () => {
    expect(toggleTag([], 'chill')).toEqual(['chill']);
  });

  it('removes a selected tag', () => {
    expect(toggleTag(['chill', 'jazz'], 'chill')).toEqual(['jazz']);
  });

  it('caps the selection at max', () => {
    const full = ['a', 'b', 'c', 'd', 'e', 'f'];
    expect(toggleTag(full, 'g')).toEqual(full);
    expect(toggleTag(full, 'g', 8)).toEqual([...full, 'g']);
  });

  it('does not mutate the input', () => {
    const input = ['chill'];
    toggleTag(input, 'jazz');
    expect(input).toEqual(['chill']);
  });
});

describe('STYLE_TAG_GROUPS (WI-G5)', () => {
  it('exposes mood/genre/use groups with tag keys', () => {
    expect(STYLE_TAG_GROUPS.map((g) => g.id)).toEqual(['mood', 'genre', 'use']);
    for (const g of STYLE_TAG_GROUPS) {
      expect(g.keys.length).toBeGreaterThan(0);
    }
  });

  it('keeps the cap positive', () => {
    expect(MAX_STYLE_TAGS).toBeGreaterThan(0);
  });
});
