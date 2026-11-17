// Self-contained in-page metrics collector (serialized into the page; no imports).
export async function collectMetrics() {
  const de = document.documentElement;
  const vis = (el) => {
    const r = el.getBoundingClientRect();
    const s = getComputedStyle(el);
    return r.width > 0 && r.height > 0 && s.visibility !== 'hidden' && s.display !== 'none';
  };
  const clickables = [...document.querySelectorAll('a,button,[role=button],input,select,textarea')].filter(vis);
  const box = (el) => el.getBoundingClientRect();
  const small44 = clickables.filter((el) => {
    const r = box(el);
    return (r.width < 44 || r.height < 44) && el.type !== 'hidden';
  });
  const tiny24 = clickables.filter((el) => {
    const r = box(el);
    return (r.width < 24 || r.height < 24) && el.type !== 'hidden';
  });
  const unlabeled = [...document.querySelectorAll('button,a')].filter(vis).filter(
    (el) => !(el.innerText || '').trim() && !el.getAttribute('aria-label') && !el.getAttribute('title'),
  );
  const imgsNoAlt = [...document.querySelectorAll('img')].filter((i) => !i.hasAttribute('alt')).length;
  const overflowers = [...document.querySelectorAll('body *')]
    .filter(vis)
    .filter((el) => box(el).right > window.innerWidth + 1)
    .slice(0, 5)
    .map((el) => `${el.tagName}.${String(el.className).slice(0, 60)}`);
  const textNodes = [...document.querySelectorAll('body *')].filter(
    (el) => vis(el) && el.childNodes.length && [...el.childNodes].some((n) => n.nodeType === 3 && n.textContent.trim()),
  );
  const tiny = textNodes.filter((el) => parseFloat(getComputedStyle(el).fontSize) < 12);
  // Contrast probe: sample up to 40 text elements, compute ratio vs opaque background.
  const lum = (c) => {
    const m = c.match(/rgba?\(([^)]+)\)/);
    if (!m) return null;
    const [r, g, b] = m[1].split(',').slice(0, 3).map((v) => parseFloat(v) / 255);
    const f = (v) => (v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4));
    return 0.2126 * f(r) + 0.7152 * f(g) + 0.0722 * f(b);
  };
  const bg = (el) => {
    let n = el;
    while (n && n !== document.body) {
      const c = getComputedStyle(n).backgroundColor;
      if (c && !c.includes('rgba(0, 0, 0, 0)') && !c.endsWith(', 0)')) return c;
      n = n.parentElement;
    }
    return getComputedStyle(document.body).backgroundColor || 'rgb(0,0,0)';
  };
  let lowContrast = 0;
  for (const el of textNodes.slice(0, 60)) {
    const fg = getComputedStyle(el).color;
    const l1 = lum(fg);
    const l2 = lum(bg(el));
    if (l1 === null || l2 === null) continue;
    const ratio = (Math.max(l1, l2) + 0.05) / (Math.min(l1, l2) + 0.05);
    const size = parseFloat(getComputedStyle(el).fontSize);
    if (ratio < (size >= 18 ? 3.0 : 4.5)) lowContrast++;
  }
  const lcp = (() => {
    if (typeof window.__lcp === 'number') return window.__lcp;
    try {
      const es = performance.getEntriesByType('largest-contentful-paint');
      return es.length ? Math.round(es[es.length - 1].startTime) : null;
    } catch {
      return null;
    }
  })();
  return {
    url: location.pathname + location.search,
    title: document.title,
    lang: de.lang,
    hScroll: de.scrollWidth > window.innerWidth,
    scrollW: de.scrollWidth,
    pageH: de.scrollHeight,
    clickables: clickables.length,
    small44: small44.length,
    small44sample: small44.slice(0, 5).map((el) => `${el.tagName}.${String(el.className).slice(0, 40)}`),
    tiny24: tiny24.length,
    tinyText: tiny.length,
    tinySample: tiny.slice(0, 5).map((el) => `${el.tagName}:${(el.innerText || '').trim().slice(0, 30)}`),
    unlabeled: unlabeled.length,
    imgsNoAlt,
    overflowers,
    lowContrast,
    h1: [...document.querySelectorAll('h1')].map((h) => h.innerText.trim().slice(0, 60)),
    lcp,
  };
}
