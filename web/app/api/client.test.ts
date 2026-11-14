import { AxiosError, AxiosHeaders } from 'axios';
import { beforeEach, describe, expect, it } from 'vitest';
import { ERR_MESSAGES, errCodeOf, errMessageOf, tokenStore, unwrap } from './client';
import type { ApiErrorBody } from './types';

function axiosErrorWith(code: string, message: string): AxiosError<ApiErrorBody> {
  return new AxiosError<ApiErrorBody>(
    message,
    'ERR_BAD_REQUEST',
    undefined,
    undefined,
    {
      status: 400,
      statusText: 'Bad Request',
      headers: {},
      config: { headers: new AxiosHeaders() },
      data: { error: { code, message } },
    },
  );
}

describe('errCodeOf', () => {
  it('extracts the backend error code', () => {
    expect(errCodeOf(axiosErrorWith('ERR_EMAIL_TAKEN', 'taken'))).toBe('ERR_EMAIL_TAKEN');
  });

  it('falls back to ERR_INTERNAL for non-API errors', () => {
    expect(errCodeOf(new Error('boom'))).toBe('ERR_INTERNAL');
    expect(errCodeOf(new AxiosError('network down'))).toBe('ERR_INTERNAL');
  });
});

describe('errMessageOf', () => {
  it('maps known codes to Korean copy', () => {
    expect(errMessageOf(axiosErrorWith('ERR_INVALID_CREDENTIALS', 'bad'))).toBe(
      '이메일 또는 비밀번호가 올바르지 않습니다.',
    );
  });

  it('passes through unknown backend messages', () => {
    expect(errMessageOf(axiosErrorWith('ERR_SOME_NEW_CODE', 'fresh message'))).toBe('fresh message');
  });

  it('maps opaque errors to the generic server message', () => {
    expect(errMessageOf(new Error('boom'))).toBe('서버 오류가 발생했습니다.');
  });

  it('uses the fallback for unknown codes without a backend message', () => {
    expect(errMessageOf(axiosErrorWith('ERR_SOME_NEW_CODE', ''))).toBe('요청 처리에 실패했습니다.');
    expect(errMessageOf(axiosErrorWith('ERR_SOME_NEW_CODE', ''), 'custom')).toBe('custom');
  });

  it('covers every code the client claims to translate', () => {
    for (const code of Object.keys(ERR_MESSAGES)) {
      expect(errMessageOf(axiosErrorWith(code, 'x'))).toBe(ERR_MESSAGES[code]);
    }
  });
});

describe('tokenStore', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('round-trips tokens', () => {
    expect(tokenStore.getAccess()).toBeNull();
    tokenStore.set('a1', 'r1');
    expect(tokenStore.getAccess()).toBe('a1');
    expect(tokenStore.getRefresh()).toBe('r1');
    tokenStore.clear();
    expect(tokenStore.getAccess()).toBeNull();
    expect(tokenStore.getRefresh()).toBeNull();
  });
});

describe('unwrap', () => {
  it('extracts nested data payloads', async () => {
    await expect(unwrap(Promise.resolve({ data: { data: { id: 't-1' } } }))).resolves.toEqual({
      id: 't-1',
    });
  });
});
