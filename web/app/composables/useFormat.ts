import { useI18n } from 'vue-i18n';
import { intlLocale } from '@/utils/locale';

/** Locale-aware date/number/currency formatting (Intl). Replaces direct toLocaleString calls. */
export function useFormat() {
  const { locale } = useI18n();
  const tag = () => intlLocale(locale.value);

  function date(v: string | number | Date, style: 'short' | 'long' = 'short'): string {
    const d = v instanceof Date ? v : new Date(v);
    if (Number.isNaN(d.getTime())) return '';
    return new Intl.DateTimeFormat(tag(), style === 'long'
      ? { dateStyle: 'full', timeStyle: 'short' }
      : { dateStyle: 'medium' }).format(d);
  }

  function number(v: number): string {
    return new Intl.NumberFormat(tag(), { style: 'decimal' }).format(v);
  }

  /** KRW amount rendered in the user's locale format (single-currency phase). */
  function currencyKRW(v: number): string {
    return new Intl.NumberFormat(tag(), { style: 'currency', currency: 'KRW' }).format(v);
  }

  return { date, number, currencyKRW };
}
