import { describe, expect, it } from 'vitest';
import { authDecision, layoutFor, redirectFor } from './routeGuards';

describe('layoutFor', () => {
  it('routes consoles and special shells', () => {
    expect(layoutFor('/studio/upload')).toBe('studio');
    expect(layoutFor('/business/console/stores')).toBe('business');
    expect(layoutFor('/admin/users')).toBe('admin');
    expect(layoutFor('/embed/track/x')).toBe('embed');
    expect(layoutFor('/store/player')).toBe(false);
    expect(layoutFor('/500')).toBe(false);
    expect(layoutFor('/login')).toBe('auth');
    expect(layoutFor('/signup/listener')).toBe('auth');
    expect(layoutFor('/forgot-password')).toBe('auth');
    expect(layoutFor('/reset-password')).toBe('auth');
    expect(layoutFor('/verify-email')).toBe('auth');
    expect(layoutFor('/oauth/callback')).toBe('auth');
    expect(layoutFor('/explore')).toBe('app');
    expect(layoutFor('/')).toBe('app');
  });
});

describe('redirectFor', () => {
  const base = { name: 'x', queryTab: undefined as unknown, queryToken: undefined as unknown, fullPath: '' };
  it('strips trailing slashes preserving the query', () => {
    expect(redirectFor({ ...base, path: '/explore/', fullPath: '/explore/?q=a' })).toEqual({
      to: '/explore?q=a',
      code: 301,
    });
    expect(redirectFor({ ...base, path: '/', fullPath: '/' })).toBeNull();
  });
  it('keeps locale roots slashed', () => {
    for (const loc of ['en', 'ja', 'zh-cn', 'th', 'en-XA']) {
      expect(redirectFor({ ...base, path: `/${loc}/`, fullPath: `/${loc}/` })).toBeNull();
    }
  });
  it('maps legacy login tabs and tokenless resets', () => {
    expect(redirectFor({ path: '/login', name: 'login', queryTab: 'register', queryToken: undefined, fullPath: '/login?tab=register' })).toEqual({
      to: '/signup/listener',
      code: 301,
    });
    expect(redirectFor({ path: '/login', name: 'login', queryTab: 'partner', queryToken: undefined, fullPath: '/login' })).toEqual({
      to: '/signup/creator',
      code: 301,
    });
    expect(redirectFor({ path: '/reset-password', name: 'reset-password', queryTab: undefined, queryToken: '', fullPath: '/reset-password' })).toEqual({
      to: '/forgot-password',
      code: 301,
    });
    expect(
      redirectFor({ path: '/reset-password', name: 'reset-password', queryTab: undefined, queryToken: 'tok', fullPath: '/reset-password?token=tok' }),
    ).toBeNull();
  });
  it('maps moved pages and non-en admin consoles', () => {
    expect(redirectFor({ ...base, path: '/my-library', fullPath: '/my-library' })).toEqual({ to: '/library', code: 301 });
    expect(redirectFor({ ...base, path: '/disputes', fullPath: '/disputes' })).toEqual({ to: '/rights-center', code: 301 });
    expect(redirectFor({ ...base, path: '/ja/admin/users', fullPath: '/ja/admin/users?p=2' })).toEqual({
      to: '/en/admin/users?p=2',
      code: 302,
    });
    expect(redirectFor({ ...base, path: '/en/admin/users', fullPath: '/en/admin/users' })).toBeNull();
  });
  it('maps the old partner console', () => {
    expect(redirectFor({ ...base, path: '/partner', fullPath: '/partner' })).toEqual({ to: '/studio', code: 301 });
    expect(redirectFor({ ...base, path: '/partner/upload', fullPath: '/partner/upload' })).toEqual({ to: '/studio/upload', code: 301 });
    expect(redirectFor({ ...base, path: '/partner/stores', fullPath: '/partner/stores' })).toEqual({
      to: '/business/console/stores',
      code: 301,
    });
    expect(redirectFor({ ...base, path: '/partner/xyz', fullPath: '/partner/xyz' })).toEqual({ to: '/studio/xyz', code: 301 });
  });
});

describe('authDecision', () => {
  const roles =
    (mine: string[]) =>
    (r: string): boolean =>
      mine.includes(r);
  it('bounces signed-in users off entry forms', () => {
    expect(authDecision('/login', 'login', true, roles(['USER']), 'USER')).toEqual({ kind: 'guest-redirect' });
    expect(authDecision('/login', 'login', false, roles([]), null)).toEqual({ kind: 'pass' });
  });
  it('sends guests to login for guarded routes', () => {
    expect(authDecision('/library', 'library', false, roles([]), null)).toEqual({ kind: 'login' });
    expect(authDecision('/account', 'account-index', false, roles([]), null)).toEqual({ kind: 'login' });
    expect(authDecision('/studio', 'studio', false, roles([]), null)).toEqual({ kind: 'login' });
    expect(authDecision('/explore', 'explore', false, roles([]), null)).toEqual({ kind: 'pass' });
    expect(authDecision('/library', 'library', true, roles(['USER']), 'USER')).toEqual({ kind: 'pass' });
  });
  it('enforces console roles', () => {
    expect(authDecision('/admin', 'admin', true, roles(['USER']), 'USER')).toEqual({ kind: 'home', toastKey: 'common.auth.needAdmin' });
    expect(authDecision('/admin', 'admin', true, roles(['ADMIN']), 'ADMIN')).toEqual({ kind: 'pass' });
    expect(authDecision('/studio', 'studio', true, roles(['PARTNER']), 'PARTNER')).toEqual({ kind: 'pass' });
    expect(authDecision('/studio', 'studio', true, roles(['USER']), 'USER')).toEqual({ kind: 'home', toastKey: 'common.auth.needPartner' });
    expect(authDecision('/business/console', 'console', true, roles(['USER']), 'USER')).toEqual({
      kind: 'home',
      toastKey: 'common.auth.needBusiness',
    });
  });
});
