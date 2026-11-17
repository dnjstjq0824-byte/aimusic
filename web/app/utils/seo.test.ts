import { describe, expect, it } from 'vitest';
import {
  SITE_URL,
  breadcrumbLd,
  canonicalFor,
  canonicalPlaylistPath,
  canonicalTrackPath,
  creatorTitle,
  describeCreator,
  describeLanding,
  describePlaylist,
  describeTrack,
  detailParam,
  embedPlaylistUrl,
  embedTrackUrl,
  guideTitle,
  hreflangLinks,
  landingTitle,
  localizedPath,
  musicPlaylistLd,
  musicRecordingLd,
  needsCjkThaiFont,
  normalizeSeoLocale,
  oembedDiscoveryUrl,
  parseSlugParam,
  personLd,
  playlistLink,
  playlistTitle,
  shareLinks,
  stripLocalePrefix,
  toISO8601Duration,
  trackLink,
  trackTitle,
  type TranslateFn,
} from './seo';
import koSeo from '../../i18n/locales/ko/seo.json';
import enSeo from '../../i18n/locales/en/seo.json';
import jaSeo from '../../i18n/locales/ja/seo.json';
import zhSeo from '../../i18n/locales/zh-CN/seo.json';
import thSeo from '../../i18n/locales/th/seo.json';

type Dict = Record<string, unknown>;
const DICTS: Record<string, Dict> = {
  ko: koSeo.seo as unknown as Dict,
  en: enSeo.seo as unknown as Dict,
  ja: jaSeo.seo as unknown as Dict,
  'zh-cn': zhSeo.seo as unknown as Dict,
  th: thSeo.seo as unknown as Dict,
};

function makeT(locale: string): TranslateFn {
  const dict = DICTS[locale] ?? DICTS.en;
  return (key: string, params?: Record<string, string | number>) => {
    const path = key.replace(/^seo\./, '').split('.');
    let cur: unknown = dict;
    for (const seg of path) cur = (cur as Dict)[seg];
    let out = String(cur ?? key);
    for (const [k, v] of Object.entries(params ?? {})) out = out.replaceAll(`{${k}}`, String(v));
    return out.replaceAll("{'|'}", '|');
  };
}

const tagName = (key: string) => ({ cafe: 'Cafe', jazz: 'Jazz', focus: 'Focus' })[key] ?? key;

describe('seo locales', () => {
  it('normalizes locales', () => {
    expect(normalizeSeoLocale('EN')).toBe('en');
    expect(normalizeSeoLocale('zh-cn')).toBe('zh-cn');
    expect(normalizeSeoLocale('xx')).toBe('ko');
    expect(normalizeSeoLocale(null)).toBe('ko');
  });
  it('localizes paths', () => {
    expect(localizedPath('/tracks/a', 'ko')).toBe('/tracks/a');
    expect(localizedPath('/tracks/a', 'en')).toBe('/en/tracks/a');
    expect(localizedPath('/', 'th')).toBe('/th/');
    expect(stripLocalePrefix('/ja/explore')).toBe('/explore');
    expect(stripLocalePrefix('/explore')).toBe('/explore');
  });
  it('builds canonical + hreflang', () => {
    expect(canonicalFor('/tracks/a', 'en')).toBe(`${SITE_URL}/en/tracks/a`);
    const links = hreflangLinks('/en/tracks/a?x=1');
    expect(links).toHaveLength(6);
    expect(links.map((l) => l.hreflang)).toEqual(['ko', 'en', 'ja', 'zh-CN', 'th', 'x-default']);
    expect(links[0]?.href).toBe(`${SITE_URL}/tracks/a`);
    expect(links[5]?.href).toBe(`${SITE_URL}/en/tracks/a`);
  });
});

describe('seo slugs', () => {
  it('parses params', () => {
    expect(parseSlugParam('123e4567-e89b-12d3-a456-426614174000').kind).toBe('uuid');
    expect(parseSlugParam('Ab3xK9pQ')).toEqual({ slug: '', shortId: 'Ab3xK9pQ', kind: 'short' });
    expect(parseSlugParam('night-drive-Ab3xK9pQ')).toEqual({ slug: 'night-drive', shortId: 'Ab3xK9pQ', kind: 'slugshort' });
    expect(parseSlugParam('nope').kind).toBe('invalid');
  });
  it('builds canonical paths and links', () => {
    expect(canonicalTrackPath('night-drive', 'Ab3xK9pQ')).toBe('/tracks/night-drive-Ab3xK9pQ');
    expect(canonicalPlaylistPath('cafe', 'Xy12Zz99')).toBe('/playlists/cafe-Xy12Zz99');
    expect(trackLink({ id: 'u', slug: 's', shortId: 'Ab3xK9pQ' })).toBe('/tracks/s-Ab3xK9pQ');
    expect(trackLink({ id: 'u' })).toBe('/tracks/u');
    expect(playlistLink({ id: 'u', slug: 's', shortId: 'Ab3xK9pQ' })).toBe('/playlists/s-Ab3xK9pQ');
    expect(detailParam({ id: 'u', slug: 's', shortId: 'Ab3xK9pQ' })).toBe('s-Ab3xK9pQ');
    expect(detailParam({ id: 'u' })).toBe('u');
  });
  it('detects CJK/Thai scripts', () => {
    expect(needsCjkThaiFont('Midnight Drive')).toBe(false);
    expect(needsCjkThaiFont('밤 산책')).toBe(true);
    expect(needsCjkThaiFont('เพลงช้า')).toBe(true);
    expect(needsCjkThaiFont('カフェ')).toBe(true);
    expect(needsCjkThaiFont('咖啡馆')).toBe(true);
  });
  it('formats durations', () => {
    expect(toISO8601Duration(227000)).toBe('PT3M47S');
    expect(toISO8601Duration(60000)).toBe('PT1M');
    expect(toISO8601Duration(null)).toBeNull();
  });
});

