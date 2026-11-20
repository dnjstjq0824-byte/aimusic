// Console-only i18n namespaces load on demand (bundle diet, 2026-11-21).
// admin / studio-console / business-console messages were ~40% of every
// page's messages.json; public pages never use them. The global middleware
// awaits ensureConsoleMessages() before a console route renders, and
// ConsoleShell re-runs it when the locale changes inside a console.

type Messages = Record<string, unknown>;

/** Minimal surface of the vue-i18n global composer we need. */
export interface ConsoleI18n {
  mergeLocaleMessage: (locale: string, messages: Messages) => void;
}

const loaders = import.meta.glob<{ default: Messages }>(
  '../../i18n/locales/*/{admin,studio-console,business-console}.json',
);

// Locale code -> folder name (codes are lowercase, folders keep region case).
const folderOf = (code: string): string => (code === 'zh-cn' ? 'zh-CN' : code);

// Per i18n instance (one per SSR request / one in the browser).
const loaded = new WeakMap<object, Map<string, Promise<void>>>();

const CONSOLE_PREFIXES = ['/studio', '/admin', '/business/console'];

/** True for console paths (locale prefix already stripped). */
export function isConsolePath(path: string): boolean {
  return CONSOLE_PREFIXES.some((p) => path === p || path.startsWith(`${p}/`));
}

export function ensureConsoleMessages(i18n: ConsoleI18n, locale: string): Promise<void> {
  let perInstance = loaded.get(i18n);
  if (!perInstance) {
    perInstance = new Map();
    loaded.set(i18n, perInstance);
  }
  const hit = perInstance.get(locale);
  if (hit) return hit;

  const dir = `/locales/${folderOf(locale)}/`;
  const jobs = Object.entries(loaders)
    .filter(([file]) => file.includes(dir))
    .map(([, load]) => load().then((mod) => i18n.mergeLocaleMessage(locale, mod.default)));
  const done = Promise.all(jobs).then(
    () => undefined,
    (err: unknown) => {
      perInstance!.delete(locale); // allow a retry on the next navigation
      throw err;
    },
  );
  perInstance.set(locale, done);
  return done;
}
