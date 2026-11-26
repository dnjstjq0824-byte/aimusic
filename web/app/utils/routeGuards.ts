// Pure route decisions for app.global.ts (P1): layout, redirects, guards.
// The middleware stays a thin adapter over these; behavior is pinned here.
// NOTE: the en-XA dev locale rides the same rules as the shipped locales.

import type { Role } from '@/api/types';

export function layoutFor(path: string): string | false {
  if (path.startsWith('/studio')) return 'studio';
  if (path.startsWith('/business/console')) return 'business';
  if (path.startsWith('/admin')) return 'admin';
  if (path.startsWith('/embed')) return 'embed';
  if (path === '/store/player' || path === '/500') return false;
  if (
    path === '/login' || path.startsWith('/signup') || path === '/forgot-password' ||
    path === '/reset-password' || path === '/verify-email' || path.startsWith('/oauth/')
  ) {
    return 'auth';
  }
  return 'app';
}

// Entry screens that make no sense once signed in (R3: login + signup + forgot).
const GUEST_ONLY = new Set([
  'login',
  'signup',
  'signup-listener',
  'signup-creator',
  'signup-business',
  'forgot-password',
]);

const AUTH_REQUIRED = new Set([
  'library',
  'me-playlists',
  'checkout',
  'checkout-confirm',
  'account-index',
  'account-profile',
  'account-security',
  'account-billing',
  'account-language',
  'account-privacy',
  'account-connections',
]);

export interface RedirectRule {
  to: string;
  code: 301 | 302;
}

export function redirectFor(args: {
  path: string;
  name: string;
  queryTab?: unknown;
  queryToken?: unknown;
  fullPath: string;
}): RedirectRule | null {
  const { path, name, queryTab, queryToken, fullPath } = args;
  const q = fullPath.slice(path.length);
  // Trailing slash: single canonical without it (SEO §3).
  // Locale roots (/en/ …) keep their slash — they have no shorter form.
  const isLocaleRoot = /^\/(en|ja|zh-cn|th|en-XA)\/$/.test(path);
  if (path !== '/' && path.endsWith('/') && !isLocaleRoot) {
    return { to: `${path.slice(0, -1)}${q}`, code: 301 };
  }
  // Legacy tab URLs keep working (R3 split): single-URL 3-tab login is gone.
  if (name === 'login' && typeof queryTab === 'string') {
    if (queryTab === 'register') return { to: '/signup/listener', code: 301 };
    if (queryTab === 'partner') return { to: '/signup/creator', code: 301 };
  }
  // Tokenless reset is the forgot screen (R3 split).
  if (name === 'reset-password' && (typeof queryToken !== 'string' || queryToken === '')) {
    return { to: '/forgot-password', code: 301 };
  }
  // R6 split: the old combined library lives at /library now.
  if (path === '/my-library') {
    return { to: '/library', code: 301 };
  }
  // R-2 (WI-N6): the admin console ships ko/en only. Non-en locale
  // prefixes have no admin routes ([...slug] would 404); send them to
  // the en console instead of a dead page.
  {
    const m = /^\/(ja|th|zh-cn|en-XA)(\/admin(?:\/|$).*)$/.exec(path);
    if (m) {
      return { to: `/en${m[2]}${q}`, code: 302 };
    }
  }
  // R8 split: the disputes page lives at /rights-center now.
  if (path === '/disputes') {
    return { to: '/rights-center', code: 301 };
  }
  // R9/R10 split: the partner console moved to /studio/** (creator) and
  // /business/console/** (stores). Old bookmarks keep working (R-3, R-S).
  if (path === '/partner' || path.startsWith('/partner/')) {
    const rest = path === '/partner' ? '' : path.slice('/partner'.length);
    const map: Record<string, string> = {
      '': '/studio',
      '/upload': '/studio/upload',
      '/subscriptions': '/studio/proofs',
      '/stores': '/business/console/stores',
      '/generations': '/studio/create',
    };
    return { to: `${map[rest] ?? `/studio${rest}`}${q}`, code: 301 };
  }
  return null;
}

export type AuthDecision =
  | { kind: 'pass' }
  | { kind: 'guest-redirect' }
  | { kind: 'login' }
  | { kind: 'home'; toastKey: string };

export function authDecision(
  path: string,
  name: string,
  isAuthenticated: boolean,
  hasRole: (r: Role) => boolean,
  role: unknown,
): AuthDecision {
  // Already signed in: never show entry forms again; continue to the target.
  if (GUEST_ONLY.has(name) && isAuthenticated) return { kind: 'guest-redirect' };
  const needsAuth =
    AUTH_REQUIRED.has(name) ||
    path.startsWith('/studio') ||
    path.startsWith('/business/console') ||
    path.startsWith('/admin');
  if (needsAuth && !isAuthenticated) return { kind: 'login' };
  // R-3: listeners reach neither console; migrated accounts hold both grants.
  const guard = path.startsWith('/admin')
    ? { roles: ['ADMIN'], key: 'common.auth.needAdmin' }
    : path.startsWith('/studio')
      ? { roles: ['PARTNER', 'ADMIN'], key: 'common.auth.needPartner' }
      : path.startsWith('/business/console')
        ? { roles: ['BUSINESS', 'ADMIN'], key: 'common.auth.needBusiness' }
        : undefined;
  if (guard && role && !guard.roles.some((r) => hasRole(r as Role))) {
    return { kind: 'home', toastKey: guard.key };
  }
  return { kind: 'pass' };
}
