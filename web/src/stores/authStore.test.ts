import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useAuthStore } from './authStore';
import { authApi } from '@/api/endpoints';
import { tokenStore } from '@/api/client';

vi.mock('@/api/endpoints', () => ({
  authApi: {
    me: vi.fn(),
    login: vi.fn(),
    confirmAge: vi.fn(),
  },
  legalApi: {
    pendingConsents: vi.fn(),
    consent: vi.fn(),
  },
}));

const meMock = vi.mocked(authApi.me);
const loginMock = vi.mocked(authApi.login);
const confirmAgeMock = vi.mocked(authApi.confirmAge);

describe('useAuthStore', () => {
  beforeEach(() => {
    localStorage.clear();
    vi.resetAllMocks();
    setActivePinia(createPinia());
  });

  it('hydrate without tokens only marks initialized', async () => {
    const store = useAuthStore();
    await store.hydrate();
    expect(store.initialized).toBe(true);
    expect(store.isAuthenticated).toBe(false);
    expect(meMock).not.toHaveBeenCalled();
  });

  it('hydrate resolves the principal from /me', async () => {
    tokenStore.set('a', 'r');
    meMock.mockResolvedValue({ userId: 'u-1', role: 'PARTNER' } as never);
    const store = useAuthStore();
    await store.hydrate();
    expect(store.userId).toBe('u-1');
    expect(store.isAuthenticated).toBe(true);
    expect(store.isPartner).toBe(true);
    expect(store.isAdmin).toBe(false);
  });

  it('hydrate clears tokens when /me fails', async () => {
    tokenStore.set('a', 'r');
    meMock.mockRejectedValue(new Error('expired'));
    const store = useAuthStore();
    await store.hydrate();
    expect(tokenStore.getAccess()).toBeNull();
    expect(store.isAuthenticated).toBe(false);
    expect(store.initialized).toBe(true);
  });

  it('login stores tokens and the user', async () => {
    loginMock.mockResolvedValue({
      tokens: { accessToken: 'a', refreshToken: 'r' },
      user: { id: 'u-9', role: 'ADMIN' },
    } as never);
    const store = useAuthStore();
    await store.login('a@x.io', 'password-1');
    expect(tokenStore.getAccess()).toBe('a');
    expect(store.userId).toBe('u-9');
    expect(store.isAdmin).toBe(true);
  });

  it('loginWithTokens resolves role from /me, not the redirect', async () => {
    meMock.mockResolvedValue({ userId: 'u-2', role: 'USER' } as never);
    const store = useAuthStore();
    await store.loginWithTokens('oa', 'or');
    expect(store.userId).toBe('u-2');
    expect(store.role).toBe('USER');
    expect(store.isPartner).toBe(false);
  });

  it('loginWithTokens clears tokens and throws when /me fails', async () => {
    meMock.mockRejectedValue(new Error('nope'));
    const store = useAuthStore();
    await expect(store.loginWithTokens('oa', 'or')).rejects.toThrow('소셜 로그인 세션');
    expect(tokenStore.getAccess()).toBeNull();
    expect(store.userId).toBeNull();
  });

  it('signs out in the UI when tokens are wiped elsewhere (failed refresh)', async () => {
    tokenStore.set('a', 'r');
    meMock.mockResolvedValue({ userId: 'u-1', role: 'USER' } as never);
    const store = useAuthStore();
    await store.hydrate();
    expect(store.isAuthenticated).toBe(true);

    tokenStore.clear(); // what the axios interceptor does when refresh fails

    expect(store.isAuthenticated).toBe(false);
    expect(store.userId).toBeNull();
  });

  it('flags social accounts missing the 14+ declaration', async () => {
    tokenStore.set('a', 'r');
    meMock.mockResolvedValue({ userId: 'u-1', role: 'USER', ageConfirmedAt: null } as never);
    const store = useAuthStore();
    await store.hydrate();
    expect(store.needsAgeConfirm).toBe(true);

    confirmAgeMock.mockResolvedValue({ confirmed: true } as never);
    await store.confirmAge(1995);
    expect(confirmAgeMock).toHaveBeenCalledWith(true, 1995);
    expect(store.needsAgeConfirm).toBe(false);
  });

  it('skips the age gate when already confirmed', async () => {
    tokenStore.set('a', 'r');
    meMock.mockResolvedValue({ userId: 'u-1', role: 'USER', ageConfirmedAt: '2026-01-01T00:00:00Z' } as never);
    const store = useAuthStore();
    await store.hydrate();
    expect(store.needsAgeConfirm).toBe(false);
  });

  it('loads pending terms consents on hydrate and clears them on consent', async () => {
    const { legalApi } = await import('@/api/endpoints');
    const pendingMock = vi.mocked(legalApi.pendingConsents);
    const consentMock = vi.mocked(legalApi.consent);
    tokenStore.set('a', 'r');
    meMock.mockResolvedValue({ userId: 'u-1', role: 'USER', ageConfirmedAt: '2026-01-01' } as never);
    pendingMock.mockResolvedValue([{ id: 't-1', code: 'TOS', version: 'v2' }] as never);
    const store = useAuthStore();
    await store.hydrate();
    expect(store.needsTermsConsent).toBe(true);
    consentMock.mockResolvedValue({ consented: true } as never);
    await store.consentTerms('t-1');
    expect(consentMock).toHaveBeenCalledWith('t-1');
    expect(store.needsTermsConsent).toBe(false);
  });

  it('logout resets everything', async () => {
    tokenStore.set('a', 'r');
    meMock.mockResolvedValue({ userId: 'u-1', role: 'USER' } as never);
    const store = useAuthStore();
    await store.hydrate();
    store.logout();
    expect(tokenStore.getAccess()).toBeNull();
    expect(store.userId).toBeNull();
    expect(store.role).toBeNull();
    expect(store.isAuthenticated).toBe(false);
  });
});
