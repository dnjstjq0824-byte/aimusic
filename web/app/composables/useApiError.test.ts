import { describe, expect, it, vi } from 'vitest';

const MESSAGES: Record<string, string> = {
  'errors.ERR_INVALID_INPUT': 'bad input (en)',
  'errors.ERR_FALLBACK': 'fallback (en)',
};
vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (k: string) => MESSAGES[k] ?? k,
    te: (k: string) => k in MESSAGES,
  }),
}));

import { useApiError } from './useApiError';

function axiosErr(code: string) {
  return { isAxiosError: true, response: { data: { error: { code, message: 'dev-msg' } } } };
}

describe('useApiError', () => {
  it('translates known codes', () => {
    expect(useApiError()(axiosErr('ERR_INVALID_INPUT'))).toBe('bad input (en)');
  });

  it('falls back for unknown codes without leaking dev messages', () => {
    expect(useApiError()(axiosErr('ERR_SOMETHING_NEW'))).toBe('fallback (en)');
    expect(useApiError()(new Error('boom'))).toBe('fallback (en)');
  });
});
