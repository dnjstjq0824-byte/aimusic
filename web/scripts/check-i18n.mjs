// i18n gate: Hangul hardcoding (R13: global enforce, zero allowlist),
// key parity, ICU syntax, backend error-code coverage.
// Unused keys are warnings.
// Usage: node scripts/check-i18n.mjs
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..');
const APP = path.join(ROOT, 'web', 'app');
const LOCALES = path.join(ROOT, 'web', 'i18n', 'locales');
const BACKEND = path.join(ROOT, 'backend');

const walk = (d, ext) => {
  const out = [];
  for (const f of fs.readdirSync(d, { withFileTypes: true })) {
    const p = path.join(d, f.name);
    if (f.isDirectory()) { out.push(...walk(p, ext)); continue; }
    if (ext.some((e) => f.name.endsWith(e))) out.push(p);
  }
  return out;
};

let fail = 0;
// 1. Hangul in app source (tests + dev showcase + allowlist excluded).
for (const f of walk(APP, ['.vue', '.ts'])) {
  const rel = path.relative(ROOT, f).replace(/\\/g, '/');
  if (rel.endsWith('.test.ts') || rel.includes('/dev/')) continue;
  const t = fs.readFileSync(f, 'utf8');
  // Strip comments to avoid flagging explanations.
  const code = t
    .replace(/<!--[\s\S]*?-->/g, '')
    .replace(/\/\/[^\n]*/g, '')
    .replace(/\/\*[\s\S]*?\*\//g, '')
    // Language self-names are locale-invariant by design (never translated).
    .replace(/한국어|日本語|简体中文|ไทย/g, '');
  const m = code.match(/[\uAC00-\uD7A3]/);
  if (m) {
    console.log(`HANGUL ${rel}`);
    fail++;
  }
}

// 2. Key parity across locales (merged namespaces).
const flat = (o, p = '', acc = {}) => {
  for (const [k, v] of Object.entries(o)) {
    const key = p ? `${p}.${k}` : k;
    if (v && typeof v === 'object' && !Array.isArray(v)) flat(v, key, acc);
    else acc[key] = v;
  }
  return acc;
};
const deepMerge = (a, b) => {
  const o = { ...a };
  for (const [k, v] of Object.entries(b)) {
    o[k] = v && typeof v === 'object' && !Array.isArray(v) && a[k] && typeof a[k] === 'object' ? deepMerge(a[k], v) : v;
  }
  return o;
};
const localeCodes = fs.readdirSync(LOCALES).filter((d) => fs.statSync(path.join(LOCALES, d)).isDirectory());
const merged = {};
for (const loc of localeCodes) {
  let m = {};
  for (const f of fs.readdirSync(path.join(LOCALES, loc)).filter((x) => x.endsWith('.json'))) {
    m = deepMerge(m, JSON.parse(fs.readFileSync(path.join(LOCALES, loc, f), 'utf8')));
  }
  merged[loc] = flat(m);
}
const base = new Set(Object.keys(merged.ko ?? {}));
// R11: the admin console ships ko/en only (R-2); admin.* keys are required
// in ko+en and skipped elsewhere. The admin layout forces ko/en at runtime.
const adminExempt = (loc, k) => k.startsWith('admin.') && loc !== 'ko' && loc !== 'en';
for (const loc of localeCodes) {
  if (loc === 'ko') continue;
  const keys = new Set(Object.keys(merged[loc]));
  for (const k of base) {
    if (!keys.has(k) && !adminExempt(loc, k)) { console.log(`PARITY missing ${loc}: ${k}`); fail++; }
  }
  for (const k of keys) {
    if (!base.has(k) && !adminExempt(loc, k)) { console.log(`PARITY extra ${loc}: ${k}`); fail++; }
  }
}

// 3. ICU syntax: balanced braces, simple placeholder names.
for (const loc of localeCodes) {
  for (const [k, v] of Object.entries(merged[loc])) {
    if (typeof v !== 'string') continue;
    const open = (v.match(/\{/g) || []).length;
    const close = (v.match(/\}/g) || []).length;
    if (open !== close) { console.log(`ICU unbalanced ${loc}: ${k}`); fail++; continue; }
    for (const m of v.match(/\{([^}]*)\}/g) || []) {
      if (m === "{'@'}") continue; // vue-i18n literal @ (R3: email placeholders)
      if (m === "{'|'}") continue; // vue-i18n literal | (S2: title pipes are not plurals)
      if (!/^\{[A-Za-z0-9_]+\}$/.test(m)) {
        console.log(`ICU bad placeholder ${loc}: ${k} :: ${m}`);
        fail++;
      }
    }
  }
}

// 4. Backend error codes covered by errors.*.
const codes = new Set(['LOGIN_LOCKED']);
for (const f of walk(BACKEND, ['.go'])) {
  if (f.endsWith('_test.go')) continue;
  const t = fs.readFileSync(f, 'utf8');
  for (const m of t.match(/ERR_[A-Z0-9_]+/g) || []) codes.add(m);
}
for (const code of [...codes].sort()) {
  if (!(`errors.${code}` in (merged.ko ?? {}))) {
    if (code !== 'ERR_FALLBACK') { console.log(`CODES missing errors.${code}`); fail++; }
  }
}

// 5. Unused keys (warning only).
const used = new Set();
for (const f of walk(APP, ['.vue', '.ts'])) {
  if (f.endsWith('.test.ts')) continue;
  const t = fs.readFileSync(f, 'utf8');
  for (const m of t.match(/(?:\bt|\$t|te)\(\s*['"`]([A-Za-z0-9_.]+)['"`]/g) || []) {
    const k = m.replace(/(?:\bt|\$t|te)\(\s*['"`]/, '').replace(/['"`]$/, '');
    used.add(k);
  }
  // dynamic errors.<code> via useApiError covers the whole errors namespace
  if (t.includes('useApiError')) {
    for (const k of Object.keys(merged.ko ?? {})) if (k.startsWith('errors.')) used.add(k);
  }
  // middleware titles use common.titles.* dynamically
  if (f.endsWith('app.global.ts')) {
    for (const k of Object.keys(merged.ko ?? {})) if (k.startsWith('common.titles.')) used.add(k);
  }
}
let unused = 0;
for (const k of Object.keys(merged.ko ?? {})) {
  if (!used.has(k) && !k.startsWith('common.locale.')) { unused++; }
}
if (unused > 0) console.log(`UNUSED-KEYS warn: ${unused} (namespaces fill in later WIs)`);

console.log(fail === 0 ? 'check-i18n PASS' : `check-i18n FAIL (${fail})`);
process.exit(fail === 0 ? 0 : 1);
