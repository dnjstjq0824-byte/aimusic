import axios, { AxiosError, type AxiosInstance, type InternalAxiosRequestConfig } from 'axios';
import type { ApiErrorBody } from './types';

// URIs match backend/internal/handler/router.go (/api/v1/...) exactly.
export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '/api/v1';

// Stable backend ERR codes (middleware.go AbortError + domain GateError).
export const ERR_MESSAGES: Record<string, string> = {
  ERR_INVALID_INPUT: '입력값을 확인해 주세요.',
  ERR_UNAUTHORIZED: '로그인이 필요합니다.',
  ERR_FORBIDDEN: '접근 권한이 없습니다.',
  ERR_NOT_FOUND: '요청한 리소스를 찾을 수 없습니다.',
  ERR_EMAIL_TAKEN: '이미 등록된 이메일입니다.',
  ERR_INVALID_CREDENTIALS: '이메일 또는 비밀번호가 올바르지 않습니다.',
  ERR_USER_INACTIVE: '비활성화된 계정입니다.',
  ERR_NOT_STREAMABLE: '재생할 수 없는 음원입니다. (심사/차단 상태)',
  ERR_RATE_LIMITED: '요청이 너무 많습니다. 잠시 후 다시 시도해 주세요.',
  ERR_AGE_CONFIRM_REQUIRED: '만 14세 이상 확인이 필요합니다.',
  ERR_CONSENT_REQUIRED: '필수 약관 동의가 필요합니다.',
  ERR_TOTP_REQUIRED: '2단계 인증 코드가 필요합니다.',
  ERR_TOTP_INVALID: '인증 코드가 올바르지 않습니다. 다시 입력해 주세요.',
  LOGIN_LOCKED: '로그인 시도가 너무 많습니다. 15분 후 다시 시도해 주세요.',
  ERR_NON_COMMERCIAL_TIER: '상업용 플랜이 아닙니다. 상업용 구독 증빙을 등록해 주세요.',
  ERR_RETROACTIVE_GENERATION_DATE: '구독 시작일 이전에 생성된 음원은 등록할 수 없습니다.',
  ERR_SYNC_ONLY_LICENSE: '이 제공자는 단독 스트리밍이 불가한 Sync-Only 라이선스입니다.',
  ERR_DUPLICATE_AUDIO_FILE: '이미 등록된 음원 파일(SHA-256 중복)입니다.',
  ERR_PROVIDER_NOT_ALLOWED: '허용 목록에 없는 제공자입니다. 허용 제휴 제공자를 선택해 주세요.',
  ERR_PAYMENT_UNAVAILABLE: '결제 모듈이 아직 설정되지 않았습니다. 관리자에게 문의해 주세요.',
  ERR_INTERNAL: '서버 오류가 발생했습니다.',
};

export function errCodeOf(err: unknown): string {
  if (axios.isAxiosError(err)) {
    const body = (err as AxiosError<ApiErrorBody>).response?.data;
    if (body?.error?.code) return body.error.code;
  }
  return 'ERR_INTERNAL';
}

export function errMessageOf(err: unknown, fallback = '요청 처리에 실패했습니다.'): string {
  const code = errCodeOf(err);
  if (code in ERR_MESSAGES) return ERR_MESSAGES[code] as string;
  if (axios.isAxiosError(err)) {
    const msg = (err as AxiosError<ApiErrorBody>).response?.data?.error?.message;
    if (msg) return msg;
  }
  return fallback;
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

export const tokenStore = {
  getAccess(): string | null {
    return localStorage.getItem(ACCESS_KEY);
  },
  getRefresh(): string | null {
    return localStorage.getItem(REFRESH_KEY);
  },
  set(access: string, refresh: string): void {
    localStorage.setItem(ACCESS_KEY, access);
    localStorage.setItem(REFRESH_KEY, refresh);
    announceTokens();
  },
  clear(): void {
    localStorage.removeItem(ACCESS_KEY);
    localStorage.removeItem(REFRESH_KEY);
    announceTokens();
  },
};

let refreshPromise: Promise<string> | null = null;

async function refreshAccessToken(client: AxiosInstance): Promise<string> {
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

export function createApiClient(): AxiosInstance {
  const client = axios.create({ baseURL: API_BASE_URL, timeout: 20000 });

  client.interceptors.request.use((config: InternalAxiosRequestConfig) => {
    const token = tokenStore.getAccess();
    if (token) config.headers.set('Authorization', `Bearer ${token}`);
    return config;
  });

  client.interceptors.response.use(
    (res) => res,
    async (err: AxiosError) => {
      const original = err.config as (InternalAxiosRequestConfig & { _retried?: boolean }) | undefined;
      if (
        err.response?.status === 401 &&
        original &&
        !original._retried &&
        !original.url?.includes('/auth/refresh') &&
        !original.url?.includes('/auth/login') &&
        tokenStore.getRefresh()
      ) {
        original._retried = true;
        try {
          const access = await refreshAccessToken(client);
          original.headers.set('Authorization', `Bearer ${access}`);
          return client.request(original);
        } catch {
          tokenStore.clear();
          if (typeof window !== 'undefined') window.dispatchEvent(new Event(SESSION_EXPIRED_EVENT));
        }
      }
      return Promise.reject(err);
    },
  );

  return client;
}

export const api = createApiClient();

const DEVICE_KEY = 'avibe.deviceToken';

/** Store player device token (separate from user JWTs; no refresh flow). */
export const deviceTokenStore = {
  get(): string | null {
    return localStorage.getItem(DEVICE_KEY);
  },
  set(token: string): void {
    localStorage.setItem(DEVICE_KEY, token);
  },
  clear(): void {
    localStorage.removeItem(DEVICE_KEY);
  },
};

/** Device client: Bearer [REDACTED] device token, no user-JWT refresh. A 401 means re-claim. */
export function createDeviceClient(): AxiosInstance {
  const client = axios.create({ baseURL: API_BASE_URL, timeout: 20000 });
  client.interceptors.request.use((config: InternalAxiosRequestConfig) => {
    const token = deviceTokenStore.get();
    if (token) config.headers.set('Authorization', `Bearer ${token}`);
    return config;
  });
  return client;
}

export const deviceApiClient = createDeviceClient();

export async function unwrap<T>(p: Promise<{ data: { data: T } }>): Promise<T> {
  const res = await p;
  return res.data.data;
}
