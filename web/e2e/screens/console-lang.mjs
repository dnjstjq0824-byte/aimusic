// WI-N6 console language scenario: switch locale INSIDE the 3 consoles
// via the footer language menu (consoles have no locale-prefixed routes by
// design, R-3) and assert zero raw translation keys in the body text.
// Also asserts admin falls back to en when ja is picked (ko/en only).
// Usage: node e2e/screens/setup.mjs && node e2e/screens/console-lang.mjs
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium } from '@playwright/test';
import { WEB, API } from './config.mjs';

const DIR = path.dirname(fileURLToPath(import.meta.url));
const AUTH = path.join(DIR, '.auth.json');
if (!fs.existsSync(AUTH)) {
  console.error('missing .auth.json — run setup.mjs first');
  process.exit(1);
}
const auth = JSON.parse(fs.readFileSync(AUTH, 'utf8'));

// Raw key shape: admin.* / studio.* / business.* dotted keys.
const KEY_RE = /\b(admin|studio|business)\.[A-Za-z0-9_.]+/g;

function chromeExe() {
  const base = path.join(process.env.LOCALAPPDATA || '', 'ms-playwright');
  if (!fs.existsSync(base)) return undefined;
  for (const d of fs.readdirSync(base)) {
    if (!d.startsWith('chromium-')) continue;
    for (const sub of ['chrome-win/chrome.exe', 'chrome-win64/chrome.exe']) {
      const p = path.join(base, d, sub);
      if (fs.existsSync(p)) return p;
    }
  }
  return undefined;
}

// [role, console path (unprefixed, no locale routes exist)]
const CASES = [
  ['creator', '/studio'],
  ['business', '/business/console'],
  ['admin', '/admin'],
];
const LANG_NAMES = { en: 'English', ja: '日本語', th: 'ไทย', 'zh-cn': '简体中文' };

const browser = await chromium.launch({ executablePath: chromeExe() });
let pass = 0;
let fail = 0;
const failures = [];

async function freshCreds(creds) {
  if (!creds?.refreshToken) return creds;
  try {
    const res = await fetch(`${API}/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refreshToken: creds.refreshToken }),
    });
    if (!res.ok) return creds;
    const t = (await res.json())?.data?.tokens;
    if (!t?.accessToken || !t?.refreshToken) return creds;
    return { ...creds, accessToken: t.accessToken, refreshToken: t.refreshToken };
  } catch {
    return creds;
  }
}

for (const [role, base] of CASES) {
  let creds = auth.roles[role];
  if (!creds?.accessToken) {
    console.log(`SKIP ${role}: no creds`);
    continue;
  }
  creds = await freshCreds(creds);
  const ctx = await browser.newContext({ locale: 'ko-KR' });
  await ctx.addInitScript(
    ({ a, r }) => {
      localStorage.setItem('avibe.accessToken', a);
      localStorage.setItem('avibe.refreshToken', r);
    },
    { a: creds.accessToken, r: creds.refreshToken },
  );
  const page = await ctx.newPage();
  await page.goto(`${WEB}${base}`, { waitUntil: 'networkidle', timeout: 30000 }).catch(() => null);
  await page.waitForTimeout(1500);
  for (const lang of Object.keys(LANG_NAMES)) {
    // Open the footer language menu and pick the language.
    const menu = page.locator('button[aria-haspopup="listbox"]').first();
    try {
      await menu.click({ timeout: 8000 });
      await page.getByRole('option', { name: LANG_NAMES[lang] }).first().click({ timeout: 8000 });
    } catch {
      fail++;
      failures.push(`${role}/${lang}: language menu not operable`);
      console.log(`FAIL ${role}/${lang}: menu not operable`);
      continue;
    }
    await page.waitForTimeout(1500);
    const body = await page.evaluate(() => document.body?.innerText ?? '');
    const keys = [...new Set(body.match(KEY_RE) || [])];
    if (role === 'admin') {
      // ko/en only: ja capture would stay broken-locale; only en+ko render.
      const htmlLang = await page.evaluate(() => document.documentElement.lang);
      const wantEn = htmlLang === 'en' || lang === 'en';
      if (!wantEn && lang === 'ja') {
        // ja pick must fall back to en content (layouts/admin.vue).
        if (htmlLang !== 'en') {
          fail++;
          failures.push(`${role}/${lang}: html lang=${htmlLang}, want en fallback`);
          console.log(`FAIL ${role}/${lang}: html lang=${htmlLang}`);
          continue;
        }
      }
    }
    if (keys.length > 0) {
      fail++;
      failures.push(`${role}/${lang}: ${keys.length} raw keys (${keys.slice(0, 5).join(', ')})`);
      console.log(`FAIL ${role}/${lang}: ${keys.length} raw keys`);
    } else {
      pass++;
      console.log(`PASS ${role}/${lang}: 0 raw keys`);
    }
  }
  await ctx.close();
}
await browser.close();
console.log(`\nconsole-lang: PASS=${pass} FAIL=${fail}`);
for (const f of failures) console.log('  ' + f);
process.exit(fail === 0 ? 0 : 1);
