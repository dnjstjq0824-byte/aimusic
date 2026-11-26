import { describe, expect, it } from 'vitest';
import ko from '../../content/site.ko.json';
import en from '../../content/site.en.json';
import ja from '../../content/site.ja.json';
import zhCn from '../../content/site.zh-cn.json';
import th from '../../content/site.th.json';
import { MODEL_METAS, modelBySlug, modelSlugForProvider } from './models';

const flatKeys = (o: unknown, p = ''): string[] => {
  if (Array.isArray(o)) return [`${p}[${o.length > 0 ? 'n' : '0'}]`];
  if (o && typeof o === 'object') {
    return Object.entries(o as Record<string, unknown>).flatMap(([k, v]) => flatKeys(v, p ? `${p}.${k}` : k));
  }
  return [p];
};

describe('site content', () => {
  const base = flatKeys(ko).sort();
  it.each([['en', en], ['ja', ja], ['zh-cn', zhCn], ['th', th]])('%s has the same shape as ko', (_name, data) => {
    expect(flatKeys(data).sort()).toEqual(base);
  });

  it('help categories keep the same ids and question counts in every locale', () => {
    const shape = (d: typeof ko) => d.help.categories.map((c) => `${c.id}:${c.items.length}`);
    for (const d of [en, ja, zhCn, th]) expect(shape(d as typeof ko)).toEqual(shape(ko));
  });

  it('every model slug has localized content', () => {
    for (const d of [ko, en, ja, zhCn, th]) {
      for (const m of MODEL_METAS) expect((d.models.items as Record<string, unknown>)[m.slug]).toBeTruthy();
    }
  });
});

describe('models', () => {
  it('maps engine provider codes to slugs', () => {
    expect(modelSlugForProvider('AVIBE')).toBe('avibe');
    expect(modelSlugForProvider('STABLE_AUDIO')).toBe('stable-audio');
    expect(modelSlugForProvider('UNKNOWN')).toBeNull();
    expect(modelSlugForProvider(null)).toBeNull();
  });
  it('finds models by slug', () => {
    expect(modelBySlug('suno')?.code).toBe('SUNO');
    expect(modelBySlug('nope')).toBeUndefined();
  });
});
