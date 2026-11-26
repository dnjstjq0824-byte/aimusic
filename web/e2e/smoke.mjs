// Browser smoke (P1): turns the orphan @playwright/test dependency into a
// real gate. Loads the key public routes against a live stack (CI e2e job),
// asserting status codes, SSR markers, and zero uncaught page errors.
// Browser: system Chrome (channel) on CI; SMOKE_CHROMIUM_PATH locally.
import { chromium } from '@playwright/test';

const BASE = process.env.SMOKE_BASE_URL || 'http://localhost:3000';

const browser = await chromium.launch(
  process.env.SMOKE_CHROMIUM_PATH
    ? { executablePath: process.env.SMOKE_CHROMIUM_PATH }
    : { channel: 'chrome' },
);
const page = await browser.newPage({ viewport: { width: 390, height: 844 } });
const pageErrors = [];
const consoleErrors = [];
page.on('pageerror', (e) => pageErrors.push(String(e).slice(0, 200)));
page.on('console', (m) => {
  if (m.type() === 'error') consoleErrors.push(m.text().slice(0, 200));
});

let failed = 0;
async function check(name, path, marker) {
  const res = await page.goto(BASE + path, { waitUntil: 'domcontentloaded' });
  const status = res?.status() ?? -1;
  const body = await page.content();
  const ok = status === 200 && body.includes(marker);
  console.log(`${ok ? 'PASS' : 'FAIL'} ${name}: status=${status} marker=${body.includes(marker)}`);
  if (!ok) failed++;
}

await check('home SSR', '/', '__NUXT');
await check('locale nav', '/en', '/en/explore');
await check('explore SSR payload', '/explore', 'explore-initial');

console.log(`console errors: ${consoleErrors.length}`);
for (const c of consoleErrors.slice(0, 5)) console.log(`  console: ${c}`);
console.log(`page errors: ${pageErrors.length}`);
for (const e of pageErrors.slice(0, 5)) console.log(`  page: ${e}`);
if (pageErrors.length > 0) {
  console.error('FAIL: uncaught page errors');
  failed++;
}
await browser.close();
if (failed > 0) process.exit(1);
console.log('smoke PASS');
