// Console-only messages for a locale switch inside a console (footer language
// menu, admin's ko/en enforcement). nuxt-i18n awaits this hook before the
// locale flips, so the new locale's console set is merged before re-render.
// (Merging after the flip does not re-render: vue-i18n's mergeLocaleMessage
// mutates messages in place.) Route entry is covered by app.global.ts.
import { ensureConsoleMessages, isConsolePath, type ConsoleI18n } from '@/utils/consoleMessages';
import { stripLocalePrefix } from '@/utils/seo';

export default defineNuxtPlugin((nuxtApp) => {
  nuxtApp.hook('i18n:beforeLocaleSwitch', async ({ newLocale }) => {
    const path = stripLocalePrefix(nuxtApp._route?.path ?? window.location.pathname);
    if (!isConsolePath(path)) return;
    await ensureConsoleMessages(nuxtApp.$i18n as unknown as ConsoleI18n, newLocale).catch(() => {
      // Raw keys stay visible on failure; the next navigation retries.
    });
  });
});
