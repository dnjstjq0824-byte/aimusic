import { createHttpClient, isHttpError, type HttpClient, type HttpError, type HttpRequestConfig } from './http';
import type { ApiErrorBody } from './types';

// URIs match backend/internal/handler/router.go (/api/v1/...) exactly.
export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '/api/v1';

// Store-safe translation bridge (R13): plain TS (stores, api) cannot call
// useI18n, so a Nuxt plugin installs the app-locale translator at startup
// (web/app/plugins/i18n-keys.ts). Outside the app (unit tests) the English
// fallbacks below apply.
export type KeyTranslator = (key: string, params?: Record<string, string | number>) => string | null;

let keyTranslator: KeyTranslator | null = null;

export function setKeyTranslator(fn: KeyTranslator | null): void {
  keyTranslator = fn;
}

export function tKey(key: string, fallback: string, params?: Record<string, string | number>): string {
  const hit = keyTranslator?.(key, params);
  if (hit) return hit;
  if (!params) return fallback;
  return fallback.replace(/\{(\w+)\}/g, (_, name: string) => String(params[name] ?? ''));
}

// English fallbacks mirroring errors.en.json (stable backend ERR codes from
// middleware.go AbortError + domain GateError). The app resolves errors.*
// in the active locale first via the translator above.
export const ERR_MESSAGES: Record<string, string> = {
  ERR_INVALID_INPUT: 'Please check your input.',
  ERR_UNAUTHORIZED: 'Please log in.',
  ERR_FORBIDDEN: "You don't have permission to do that.",
  ERR_NOT_FOUND: 'The requested item was not found.',
  ERR_EMAIL_TAKEN: 'This email is already registered.',
  ERR_INVALID_CREDENTIALS: 'The email or password is incorrect.',
  ERR_USER_INACTIVE: 'This account is deactivated.',
  ERR_NOT_STREAMABLE: "This track can't be played. (review/blocked status)",
  ERR_RATE_LIMITED: 'Too many requests. Please try again later.',
  ERR_AGE_CONFIRM_REQUIRED: 'Age confirmation (14+) is required.',
  ERR_CONSENT_REQUIRED: 'Required terms consent is missing.',
  ERR_TOTP_REQUIRED: 'A 2-step verification code is required.',
  ERR_TOTP_INVALID: 'The code is incorrect. Please try again.',
  LOGIN_LOCKED: 'Too many login attempts. Please try again in 15 minutes.',
  ERR_NON_COMMERCIAL_TIER: 'Not a commercial plan. Please register commercial subscription proof.',
  ERR_RETROACTIVE_GENERATION_DATE: "Tracks generated before your subscription started can't be registered.",
  ERR_SYNC_ONLY_LICENSE: "This provider's license is sync-only and can't be streamed standalone.",
  ERR_DUPLICATE_AUDIO_FILE: 'This audio file is already registered (duplicate).',
  ERR_PROVIDER_NOT_ALLOWED: 'This provider is not allowlisted. Please pick an allowed partner provider.',
  ERR_PAYMENT_UNAVAILABLE: "Payments aren't set up yet. Please contact support.",
  ERR_ACTIVE_PASS_EXISTS: 'You already have an active pass. Buy again after it expires, or cancel it immediately in your library first.',
  ERR_TOTP_ENROLL_REQUIRED: 'Admin accounts must enroll 2-step verification. Please enroll in security settings.',
  ERR_OAUTH_LINK_LOGIN_REQUIRED: 'This email is already registered. Sign in with email first, then link the social account under Connected accounts.',
  ERR_OAUTH_LINK_TAKEN: 'This social account is already linked to another account.',
  ERR_OAUTH_LAST_METHOD: "You can't unlink your last sign-in method.",
  EMAIL_NOT_VERIFIED: 'Email verification is required. Open the verification link in your inbox.',
  ERR_INTERNAL: 'A server error occurred.',
  ERR_FALLBACK: 'The request failed.',
};

export function errCodeOf(err: unknown): string {
  if (isHttpError<ApiErrorBody>(err)) {
    const body = err.response?.data;
    if (body?.error?.code) return body.error.code;
  }
  return 'ERR_INTERNAL';
}

export function errMessageOf(err: unknown, fallback?: string): string {
  const code = errCodeOf(err);
  if (code in ERR_MESSAGES) return tKey(`errors.${code}`, ERR_MESSAGES[code] as string);
  if (isHttpError<ApiErrorBody>(err)) {
    const msg = err.response?.data?.error?.message;
    if (msg) return msg;
  }
  if (fallback !== undefined) return fallback;
  return tKey('errors.ERR_FALLBACK', ERR_MESSAGES.ERR_FALLBACK as string);
}

