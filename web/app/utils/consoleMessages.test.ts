import { describe, expect, it } from 'vitest';
import { ensureConsoleMessages, isConsolePath, type ConsoleI18n } from './consoleMessages';

function fakeI18n() {
  const merged: Record<string, Record<string, unknown>[]> = {};
  const i18n: ConsoleI18n = {
    mergeLocaleMessage(locale, messages) {
      (merged[locale] ??= []).push(messages);
    },
  };
  return { i18n, merged };
}

const topKeys = (list: Record<string, unknown>[] = []) =>
  list.flatMap((m) => Object.entries(m).flatMap(([ns, v]) => Object.keys(v as object).map((k) => `${ns}.${k}`))).sort();

describe('isConsolePath', () => {
  it('matches the three consoles and nothing public', () => {
    expect(isConsolePath('/studio')).toBe(true);
    expect(isConsolePath('/studio/upload')).toBe(true);
    expect(isConsolePath('/admin/users')).toBe(true);
    expect(isConsolePath('/business/console/billing')).toBe(true);
    expect(isConsolePath('/business')).toBe(false);
    expect(isConsolePath('/studios')).toBe(false);
    expect(isConsolePath('/')).toBe(false);
  });
});

describe('ensureConsoleMessages', () => {
  it('merges admin, studio-console and business-console for the locale', async () => {
    const { i18n, merged } = fakeI18n();
    await ensureConsoleMessages(i18n, 'ko');
    const keys = topKeys(merged.ko);
    expect(keys).toContain('admin.nav');
    expect(keys).toContain('studio.dash');
    expect(keys).toContain('business.console');
    // Public studio keys stay in the eager studio.json.
    expect(keys).not.toContain('studio.status');
  });

  it('maps zh-cn to the zh-CN folder', async () => {
    const { i18n, merged } = fakeI18n();
    await ensureConsoleMessages(i18n, 'zh-cn');
    expect(topKeys(merged['zh-cn'])).toContain('studio.dash');
  });

  it('loads each locale once per i18n instance', async () => {
    const { i18n, merged } = fakeI18n();
    await ensureConsoleMessages(i18n, 'en');
    const first = merged.en?.length ?? 0;
    await ensureConsoleMessages(i18n, 'en');
    expect(merged.en?.length).toBe(first);
    expect(first).toBeGreaterThan(0);
  });
});
