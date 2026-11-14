export default defineI18nConfig(() => ({
  fallbackLocale: 'en',
  datetimeFormats: {
    ko: { short: { dateStyle: 'medium' }, long: { dateStyle: 'full', timeStyle: 'short' } },
    en: { short: { dateStyle: 'medium' }, long: { dateStyle: 'full', timeStyle: 'short' } },
    ja: { short: { dateStyle: 'medium' }, long: { dateStyle: 'full', timeStyle: 'short' } },
    'zh-cn': { short: { dateStyle: 'medium' }, long: { dateStyle: 'full', timeStyle: 'short' } },
    th: { short: { dateStyle: 'medium' }, long: { dateStyle: 'full', timeStyle: 'short' } },
  },
  numberFormats: {
    ko: { currency: { style: 'currency', currency: 'KRW' }, decimal: { style: 'decimal' } },
    en: { currency: { style: 'currency', currency: 'KRW' }, decimal: { style: 'decimal' } },
    ja: { currency: { style: 'currency', currency: 'KRW' }, decimal: { style: 'decimal' } },
    'zh-cn': { currency: { style: 'currency', currency: 'KRW' }, decimal: { style: 'decimal' } },
    th: { currency: { style: 'currency', currency: 'KRW' }, decimal: { style: 'decimal' } },
  },
}));
