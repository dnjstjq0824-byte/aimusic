import { useI18n } from 'vue-i18n';
import { errCodeOf } from '@/api/client';

/** Translate API errors by code (R2). Components use this; stores keep errMessageOf (ko fallback). */
export function useApiError() {
  const { t, te } = useI18n();
  return (err: unknown): string => {
    const key = `errors.${errCodeOf(err)}`;
    return te(key) ? String(t(key)) : String(t('errors.ERR_FALLBACK'));
  };
}
