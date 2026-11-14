// Screen harness main loop: viewports x roles x routes -> screenshots + metrics JSON.
// Usage: node e2e/screens/setup.mjs && node e2e/screens/run.mjs [outDir]
// Defaults to docs/handoff/20261114/evidence/baseline/.
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium } from '@playwright/test';
import { WEB, VIEWPORTS, MATRIX } from './config.mjs';
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
const fill = (p) => p.replace('{track}', auth.ids.track).replace('{playlist}', auth.ids.playlist).replace('{cert}', auth.ids.cert);

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
for (const [vk, vp] of Object.entries(VIEWPORTS)) {
  for (const [role, routes] of Object.entries(MATRIX)) {
    const ctx = await browser.newContext({
      viewport: { width: vp.width, height: vp.height },
      deviceScaleFactor: 1,
      isMobile: vk === 'm390',
      hasTouch: vp.touch,
    });
    const creds = auth.roles[role];
    if (creds?.accessToken) {
      await ctx.addInitScript(
        ({ a, r }) => {
          localStorage.setItem('avibe.accessToken', a);
          localStorage.setItem('avibe.refreshToken', r);
        },
        { a: creds.accessToken, r: creds.refreshToken },
      );
    }
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
      const urlPath = fill(raw);
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
      const shot = `${vk}-${role}-${name}.png`;
      await page.screenshot({ path: path.join(OUT, shot), fullPage: true });
      results.push({
        vp: vk, role, name, path: urlPath, status, ms, shot,
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
await browser.close();
fs.writeFileSync(path.join(OUT, 'baseline.json'), JSON.stringify(results, null, 1));
const screens = results.map((r) => `${r.vp} ${r.role} ${r.name} ${r.path}`);
fs.writeFileSync(path.join(OUT, 'screens.txt'), screens.join('\n') + '\n');
// Summary table
for (const r of results) {
  console.log(
    `${r.vp} ${r.role.padEnd(12)} ${r.name.padEnd(22)} ${String(r.status).padEnd(4)} ${String(r.ms).padStart(5)}ms ` +
      `url=${r.url} hScroll=${r.hScroll} s44=${r.small44}/${r.clickables} t24=${r.tiny24} tiny=${r.tinyText} ` +
      `unlabeled=${r.unlabeled} noAlt=${r.imgsNoAlt} lowC=${r.lowContrast} lcp=${r.lcp} err=${r.errors.length} bad=${r.badResponses.length}`,
  );
  for (const e of r.errors) console.log(`   ERR: ${e}`);
  for (const b of r.badResponses) console.log(`   BAD: ${b}`);
}
const withErr = results.filter((r) => r.errors.length || r.badResponses.length).length;
console.log(`\ndone: ${results.length} screens, ${withErr} with errors/bad-responses -> ${OUT}`);
process.exit(0);
