// Global route middleware: layout shell + auth/role guards + tab title (WI-R2 i18n).
import { computed } from 'vue';
import { useAuthStore } from '@/stores/authStore';
import { useToast } from '@/composables/useToast';
import { safeRedirect } from '@/utils/redirect';
import { authDecision, layoutFor, redirectFor } from '@/utils/routeGuards';
import { shouldApplyServerLocale } from '@/utils/locale';
import { ensureConsoleMessages, isConsolePath, type ConsoleI18n } from '@/utils/consoleMessages';
import { stripLocalePrefix } from '@/utils/seo';

/** Strip the i18n route-name suffix (home___en -> home). */
export function baseRouteName(name: unknown): string {
  return String(name ?? '').replace(/___.+$/, '');
}

export default defineNuxtRouteMiddleware(async (to) => {
  const name = baseRouteName(to.name);
  const rule = redirectFor({
    path: to.path,
    name,
    queryTab: to.query.tab,
    queryToken: to.query.token,
    fullPath: to.fullPath,
  });
  if (rule) {
    return navigateTo(rule.to, { redirectCode: rule.code });
  }

  // [...slug] catch-all renders bare.
  const layout = name === 'not-found' ? false : layoutFor(to.path);
  if (layout === false) setPageLayout(false);
  else setPageLayout(layout as 'app' | 'studio' | 'business' | 'admin' | 'auth' | 'embed');

  const localePath = useLocalePath();
  const auth = useAuthStore();
  if (!auth.initialized && import.meta.client) {
    await auth.hydrate();
    // A2: a fresh principal applies the server locale (detection order ①).
    const { $i18n } = useNuxtApp();
    const apply = shouldApplyServerLocale(auth.locale, $i18n.locale.value, to.path, name);
    if (apply) await $i18n.setLocale(apply);
  }
  const decision = authDecision(to.path, name, auth.isAuthenticated, (r) => auth.hasRole(r), auth.role);
  if (decision.kind === 'guest-redirect') {
    const redirect = safeRedirect(to.query.redirect);
    return redirect === to.fullPath ? localePath({ name: 'home' }) : redirect;
  }
  if (decision.kind === 'login') {
    return { name: 'login', query: { redirect: to.fullPath } };
  }
  if (decision.kind === 'home') {
    if (import.meta.client) {
      const { $i18n } = useNuxtApp();
      useToast().warning($i18n.t(decision.toastKey));
    }
    return localePath({ name: 'home' });
  }

  const { $i18n } = useNuxtApp();
  // Bundle diet: console-only namespaces load before a console route renders.
  if (isConsolePath(stripLocalePrefix(to.path))) {
    await ensureConsoleMessages($i18n as unknown as ConsoleI18n, $i18n.locale.value).catch(() => {
      // A failed chunk leaves raw keys visible; the next navigation retries.
    });
  }
  const key = `common.titles.${name}`;
  // Reactive: client-side locale switches (after this middleware runs) refresh title + lang.
  const pageTitle = computed(() => ($i18n.te(key) ? `${$i18n.t(key)} · AVIBE` : 'AVIBE — Beyond Your Music'));
  const htmlLang = computed(() => ($i18n.locale.value === 'zh-cn' ? 'zh-CN' : $i18n.locale.value));
  useHead({ htmlAttrs: { lang: htmlLang }, title: pageTitle });
});
