// Shared SEO builders (WI-S1/S2/S4): canonical + hreflang, localized
// title/description templates, language-specific track descriptions (§3.1),
// JSON-LD payloads, share links, and slug-param parsing. Pure (no Nuxt
// imports) so vitest covers every branch.
import type { Track, Playlist } from '@/api/types';

export const SITE_URL = 'https://avibemusic.com';
export const SEO_LOCALES = ['ko', 'en', 'ja', 'zh-cn', 'th'] as const;
export type SeoLocale = (typeof SEO_LOCALES)[number];

export function normalizeSeoLocale(input: string | null | undefined): SeoLocale {
  const v = String(input ?? '').toLowerCase();
  return (SEO_LOCALES as readonly string[]).includes(v) ? (v as SeoLocale) : 'ko';
}

/** Prefix an unprefixed page path for a locale (ko has none). */
export function localizedPath(path: string, locale: string): string {
  const p = path.startsWith('/') ? path : `/${path}`;
  const l = normalizeSeoLocale(locale);
  if (l === 'ko') return p;
  if (p === '/') return `/${l}/`;
  return `/${l}${p}`;
}

/** Strip one locale prefix from a page path. */
export function stripLocalePrefix(path: string): string {
  for (const l of SEO_LOCALES) {
    if (l === 'ko') continue;
    if (path === `/${l}`) return '/';
    if (path.startsWith(`/${l}/`)) return path.slice(l.length + 1);
  }
  return path;
}

/** Absolute canonical URL for a page path + locale. */
export function canonicalFor(path: string, locale: string): string {
  return `${SITE_URL}${localizedPath(path, locale)}`;
}

export interface HreflangLink {
  rel: 'alternate';
  hreflang: string;
  href: string;
}

/** 5 hreflang alternates + x-default (spec: x-default = /en/…). */
export function hreflangLinks(path: string): HreflangLink[] {
  const clean = stripLocalePrefix(path.split('?')[0] || '/');
  const links = SEO_LOCALES.map((l) => ({
    rel: 'alternate' as const,
    hreflang: l === 'zh-cn' ? 'zh-CN' : l,
    href: canonicalFor(clean, l),
  }));
  links.push({ rel: 'alternate', hreflang: 'x-default', href: canonicalFor(clean, 'en') });
  return links;
}

/** True when text needs CJK/Thai glyphs (OG textless-fallback trigger). */
export function needsCjkThaiFont(text: string): boolean {
  return /[\u0E00-\u0E7F\u3040-\u30FF\u3400-\u4DBF\u4E00-\u9FFF\uAC00-\uD7AF\uFF00-\uFFEF]/.test(text);
}

/** Split a detail param into slug + short ID (mirrors domain.SplitSlugID). */
export function parseSlugParam(param: string): { slug: string; shortId: string; kind: 'uuid' | 'short' | 'slugshort' | 'invalid' } {
  const p = param.trim();
  if (/^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$/.test(p)) {
    return { slug: '', shortId: '', kind: 'uuid' };
  }
  if (/^[0-9A-Za-z]{8}$/.test(p)) return { slug: '', shortId: p, kind: 'short' };
  const i = p.lastIndexOf('-');
  if (i > 0 && i + 1 < p.length && /^[0-9A-Za-z]{8}$/.test(p.slice(i + 1))) {
    return { slug: p.slice(0, i), shortId: p.slice(i + 1), kind: 'slugshort' };
  }
  return { slug: '', shortId: '', kind: 'invalid' };
}

export function canonicalTrackPath(slug: string, shortId: string): string {
  return `/tracks/${slug}-${shortId}`;
}

export function canonicalPlaylistPath(slug: string, shortId: string): string {
  return `/playlists/${slug}-${shortId}`;
}

/** Canonical detail link with uuid fallback for legacy objects. */
export function trackLink(t: { id: string; slug?: string; shortId?: string }): string {
  return t.slug && t.shortId ? canonicalTrackPath(t.slug, t.shortId) : `/tracks/${t.id}`;
}

export function playlistLink(p: { id: string; slug?: string; shortId?: string }): string {
  return p.slug && p.shortId ? canonicalPlaylistPath(p.slug, p.shortId) : `/playlists/${p.id}`;
}

/** Named-route `id` param: slug-shortId when known, uuid otherwise (301s). */
export function detailParam(o: { id: string; slug?: string; shortId?: string }): string {
  return o.slug && o.shortId ? `${o.slug}-${o.shortId}` : o.id;
}

export function formatDuration(ms: number | null | undefined): string {
  if (!ms || ms <= 0) return '';
  const total = Math.round(ms / 1000);
  return `${Math.floor(total / 60)}:${String(total % 60).padStart(2, '0')}`;
}

