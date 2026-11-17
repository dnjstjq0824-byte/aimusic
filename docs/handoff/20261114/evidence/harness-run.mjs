import { chromium } from 'playwright-core';
import fs from 'node:fs';

const BASE = process.env.BASE || 'https://avibemusic.com';
const OUT = new URL('./shots/', import.meta.url).pathname.replace(/^\/(\w:)/, '$1');
fs.mkdirSync(OUT, { recursive: true });
const exe = process.env.LOCALAPPDATA + '/ms-playwright/chromium-1254/chrome-win64/chrome.exe';

const routes = [
  ['home', '/'],
  ['explore', '/explore'],
  ['pricing', '/pricing'],
  ['playlists', '/playlists'],
  ['login', '/login'],
  ['register', '/login?tab=register'],
  ['partner-register', '/login?tab=partner'],
  ['reset', '/reset-password'],
  ['legal', '/legal'],
  ['legal-tos', '/legal/TOS'],
  ['cert-invalid', '/certificate/AVB-0000'],
  ['store-player', '/store/player'],
  ['notfound', '/no-such-page'],
  ['library-guard', '/my-library'],
  ['partner-guard', '/partner'],
];
const viewports = { d: { width: 1440, height: 900 }, m: { width: 390, height: 844 } };

const browser = await chromium.launch({ executablePath: exe });
const results = [];
for (const [vk, vp] of Object.entries(viewports)) {
  const ctx = await browser.newContext({ viewport: vp, deviceScaleFactor: 1, isMobile: vk === 'm', hasTouch: vk === 'm' });
  const page = await ctx.newPage();
  let trackPath = null;
  const list = [...routes];
  for (let i = 0; i < list.length; i++) {
    const [name, path] = list[i];
    const errors = [];
    const onErr = (m) => { if (m.type() === 'error') errors.push(m.text().slice(0, 160)); };
    const onPageErr = (e) => errors.push('pageerror: ' + String(e).slice(0, 160));
    const failed = [];
    const onFail = (r) => failed.push(r.url().slice(0, 120));
    page.on('console', onErr); page.on('pageerror', onPageErr); page.on('requestfailed', onFail);
    const t0 = Date.now();
    let status = null;
    try {
      const resp = await page.goto(BASE + path, { waitUntil: 'networkidle', timeout: 30000 });
      status = resp?.status();
    } catch (e) { errors.push('goto: ' + String(e).slice(0, 120)); }
    await page.waitForTimeout(800);
    const ms = Date.now() - t0;
    const info = await page.evaluate(() => {
      const de = document.documentElement;
      const vis = (el) => { const r = el.getBoundingClientRect(); const s = getComputedStyle(el); return r.width > 0 && r.height > 0 && s.visibility !== 'hidden' && s.display !== 'none'; };
      const clickables = [...document.querySelectorAll('a,button,[role=button],input,select')].filter(vis);
      const small = clickables.filter((el) => { const r = el.getBoundingClientRect(); return (r.width < 44 || r.height < 44) && el.type !== 'hidden'; });
      const unlabeled = [...document.querySelectorAll('button,a')].filter(vis).filter((el) => !(el.innerText || '').trim() && !el.getAttribute('aria-label') && !el.getAttribute('title'));
      const imgsNoAlt = [...document.querySelectorAll('img')].filter((i) => !i.hasAttribute('alt')).length;
      const overflowers = [...document.querySelectorAll('body *')].filter(vis).filter((el) => el.getBoundingClientRect().right > window.innerWidth + 1).slice(0, 5).map((el) => el.tagName + '.' + String(el.className).slice(0, 60));
      const h1 = [...document.querySelectorAll('h1')].map((h) => h.innerText.trim().slice(0, 60));
      const text = document.body.innerText;
      const hangul = (text.match(/[가-힣]+/g) || []).length;
      const fontSizes = new Set([...document.querySelectorAll('body *')].filter(vis).map((el) => getComputedStyle(el).fontSize));
      const tiny = [...document.querySelectorAll('body *')].filter(vis).filter((el) => el.childNodes.length && [...el.childNodes].some((n) => n.nodeType === 3 && n.textContent.trim()) && parseFloat(getComputedStyle(el).fontSize) < 12).length;
      const track = document.querySelector('a[href^="/tracks/"]')?.getAttribute('href') || null;
      return {
        url: location.pathname + location.search, title: document.title, lang: de.lang,
        hScroll: de.scrollWidth > window.innerWidth, scrollW: de.scrollWidth, pageH: de.scrollHeight,
        clickables: clickables.length, smallTargets: small.length, unlabeled: unlabeled.length, imgsNoAlt,
        overflowers, h1, hangulWords: hangul, distinctFontSizes: fontSizes.size, tinyTextNodes: tiny, track,
        buttons: [...document.querySelectorAll('button')].filter(vis).map((b) => (b.innerText || b.getAttribute('aria-label') || '').trim().slice(0, 30)).slice(0, 40),
      };
    });
    if (!trackPath && info.track) { trackPath = info.track; list.push(['track-detail', trackPath]); }
    await page.screenshot({ path: `${OUT}${vk}-${name}.png`, fullPage: true });
    results.push({ vp: vk, name, path, status, ms, errors: errors.slice(0, 6), failed: failed.slice(0, 6), ...info });
    page.off('console', onErr); page.off('pageerror', onPageErr); page.off('requestfailed', onFail);
  }
  await ctx.close();
}
await browser.close();
fs.writeFileSync(OUT + 'results.json', JSON.stringify(results, null, 1));
for (const r of results) {
  console.log(`${r.vp} ${r.name.padEnd(17)} ${String(r.status).padEnd(4)} ${String(r.ms).padStart(5)}ms url=${r.url} lang=${r.lang} hScroll=${r.hScroll} small=${r.smallTargets}/${r.clickables} unlabeled=${r.unlabeled} noAlt=${r.imgsNoAlt} tiny=${r.tinyTextNodes} fonts=${r.distinctFontSizes} err=${r.errors.length} h1=${JSON.stringify(r.h1)}`);
  if (r.errors.length) console.log('   errors:', r.errors.join(' | '));
  if (r.overflowers.length && r.hScroll) console.log('   overflow:', r.overflowers.join(', '));
}