describe('seo titles', () => {
  it('tracks', () => {
    expect(trackTitle(makeT('ko'), '밤 산책', 'DJ Kim')).toBe('밤 산책 – DJ Kim | AVIBE');
    expect(trackTitle(makeT('en'), 'T', '')).toBe('T | AVIBE');
  });
  it('playlists per locale', () => {
    expect(playlistTitle(makeT('ko'), 'Cafe', 'Mina')).toContain('Mina의 플레이리스트');
    expect(playlistTitle(makeT('en'), 'Cafe', 'Mina')).toContain('playlist by Mina');
    expect(playlistTitle(makeT('en'), 'Cafe', '')).toBe('Cafe | AVIBE');
  });
  it('creators/landings/guides', () => {
    expect(creatorTitle(makeT('th'), 'Mina')).toContain('ครีเอเตอร์');
    expect(landingTitle(makeT('ko'), '카페')).toBe('카페 AI 음악 – 상업적 이용권 확인 | AVIBE');
    expect(landingTitle(makeT('en'), 'Cafe')).toContain('verified commercial use');
    expect(guideTitle(makeT('ko'), '매장 음악')).toBe('매장 음악 | AVIBE 가이드');
  });
});

describe('seo descriptions', () => {
  const track = {
    title: 'Night Drive', creatorName: 'Mina', genre: 'jazz',
    moodTags: ['cafe', 'focus'], bpm: 120, musicalKey: 'A minor', durationMs: 227000,
  };
  it('differs per locale (§3.1)', () => {
    const texts = (['ko', 'en', 'ja', 'zh-cn', 'th'] as const).map((l) => describeTrack(makeT(l), track, tagName));
    expect(new Set(texts).size).toBe(5);
    for (const t of texts) {
      expect(t).toContain('Night Drive');
      expect(t.length).toBeGreaterThan(60);
    }
  });
  it('carries facts', () => {
    expect(describeTrack(makeT('en'), track, tagName)).toContain('120 BPM');
    expect(describeTrack(makeT('ko'), track, tagName)).toContain('BPM 120');
  });
  it('degrades without optionals', () => {
    const bare = describeTrack(makeT('en'), { title: 'T' }, tagName);
    expect(bare).toContain('AI track');
    expect(bare).not.toContain('{');
  });
  it('covers playlist/creator/landing', () => {
    expect(describePlaylist(makeT('ko'), { title: 'C', ownerName: 'M' } as never, 12)).toContain('12곡');
    expect(describeCreator(makeT('en'), 'M', 5)).toContain('5 AI tracks');
    expect(describeLanding(makeT('th'), 'mood', 'Cafe', 12)).toContain('12');
    expect(describeLanding(makeT('en'), 'use', 'Cafe', 12)).toContain('use-case');
  });
});

describe('seo share/embed', () => {
  it('share links carry utm', () => {
    const s = shareLinks('https://avibemusic.com/tracks/a', 'T');
    expect(s.copy).toContain('utm_source=share');
    expect(s.line).toContain('lineit');
    expect(s.x).toContain('intent/tweet');
    expect(s.facebook).toContain('sharer');
    expect(s.kakao).toContain('sharer.kakao.com');
  });
  it('embed + oembed urls', () => {
    expect(embedTrackUrl('Ab3xK9pQ', 'th')).toBe(`${SITE_URL}/embed/track/Ab3xK9pQ?lang=th`);
    expect(embedPlaylistUrl('Xy12Zz99')).toBe(`${SITE_URL}/embed/playlist/Xy12Zz99`);
    expect(oembedDiscoveryUrl(`${SITE_URL}/tracks/a`)).toContain('/oembed?url=');
  });
});

describe('seo json-ld', () => {
  it('music recording', () => {
    const ld = musicRecordingLd({
      track: { title: 'T', creatorName: 'C', genre: 'jazz', durationMs: 60000, isrc: 'KRAA12500001' } as never,
      pageUrl: `${SITE_URL}/tracks/t`,
      creatorUrl: `${SITE_URL}/creators/c`,
      certNumber: 'AVB-1',
      datePublished: '2026-11-01',
    });
    expect(ld['@type']).toBe('MusicRecording');
    expect(ld.isrcCode).toBe('KRAA12500001');
    expect(ld.license).toContain('/certificate/AVB-1');
  });
  it('playlist caps at 50', () => {
    const tracks = Array.from({ length: 60 }, (_, i) => ({ title: `T${i}`, url: `${SITE_URL}/t${i}` }));
    const ld = musicPlaylistLd({ playlist: { title: 'P' } as never, pageUrl: `${SITE_URL}/p`, tracks, trackCount: 60 });
    expect(ld.numTracks).toBe(60);
    expect((ld.track as unknown[])).toHaveLength(50);
  });
  it('person/breadcrumb', () => {
    expect(personLd('C', `${SITE_URL}/c`, [])['@type']).toContain('ProfilePage');
    const bc = breadcrumbLd([{ name: 'H', url: SITE_URL }]);
    expect((bc.itemListElement as unknown[])).toHaveLength(1);
  });
});