/** ISO 8601 duration for MusicRecording (PT3M47S). */
export function toISO8601Duration(ms: number | null | undefined): string | null {
  if (!ms || ms <= 0) return null;
  const total = Math.round(ms / 1000);
  const m = Math.floor(total / 60);
  const s = total % 60;
  if (m === 0) return `PT${s}S`;
  if (s === 0) return `PT${m}M`;
  return `PT${m}M${s}S`;
}

// --- Localized title templates (spec §5, seo.json) ---

export type TranslateFn = (key: string, params?: Record<string, string | number>) => string;

export function trackTitle(t: TranslateFn, title: string, creator: string): string {
  return creator ? t('seo.trackTitle', { title, creator }) : t('seo.trackTitleBare', { title });
}

export function playlistTitle(t: TranslateFn, title: string, owner: string): string {
  return owner ? t('seo.playlistTitle', { title, owner }) : t('seo.playlistTitleBare', { title });
}

export function creatorTitle(t: TranslateFn, displayName: string): string {
  return t('seo.creatorTitle', { name: displayName });
}

export type LandingKind = 'mood' | 'genre' | 'use';

export function landingTitle(t: TranslateFn, tagLabel: string): string {
  return t('seo.landingTitle', { label: tagLabel });
}

export function guideTitle(t: TranslateFn, headline: string): string {
  return t('seo.guideTitle', { headline });
}

// --- Language-specific descriptions (spec §3.1: no cross-locale dupes) ---

export type TagNameFn = (key: string) => string;

export interface TrackDescribeInput {
  title: string;
  creatorName?: string;
  genre?: string | null;
  moodTags?: string[];
  bpm?: number | null;
  musicalKey?: string | null;
  durationMs?: number | null;
}

/**
 * Build the per-locale track description from server facts (mood/genre/BPM/
 * key/duration). Sentence frames live in seo.json so each locale reads
 * natively and the 5 pages are never shell-identical (§3.1).
 */
export function describeTrack(tr: TranslateFn, input: TrackDescribeInput, tagName: TagNameFn): string {
  const t = tr;
  const moods = (input.moodTags ?? []).map(tagName).filter(Boolean).slice(0, 2);
  const genre = tagName(input.genre ?? '');
  const parts: string[] = [];
  if (moods.length > 0) parts.push(t('seo.tdMood', { moods: moods.join(' ') }));
  if (genre) parts.push(t('seo.tdGenre', { genre }));
  const head = parts.join(t('seo.tdHeadJoin'));
  const music = head ? t('seo.tdMusicHead', { head }) : t('seo.tdMusicBare');
  const facts: string[] = [];
  if (input.bpm) facts.push(t('seo.tdBpm', { bpm: String(Math.round(input.bpm * 10) / 10) }));
  if (input.musicalKey) facts.push(t('seo.tdKey', { key: input.musicalKey }));
  const dur = formatDuration(input.durationMs);
  if (dur) facts.push(t('seo.tdDur', { dur }));
  const factsPart = facts.length > 0 ? t('seo.tdFactsWrap', { facts: facts.join(t('seo.tdFactJoin')) }) : '';
  const creatorPart = input.creatorName ? t('seo.tdCreatorPart', { creator: input.creatorName }) : '';
  return `${input.title}${creatorPart}${t('seo.tdSep')}${music}${factsPart}${t('seo.tdTail')}`;
}

export function describePlaylist(t: TranslateFn, p: Playlist, count: number): string {
  return p.ownerName
    ? t('seo.pdDesc', { title: p.title, owner: p.ownerName, count })
    : t('seo.pdDescBare', { title: p.title, count });
}

export function describeCreator(t: TranslateFn, displayName: string, trackCount: number): string {
  return t('seo.cdDesc', { name: displayName, count: trackCount });
}

export function describeLanding(t: TranslateFn, kind: LandingKind, tagLabel: string, count: number): string {
  let kindWord: string;
  switch (kind) {
    case 'genre':
      kindWord = t('seo.ldKindGenre');
      break;
    case 'use':
      kindWord = t('seo.ldKindUse');
      break;
    default:
      kindWord = t('seo.ldKindMood');
      break;
  }
  return t('seo.ldDesc', { label: tagLabel, kindWord, count });
}

// --- Share links (WI-S4: link + Kakao + LINE + X + Facebook, utm) ---

export interface ShareTargets {
  copy: string;
  kakao: string;
  line: string;
  x: string;
  facebook: string;
}

