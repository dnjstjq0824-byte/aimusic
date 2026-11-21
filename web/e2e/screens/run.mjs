// Screen harness main loop: viewports x roles x langs x routes -> screenshots + metrics JSON.
// R13: full matrix = routes x 5 roles x 3 viewports x 5 langs.
// Usage: node e2e/screens/setup.mjs && node e2e/screens/run.mjs [outDir]
// Defaults to docs/handoff/20261114/evidence/baseline/.
// HARNESS_LANGS=ko,en runs a language subset.
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium } from '@playwright/test';
import { WEB, API, VIEWPORTS, MATRIX, LANGS } from './config.mjs';
import { collectMetrics } from './collect.mjs';

const DIR = path.dirname(fileURLToPath(import.meta.url));
const ROOT = path.resolve(DIR, '../../..');
const AUTH = path.join(DIR, '.auth.json');
const OUT = process.argv[2] || path.join(ROOT, 'docs/handoff/20261114/evidence/baseline');
fs.mkdirSync(OUT, { recursive: true });
if (!fs.existsSync(AUTH)) {
  console.error('missing .auth.json — run setup.mjs first');
  process.exit(1);
}
const auth = JSON.parse(fs.readFileSync(AUTH, 'utf8'));
const fill = (p) => p
  .replace('{trackShort}', auth.ids.trackShort)
  .replace('{playlistShort}', auth.ids.playlistShort)
  .replace('{track}', auth.ids.track)
  .replace('{playlist}', auth.ids.playlist)
  .replace('{creator}', auth.ids.creator)
  .replace('{cert}', auth.ids.cert);

// prefix_except_default: ko rides the bare path, others take /{lang}.
const withLang = (p, lang) => {
  if (lang === 'ko') return p;
  const qi = p.indexOf('?');
  const pathOnly = qi === -1 ? p : p.slice(0, qi);
  const q = qi === -1 ? '' : p.slice(qi);
  return `/${lang}${pathOnly === '/' ? '' : pathOnly}${q}`;
};

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

const browser = await chromium.launch({ executablePath: chromeExe() });
const results = [];

// WI-N6: access tokens live 15min but a full run takes hours. Refresh
// each role's tokens when its first context starts (and persist back to
// .auth.json), so late contexts don't land on the login screen.
async function freshCreds(role, creds) {
  if (!creds?.refreshToken) return creds;
  try {
    const res = await fetch(`${API}/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refreshToken: creds.refreshToken }),
    });
    if (!res.ok) return creds;
    const json = await res.json();
    const t = json?.data?.tokens;
    if (!t?.accessToken || !t?.refreshToken) return creds;
    const next = { ...creds, accessToken: t.accessToken, refreshToken: t.refreshToken };
    auth.roles[role] = next;
    fs.writeFileSync(AUTH, JSON.stringify(auth, null, 1));
    return next;
  } catch {
    return creds;
  }
}

for (const [vk, vp] of Object.entries(VIEWPORTS)) {
  for (const [role, routes] of Object.entries(MATRIX)) {
    let creds = auth.roles[role];
    for (const lang of LANGS) {
      creds = await freshCreds(role, creds);
      const ctx = await browser.newContext({
        viewport: { width: vp.width, height: vp.height },
        deviceScaleFactor: 1,
        isMobile: vk === 'm390',
        hasTouch: vp.touch,
        locale: lang === 'zh-cn' ? 'zh-CN' : lang,
      });
      await ctx.addInitScript(
        ({ a, r, l }) => {
          // Stored locale pins private screens (locale-init plugin);
          // public screens resolve from the URL prefix.
          localStorage.setItem('avibe.locale', l);
          if (a) {
            localStorage.setItem('avibe.accessToken', a);
            localStorage.setItem('avibe.refreshToken', r);
          }
        },
        { a: creds?.accessToken ?? null, r: creds?.refreshToken ?? null, l: lang },
      );
      const page = await ctx.newPage();
      await page.addInitScript(() => {
        window.__lcp = null;
        try {
          new PerformanceObserver((list) => {
            const es = list.getEntries();
            if (es.length) window.__lcp = Math.round(es[es.length - 1].startTime);
          }).observe({ type: 'largest-contentful-paint', buffered: true });
        } catch {
          /* ignore */
        }
      });
      for (const [name, raw] of routes) {
        const urlPath = withLang(fill(raw), lang);
        const consoleErrs = [];
        const badResponses = [];
        const failed = [];
        page.on('console', (m) => {
          if (m.type() === 'error') consoleErrs.push(m.text().slice(0, 200));
        });
        page.on('pageerror', (e) => consoleErrs.push('pageerror: ' + String(e).slice(0, 200)));
        page.on('response', (r) => {
          if (r.status() >= 400) badResponses.push(`${r.status()} ${r.url().slice(0, 140)}`);
        });
        page.on('requestfailed', (r) => failed.push(r.url().slice(0, 140)));
        const t0 = Date.now();
        let status = null;
        try {
          const resp = await page.goto(WEB + urlPath, { waitUntil: 'networkidle', timeout: 30000 });
          status = resp?.status() ?? null;
        } catch (e) {
          consoleErrs.push('goto: ' + String(e).slice(0, 160));
        }
        await page.waitForTimeout(900);
        const ms = Date.now() - t0;
        let info = {};
        try {
          info = await page.evaluate(collectMetrics);
        } catch (e) {
          consoleErrs.push('collect: ' + String(e).slice(0, 160));
        }
        const shot = `${vk}-${role}-${lang}-${name}.png`;
        await page.screenshot({ path: path.join(OUT, shot), fullPage: true });
        results.push({
          vp: vk, role, lang, name, path: urlPath, status, ms, shot,
          errors: consoleErrs.slice(0, 8),
          badResponses: badResponses.slice(0, 8),
          failed: failed.slice(0, 6),
          ...info,
        });
        page.removeAllListeners();
      }
      await ctx.close();
    }
  }
}
await browser.close();
fs.writeFileSync(path.join(OUT, 'baseline.json'), JSON.stringify(results, null, 1));
const screens = results.map((r) => `${r.vp} ${r.role} ${r.lang} ${r.name} ${r.path}`);
fs.writeFileSync(path.join(OUT, 'screens.txt'), screens.join('\n') + '\n');
// Summary table
for (const r of results) {
  console.log(
    `${r.vp} ${r.role.padEnd(12)} ${r.lang.padEnd(6)} ${r.name.padEnd(22)} ${String(r.status).padEnd(4)} ${String(r.ms).padStart(5)}ms ` +
      `url=${r.url} hScroll=${r.hScroll} s44=${r.small44}/${r.clickables} t24=${r.tiny24} tiny=${r.tinyText} ` +
      `unlabeled=${r.unlabeled} noAlt=${r.imgsNoAlt} lowC=${r.lowContrast} lcp=${r.lcp} err=${r.errors.length} bad=${r.badResponses.length}`,
  );
  for (const e of r.errors) console.log(`   ERR: ${e}`);
  for (const b of r.badResponses) console.log(`   BAD: ${b}`);
}
const withErr = results.filter((r) => r.errors.length || r.badResponses.length).length;
console.log(`\ndone: ${results.length} screens, ${withErr} with errors/bad-responses -> ${OUT}`);
process.exit(0);
