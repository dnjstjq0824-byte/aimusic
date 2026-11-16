// Installs the app-locale translator for plain TS (stores, api) so
// errMessageOf/tKey resolve keys in the active locale (R13). Components
// keep using useI18n/useApiError directly.
import { setKeyTranslator } from '@/api/client';

export default defineNuxtPlugin((nuxtApp) => {
  // NOTE: useI18n() is setup-only; in plugins use nuxtApp.$i18n (cf. app.global.ts).
  const i18n = nuxtApp.$i18n as {
    te: (key: string) => boolean;
    t: (key: string, params?: Record<string, string | number>) => unknown;
  };
  setKeyTranslator((key, params) => (i18n.te(key) ? String(i18n.t(key, params ?? {})) : null));
});