export function shareLinks(pageUrl: string, title: string): ShareTargets {
  const withUtm = (channel: string) => {
    const sep = pageUrl.includes('?') ? '&' : '?';
    return `${pageUrl}${sep}utm_source=share&utm_medium=${channel}`;
  };
  const enc = encodeURIComponent;
  return {
    copy: withUtm('copy'),
    kakao: `https://sharer.kakao.com/talk/friends/picker/shortlink?link=${enc(withUtm('kakao'))}`,
    line: `https://social-plugins.line.me/lineit/share?url=${enc(withUtm('line'))}&text=${enc(title)}`,
    x: `https://twitter.com/intent/tweet?url=${enc(withUtm('x'))}&text=${enc(title)}`,
    facebook: `https://www.facebook.com/sharer/sharer.php?u=${enc(withUtm('facebook'))}`,
  };
}

export function embedTrackUrl(shortId: string, locale = ''): string {
  const q = locale && locale !== 'ko' ? `?lang=${locale}` : '';
  return `${SITE_URL}/embed/track/${shortId}${q}`;
}

export function embedPlaylistUrl(shortId: string, locale = ''): string {
  const q = locale && locale !== 'ko' ? `?lang=${locale}` : '';
  return `${SITE_URL}/embed/playlist/${shortId}${q}`;
}

export function oembedDiscoveryUrl(pageUrl: string): string {
  return `${SITE_URL}/oembed?url=${encodeURIComponent(pageUrl)}`;
}

// --- JSON-LD payloads (fed to nuxt-schema-org useSchemaOrg) ---

export function organizationLd(): Record<string, unknown> {
  return {
    '@type': 'Organization',
    name: 'AVIBE',
    url: SITE_URL,
    logo: `${SITE_URL}/favicon.svg`,
  };
}

export function websiteLd(): Record<string, unknown> {
  return {
    '@type': 'WebSite',
    name: 'AVIBE',
    url: SITE_URL,
    inLanguage: [...SEO_LOCALES],
  };
}

export interface RecordingLdInput {
  track: Track;
  pageUrl: string;
  creatorUrl: string | null;
  certNumber: string | null;
  datePublished: string;
}

export function musicRecordingLd(input: RecordingLdInput): Record<string, unknown> {
  const { track, pageUrl, creatorUrl, certNumber, datePublished } = input;
  const byArtist: Record<string, unknown> = {
    '@type': 'MusicGroup',
    name: track.creatorName || 'AVIBE Creator',
  };
  if (creatorUrl) byArtist.url = creatorUrl;
  const out: Record<string, unknown> = {
    '@type': 'MusicRecording',
    name: track.title,
    url: pageUrl,
    duration: toISO8601Duration(track.durationMs) ?? undefined,
    byArtist,
    genre: track.genre ?? undefined,
    datePublished,
  };
  if (track.isrc) out.isrcCode = track.isrc;
  if (certNumber) out.license = `${SITE_URL}/certificate/${certNumber}`;
  return out;
}

export interface PlaylistLdInput {
  playlist: Playlist;
  pageUrl: string;
  tracks: { title: string; url: string; durationMs?: number | null }[];
  trackCount: number;
}

export function musicPlaylistLd(input: PlaylistLdInput): Record<string, unknown> {
  return {
    '@type': 'MusicPlaylist',
    name: input.playlist.title,
    url: input.pageUrl,
    numTracks: input.trackCount,
    track: input.tracks.slice(0, 50).map((t) => ({
      '@type': 'MusicRecording',
      name: t.title,
      url: t.url,
      duration: toISO8601Duration(t.durationMs ?? null) ?? undefined,
    })),
  };
}

export function personLd(displayName: string, pageUrl: string, topTrackUrls: string[]): Record<string, unknown> {
  return {
    '@type': ['Person', 'MusicGroup', 'ProfilePage'],
    name: displayName,
    url: pageUrl,
    mainEntity: topTrackUrls.slice(0, 10).map((u) => ({ '@type': 'MusicRecording', url: u })),
  };
}

export function breadcrumbLd(items: { name: string; url: string }[]): Record<string, unknown> {
  return {
    '@type': 'BreadcrumbList',
    itemListElement: items.map((it, i) => ({
      '@type': 'ListItem',
      position: i + 1,
      name: it.name,
      item: it.url,
    })),
  };
}

export function articleLd(headline: string, pageUrl: string, dateModified: string): Record<string, unknown> {
  return {
    '@type': 'Article',
    headline,
    url: pageUrl,
    dateModified,
    author: { '@type': 'Organization', name: 'AVIBE', url: SITE_URL },
  };
}

export function faqLd(pairs: { q: string; a: string }[]): Record<string, unknown> {
  return {
    '@type': 'FAQPage',
    mainEntity: pairs.map((p) => ({
      '@type': 'Question',
      name: p.q,
      acceptedAnswer: { '@type': 'Answer', text: p.a },
    })),
  };
}