const ACCESS_KEY = 'avibe.accessToken';
const REFRESH_KEY = 'avibe.refreshToken';

/** Fired on window whenever stored tokens change, so UI auth state can follow. */
export const AUTH_TOKENS_EVENT = 'avibe:auth-tokens';

/** Fired when a refresh-token rotation fails (revoked/expired): the user must sign in again. */
export const SESSION_EXPIRED_EVENT = 'avibe:session-expired';

function announceTokens(): void {
  if (typeof window !== 'undefined') window.dispatchEvent(new Event(AUTH_TOKENS_EVENT));
}

const hasStorage = () => typeof localStorage !== 'undefined';

export const tokenStore = {
  getAccess(): string | null {
    return hasStorage() ? localStorage.getItem(ACCESS_KEY) : null;
  },
  getRefresh(): string | null {
    return hasStorage() ? localStorage.getItem(REFRESH_KEY) : null;
  },
  set(access: string, refresh: string): void {
    if (!hasStorage()) return;
    localStorage.setItem(ACCESS_KEY, access);
    localStorage.setItem(REFRESH_KEY, refresh);
    announceTokens();
  },
  clear(): void {
    if (!hasStorage()) return;
    localStorage.removeItem(ACCESS_KEY);
    localStorage.removeItem(REFRESH_KEY);
    announceTokens();
  },
};

let refreshPromise: Promise<string> | null = null;

async function refreshAccessToken(client: HttpClient): Promise<string> {
  if (!refreshPromise) {
    const refreshToken = tokenStore.getRefresh();
    refreshPromise = client
      .post<{ data: { tokens: { accessToken: string; refreshToken: string } } }>('/auth/refresh', {
        refreshToken,
      })
      .then((res) => {
        const t = res.data.data.tokens;
        tokenStore.set(t.accessToken, t.refreshToken);
        return t.accessToken;
      })
      .finally(() => {
        refreshPromise = null;
      });
  }
  return refreshPromise;
}

export function createApiClient(): HttpClient {
  return createHttpClient({
    baseURL: API_BASE_URL,
    timeout: 20000,
    onRequest(config) {
      const token = tokenStore.getAccess();
      if (token) (config.headers ??= {}).Authorization = `Bearer ${token}`;
    },
    async onError(err: HttpError, client) {
      const original: HttpRequestConfig = err.config;
      if (
        err.response?.status === 401 &&
        !original._retried &&
        !original.url?.includes('/auth/refresh') &&
        !original.url?.includes('/auth/login') &&
        tokenStore.getRefresh()
      ) {
        original._retried = true;
        let access: string | null = null;
        try {
          access = await refreshAccessToken(client);
        } catch {
          tokenStore.clear();
          if (typeof window !== 'undefined') window.dispatchEvent(new Event(SESSION_EXPIRED_EVENT));
        }
        // onRequest re-reads the rotated access token from tokenStore.
        if (access) return client.request(original);
      }
      // S-03: admins without a second factor are sent to enroll it.
      const code = (err.response?.data as ApiErrorBody | undefined)?.error?.code;
      if (code === 'ERR_TOTP_ENROLL_REQUIRED' && typeof window !== 'undefined' &&
          !window.location.pathname.startsWith('/account/security')) {
        const back = encodeURIComponent(window.location.pathname + window.location.search);
        window.location.assign('/account/security?enroll=admin&redirect=' + back);
      }
      throw err;
    },
  });
}

export const api = createApiClient();

const DEVICE_KEY = 'avibe.deviceToken';

/** Store player device token (separate from user JWTs; no refresh flow). */
export const deviceTokenStore = {
  get(): string | null {
    return hasStorage() ? localStorage.getItem(DEVICE_KEY) : null;
  },
  set(token: string): void {
    if (hasStorage()) localStorage.setItem(DEVICE_KEY, token);
  },
  clear(): void {
    if (hasStorage()) localStorage.removeItem(DEVICE_KEY);
  },
};

/** Device client: Bearer [REDACTED] device token, no user-JWT refresh. A 401 means re-claim. */
export function createDeviceClient(): HttpClient {
  return createHttpClient({
    baseURL: API_BASE_URL,
    timeout: 20000,
    onRequest(config) {
      const token = deviceTokenStore.get();
      if (token) (config.headers ??= {}).Authorization = `Bearer ${token}`;
    },
  });
}

export const deviceApiClient = createDeviceClient();

export async function unwrap<T>(p: Promise<{ data: { data: T } }>): Promise<T> {
  const res = await p;
  return res.data.data;
}
